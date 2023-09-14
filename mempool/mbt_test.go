package mempool

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/cometbft/cometbft/abci/example/kvstore"
	abci "github.com/cometbft/cometbft/abci/types"
	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/internal/test"
	"github.com/cometbft/cometbft/libs/clist"
	"github.com/cometbft/cometbft/libs/log"
	cmtrand "github.com/cometbft/cometbft/libs/rand"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/p2p/mock"
	protomem "github.com/cometbft/cometbft/proto/tendermint/mempool"
	"github.com/cometbft/cometbft/proxy"
	"github.com/cometbft/cometbft/test/mbt/itf"
	"github.com/cometbft/cometbft/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
)

func TestAllTraces(t *testing.T) {
	// if os.Getenv("MODEL_BASED_TESTING") == "" {
	// 	t.Skip("skipping test; $MODEL_BASED_TESTING not set")
	// }

	dir := tracesDir()
	traces, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("Error reading trace directory: %s", err)
	}

	for _, trace := range traces {
		testTrace(t, dir+"/"+trace.Name(), 1)
	}
}

func TestOneTrace(t *testing.T) {
	testTrace(t, tracesDir()+"/notFullChain_trace-1.itf.json", 1)
}

// txMultiplier will convert each transaction in the model by to txMultiplier
// transactions in the test.
func testTrace(t *testing.T, path string, txMultiplier int) {
	t.Logf("🟡 Testing trace %s (mult=%d)", path, txMultiplier)
	var err error

	// Load trace
	trace := &itf.Trace{}
	if err := trace.LoadFromFile(path); err != nil {
		t.Fatalf("Error loading trace file: %s", err)
	}

	// Model parameters
	// TODO: take these values from the trace (params field, which is currently empty)
	txs := []string{"tx1", "tx2", "tx3", "tx4"}
	nodeIds := []string{"n1", "n2"}
	mempoolMaxSize := 2
	// configs = NodeIds.mapBy(_ => { keepInvalidTxsInCache: false })
	numNodes := len(nodeIds)

	// Create transactions and map them to model values.
	txsMap := make(map[string][]types.Tx, numNodes)
	for i, tx := range txs {
		txs := make([]types.Tx, txMultiplier)
		for j := 0; j < txMultiplier; j++ {
			txs[j] = mkTx(i, j)
		}
		txsMap[tx] = txs

	}

	// Create peers and map them to model values.
	peersMap := make(map[string]p2p.Peer, numNodes)
	for i, id := range nodeIds {
		ip := net.IP{127, 0, 0, byte(i)}
		peersMap[id] = mock.NewPeer(ip)
	}

	// Create reactors and map them to model values.
	// Note that we don't need to connect the nodes; we'll handle messages manually.
	config := cfg.TestConfig()
	config.Mempool.Size = mempoolMaxSize
	config.Mempool.CacheSize = mempoolMaxSize
	reactors := makeReactors(t, config, numNodes)
	reactorsMap := make(map[string]*Reactor, numNodes)
	for i, id := range nodeIds {
		reactorsMap[id] = reactors[i]
	}

	// Execute trace
	for i, state := range trace.States {
		// Get step info
		step, ok := state.VarValues["_step"]
		require.True(t, ok)

		stepMap, ok := step.Value.(itf.MapExprType)
		require.True(t, ok)

		nodeId, ok := stepMap["node"].Value.(string) // node executing the action
		require.True(t, ok)

		stepName, ok := stepMap["name"].Value.(string)
		require.True(t, ok)

		args, ok := stepMap["args"].Value.(itf.MapExprType)
		require.True(t, ok)

		// Get step error
		stepError, ok := state.VarValues["_error"].Value.(string)
		require.True(t, ok)

		if stepName == "init" || nodeId == "no-node" {
			continue
		}

		// Get step arguments
		a_tx, _ := args["tx"].Value.(string)
		a_error, _ := args["error"].Value.(string)
		a_validTxs := args["validTxs"].Value.(itf.ListExprType)
		a_invalidTxs := args["invalidTxs"].Value.(itf.ListExprType)
		a_height, _ := args["height"].Value.(float64)
		a_peerId, _ := args["node"].Value.(string)

		// State
		// caches := state.VarValues["_error"].GetValue().(map[string]*itf.Expr)
		// mempools := state.VarValues["mempool"].GetValue().(map[string]*itf.Expr)
		// heights := state.VarValues["mempoolHeight"].GetValue().(map[string]int)
		// chain := state.VarValues["Chain::chain"].GetValue().([]map[*itf.Expr]struct{})

		reactor := reactorsMap[nodeId]
		mp := reactor.mempool

		switch stepName {
		case "ReceiveTxViaRPC":
			t.Logf("🔵 #%d node=%s step=%s(%s) -> error=\"%s\"\n",
				i, nodeId, stepName, a_tx, stepError)

			// build parameters
			txs, ok := txsMap[a_tx]
			require.True(t, ok)

			// try to add transactions and check results
			switch stepError {
			case "err:tx-in-cache":
				for _, tx := range txs {
					_, err = mp.CheckTx(tx)
					require.True(t, mp.cache.Has(tx))
					require.Equal(t, ErrTxInCache, err)
				}
			case "warn:invalid-tx":
				for _, tx := range txs {
					_, err = mp.CheckTx(tx)
					require.False(t, mp.cache.Has(tx))
					require.False(t, mp.InMempool(tx.Key()))
					require.NoError(t, err)
				}
			case "none":
				for _, tx := range txs {
					_, err = mp.CheckTx(tx)
					require.True(t, mp.cache.Has(tx))
					require.False(t, mp.InMempool(tx.Key()))
					require.NoError(t, err)
				}
			}

		case "ReceiveCheckTxResponse":
			// t.Logf("🔵 #%d node=%s step=%s(%s, responseError:\"%s\") -> error=\"%s\"\n",
			// 	i, nodeId, stepName, a_tx, a_error, stepError)

			// build parameters
			txs, ok := txsMap[a_tx]
			require.True(t, ok)
			res := mkResponse(a_error)

			// process response and check results
			switch stepError {
			case "err:mempool-full":
				for _, tx := range txs {
					mp.resCbFirstTime(tx, res)
					require.True(t, mp.cache.Has(tx))
					require.False(t, mp.InMempool(tx.Key()))
				}
			case "warn:invalid-tx":
				for _, tx := range txs {
					mp.resCbFirstTime(tx, res)
					require.False(t, mp.cache.Has(tx))
					require.False(t, mp.InMempool(tx.Key()))
				}
			case "none":
				for _, tx := range txs {
					mp.resCbFirstTime(tx, res)
					require.True(t, mp.cache.Has(tx))
					require.True(t, mp.InMempool(tx.Key()))
				}
			}

		case "ReceiveRecheckTxResponse":
			t.Logf("🔵 #%d node=%s step=%s(%s, responseError:\"%s\") -> error=\"%s\"\n",
				i, nodeId, stepName, a_tx, a_error, stepError)
			require.Equal(t, "none", stepError)

			// build parameters
			txs, ok := txsMap[a_tx]
			require.True(t, ok)
			res := mkResponse(a_error)

			// build request, process request and response, and check results
			switch a_error {
			case "err:mempool-full":
				for _, tx := range txs {
					req := mkRequest(tx, abci.CheckTxType_Recheck)
					mp.resCbRecheck(req, res)
					require.True(t, mp.cache.Has(tx))
					require.False(t, mp.InMempool(tx.Key()))
				}
			case "err:invalid-tx":
				for _, tx := range txs {
					req := mkRequest(tx, abci.CheckTxType_Recheck)
					mp.resCbRecheck(req, res)
					require.False(t, mp.cache.Has(tx))
					require.False(t, mp.InMempool(tx.Key()))
				}
			case "warn:invalid-tx":
				for _, tx := range txs {
					req := mkRequest(tx, abci.CheckTxType_Recheck)
					mp.resCbRecheck(req, res)
					require.False(t, mp.cache.Has(tx))
					require.False(t, mp.InMempool(tx.Key()))
				}
			case "none":
				for _, tx := range txs {
					req := mkRequest(tx, abci.CheckTxType_Recheck)
					mp.resCbRecheck(req, res)
					require.True(t, mp.cache.Has(tx))
					require.True(t, mp.InMempool(tx.Key()))
				}
			}

		case "Update":
			t.Logf("🔵 #%d node=%s step=%s(%d, valid:%s, invalid:%s) -> error=\"%s\"\n",
				i, nodeId, stepName, int(a_height), a_validTxs, a_invalidTxs, stepError)
			require.Equal(t, "none", stepError)

			// build parameters
			validTxs := make([]string, 0, len(a_validTxs))
			for _, tx := range a_validTxs {
				validTxs = append(validTxs, tx.Value.(string))
			}

			invalidTxs := make([]string, 0, len(a_invalidTxs))
			for _, tx := range a_invalidTxs {
				invalidTxs = append(invalidTxs, tx.Value.(string))
			}

			txs, results := mkTxsResults(txsMap, txMultiplier, validTxs, invalidTxs)

			// update mempool
			err = mp.Update(int64(a_height), txs, results, nil, nil)
			require.NoError(t, err)

			// check results
			for i, tx := range txs {
				if results[i].Code == abci.CodeTypeOK {
					require.True(t, mp.cache.Has(tx))
				} else {
					require.False(t, mp.cache.Has(tx))
				}
				require.False(t, mp.InMempool(tx.Key()))
				for _, tx := range txs {
					require.Equal(t, 0, len(reactor.txSenders[tx.Key()]))
				}
			}

		case "P2P_ReceiveTx":
			t.Logf("🔵 #%d node=%s step=%s(%s) from %s -> error=\"%s\"\n",
				i, nodeId, stepName, a_tx, a_peerId, stepError)

			// build parameters
			peer := peersMap[a_peerId]
			txs, ok := txsMap[a_tx]
			require.True(t, ok)

			// build and receive message; check results
			switch stepError {
			case "none":
				for _, tx := range txs {
					reactor.Receive(p2p.Envelope{Src: peer, Message: mkTxMsg(tx), ChannelID: MempoolChannel})
					require.True(t, mp.cache.Has(tx))
					time.Sleep(100 * time.Millisecond) // wait until CheckTx's reqRes callback is invoked :-(
					require.True(t, reactor.isSender(tx.Key(), peer.ID()), fmt.Sprintf("%s [%s] is not a sender of %s [%s]", a_peerId, peer.ID(), a_tx, tx.Key()))
				}
			case "err:mempool-full":
				for _, tx := range txs {
					reactor.Receive(p2p.Envelope{Src: peer, Message: mkTxMsg(tx), ChannelID: MempoolChannel})
					// nothing to check here; the tx could be or not in the cache
					// TODO: check in the model's state
				}
			case "err:tx-in-cache":
				for _, tx := range txs {
					require.True(t, mp.cache.Has(tx))
					reactor.Receive(p2p.Envelope{Src: peer, Message: mkTxMsg(tx), ChannelID: MempoolChannel})
					require.True(t, mp.cache.Has(tx))
				}
			}

		case "P2P_BroadcastTx":
			t.Logf("🔵 #%d node=%s step=%s(%s) -> error=\"%s\"\n",
				i, nodeId, stepName, a_tx, stepError)

			// build parameters
			txs, ok := txsMap[a_tx]
			require.True(t, ok)

			// broadcast
			for id, peer := range peersMap {
				if id != nodeId {
					for _, tx := range txs {
						reactor.send(tx, peer)
					}
				}
			}

			// nothing to check here
		}
	}
}

func tracesDir() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return wd + "/traces"
}

func mkTx(i int, j int) []byte {
	return []byte(fmt.Sprintf("%d.%d=x", i, j))
}

func (reactor *Reactor) send(tx types.Tx, peer p2p.Peer) {
	if !reactor.isSender(tx.Key(), peer.ID()) {
		peer.Send(p2p.Envelope{
			ChannelID: MempoolChannel,
			Message:   &protomem.Txs{Txs: [][]byte{tx}},
		})
	}
}

// connect N mempool reactors through N switches
func makeReactors(t *testing.T, config *cfg.Config, n int) []*Reactor {
	logger := mempoolLogger()
	reactors := make([]*Reactor, n)
	for i := 0; i < n; i++ {
		sockPath := fmt.Sprintf("unix:///tmp/echo_%v.sock", cmtrand.Str(6))
		app := kvstore.NewInMemoryApplication()
		newRemoteApp(t, sockPath, app) // TODO: defer close created server returns error

		cc := proxy.NewRemoteClientCreator(sockPath, "socket", true)
		cfg := test.ResetTestRoot("mempool_test")
		appConnMem, err := cc.NewABCIClient()
		if err != nil {
			panic(err)
		}
		appConnMem.SetLogger(log.TestingLogger().With("module", "abci-client", "connection", "mempool"))
		if err := appConnMem.Start(); err != nil {
			panic(err)
		}

		// We do not want to SetResponseCallback, so we do not call
		// NewCListMempool.
		mempool := &CListMempool{
			config:        cfg.Mempool,
			proxyAppConn:  appConnMem,
			txs:           clist.New(),
			height:        0,
			recheckCursor: nil,
			recheckEnd:    nil,
			logger:        log.NewNopLogger(),
			metrics:       NopMetrics(),
		}
		mempool.cache = NewLRUTxCache(cfg.Mempool.CacheSize)
		mempool.SetLogger(log.TestingLogger())
		defer func() { os.RemoveAll(cfg.RootDir) }()

		appConnMem.SetResponseCallback(func(*abci.Request, *abci.Response) {})

		config.Mempool.Broadcast = false                  // We will send messages only when the trace commands to.
		reactors[i] = NewReactor(config.Mempool, mempool) // so we dont start the consensus states
		reactors[i].SetLogger(logger.With("validator", i))
	}
	return reactors
}

func mkTxsResults(
	txsMap map[string][]types.Tx,
	txMultiplier int,
	validTxs []string,
	invalidTxs []string,
) ([]types.Tx, []*abci.ExecTxResult) {
	size := (len(validTxs) + len(invalidTxs)) * txMultiplier
	allTxs := make([]types.Tx, 0, size)
	results := make([]*abci.ExecTxResult, 0, size)
	for _, txName := range validTxs {
		for _, tx := range txsMap[txName] {
			allTxs = append(allTxs, tx)
			results = append(results, &abci.ExecTxResult{Code: abci.CodeTypeOK})
		}
	}
	for _, txName := range invalidTxs {
		for _, tx := range txsMap[txName] {
			allTxs = append(allTxs, tx)
			results = append(results, &abci.ExecTxResult{Code: 1})
		}
	}
	return allTxs, results
}

func mkRequest(tx types.Tx, checkType abci.CheckTxType) *abci.Request {
	return abci.ToRequestCheckTx(&abci.RequestCheckTx{Tx: tx, Type: checkType})
}

func mkResponse(respErr string) *abci.Response {
	var re abci.ResponseCheckTx
	if respErr == "none" {
		re = abci.ResponseCheckTx{Code: abci.CodeTypeOK}
	} else {
		re = abci.ResponseCheckTx{Code: 1}
	}
	return abci.ToResponseCheckTx(&re)
}

func mkTxMsg(tx types.Tx) proto.Message {
	txsMsg := protomem.Txs{Txs: [][]byte{tx}}
	msg := txsMsg.Wrap()
	var err error
	if w, ok := msg.(p2p.Unwrapper); ok {
		msg, err = w.Unwrap()
		if err != nil {
			panic(fmt.Errorf("unwrapping message: %s", err))
		}
	}
	return msg
}

// func printCList(txs *clist.CList) {
// 	for el := txs.Front(); el != nil; el = el.Next() {
// 		memTx := el.Value.(*mempoolTx)
// 		fmt.Printf("memTx: %s - %d\n", memTx.tx.Key().String(), memTx.height)
// 	}
// }
