package client

import (
	"context"

	legacygrpc "github.com/cometbft/cometbft/rpc/grpc"
	grpc "github.com/cosmos/gogoproto/grpc"
	grpc1 "google.golang.org/grpc"
)

// TODO This whole needs to be used somewhere in tests and rethought as to whether this is how the implementation of the functions should be
type broadcastAPIClient struct {
	client legacygrpc.BroadcastAPIClient
}

func newBroadcastAPIClient(conn grpc.ClientConn) legacygrpc.BroadcastAPIClient {
	return &broadcastAPIClient{
		client: legacygrpc.NewBroadcastAPIClient(conn),
	}
}

// BroadacstTx implements the BroadcastTx of the BroadcastClientAPI
func (c *broadcastAPIClient) BroadcastTx(ctx context.Context, req *legacygrpc.RequestBroadcastTx, opts ...grpc1.CallOption) (*legacygrpc.ResponseBroadcastTx, error) {

	// req := legacygrpc.RequestBroadcastTx{
	// 	Tx: tx,
	// }

	res, err := c.client.BroadcastTx(ctx, req)

	if err != nil {
		return nil, err
	}

	// TODO: Maybe do something smarter here and extract info from the response instead of just returning the response
	return res, nil
}

func (c *broadcastAPIClient) Ping(ctx context.Context, req *legacygrpc.RequestPing, opts ...grpc1.CallOption) (*legacygrpc.ResponsePing, error) {
	res, err := c.client.Ping(ctx, req)
	if err != nil || res == nil {
		return nil, err
	}
	return res, err

}

type disabledBroadcastAPIServiceClient struct{}

func newDisabledBroadcastAPIServiceClient() legacygrpc.BroadcastAPIClient {
	return &disabledBroadcastAPIServiceClient{}
}

// BroadacstTx implements the BroadcastTx of the BroadcastClientAPI
func (c *disabledBroadcastAPIServiceClient) BroadcastTx(ctx context.Context, req *legacygrpc.RequestBroadcastTx, opts ...grpc1.CallOption) (*legacygrpc.ResponseBroadcastTx, error) {
	panic("version service client is disabled")
}

func (c *disabledBroadcastAPIServiceClient) Ping(ctx context.Context, req *legacygrpc.RequestPing, opts ...grpc1.CallOption) (*legacygrpc.ResponsePing, error) {
	panic("version service client is disabled")

}
