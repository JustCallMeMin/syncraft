package browser

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/JustCallMeMin/syncraft/internal/core/model"
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

func TestBrowserShellRejectsCrossOriginWebsocketUpgrade(t *testing.T) {
	server := newTestServer(t)
	handler, err := server.Handler()
	if err != nil {
		t.Fatalf("Handler() error = %v, want nil", err)
	}
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/ws"
	header := http.Header{}
	header.Set("Origin", "http://evil.example")
	_, response, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err == nil {
		t.Fatal("Dial() error = nil, want cross-origin rejection")
	}
	if response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("StatusCode = %v, want %d", response, http.StatusForbidden)
	}
}

func TestBrowserShellUsesPersistedNextActorCounterFromInit(t *testing.T) {
	server := newTestServer(t)
	handler, err := server.Handler()
	if err != nil {
		t.Fatalf("Handler() error = %v, want nil", err)
	}
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	client := mustDial(t, httpServer.URL)
	defer client.Close()
	mustSend(t, client, browserCommand{
		Type:             "init",
		ActorID:          "actor_a",
		DocumentID:       "demo-doc",
		NextActorCounter: 10,
	})
	_ = mustReadState(t, client)

	mustSend(t, client, browserCommand{
		Type:              "insert_text",
		Index:             0,
		Value:             "a",
		ActorCounterStart: 10,
	})
	state := mustReadStateUntilText(t, client, "a")
	if state.NextActorCounter != 11 {
		t.Fatalf("NextActorCounter = %d, want 11", state.NextActorCounter)
	}
}

func TestBrowserShellAcceptsCanonicalSubmitOperationReplay(t *testing.T) {
	server := newTestServer(t)
	handler, err := server.Handler()
	if err != nil {
		t.Fatalf("Handler() error = %v, want nil", err)
	}
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	client := mustDial(t, httpServer.URL)
	defer client.Close()
	mustSend(t, client, browserCommand{
		Type:       "init",
		ActorID:    "actor_a",
		DocumentID: "demo-doc",
	})
	_ = mustReadState(t, client)

	mustSend(t, client, browserCommand{
		Type: "submit_operation",
		Operation: &browserOperation{
			DocumentID:   "demo-doc",
			OperationID:  "op_actor_a_1",
			ActorID:      "actor_a",
			ActorCounter: 1,
			Type:         "insert",
			InsertPayload: &browserInsertPayload{
				ElementID: "elem_actor_a_1",
				Value:     "a",
			},
		},
	})
	state := mustReadStateUntilText(t, client, "a")
	if len(state.VisibleElements) != 1 {
		t.Fatalf("len(VisibleElements) = %d, want 1", len(state.VisibleElements))
	}
	if state.VisibleElements[0].ID != "elem_actor_a_1" {
		t.Fatalf("VisibleElements[0].ID = %q, want %q", state.VisibleElements[0].ID, "elem_actor_a_1")
	}
}

func TestBrowserShellCanonicalSubmitOperationAdvancesActorCounterAcrossSequentialInputs(t *testing.T) {
	server := newTestServer(t)
	handler, err := server.Handler()
	if err != nil {
		t.Fatalf("Handler() error = %v, want nil", err)
	}
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	client := mustDial(t, httpServer.URL)
	defer client.Close()
	mustSend(t, client, browserCommand{
		Type:       "init",
		ActorID:    "actor_a",
		DocumentID: "demo-doc",
	})
	_ = mustReadState(t, client)

	mustSend(t, client, browserCommand{
		Type: "submit_operation",
		Operation: &browserOperation{
			DocumentID:   "demo-doc",
			OperationID:  "op_actor_a_1",
			ActorID:      "actor_a",
			ActorCounter: 1,
			Type:         "insert",
			InsertPayload: &browserInsertPayload{
				ElementID: "elem_actor_a_1",
				Value:     "â",
			},
		},
	})
	state := mustReadStateUntilText(t, client, "â")
	if state.NextActorCounter != 2 {
		t.Fatalf("NextActorCounter after first canonical submit = %d, want 2", state.NextActorCounter)
	}

	mustSend(t, client, browserCommand{
		Type: "submit_operation",
		Operation: &browserOperation{
			DocumentID:   "demo-doc",
			OperationID:  "op_actor_a_2",
			ActorID:      "actor_a",
			ActorCounter: 2,
			Type:         "insert",
			InsertPayload: &browserInsertPayload{
				ElementID:     "elem_actor_a_2",
				Value:         "b",
				LeftOriginID:  ptrElementID("elem_actor_a_1"),
				RightOriginID: nil,
			},
		},
	})
	state = mustReadStateUntilText(t, client, "âb")
	if state.NextActorCounter != 3 {
		t.Fatalf("NextActorCounter after second canonical submit = %d, want 3", state.NextActorCounter)
	}
}

func TestBrowserShellServesFavicon(t *testing.T) {
	server := newTestServer(t)
	handler, err := server.Handler()
	if err != nil {
		t.Fatalf("Handler() error = %v, want nil", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "image/svg+xml" {
		t.Fatalf("Content-Type = %q, want %q", contentType, "image/svg+xml")
	}
	if body := response.Body.String(); !strings.Contains(body, "<svg") {
		t.Fatalf("favicon body = %q, want embedded svg payload", body)
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
	header := http.Header{}
	header.Set("Origin", baseURL)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
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

func ptrElementID(value model.ElementID) *model.ElementID {
	return &value
}
