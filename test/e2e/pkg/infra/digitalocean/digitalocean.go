package digitalocean

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	_ "embed"

	e2e "github.com/cometbft/cometbft/test/e2e/pkg"
	"github.com/cometbft/cometbft/test/e2e/pkg/exec"
	"github.com/cometbft/cometbft/test/e2e/pkg/infra"
)

var _ infra.Provider = (*Provider)(nil)

// Provider implements a DigitalOcean-backed infrastructure provider.
type Provider struct {
	infra.ProviderData
}

// Setup generates any necessary configuration for the infrastructure
// provider during testnet setup.
func (p *Provider) Setup() error {
	// Create directory for Ansible files.
	if err := os.MkdirAll(filepath.Join(p.Testnet.Dir, ".ansible"), 0o755); err != nil {
		return err
	}

	// Generate the file mapping IPs to zones, needed for emulating latencies.
	if err := infra.GenerateIPZonesTable(p.Testnet.Nodes, p.IPZonesFilePath(), false); err != nil {
		return err
	}

	return nil
}

//go:embed ansible/testapp-set-latency.yaml
var ansibleSetLatencyContent []byte

// SetLatency prepares and executes the latency-setter script in the given list of nodes.
func (p Provider) SetLatency(ctx context.Context, nodes ...*e2e.Node) error {
	// Execute Ansible playbook on all nodes.
	externalIPs := make([]string, len(nodes))
	for i, n := range nodes {
		externalIPs[i] = n.ExternalIP.String()
	}
	return p.execAnsible(ctx, "testapp-set-latency.yaml", ansibleSetLatencyContent, externalIPs)
}

//go:embed ansible/testapp-start.yaml
var ansibleStartContent []byte

func (p Provider) StartNodes(ctx context.Context, nodes ...*e2e.Node) error {
	nodeIPs := make([]string, len(nodes))
	for i, n := range nodes {
		nodeIPs[i] = n.ExternalIP.String()
	}
	return p.execAnsible(ctx, "testapp-start.yaml", ansibleStartContent, nodeIPs)
}

//go:embed ansible/testapp-start.yaml
var ansibleStopContent []byte

func (p Provider) StopTestnet(ctx context.Context) error {
	nodeIPs := make([]string, len(p.Testnet.Nodes))
	for i, n := range p.Testnet.Nodes {
		nodeIPs[i] = n.ExternalIP.String()
	}
	return p.execAnsible(ctx, "testapp-stop.yaml", ansibleStopContent, nodeIPs)
}

//go:embed ansible/testapp-disconnect.yaml
var ansibleDisconnectContent []byte

func (p Provider) Disconnect(ctx context.Context, _ string, ip string) error {
	return p.execAnsible(ctx, "testapp-disconnect.yaml", ansibleDisconnectContent, []string{ip})
}

//go:embed ansible/testapp-reconnect.yaml
var ansibleReconnectContent []byte

func (p Provider) Reconnect(ctx context.Context, _ string, ip string) error {
	return p.execAnsible(ctx, "testapp-reconnect.yaml", ansibleReconnectContent, []string{ip})
}

func (p Provider) CheckUpgraded(_ context.Context, node *e2e.Node) (string, bool, error) {
	// Upgrade not supported yet by DO provider
	return node.Name, false, nil
}

// ExecCompose runs a Docker Compose command for a testnet.
func (p Provider) execAnsible(ctx context.Context, playbook string, content []byte, nodeIPs []string, args ...string) error { //nolint:unparam
	// Write playbook file to Ansible testnet directory.
	playbookPath := filepath.Join(p.Testnet.Dir, ".ansible", playbook)
	//nolint: gosec // G306: Expect WriteFile permissions to be 0600 or less
	if err := os.WriteFile(playbookPath, content, 0o644); err != nil {
		return err
	}
	// Execute playbook.
	return exec.CommandVerbose(ctx, append(
		[]string{"ansible-playbook", playbookPath, "--limit", strings.Join(nodeIPs, ",")},
		args...)...)
}
