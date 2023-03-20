package coregrpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// DialContext constructs an insecure gRPC client connection.
func DialContext(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	return grpc.DialContext(
		ctx,
		addr,
		grpc.WithContextDialer(dialerFunc),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}
