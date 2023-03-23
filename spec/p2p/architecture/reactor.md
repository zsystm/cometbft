# Reactors

Reactor is the generic name for a component that employs the p2p communication layer.

To become a reactor, a component has to:

1. Implement the `Reactor` interface
2. Register with the p2p layer using the `Switch.AddReactor` method

The p2p layer invokes a registered reactor to:

- `Receive` messages sent by peers and destined to the reactor
   - The reactor is responsible for incoming messages on one or more `Channels`
- Add a `Peer` with which the node has established a connection
   - The reactor can use the provided `Peer` handler to send messages to that peer
- Remove a `Peer` from which the node has for some reason been disconnected

The p2p layer supports the operation of multiple reactors that exchange messages
with multiple peers.


## Channels

In order to route incoming messages to the appropriate reactors, the p2p layer
adopts the abstraction of channels.
This feature is implemented by the multiplex connections used by a node to
exchange message with peers.

Each `Channel` has a globally unique ID.
When a reactor registers with the p2p layer, it informs the list of channels 
it is responsible for.
Incoming messages on the registered channels will then be delivered to the reactor.

More specifically, when a reactor is registered the following method of the
`Reactor` interface is invoked:

        // GetChannels returns the list of MConnection.ChannelDescriptor. Make sure
        // that each ID is unique across all the reactors added to the switch.
        GetChannels() []*conn.ChannelDescriptor

The method should return a list of channels the reactor is responsible for.
A `ChannelDescriptor` consists of a global byte `ID` and some local parameters
for the channel, employed by the multiplex connection.

> The `Switch` panics when the same channel ID is registered by two reactors.

### Priorities

TODO


## Receive

The main duty of a reactor is to handle incoming messages on the channels it
has registered with the p2p layer.

When a message is received from any connected `Peer` on any of the channels
registered by the reactor, the node will deliver the message to the reactor
invoking the following method of the `Reactor` interface:


        // Receive is called by the switch when an envelope is received from any connected
        // peer on any of the channels registered by the reactor
        Receive(Envelope)

The reactor receives a message packed into an `Envelope` with the following content:

- `ChannelID`: the channel the message belongs to
- `Src`: the `Peer` from which the message was received
- `Message`: the message's payload, unmarshalled using protocol buffers

> On the `v0.34.x` and `v0.37.x` branches, the receive method is `ReceiveEnvelope(Envelope)`.
> The `v0.34.x` branch also includes a previous version of this method:
> `Receive(chID byte, peer Peer, msgBytes []byte)`

Two important observations regarding the implementation of the `Receive` method:

1. Concurrency: the implementation should consider concurrent invocations of
   the `Receive` method, as messages received from different peers about at the
same time can be delivered to the reactor concurrently.

1. The implementation should be non-blocking, as it is directly invoked
   by the receive routines of peers.


## Add Peer

The p2p layer informs all registered reactors when it successfully establishes
a connection with a `Peer`.

It is up to the reactor to define how to process this event.
The typical behavior is to create routines to, upon some conditions or events,
send messages to the added peer, using the provided `Peer` handler.

Adding a peer has two steps.
The first step is optional and should be used to initialize data related to the `Peer`.
Some reactors (block sync, evidence, state sync) extend `BaseReactor` and
do not overwrite this method:


        // InitPeer is called by the switch before the peer is started. Use it to
        // initialize data for the peer (e.g. peer state).
        InitPeer(peer Peer) Peer

The second step is performed after the `Peer`'s send and receive routines are
successfully started.
The updated `Peer` handler provided to this method can be used, for instance,
to send messages to the peer:

        // AddPeer is called by the switch after the peer is added and successfully
        // started. Use it to start goroutines communicating with the peer.
        AddPeer(peer Peer)


## Send messages


	Send(Envelope) bool
	TrySend(Envelope) bool

The methods receive an `Envelope`, which consists of the byte `ChannelID`
and the `Message` payload.

FIXME: on `v0.34.x`, it is `Send/TrySend(chID byte, msgBytes []byte) bool`


## Remove Peer


        // RemovePeer is called by the switch when the peer is stopped (due to error
        // or other reason).
        RemovePeer(peer Peer, reason interface{})

## Code

`Reactor` interface declared on `p2p/base_reactor.go`.

## References

https://github.com/cometbft/cometbft/blob/main/spec/p2p/connection.md#switchreactor
