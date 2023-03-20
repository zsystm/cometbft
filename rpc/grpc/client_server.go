package coregrpc

import (
	"net"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	cmtnet "github.com/tendermint/tendermint/libs/net"
	proto "github.com/tendermint/tendermint/proto/tendermint/rpc/grpc"
)

// StartGRPCServer starts a new gRPC server using the given net.Listener.
//
// NOTE: This function blocks - you may want to call it in a go-routine.
//
// Deprecated. Please use ServerBuilder instead.
func StartGRPCServer(ln net.Listener) error {
	grpcServer := grpc.NewServer()
	proto.RegisterBroadcastAPIServer(grpcServer, &broadcastAPI{})
	return grpcServer.Serve(ln)
}

// StartGRPCClient dials the gRPC server using protoAddr and returns a new
// BroadcastAPIClient.
//
// Deprecated.
func StartGRPCClient(protoAddr string) proto.BroadcastAPIClient {
	//nolint:staticcheck // SA1019 Existing use of deprecated but supported dial option.
	conn, err := grpc.Dial(protoAddr, grpc.WithInsecure(), grpc.WithContextDialer(dialerFunc))
	if err != nil {
		panic(err)
	}
	return proto.NewBroadcastAPIClient(conn)
}

func dialerFunc(ctx context.Context, addr string) (net.Conn, error) {
	return cmtnet.Connect(addr)
}
