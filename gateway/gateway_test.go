package gateway_test

import (
	"os"
	"testing"

	"github.com/textileio/go-textile/core"
	. "github.com/textileio/go-textile/gateway"
	"github.com/textileio/go-textile/keypair"
)

var repoPath = "testdata/.textile"

func TestGateway_Creation(t *testing.T) {
	_ = os.RemoveAll(repoPath)

	err := core.InitRepo(core.InitConfig{
		Account:     keypair.Random(),
		RepoPath:    repoPath,
		GatewayAddr: "127.0.0.1:9998",
	})
	if err != nil {
		t.Errorf("init node failed: %s", err)
		return
	}

	node, err := core.NewTextile(core.RunConfig{
		RepoPath: repoPath,
	})
	if err != nil {
		t.Errorf("create node failed: %s", err)
		return
	}

	Host = &Gateway{Node: node}
	Host.Start(node.Config().Addresses.Gateway)
}

func TestGateway_Addr(t *testing.T) {
	if len(Host.Addr()) == 0 {
		t.Error("get gateway address failed")
		return
	}
}

func TestGateway_Stop(t *testing.T) {
	err := Host.Stop()
	if err != nil {
		t.Errorf("stop gateway failed: %s", err)
	}
	_ = os.RemoveAll(repoPath)
}
