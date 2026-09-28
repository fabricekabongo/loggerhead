package main

import (
	"bytes"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fabricekabongo/loggerhead/clustering"
	"github.com/fabricekabongo/loggerhead/config"
	"github.com/hashicorp/memberlist"
)

type stubMemberList struct {
	node *memberlist.Node
}

func (s stubMemberList) LocalNode() *memberlist.Node {
	return s.node
}

type stubCluster struct {
	memberList nodeInfoProvider
}

func (s stubCluster) MemberList() nodeInfoProvider {
	return s.memberList
}

func TestPrintWelcomeMessage(t *testing.T) {
	cfg := config.Config{
		ClusterDNS:     "cluster.local",
		MaxConnections: 15,
		SeedNode:       "10.0.0.1",
		ReadPort:       1111,
		WritePort:      2222,
		HttpPort:       3333,
		ClusterPort:    4444,
		MaxEOFWait:     5 * time.Second,
	}

	cluster := stubCluster{
		memberList: stubMemberList{node: &memberlist.Node{
			Name:  "node-1",
			Addr:  net.ParseIP("192.168.0.1"),
			State: memberlist.StateAlive,
		}},
	}

	readEnd, writeEnd, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	originalStdout := os.Stdout
	os.Stdout = writeEnd
	defer func() {
		os.Stdout = originalStdout
	}()

	printWelcomeMessage(cfg, cluster)
	writeEnd.Close()

	var output bytes.Buffer
	if _, err := output.ReadFrom(readEnd); err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	got := output.String()
	expectedSnippets := []string{
		"Read Port:  1111",
		"Write Port:  2222",
		"Cluster Port:  4444",
		"Admin & Prometheus Port: 3333",
		"Max Connections:  15",
		"Max EOF Wait:  5s",
		"Cluster DNS:  cluster.local",
		"Seed Node:  10.0.0.1",
		"My IP:  192.168.0.1",
		"Node Name:  node-1",
		"Node State:  " + clustering.StateToString(memberlist.StateAlive),
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(got, snippet) {
			t.Fatalf("expected output to contain %q, got %q", snippet, got)
		}
	}
}
