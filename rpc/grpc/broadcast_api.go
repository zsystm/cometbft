package coregrpc

import (
	"context"

	gogogrpc "github.com/cosmos/gogoproto/grpc"
	abci "github.com/tendermint/tendermint/abci/types"
	proto "github.com/tendermint/tendermint/proto/tendermint/rpc/grpc"
	core "github.com/tendermint/tendermint/rpc/core"
	rpctypes "github.com/tendermint/tendermint/rpc/jsonrpc/types"
)

// Type aliases to prevent breaking the Go APIs while avoiding moving generated
// proto stub files around.
//
// TODO(thane): Remove in v0.38+.
type (
	BroadcastAPIServer  = proto.BroadcastAPIServer
	RequestPing         = proto.RequestPing
	ResponsePing        = proto.ResponsePing
	RequestBroadcastTx  = proto.RequestBroadcastTx
	ResponseBroadcastTx = proto.ResponseBroadcastTx
)

// Deprecated.
type broadcastAPI struct{}

// RegisterBroadcastAPIServer is a function alias for the equivalent
// auto-generated function in the proto/tendermint/rpc/grpc directory.
//
// Deprecated. Will be removed in CometBFT v0.38+.
func RegisterBroadcastAPIServer(s gogogrpc.Server, svr proto.BroadcastAPIServer) {
	proto.RegisterBroadcastAPIServer(s, svr)
}

// NewBroadcastAPIServer creates a new gRPC BroadcastAPIServer.
//
// Deprecated. Will be removed in CometBFT v0.38+.
func NewBroadcastAPIServer() proto.BroadcastAPIServer {
	return &broadcastAPI{}
}

func (bapi *broadcastAPI) Ping(ctx context.Context, req *proto.RequestPing) (*proto.ResponsePing, error) {
	// kvstore so we can check if the server is up
	return &proto.ResponsePing{}, nil
}

func (bapi *broadcastAPI) BroadcastTx(ctx context.Context, req *proto.RequestBroadcastTx) (*proto.ResponseBroadcastTx, error) {
	// NOTE: there's no way to get client's remote address
	// see https://stackoverflow.com/questions/33684570/session-and-remote-ip-address-in-grpc-go
	res, err := core.BroadcastTxCommit(&rpctypes.Context{}, req.Tx)
	if err != nil {
		return nil, err
	}

	return &proto.ResponseBroadcastTx{
		CheckTx: &abci.ResponseCheckTx{
			Code: res.CheckTx.Code,
			Data: res.CheckTx.Data,
			Log:  res.CheckTx.Log,
		},
		DeliverTx: &abci.ResponseDeliverTx{
			Code: res.DeliverTx.Code,
			Data: res.DeliverTx.Data,
			Log:  res.DeliverTx.Log,
		},
	}, nil
}
