package broadcastxservice

import (
	context "context"

	abci "github.com/cometbft/cometbft/abci/types"
	core "github.com/cometbft/cometbft/rpc/core"
	legacygrpc "github.com/cometbft/cometbft/rpc/grpc"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
)

type broadcastAPIServer struct {
	env *core.Environment
}

// New creates a new CometBFT version service server.
func New(e *core.Environment) legacygrpc.BroadcastAPIServer {
	return &broadcastAPIServer{env: e}
}

// GetVersion implements v1.VersionServiceServer
func (s *broadcastAPIServer) BroadcastTx(ctx context.Context, req *legacygrpc.RequestBroadcastTx) (*legacygrpc.ResponseBroadcastTx, error) {
	// NOTE: there's no way to get client's remote address
	// see https://stackoverflow.com/questions/33684570/session-and-remote-ip-address-in-grpc-go
	res, err := s.env.BroadcastTxCommit(&rpctypes.Context{}, req.Tx)
	if err != nil {
		return nil, err
	}

	return &legacygrpc.ResponseBroadcastTx{
		CheckTx: &abci.ResponseCheckTx{
			Code: res.CheckTx.Code,
			Data: res.CheckTx.Data,
			Log:  res.CheckTx.Log,
		},
		TxResult: &abci.ExecTxResult{
			Code: res.TxResult.Code,
			Data: res.TxResult.Data,
			Log:  res.TxResult.Log,
		},
	}, nil
}

func (s *broadcastAPIServer) Ping(context.Context, *legacygrpc.RequestPing) (*legacygrpc.ResponsePing, error) {
	// kvstore so we can check if the server is up
	return &legacygrpc.ResponsePing{}, nil
}
