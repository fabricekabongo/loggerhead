package admin

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fabricekabongo/loggerhead/config"
	"github.com/hashicorp/memberlist"
)

type mockMemberList struct {
	local   *memberlist.Node
	members []*memberlist.Node
	health  int
}

func (m *mockMemberList) LocalNode() *memberlist.Node { return m.local }
func (m *mockMemberList) NumMembers() int             { return len(m.members) }
func (m *mockMemberList) Members() []*memberlist.Node { return m.members }
func (m *mockMemberList) GetHealthScore() int         { return m.health }

type mockQueue struct{ queued int }

func (m mockQueue) NumQueued() int { return m.queued }

type mockCluster struct {
	ml    MemberListProvider
	queue BroadcastQueue
}

func (m mockCluster) MemberList() MemberListProvider { return m.ml }
func (m mockCluster) Broadcasts() BroadcastQueue     { return m.queue }

func TestAdminData(t *testing.T) {
	localNode := &memberlist.Node{
		Name:  "node-a",
		Addr:  net.ParseIP("127.0.0.1"),
		State: memberlist.StateAlive,
	}

	cluster := mockCluster{
		ml: &mockMemberList{
			local:   localNode,
			members: []*memberlist.Node{localNode},
			health:  2,
		},
		queue: mockQueue{queued: 3},
	}

	server := NewOpsServer(cluster, configForTests())

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin-data", nil)
	server.AdminData().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected application/json content type, got %s", got)
	}

	var data Data
	if err := json.NewDecoder(rr.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if data.Name != "node-a" {
		t.Fatalf("expected node name to be propagated, got %s", data.Name)
	}

	if data.Address != "127.0.0.1" {
		t.Fatalf("expected address to match node address, got %s", data.Address)
	}

	if data.NodesAlive != 1 {
		t.Fatalf("expected one node alive, got %d", data.NodesAlive)
	}

	if data.Health != 2 {
		t.Fatalf("expected health score from cluster, got %d", data.Health)
	}

	if data.QueueCount != 3 {
		t.Fatalf("expected queue count to be included, got %d", data.QueueCount)
	}
}

func TestAdminDataProxySkip(t *testing.T) {
	localNode := &memberlist.Node{
		Name:  "node-a",
		Addr:  net.ParseIP("127.0.0.1"),
		State: memberlist.StateAlive,
	}
	remoteNode := &memberlist.Node{
		Name:  "node-b",
		Addr:  net.ParseIP("127.0.0.2"),
		State: memberlist.StateAlive,
	}

	cluster := mockCluster{
		ml: &mockMemberList{
			local:   localNode,
			members: []*memberlist.Node{localNode, remoteNode},
			health:  1,
		},
		queue: mockQueue{queued: 0},
	}

	server := NewOpsServer(cluster, configForTests())

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin-data?proxy=true", nil)
	server.AdminData().ServeHTTP(rr, req)

	var data Data
	if err := json.NewDecoder(rr.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(data.Others) != 0 {
		t.Fatalf("expected proxy mode to skip other members, got %d entries", len(data.Others))
	}
}

func TestAdminUI(t *testing.T) {
	cluster := mockCluster{
		ml:    &mockMemberList{local: &memberlist.Node{}, members: []*memberlist.Node{{}}, health: 0},
		queue: mockQueue{queued: 0},
	}

	server := NewOpsServer(cluster, configForTests())

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	server.AdminUI().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 from admin UI, got %d", rr.Code)
	}

	if rr.Body.Len() == 0 {
		t.Fatal("expected template to render some content")
	}
}

func configForTests() config.Config {
	return config.Config{}
}
