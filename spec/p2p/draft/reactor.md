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
This feature is implemented by the multiplex connection used by nodes to
exchange message with peers.

Each `Channel` has a globally unique _byte ID_.
When a reactor registers with the p2p layer, it informs the list of channels 
it is responsible for.
Incoming messages on the registered channels will then be delivered to the reactor.

More specifically, as part of the reactor registration procedure,
the following method of the `Reactor` interface is invoked:

        // GetChannels returns the list of MConnection.ChannelDescriptor. Make sure
        // that each ID is unique across all the reactors added to the switch.
        GetChannels() []*conn.ChannelDescriptor

The method should return a list of channels the reactor is responsible for.
A `ChannelDescriptor` consists of a global byte `ID`, a `Priority`, and some
local parameters for the channel, adopted by the multiplex connection.

> The `Switch` panics when the same channel ID is registered by two reactors.

### Priorities

TODO


## Receive

The main duty of a reactor is to handle incoming messages on the channels it
has registered with the p2p layer.

When a message is received from a connected peer on any of the channels
registered by the reactor, the node will deliver the message to the reactor
invoking the following method of the `Reactor` interface:


        // Receive is called by the switch when an envelope is received from any connected
        // peer on any of the channels registered by the reactor
        Receive(Envelope)

The reactor receives a message packed into an `Envelope` with the following content:

- `ChannelID`: the channel the message belongs to
- `Src`: the `Peer` from which the message was received
- `Message`: the message's payload, unmarshalled using protocol buffers

Two important observations regarding the implementation of the `Receive` method:

1. Concurrency: the implementation should consider concurrent invocations of
   the `Receive` method, as messages received from different peers about at the
same time can be delivered to the reactor concurrently.
1. The implementation should be non-blocking, as it is directly invoked
   by the receive routines of peers.

### Previous Version

Up to `v0.34.?`, the `Reactor` interface had a distinct receive method signature:

        Receive(chID byte, peer Peer, msgBytes []byte)

The contents of an `Envelope` were in this version directly passed to the method.
An important distinction is that the provided message payload in this case was
a marshalled byte array, being up to the reactor to unmarshall it.

Due to the migration to the current interface, on the `v0.34.x` and `v0.37.x`
branches, the receive method has a different name: `ReceiveEnvelope(Envelope)`.


## Add Peer

The p2p layer informs all registered reactors when it successfully establishes
a connection with a `Peer`.

It is up to the reactor to define how to process this event.
The typical behavior is to setup routines that, given some conditions or events,
send messages to the added peer, using the provided `Peer` handler.

Adding a peer to a reactor has two steps.
In the first step, the `Peer` has not yet been started.
This step should be used to initialize state or data related to the new peer,
but not to interact with it:


        // InitPeer is called by the switch before the peer is started. Use it to
        // initialize data for the peer (e.g. peer state).
        InitPeer(peer Peer) Peer

This first step is optional.
In fact, most existing reactors (block sync, evidence, PEX, and state sync)
do not implement the `InitPeer` method.
This is possible because they extend `BaseReactor`,
which provides a standard (no-op) implementation for the method.

The second step is performed after the peer's send and receive routines are
started without errors.
The updated `Peer` handler provided to this method can then be used to interact with the peer:

        // AddPeer is called by the switch after the peer is added and successfully
        // started. Use it to start goroutines communicating with the peer.
        AddPeer(peer Peer)

In contrast to `InitPeer` method, all standard reactors implement the `AddPeer`
method.
In case of reactors extending `BaseReactor`, they overwrite the standard
(no-op) implementation it provides.


## Remove Peer

The p2p layer also informs all registered reactors when it disconnects from a `Peer`:

        // RemovePeer is called by the switch when the peer is stopped (due to error
        // or other reason).
        RemovePeer(peer Peer, reason interface{})

When this method is invoked, the peer's send and receive routine were already stopped.
This means that the reactor should not receive any further message from this
peer and should not try sending messages to the removed peer.

The reasons for disconnecting from a peer are the following:

1. The multiplex connection used to exchange messages with the peer has
   encountered an error.  The caught error is the reason passed to the
   `RemovePeer` method.
1. A registered reactor requested the removal of the peer due to an error.
   This typically happens when the peer has misbehaved.
   The reason provided by the reactor is passed to the `RemovePeer` method.
1. A registered reactor requested to disconnect a peer "gracefully".
   No reason is provided in this case.
   - This possibility is currently only used by the PEX reactor of a node
     operating in seed mode.
1. The p2p layer is shutting down. All peers are removed and no reason is provided.


## Send messages

The `Reactor` interface is used by the p2p layer to provide events to the
reactor. Thus, it does not include methods in the opposite direction, such as
for the reactor to send messages.
Instead, the p2p layer provides to the reactor `Peer` instances with methods
for sending messages to peers.

There are two methods for sending messages to a `Peer`:

        // Send the message in the envelope on the channel specified by the envelope.
        // Returns false if the connection times out trying to place the message
        // onto its internal queue.
        Peer.Send(Envelope) bool

        // TrySend attempts to sends the message in the envelope on the channel
        // specified by the envelope. Returns false immediately if the connection's
        // internal queue is full.
        Peer.TrySend(Envelope) bool

The `Send` method is blocking and waits until the message is successfully
enqueued on the channel identified by `Envelope.ChannelID`.
The method returns `false` if the message could not be enqueued after `10s`.

The `TrySend` method is non-blocking and attempts to enqueue the message on the
channel identified by `Envelope.ChannelID`.
If the queue is full, the method immediately returns `false`.

In both methods, the provided `Envelope.Message` is serialized using protocol
buffers marshalling.
If the marshalling fails, an error message is logged and the method returns `false`.

If no channel is registered with `Envelope.ChannelID`, the method immediately
returns `false`.

### Broadcast

TODO: offered by the switch, inefficient, relies on multiple `Send.Send` calls. 

### Previous Version

Up to `v0.34.?`, the `Peer` interface had distinct send method signatures:

        Peer.Send(chID byte, msgBytes []byte)
        Peer.TrySend(chID byte, msgBytes []byte)

The channel _byte ID_ and the message, as a marshalled byte array, were
directly passed to the method in this version.
The marshalling of the message was therefore up to the reactor.

Due to the migration to the current interface, on the `v0.34.x` and `v0.37.x`
branches, the send methods have different names: `Peer.SendEnvelope(Envelope)` and
`Peer.TrySendEnvelope(Envelope)`.


## Switch

The `Switch` is the core component of the p2p layer, being responsible for
establishing connections with peers and routing messages.
It is the `Switch` that invokes the `InitPeer` and `AddReactor` methods when a
new peer is connected, and the `RemovePeer` when a peer is disconnected.

When a reactor is registered, via the `Switch.AddReactor` method,
the `Switch` invokes the following method of the `Reactor` interface:

        // SetSwitch allows setting a switch.
        SetSwitch(*Switch)

The `Switch` instance provided to the reactor can be used for:

1. Broadcast messages
1. Mark a peer as good
1. Disconnect from a peer


## Grammar

The following grammar, written in case-sensitive Augmented Backus–Naur form
(ABNF, specified in [IETF rfc7405](https://datatracker.ietf.org/doc/html/rfc7405)),
specifies the expected sequence of calls from the P2P layer to a `Reactor`:


```abnf
start           = add-reactor on-start *peer-connection on-stop
add-reactor     = get-channels set-switch

; Refers to a single peer, reactor should support multiple concurrent peers
peer-connection = init-peer peer-start
peer-start      = [receive] (peer-connected / start-error)
peer-connected  = add-peer *receive peer-stop
peer-stop       = [peer-error] remove-peer

; Service interface
on-start        = %s"<OnStart>"
on-stop         = %s"<OnStop>"
; Reactor interface
get-channels    = %s"<GetChannels>"
set-switch      = %s"<SetSwitch>"
init-peer       = %s"<InitPeer>"
add-peer        = %s"<AddPeer>"
remove-peer     = %s"<RemovePeer>"
receive         = %s"<Receive>"

; Errors, for reference
start-error     = %s"Error starting peer"
peer-error      = %s"Stopping peer for error"
```


## Code

`Reactor` interface is declared on `p2p/base_reactor.go`.

## References

https://github.com/cometbft/cometbft/blob/main/spec/p2p/connection.md#switchreactor
