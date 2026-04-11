package browser

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/JustCallMeMin/syncraft/internal/backend/live"
	"github.com/JustCallMeMin/syncraft/internal/backend/persistence"
	"github.com/JustCallMeMin/syncraft/internal/backend/protocol"
	"github.com/JustCallMeMin/syncraft/internal/client/editor"
	"github.com/JustCallMeMin/syncraft/internal/core/model"

	"github.com/gorilla/websocket"
)

//go:embed web/*
var webAssets embed.FS

type browserCommand struct {
	Type       string           `json:"type"`
	ActorID    model.ActorID    `json:"actor_id,omitempty"`
	DocumentID model.DocumentID `json:"document_id,omitempty"`
	Index      int              `json:"index,omitempty"`
	Value      string           `json:"value,omitempty"`
}

type browserStateMessage struct {
	Type            string                 `json:"type"`
	ActorID         model.ActorID          `json:"actor_id,omitempty"`
	DocumentID      model.DocumentID       `json:"document_id,omitempty"`
	Text            string                 `json:"text,omitempty"`
	ConnectionState editor.ConnectionState `json:"connection_state,omitempty"`
	LastError       string                 `json:"last_error,omitempty"`
}

type browserClient struct {
	sessionID  string
	actorID    model.ActorID
	documentID model.DocumentID
	editor     *editor.Session
	conn       *websocket.Conn
	writeMu    sync.Mutex
}

// Server exposes the minimal browser collaboration shell over the live service.
type Server struct {
	service  *live.Service
	upgrader websocket.Upgrader

	mu      sync.Mutex
	nextID  uint64
	clients map[*websocket.Conn]*browserClient
	docs    map[model.DocumentID]map[*websocket.Conn]bool
}

// NewServer constructs one browser demo server backed by one persistence root.
func NewServer(persistenceRoot string) (*Server, error) {
	if persistenceRoot == "" {
		return nil, fmt.Errorf("persistence root must not be empty")
	}
	if err := os.MkdirAll(persistenceRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create browser persistence root: %w", err)
	}
	store, err := persistence.NewFileStore(filepath.Clean(persistenceRoot))
	if err != nil {
		return nil, err
	}
	return &Server{
		service: live.NewService(store),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(_ *http.Request) bool {
				return true
			},
		},
		clients: make(map[*websocket.Conn]*browserClient),
		docs:    make(map[model.DocumentID]map[*websocket.Conn]bool),
	}, nil
}

// Handler returns the HTTP handler that serves the browser shell and websocket endpoint.
func (s *Server) Handler() (http.Handler, error) {
	subtree, err := fs.Sub(webAssets, "web")
	if err != nil {
		return nil, fmt.Errorf("prepare embedded web assets: %w", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServerFS(subtree))
	mux.HandleFunc("/ws", s.handleWebSocket)
	return mux, nil
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	conn.SetReadLimit(1 << 20)
	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			s.detachClient(conn)
			return
		}
		var command browserCommand
		if err := json.Unmarshal(payload, &command); err != nil {
			s.writeError(conn, model.ActorID(""), model.DocumentID(""), fmt.Errorf("decode browser command: %w", err))
			continue
		}
		if err := s.handleCommand(r.Context(), conn, command); err != nil {
			client := s.lookupClient(conn)
			if client == nil {
				s.writeError(conn, command.ActorID, command.DocumentID, err)
				continue
			}
			s.writeError(conn, client.actorID, client.documentID, err)
		}
	}
}

func (s *Server) handleCommand(ctx context.Context, conn *websocket.Conn, command browserCommand) error {
	switch command.Type {
	case "init":
		return s.handleInit(ctx, conn, command)
	case "insert_text":
		return s.handleInsert(ctx, conn, command)
	case "delete_at":
		return s.handleDelete(ctx, conn, command)
	default:
		return fmt.Errorf("unsupported browser command type %q", command.Type)
	}
}

func (s *Server) handleInit(ctx context.Context, conn *websocket.Conn, command browserCommand) error {
	if command.ActorID == "" {
		return fmt.Errorf("actor_id must not be empty")
	}
	if command.DocumentID == "" {
		return fmt.Errorf("document_id must not be empty")
	}

	session, err := editor.NewSession(command.DocumentID, command.ActorID)
	if err != nil {
		return err
	}

	client := &browserClient{
		sessionID:  s.nextSessionID(),
		actorID:    command.ActorID,
		documentID: command.DocumentID,
		editor:     session,
		conn:       conn,
	}
	s.attachClient(conn, client)

	helloErr, err := s.service.HandleClientHello(ctx, protocol.ClientHelloMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeClientHello,
			SessionID:       client.sessionID,
			MessageID:       clientMessageID("hello"),
		},
		ActorID: command.ActorID,
	})
	if err != nil {
		return err
	}
	if helloErr != nil {
		return fmt.Errorf("%s", helloErr.ErrorMessage)
	}

	ack, subscribeErr, err := s.service.HandleSubscribe(ctx, protocol.SubscribeDocumentMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeSubscribeDoc,
			DocumentID:      command.DocumentID,
			SessionID:       client.sessionID,
			MessageID:       clientMessageID("subscribe"),
		},
	})
	if err != nil {
		return err
	}
	if subscribeErr != nil {
		return fmt.Errorf("%s", subscribeErr.ErrorMessage)
	}
	if err := client.editor.HandleSubscribeAck(*ack); err != nil {
		return err
	}
	if ack.SubscriptionMode == protocol.SubscriptionModeSnapshotGap {
		events, catchupErr, err := s.service.HandleRequestCatchup(ctx, protocol.RequestCatchupMessage{
			Envelope: protocol.Envelope{
				ProtocolVersion: protocol.VersionV1,
				MessageType:     protocol.MessageTypeRequestCatchup,
				DocumentID:      command.DocumentID,
				SessionID:       client.sessionID,
				MessageID:       clientMessageID("catchup"),
			},
		})
		if err != nil {
			return err
		}
		if catchupErr != nil {
			return fmt.Errorf("%s", catchupErr.ErrorMessage)
		}
		if err := s.applyCatchupEvents(client, events); err != nil {
			return err
		}
	}
	return s.writeState(client)
}

func (s *Server) handleInsert(ctx context.Context, conn *websocket.Conn, command browserCommand) error {
	client := s.lookupClient(conn)
	if client == nil {
		return fmt.Errorf("browser session is not initialized")
	}
	ops, err := client.editor.InsertTextAt(command.Index, command.Value)
	if err != nil {
		return err
	}
	for _, op := range ops {
		if err := s.submitAndFanout(ctx, client, op); err != nil {
			return err
		}
	}
	return s.writeState(client)
}

func (s *Server) handleDelete(ctx context.Context, conn *websocket.Conn, command browserCommand) error {
	client := s.lookupClient(conn)
	if client == nil {
		return fmt.Errorf("browser session is not initialized")
	}
	op, err := client.editor.DeleteAt(command.Index)
	if err != nil {
		return err
	}
	if err := s.submitAndFanout(ctx, client, op); err != nil {
		return err
	}
	return s.writeState(client)
}

func (s *Server) submitAndFanout(ctx context.Context, origin *browserClient, op model.Operation) error {
	broadcasts, submitErr, err := s.service.HandleSubmit(ctx, protocol.SubmitOperationMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeSubmitOperation,
			DocumentID:      origin.documentID,
			SessionID:       origin.sessionID,
			MessageID:       clientMessageID("submit"),
		},
		Operation: op,
	})
	if err != nil {
		return err
	}
	if submitErr != nil {
		return fmt.Errorf("%s", submitErr.ErrorMessage)
	}
	for _, broadcast := range broadcasts {
		if err := s.applyBroadcastToDocumentClients(broadcast.DocumentID, broadcast); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) applyBroadcastToDocumentClients(documentID model.DocumentID, broadcast protocol.BroadcastOperationMessage) error {
	clients := s.documentClients(documentID)
	for _, client := range clients {
		if err := client.editor.ApplyRemoteBroadcast(broadcast); err != nil {
			return err
		}
		if err := s.writeState(client); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) applyCatchupEvents(client *browserClient, events []any) error {
	for _, event := range events {
		switch typed := event.(type) {
		case protocol.CatchupSnapshotMessage:
			if err := client.editor.ApplyCatchupSnapshot(typed); err != nil {
				return err
			}
		case protocol.CatchupOperationsMessage:
			if err := client.editor.ApplyCatchupOperations(typed); err != nil {
				return err
			}
		case protocol.CatchupCompleteMessage:
			if err := client.editor.ApplyCatchupComplete(typed); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported catchup event %T", event)
		}
	}
	return nil
}

func (s *Server) writeState(client *browserClient) error {
	state := client.editor.ViewState()
	return s.writeJSON(client.conn, browserStateMessage{
		Type:            "state",
		ActorID:         client.actorID,
		DocumentID:      client.documentID,
		Text:            state.Text,
		ConnectionState: state.ConnectionState,
		LastError:       state.LastError,
	})
}

func (s *Server) writeError(conn *websocket.Conn, actorID model.ActorID, documentID model.DocumentID, err error) error {
	return s.writeJSON(conn, browserStateMessage{
		Type:            "state",
		ActorID:         actorID,
		DocumentID:      documentID,
		ConnectionState: editor.ConnectionStateError,
		LastError:       err.Error(),
	})
}

func (s *Server) writeJSON(conn *websocket.Conn, value any) error {
	client := s.lookupClient(conn)
	if client != nil {
		client.writeMu.Lock()
		defer client.writeMu.Unlock()
	}
	return conn.WriteJSON(value)
}

func (s *Server) attachClient(conn *websocket.Conn, client *browserClient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[conn] = client
	if s.docs[client.documentID] == nil {
		s.docs[client.documentID] = make(map[*websocket.Conn]bool)
	}
	s.docs[client.documentID][conn] = true
}

func (s *Server) detachClient(conn *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	client := s.clients[conn]
	if client == nil {
		return
	}
	delete(s.clients, conn)
	if clients := s.docs[client.documentID]; clients != nil {
		delete(clients, conn)
		if len(clients) == 0 {
			delete(s.docs, client.documentID)
		}
	}
}

func (s *Server) lookupClient(conn *websocket.Conn) *browserClient {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.clients[conn]
}

func (s *Server) documentClients(documentID model.DocumentID) []*browserClient {
	s.mu.Lock()
	defer s.mu.Unlock()
	clients := s.docs[documentID]
	out := make([]*browserClient, 0, len(clients))
	for conn := range clients {
		out = append(out, s.clients[conn])
	}
	return out
}

func (s *Server) nextSessionID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	return fmt.Sprintf("sess_browser_%d", s.nextID)
}

func clientMessageID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UTC().UnixNano())
}
