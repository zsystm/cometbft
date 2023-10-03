package coregrpc_test

import (
	"os"
	"testing"

	"github.com/cometbft/cometbft/abci/example/kvstore"

	rpctest "github.com/cometbft/cometbft/rpc/test"
)

func TestMain(m *testing.M) {
	// start a CometBFT node in the background to test against
	app := kvstore.NewInMemoryApplication()
	node := rpctest.StartTendermint(app)

	code := m.Run()

	// and shut down proper at the end
	rpctest.StopTendermint(node)
	os.Exit(code)
}

// TODO needs to be moved to grpc_test
/*
func TestBroadcastTx(t *testing.T) {
	res, err := rpctest.GetGRPCClient().BroadcastTx(
		context.Background(),
		&core_grpc.RequestBroadcastTx{Tx: kvstore.NewTx("hello", "world")},
	)
	require.NoError(t, err)
	require.EqualValues(t, 0, res.CheckTx.Code)
	require.EqualValues(t, 0, res.TxResult.Code)
}
*/
