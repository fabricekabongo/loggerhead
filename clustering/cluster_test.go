package clustering

import (
	"errors"
	"testing"

	"github.com/fabricekabongo/loggerhead/config"
	"github.com/hashicorp/memberlist"
)

func TestStateToString(t *testing.T) {
	cases := map[memberlist.NodeStateType]string{
		memberlist.StateAlive:         "Alive",
		memberlist.StateSuspect:       "Suspect",
		memberlist.StateLeft:          "Left",
		memberlist.StateDead:          "Dead",
		memberlist.NodeStateType(255): "Unknown",
	}

	for state, expected := range cases {
		if got := StateToString(state); got != expected {
			t.Fatalf("expected %s for state %v but got %s", expected, state, got)
		}
	}
}

func TestGetClusterIPsWithSeedNode(t *testing.T) {
	cfg := config.Config{SeedNode: "10.0.0.1"}

	ips, err := getClusterIPs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(ips) != 1 || ips[0] != cfg.SeedNode {
		t.Fatalf("expected seed node slice with %s, got %#v", cfg.SeedNode, ips)
	}
}

func TestGetClusterIPsWithDNS(t *testing.T) {
	cfg := config.Config{ClusterDNS: "localhost"}

	ips, err := getClusterIPs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(ips) == 0 {
		t.Fatal("expected at least one IP from localhost resolution")
	}
}

func TestGetClusterIPsDNSFailure(t *testing.T) {
	cfg := config.Config{ClusterDNS: "nonexistent.invalid."}

	_, err := getClusterIPs(cfg)
	if !errors.Is(err, ErrFailedToExtractIPsFromDNS) {
		t.Fatalf("expected ErrFailedToExtractIPsFromDNS, got %v", err)
	}
}

func TestGetIPsFromDomainNameInvalid(t *testing.T) {
	_, err := getIPsFromDomainName("nonexistent.invalid.")
	if err == nil {
		t.Fatal("expected lookup error for invalid domain")
	}
}
