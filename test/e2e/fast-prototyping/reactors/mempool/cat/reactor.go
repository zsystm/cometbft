package cat

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/crypto/tmhash"
	"github.com/cometbft/cometbft/libs/clist"
	"github.com/cometbft/cometbft/libs/log"
	cmtsync "github.com/cometbft/cometbft/libs/sync"
	"github.com/cometbft/cometbft/mempool"
	"github.com/cometbft/cometbft/p2p"
	protomem "github.com/cometbft/cometbft/proto/tendermint/mempool"
	"github.com/cometbft/cometbft/types"
)

const (
	// default duration to wait before considering a peer non-responsive
	// and searching for the tx from a new peer
	defaultGossipDelay = 200 * time.Millisecond

	// Content Addressable Tx Pool gossips state based messages (SeenTx and WantTx) on a separate channel
	// for cross compatibility
	MempoolStateChannel = byte(0x31)

	// peerHeightDiff signifies the tolerance in difference in height between the peer and the height
	// the node received the tx
	peerHeightDiff = 10
)

// Reactor handles mempool tx broadcasting amongst peers.
// It maintains a map from peer ID to counter, to prevent gossiping txs to the
// peers you received it from.
type Reactor struct {
	p2p.BaseReactor
	config   *cfg.MempoolConfig
	mempool  *mempool.CListMempool
	requests *requestScheduler

	// Thread-safe list of transactions peers have seen that we have not yet seen
	seenByPeersSet *SeenTxSet

	// `txSenders` maps every received transaction to the set of peer IDs that
	// have sent the transaction to this node. Sender IDs are used during
	// transaction propagation to avoid sending a transaction to a peer that
	// already has it. A sender ID is the internal peer ID used in the mempool
	// to identify the sender, storing two bytes with each transaction instead
	// of 20 bytes for the types.NodeID.
	txSenders    map[types.TxKey]map[p2p.ID]bool
	txSendersMtx cmtsync.Mutex
}

// NewReactor returns a new Reactor with the given config and mempool.
func NewReactor(config *cfg.MempoolConfig, mempool *mempool.CListMempool) *Reactor {
	memR := &Reactor{
		config:         config,
		mempool:        mempool,
		requests:       newRequestScheduler(defaultGossipDelay, defaultGlobalRequestTimeout),
		seenByPeersSet: NewSeenTxSet(),
		txSenders:      make(map[types.TxKey]map[p2p.ID]bool),
	}
	memR.BaseReactor = *p2p.NewBaseReactor("Mempool", memR)
	memR.mempool.SetTxRemovedCallback(func(txKey types.TxKey) {
		memR.removeSenders(txKey)
		memR.seenByPeersSet.RemoveKey(txKey)
	})
	return memR
}

// InitPeer implements Reactor by creating a state for the peer.
func (memR *Reactor) InitPeer(peer p2p.Peer) p2p.Peer {
	return peer
}

// SetLogger sets the Logger on the reactor and the underlying mempool.
func (memR *Reactor) SetLogger(l log.Logger) {
	memR.Logger = l
	memR.mempool.SetLogger(l)
}

// OnStart implements p2p.BaseReactor.
func (memR *Reactor) OnStart() error {
	if !memR.config.Broadcast {
		memR.Logger.Info("Tx broadcasting is disabled")
	} else {
		go memR.pushNewTxsRoutine()
	}
	return nil
}

func (memR *Reactor) pushNewTxsRoutine() {
	for {
		select {
		case <-memR.Quit():
			return

		// listen in for any newly verified tx via RFC, then immediately
		// broadcasts it to all connected peers.
		case nextTx := <-memR.mempool.NextNewTx():
			memR.broadcastNewTx(nextTx)
		}
	}
}

// OnStop implements Service
func (memR *Reactor) OnStop() {
	// stop all the timers tracking outbound requests
	memR.requests.Close()
}

// GetChannels implements Reactor by returning the list of channels for this
// reactor.
func (memR *Reactor) GetChannels() []*p2p.ChannelDescriptor {
	largestTx := make([]byte, memR.config.MaxTxBytes)
	batchMsg := protomem.Message{
		Sum: &protomem.Message_Txs{
			Txs: &protomem.Txs{Txs: [][]byte{largestTx}},
		},
	}

	stateMsg := protomem.Message{
		Sum: &protomem.Message_SeenTx{
			SeenTx: &protomem.SeenTx{
				TxKey: make([]byte, tmhash.Size),
			},
		},
	}

	return []*p2p.ChannelDescriptor{
		{
			ID:                  mempool.MempoolChannel,
			Priority:            6,
			RecvMessageCapacity: batchMsg.Size(),
			MessageType:         &protomem.Message{},
		},
		{
			ID:                  MempoolStateChannel,
			Priority:            5,
			RecvMessageCapacity: stateMsg.Size(),
			MessageType:         &protomem.Message{},
		},
	}
}

// AddPeer implements Reactor.
// It starts a broadcast routine ensuring all txs are forwarded to the given peer.
func (memR *Reactor) AddPeer(peer p2p.Peer) {
	if memR.config.Broadcast {
		go memR.broadcastTxRoutine(peer)
	}
}

// RemovePeer implements Reactor. For all current outbound requests to this
// peer it will find a new peer to rerequest the same transactions.
func (memR *Reactor) RemovePeer(peer p2p.Peer, reason interface{}) {
	// remove and rerequest all pending outbound requests to that peer since we know
	// we won't receive any responses from them.
	outboundRequests := memR.requests.ClearAllRequestsFrom(peer.ID())
	for key := range outboundRequests {
		// memR.mempool.metrics.RequestedTxs.Add(1)
		memR.findNewPeerToRequestTx(key)
	}
}

// Receive implements Reactor.
// It processes one of three messages: Txs, SeenTx, WantTx.
func (memR *Reactor) Receive(e p2p.Envelope) {
	memR.Logger.Debug("Receive", "src", e.Src, "chId", e.ChannelID, "msg", e.Message)
	switch msg := e.Message.(type) {

	// A peer has sent us one or more transactions. This could be either because we requested them
	// or because the peer received a new transaction and is broadcasting it to us.
	// NOTE: This setup also means that we can support older mempool implementations that simply
	// flooded the network with transactions.
	case *protomem.Txs:
		protoTxs := msg.GetTxs()
		if len(protoTxs) == 0 {
			memR.Logger.Error("received empty txs from peer", "src", e.Src)
			return
		}

		peerID := e.Src.ID()
		for _, txBytes := range protoTxs {
			tx := types.Tx(txBytes)
			key := tx.Key()

			// If we requested the transaction, we mark it as received.
			if memR.requests.Has(peerID, key) {
				memR.requests.MarkReceived(peerID, key)
				memR.Logger.Debug("received a response for a requested transaction", "peerID", peerID, "txKey", key)
			} else {
				// If we didn't request the transaction we simply mark the peer as having the
				// tx (we'd have already done it if we were requesting the tx).
				memR.markPeerHasTx(peerID, key)
				memR.Logger.Debug("received new trasaction", "peerID", peerID, "txKey", key)
			}

			reqRes, err := memR.mempool.CheckTx(tx)
			if errors.Is(err, mempool.ErrTxInCache) {
				memR.Logger.Debug("Tx already exists in cache", "tx", tx.String())
			} else if err != nil {
				memR.Logger.Info("Could not check tx", "tx", tx.String(), "err", err)
			} else {
				// Record the sender only when the transaction is valid and, as
				// a consequence, added to the mempool. Senders are stored until
				// the transaction is removed from the mempool. Note that it's
				// possible a tx is still in the cache but no longer in the
				// mempool. For example, after committing a block, txs are
				// removed from mempool but not the cache.
				reqRes.SetCallback(func(res *abci.Response) {
					if res.GetCheckTx().Code == abci.CodeTypeOK {
						memR.addSender(tx.Key(), e.Src.ID())
					}
				})
			}
			if memR.config.Broadcast {
				// We broadcast only transactions that we deem valid and actually have in our mempool.
				memR.broadcastSeenTx(key)
			}
		}

	// A peer has indicated to us that it has a transaction. We first verify the txkey and
	// mark that peer as having the transaction. Then we proceed with the following logic:
	//
	// 1. If we have the transaction, we do nothing.
	// 2. If we don't yet have the tx but have an outgoing request for it, we do nothing.
	// 3. If we recently evicted the tx and still don't have space for it, we do nothing.
	// 4. Else, we request the transaction from that peer.
	case *protomem.SeenTx:
		txKey, err := types.TxKeyFromBytes(msg.TxKey)
		if err != nil {
			memR.Logger.Error("peer sent SeenTx with incorrect tx key", "err", err)
			memR.Switch.StopPeerForError(e.Src, err)
			return
		}
		peerID := e.Src.ID()
		memR.markPeerHasTx(peerID, txKey)

		// Check if we don't already have the transaction and that it was recently rejected
		if memR.mempool.InMempool(txKey) || memR.mempool.InCache(txKey) || memR.mempool.WasRejected(txKey) {
			memR.Logger.Debug("received a seen tx for a tx we already have", "txKey", txKey)
			return
		}

		// If we are already requesting that tx, then we don't need to go any further.
		if _, exists := memR.requests.ForTx(txKey); exists {
			memR.Logger.Debug("received a SeenTx message for a transaction we are already requesting", "txKey", txKey)
			return
		}

		// We don't have the transaction, nor are we requesting it so we send the node
		// a want msg
		memR.requestTx(txKey, e.Src.ID())

	// A peer is requesting a transaction that we have claimed to have. Find the specified
	// transaction and broadcast it to the peer. We may no longer have the transaction
	case *protomem.WantTx:
		txKey, err := types.TxKeyFromBytes(msg.TxKey)
		if err != nil {
			memR.Logger.Error("peer sent WantTx with incorrect tx key", "err", err)
			memR.Switch.StopPeerForError(e.Src, err)
			return
		}
		elem, has := memR.mempool.GetCElement(txKey)
		if has && memR.config.Broadcast {
			tx := elem.Value.(*mempool.MempoolTx).GetTx()
			peerID := e.Src.ID()
			memR.Logger.Debug("sending a tx in response to a want msg", "peer", peerID)
			txsMsg := p2p.Envelope{ChannelID: mempool.MempoolChannel, Message: &protomem.Txs{Txs: [][]byte{tx}}}
			if e.Src.Send(txsMsg) {
				memR.markPeerHasTx(peerID, txKey)
			}
		}

	default:
		memR.Logger.Error("unknown message type", "src", e.Src, "chId", e.ChannelID, "msg", e.Message)
		memR.Switch.StopPeerForError(e.Src, fmt.Errorf("mempool cannot handle message of type: %T", e.Message))
		return
	}
}

// PeerHasTx marks that the transaction has been seen by a peer.
func (memR *Reactor) markPeerHasTx(peer p2p.ID, txKey types.TxKey) {
	memR.Logger.Debug("peer has tx", "peer", peer, "txKey", fmt.Sprintf("%X", txKey))
	memR.seenByPeersSet.Add(txKey, peer)
}

// PeerState describes the state of a peer.
type PeerState interface {
	GetHeight() int64
}

// Send new mempool txs to peer.
func (memR *Reactor) broadcastTxRoutine(peer p2p.Peer) {
	var next *clist.CElement

	for {
		// In case of both next.NextWaitChan() and peer.Quit() are variable at the same time
		if !memR.IsRunning() || !peer.IsRunning() {
			return
		}
		// This happens because the CElement we were looking at got garbage
		// collected (removed). That is, .NextWait() returned nil. Go ahead and
		// start from the beginning.
		if next == nil {
			select {
			case <-memR.mempool.TxsWaitChan(): // Wait until a tx is available
				if next = memR.mempool.TxsFront(); next == nil {
					continue
				}
			case <-peer.Quit():
				return
			case <-memR.Quit():
				return
			}
		}

		// Make sure the peer is up to date.
		peerState, ok := peer.Get(types.PeerStateKey).(PeerState)
		if !ok {
			// Peer does not have a state yet. We set it in the consensus reactor, but
			// when we add peer in Switch, the order we call reactors#AddPeer is
			// different every time due to us using a map. Sometimes other reactors
			// will be initialized before the consensus reactor. We should wait a few
			// milliseconds and retry.
			time.Sleep(mempool.PeerCatchupSleepIntervalMS * time.Millisecond)
			continue
		}

		// If we suspect that the peer is lagging behind, at least by more than
		// one block, we don't send the transaction immediately. This code
		// reduces the mempool size and the recheck-tx rate of the receiving
		// node. See [RFC 103] for an analysis on this optimization.
		//
		// [RFC 103]: https://github.com/cometbft/cometbft/pull/735
		memTx := next.Value.(*mempool.MempoolTx)
		if peerState.GetHeight() < memTx.Height()-1 {
			time.Sleep(mempool.PeerCatchupSleepIntervalMS * time.Millisecond)
			continue
		}

		// NOTE: Transaction batching was disabled due to
		// https://github.com/tendermint/tendermint/issues/5796

		if !memR.isSender(memTx.GetTx().Key(), peer.ID()) {
			success := peer.Send(p2p.Envelope{
				ChannelID: mempool.MempoolChannel,
				Message:   &protomem.Txs{Txs: [][]byte{memTx.GetTx()}},
			})
			if !success {
				time.Sleep(mempool.PeerCatchupSleepIntervalMS * time.Millisecond)
				continue
			}
		}

		select {
		case <-next.NextWaitChan():
			// see the start of the for loop for nil check
			next = next.Next()
		case <-peer.Quit():
			return
		case <-memR.Quit():
			return
		}
	}
}

func (memR *Reactor) isSender(txKey types.TxKey, peerID p2p.ID) bool {
	memR.txSendersMtx.Lock()
	defer memR.txSendersMtx.Unlock()

	sendersSet, ok := memR.txSenders[txKey]
	return ok && sendersSet[peerID]
}

func (memR *Reactor) addSender(txKey types.TxKey, senderID p2p.ID) bool {
	memR.txSendersMtx.Lock()
	defer memR.txSendersMtx.Unlock()

	if sendersSet, ok := memR.txSenders[txKey]; ok {
		sendersSet[senderID] = true
		return false
	}
	memR.txSenders[txKey] = map[p2p.ID]bool{senderID: true}
	return true
}

func (memR *Reactor) removeSenders(txKey types.TxKey) {
	memR.txSendersMtx.Lock()
	defer memR.txSendersMtx.Unlock()

	if memR.txSenders != nil {
		delete(memR.txSenders, txKey)
	}
}

// broadcastSeenTx broadcasts a SeenTx message to all peers unless we
// know they have already seen the transaction
func (memR *Reactor) broadcastSeenTx(txKey types.TxKey) {
	memR.Logger.Debug("broadcasting seen tx to all peers", "tx_key", txKey.String())
	msg := p2p.Envelope{ChannelID: MempoolStateChannel, Message: &protomem.Message{
		Sum: &protomem.Message_SeenTx{
			SeenTx: &protomem.SeenTx{TxKey: txKey[:]},
		},
	}}

	// Add jitter to when the node broadcasts it's seen txs to stagger when nodes
	// in the network broadcast their seenTx messages.
	time.Sleep(time.Duration(rand.Intn(10)*10) * time.Millisecond) //nolint:gosec

	peers := memR.Switch.Peers().List()
	for _, peer := range peers {
		if peerState, ok := peer.Get(types.PeerStateKey).(PeerState); ok {
			// make sure peer isn't too far behind. This can happen
			// if the peer is blocksyncing still and catching up
			// in which case we just skip sending the transaction
			if peerState.GetHeight() < memR.mempool.Height()-peerHeightDiff {
				memR.Logger.Debug("peer is too far behind us. Skipping broadcast of seen tx")
				continue
			}
		}
		// no need to send a seen tx message to a peer that already
		// has that tx.
		if memR.seenByPeersSet.Has(txKey, peer.ID()) {
			continue
		}

		peer.Send(msg)
	}
}

// broadcastNewTx broadcast new transaction to all peers unless we are already
// sure they have seen the tx.
func (memR *Reactor) broadcastNewTx(memTx *mempool.MempoolTx) {
	tx := memTx.GetTx()
	txKey := tx.Key()
	msg := p2p.Envelope{ChannelID: MempoolStateChannel, Message: &protomem.Message{
		Sum: &protomem.Message_Txs{
			Txs: &protomem.Txs{Txs: [][]byte{tx}},
		},
	}}

	peers := memR.Switch.Peers().List()
	for _, peer := range peers {
		if peerState, ok := peer.Get(types.PeerStateKey).(PeerState); ok {
			// make sure peer isn't too far behind. This can happen
			// if the peer is blocksyncing still and catching up
			// in which case we just skip sending the transaction
			if peerState.GetHeight() < memTx.Height()-peerHeightDiff {
				memR.Logger.Debug("peer is too far behind us. Skipping broadcast of seen tx")
				continue
			}
		}

		if memR.seenByPeersSet.Has(txKey, peer.ID()) {
			continue
		}

		if peer.Send(msg) {
			memR.markPeerHasTx(peer.ID(), txKey)
		}
	}
}

// requestTx requests a transaction from a peer and tracks it,
// requesting it from another peer if the first peer does not respond.
func (memR *Reactor) requestTx(txKey types.TxKey, peerID p2p.ID) {
	if !memR.Switch.Peers().Has(peerID) {
		// we have disconnected from the peer
		return
	}
	memR.Logger.Debug("requesting tx", "txKey", txKey, "peerID", peerID)
	peer := memR.Switch.Peers().Get(peerID)
	msg := p2p.Envelope{ChannelID: MempoolStateChannel, Message: &protomem.Message{
		Sum: &protomem.Message_WantTx{
			WantTx: &protomem.WantTx{TxKey: txKey[:]},
		},
	}}
	success := peer.Send(msg)
	if success {
		// memR.mempool.metrics.RequestedTxs.Add(1)
		requested := memR.requests.Add(txKey, peerID, memR.findNewPeerToRequestTx)
		if !requested {
			memR.Logger.Error("have already marked a tx as requested", "txKey", txKey, "peerID", peerID)
		}
	}
}

// findNewPeerToSendTx finds a new peer that has already seen the transaction to
// request a transaction from.
func (memR *Reactor) findNewPeerToRequestTx(txKey types.TxKey) {
	// ensure that we are connected to peers
	if memR.Switch.Peers().Size() == 0 {
		return
	}

	// pop the next peer in the list of remaining peers that have seen the tx
	// and does not already have an outbound request for that tx
	seenMap := memR.seenByPeersSet.Get(txKey)
	var peerID *p2p.ID
	for possiblePeer := range seenMap {
		if !memR.requests.Has(possiblePeer, txKey) {
			peerID = &possiblePeer
			break
		}
	}

	if peerID == nil {
		// No other free peer has the transaction we are looking for.
		// We give up 🤷‍♂️ and hope either a peer responds late or the tx
		// is gossiped again
		memR.Logger.Info("no other peer has the tx we are looking for", "txKey", txKey)
		return
	}
	if !memR.Switch.Peers().Has(*peerID) {
		// we disconnected from that peer, retry again until we exhaust the list
		memR.findNewPeerToRequestTx(txKey)
	} else {
		// memR.mempool.metrics.RerequestedTxs.Add(1)
		memR.requestTx(txKey, *peerID)
	}
}
