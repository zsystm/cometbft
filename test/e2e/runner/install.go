package main

import (
	"context"

	e2e "github.com/cometbft/cometbft/test/e2e/pkg"
	"github.com/cometbft/cometbft/test/e2e/pkg/infra"
)

func Install(ctx context.Context, testnet *e2e.Testnet, infp infra.Provider) error {
	// TODO: in DO, install CometBFT in validator nodes.

	// Set latency emulation on all nodes with a zone.
	nodesWithZone := make([]*e2e.Node, 0)
	for _, n := range testnet.Nodes {
		if n.ZoneIsSet() {
			nodesWithZone = append(nodesWithZone, n)
		}
	}
	if len(nodesWithZone) != 0 {
		logger.Info("Setting latency emulation")
		if err := infp.SetLatency(ctx, nodesWithZone...); err != nil {
			return err
		}
	}

	return nil
}
