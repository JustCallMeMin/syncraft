package browser

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestBrowserShellSyncsTwoClients(t *testing.T) {
	server := newTestServer(t)
	handler, err := server.Handler()
	if err != nil {
		t.Fatalf("Handler() error = %v, want nil", err)
	}
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	clientA := mustDial(t, httpServer.URL)
	defer clientA.Close()
	clientB := mustDial(t, httpServer.URL)
	defer clientB.Close()

	mustSend(t, clientA, browserCommand{Type: "init", ActorID: "actor_a", DocumentID: "demo-doc"})
	mustSend(t, clientB, browserCommand{Type: "init", ActorID: "actor_b", DocumentID: "demo-doc"})

	if state := mustReadState(t, clientA); state.ConnectionState != "live" {
		t.Fatalf("clientA ConnectionState = %s, want live", state.ConnectionState)
	}
	if state := mustReadState(t, clientB); state.ConnectionState != "live" {
		t.Fatalf("clientB ConnectionState = %s, want live", state.ConnectionState)
	}

	mustSend(t, clientA, browserCommand{Type: "insert_text", Index: 0, Value: "ab"})
	stateA := mustReadStateUntilText(t, clientA, "ab")
	stateB := mustReadStateUntilText(t, clientB, "ab")

	mustSend(t, clientB, browserCommand{Type: "delete_at", Index: 0})
	stateA = mustReadStateUntilText(t, clientA, "b")
	stateB = mustReadStateUntilText(t, clientB, "b")
	if stateA.Text != "b" || stateB.Text != "b" {
		t.Fatalf("final texts = (%q, %q), want (%q, %q)", stateA.Text, stateB.Text, "b", "b")
	}
}

func TestBrowserShellReconnectsWithCatchupState(t *testing.T) {
	server := newTestServer(t)
	handler, err := server.Handler()
	if err != nil {
		t.Fatalf("Handler() error = %v, want nil", err)
	}
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	clientA := mustDial(t, httpServer.URL)
	defer clientA.Close()
	clientB := mustDial(t, httpServer.URL)

	mustSend(t, clientA, browserCommand{Type: "init", ActorID: "actor_a", DocumentID: "demo-doc"})
	mustSend(t, clientB, browserCommand{Type: "init", ActorID: "actor_b", DocumentID: "demo-doc"})
	_ = mustReadState(t, clientA)
	_ = mustReadState(t, clientB)

	mustSend(t, clientA, browserCommand{Type: "insert_text", Index: 0, Value: "ab"})
	_ = mustReadStateUntilText(t, clientA, "ab")
	_ = mustReadStateUntilText(t, clientB, "ab")

	clientB.Close()

	mustSend(t, clientA, browserCommand{Type: "insert_text", Index: 2, Value: "c"})
	_ = mustReadStateUntilText(t, clientA, "abc")

	clientB = mustDial(t, httpServer.URL)
	defer clientB.Close()
	mustSend(t, clientB, browserCommand{Type: "init", ActorID: "actor_b", DocumentID: "demo-doc"})
	stateB := mustReadStateUntilText(t, clientB, "abc")
	if stateB.ConnectionState != "live" {
		t.Fatalf("clientB ConnectionState = %s, want live", stateB.ConnectionState)
	}
}

func TestBrowserShellRecoversAfterServerRestart(t *testing.T) {
	root := t.TempDir()
	serverA := mustBrowserServerWithRoot(t, root)
	handlerA, err := serverA.Handler()
	if err != nil {
		t.Fatalf("Handler() error = %v, want nil", err)
	}
	httpServerA := httptest.NewServer(handlerA)

	clientA := mustDial(t, httpServerA.URL)
	mustSend(t, clientA, browserCommand{Type: "init", ActorID: "actor_a", DocumentID: "demo-doc"})
	_ = mustReadState(t, clientA)
	mustSend(t, clientA, browserCommand{Type: "insert_text", Index: 0, Value: "ab"})
	_ = mustReadStateUntilText(t, clientA, "ab")
	clientA.Close()
	httpServerA.Close()

	serverB := mustBrowserServerWithRoot(t, root)
	handlerB, err := serverB.Handler()
	if err != nil {
		t.Fatalf("Handler() error = %v, want nil", err)
	}
	httpServerB := httptest.NewServer(handlerB)
	defer httpServerB.Close()

	clientB := mustDial(t, httpServerB.URL)
	defer clientB.Close()
	mustSend(t, clientB, browserCommand{Type: "init", ActorID: "actor_b", DocumentID: "demo-doc"})
	stateB := mustReadStateUntilText(t, clientB, "ab")
	if stateB.ConnectionState != "live" {
		t.Fatalf("clientB ConnectionState = %s, want live", stateB.ConnectionState)
	}
}

func TestBrowserShellSurfacesInvalidCommandError(t *testing.T) {
	server := newTestServer(t)
	handler, err := server.Handler()
	if err != nil {
		t.Fatalf("Handler() error = %v, want nil", err)
	}
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	client := mustDial(t, httpServer.URL)
	defer client.Close()
	mustSend(t, client, browserCommand{Type: "unknown_command"})
	state := mustReadState(t, client)
	if state.ConnectionState != "error" {
		t.Fatalf("ConnectionState = %s, want error", state.ConnectionState)
	}
	if state.LastError == "" {
		t.Fatal("LastError = empty string, want visible operator error")
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	server, err := NewServer(t.TempDir())
	if err != nil {
		t.Fatalf("NewServer() error = %v, want nil", err)
	}
	return server
}

func mustBrowserServerWithRoot(t *testing.T, root string) *Server {
	t.Helper()
	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("NewServer(%s) error = %v, want nil", root, err)
	}
	return server
}

func mustDial(t *testing.T, baseURL string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(baseURL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Dial(%s) error = %v, want nil", wsURL, err)
	}
	return conn
}

func mustSend(t *testing.T, conn *websocket.Conn, command browserCommand) {
	t.Helper()
	if err := conn.WriteJSON(command); err != nil {
		t.Fatalf("WriteJSON(%+v) error = %v, want nil", command, err)
	}
}

func mustReadState(t *testing.T, conn *websocket.Conn) browserStateMessage {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline() error = %v, want nil", err)
	}
	var state browserStateMessage
	if err := conn.ReadJSON(&state); err != nil {
		t.Fatalf("ReadJSON() error = %v, want nil", err)
	}
	if state.Type != "state" {
		t.Fatalf("state.Type = %q, want %q", state.Type, "state")
	}
	return state
}

func mustReadStateUntilText(t *testing.T, conn *websocket.Conn, want string) browserStateMessage {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		state := mustReadState(t, conn)
		if state.Text == want {
			return state
		}
	}
	t.Fatalf("did not observe text %q before timeout", want)
	return browserStateMessage{}
}
