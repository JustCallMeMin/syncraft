package browser

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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
	Type              string            `json:"type"`
	ActorID           model.ActorID     `json:"actor_id,omitempty"`
	DocumentID        model.DocumentID  `json:"document_id,omitempty"`
	Title             string            `json:"title,omitempty"`
	Index             int               `json:"index,omitempty"`
	Value             string            `json:"value,omitempty"`
	NextActorCounter  uint64            `json:"next_actor_counter,omitempty"`
	ActorCounterStart uint64            `json:"actor_counter_start,omitempty"`
	ActorCounter      uint64            `json:"actor_counter,omitempty"`
	Presence          *browserPresence  `json:"presence,omitempty"`
	Operation         *browserOperation `json:"operation,omitempty"`
}

type browserStateMessage struct {
	Type              string                  `json:"type"`
	ActorID           model.ActorID           `json:"actor_id,omitempty"`
	DocumentID        model.DocumentID        `json:"document_id,omitempty"`
	Title             string                  `json:"title,omitempty"`
	Text              string                  `json:"text,omitempty"`
	ConnectionState   editor.ConnectionState  `json:"connection_state,omitempty"`
	LastError         string                  `json:"last_error,omitempty"`
	NextActorCounter  uint64                  `json:"next_actor_counter,omitempty"`
	LastSnapshotID    *model.SnapshotID       `json:"last_snapshot_id,omitempty"`
	LastOperationID   *model.OperationID      `json:"last_operation_id,omitempty"`
	PendingQueueCount int                     `json:"pending_queue_count,omitempty"`
	VisibleElements   []browserVisibleElement `json:"visible_elements,omitempty"`
	Collaborators     []browserCollaborator   `json:"collaborators,omitempty"`
}

type browserVisibleElement struct {
	ID    model.ElementID `json:"id"`
	Value string          `json:"value"`
}

type browserPresence struct {
	DisplayName        string                  `json:"display_name,omitempty"`
	CursorAnchor       *browserPresenceAnchor  `json:"cursor_anchor,omitempty"`
	CursorFocus        *browserPresenceAnchor  `json:"cursor_focus,omitempty"`
	SelectionDirection string                  `json:"selection_direction,omitempty"`
	IsCollapsed        bool                    `json:"is_collapsed"`
}

type browserPresenceAnchor struct {
	ElementID     model.ElementID `json:"element_id,omitempty"`
	Offset        int             `json:"offset,omitempty"`
	FallbackIndex int             `json:"fallback_index,omitempty"`
}

type browserCollaborator struct {
	ActorID            model.ActorID          `json:"actor_id"`
	SessionID          string                 `json:"session_id"`
	DisplayName        string                 `json:"display_name,omitempty"`
	IsSelf             bool                   `json:"is_self,omitempty"`
	CursorAnchor       browserPresenceAnchor  `json:"cursor_anchor"`
	CursorFocus        browserPresenceAnchor  `json:"cursor_focus"`
	SelectionDirection string                 `json:"selection_direction,omitempty"`
	IsCollapsed        bool                   `json:"is_collapsed"`
	LastSeenAt         time.Time              `json:"last_seen_at"`
}

type browserOperation struct {
	DocumentID    model.DocumentID      `json:"document_id"`
	OperationID   model.OperationID     `json:"operation_id"`
	ActorID       model.ActorID         `json:"actor_id"`
	ActorCounter  uint64                `json:"actor_counter"`
	Type          model.OperationType   `json:"type"`
	InsertPayload *browserInsertPayload `json:"insert_payload,omitempty"`
	DeletePayload *browserDeletePayload `json:"delete_payload,omitempty"`
}

type browserInsertPayload struct {
	ElementID     model.ElementID  `json:"element_id"`
	Value         string           `json:"value"`
	LeftOriginID  *model.ElementID `json:"left_origin_id,omitempty"`
	RightOriginID *model.ElementID `json:"right_origin_id,omitempty"`
}

type browserDeletePayload struct {
	TargetElementID model.ElementID `json:"target_element_id"`
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
		},
		clients: make(map[*websocket.Conn]*browserClient),
		docs:    make(map[model.DocumentID]map[*websocket.Conn]bool),
	}, nil
}

// Handler returns the HTTP handler that serves the browser shell and websocket endpoint.
func (s *Server) Handler() (http.Handler, error) {
	s.upgrader.CheckOrigin = s.checkOrigin
	subtree, err := fs.Sub(webAssets, "web")
	if err != nil {
		return nil, fmt.Errorf("prepare embedded web assets: %w", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		asset, readErr := fs.ReadFile(subtree, "favicon.svg")
		if readErr != nil {
			http.Error(w, "favicon not available", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/svg+xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(asset)
	})
	mux.Handle("/", http.FileServerFS(subtree))
	mux.HandleFunc("/ws", s.handleWebSocket)
	return mux, nil
}

func (s *Server) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(originURL.Host, r.Host)
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
	case "update_title":
		return s.handleUpdateTitle(ctx, conn, command)
	case "presence_update":
		return s.handlePresenceUpdate(ctx, conn, command)
	case "insert_text":
		return s.handleInsert(ctx, conn, command)
	case "delete_at":
		return s.handleDelete(ctx, conn, command)
	case "submit_operation":
		return s.handleSubmitOperation(ctx, conn, command)
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
	if command.NextActorCounter != 0 {
		if err := session.SetNextActorCounter(command.NextActorCounter); err != nil {
			return err
		}
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

func (s *Server) handleUpdateTitle(ctx context.Context, conn *websocket.Conn, command browserCommand) error {
	client := s.lookupClient(conn)
	if client == nil {
		return fmt.Errorf("browser session is not initialized")
	}
	updates, errMsg, err := s.service.HandleUpdateTitle(ctx, protocol.UpdateDocumentTitleMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeUpdateTitle,
			DocumentID:      client.documentID,
			SessionID:       client.sessionID,
			MessageID:       clientMessageID("title"),
		},
		Title: strings.TrimSpace(command.Title),
	})
	if err != nil {
		return err
	}
	if errMsg != nil {
		return fmt.Errorf("%s", errMsg.ErrorMessage)
	}
	return s.applyTitleUpdates(client.documentID, updates)
}

func (s *Server) handlePresenceUpdate(ctx context.Context, conn *websocket.Conn, command browserCommand) error {
	client := s.lookupClient(conn)
	if client == nil {
		return fmt.Errorf("browser session is not initialized")
	}
	if command.Presence == nil {
		return fmt.Errorf("presence_update requires one presence payload")
	}
	payload := command.Presence.toProtocol(client.actorID, client.sessionID)
	updates, errMsg, err := s.service.HandlePresenceUpdate(ctx, protocol.PresenceUpdateMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypePresenceUpdate,
			DocumentID:      client.documentID,
			SessionID:       client.sessionID,
			MessageID:       clientMessageID("presence"),
		},
		Presence: payload,
	})
	if err != nil {
		return err
	}
	if errMsg != nil {
		return fmt.Errorf("%s", errMsg.ErrorMessage)
	}
	return s.applyPresenceUpdates(client.documentID, updates)
}

func (s *Server) handleInsert(ctx context.Context, conn *websocket.Conn, command browserCommand) error {
	client := s.lookupClient(conn)
	if client == nil {
		return fmt.Errorf("browser session is not initialized")
	}
	startCounter := command.ActorCounterStart
	if startCounter == 0 {
		startCounter = client.editor.ViewMetadata().NextActorCounter
	}
	ops, err := client.editor.InsertTextAtWithCounter(command.Index, command.Value, startCounter)
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
	counter := command.ActorCounter
	if counter == 0 {
		counter = client.editor.ViewMetadata().NextActorCounter
	}
	op, err := client.editor.DeleteAtWithCounter(command.Index, counter)
	if err != nil {
		return err
	}
	if err := s.submitAndFanout(ctx, client, op); err != nil {
		return err
	}
	return s.writeState(client)
}

func (s *Server) handleSubmitOperation(ctx context.Context, conn *websocket.Conn, command browserCommand) error {
	client := s.lookupClient(conn)
	if client == nil {
		return fmt.Errorf("browser session is not initialized")
	}
	if command.Operation == nil {
		return fmt.Errorf("submit_operation requires one canonical operation payload")
	}
	op, err := command.Operation.toModelOperation()
	if err != nil {
		return err
	}
	if op.DocumentID != client.documentID {
		return fmt.Errorf("submit operation document_id %q does not match session document_id %q", op.DocumentID, client.documentID)
	}
	if op.ActorID != client.actorID {
		return fmt.Errorf("submit operation actor_id %q does not match session actor_id %q", op.ActorID, client.actorID)
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

func (s *Server) applyTitleUpdates(documentID model.DocumentID, updates []protocol.DocumentTitleChangedMessage) error {
	if len(updates) == 0 {
		return nil
	}
	clients := s.documentClients(documentID)
	for _, client := range clients {
		if err := s.writeState(client); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) applyPresenceUpdates(documentID model.DocumentID, updates []protocol.PresenceBroadcastMessage) error {
	if len(updates) == 0 {
		return nil
	}
	clients := s.documentClients(documentID)
	for _, client := range clients {
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
	metadata := client.editor.ViewMetadata()
	visibleElements := client.editor.ViewVisibleElements()
	documentMetadata, err := s.service.DocumentMetadata(context.Background(), client.documentID)
	if err != nil {
		return err
	}
	presenceSnapshot := s.service.PresenceSnapshot(client.documentID)
	browserVisible := make([]browserVisibleElement, 0, len(visibleElements))
	for _, elem := range visibleElements {
		browserVisible = append(browserVisible, browserVisibleElement{
			ID:    elem.ID,
			Value: elem.Value,
		})
	}
	collaborators := make([]browserCollaborator, 0, len(presenceSnapshot))
	for _, collaborator := range presenceSnapshot {
		collaborators = append(collaborators, browserCollaborator{
			ActorID:            collaborator.ActorID,
			SessionID:          collaborator.SessionID,
			DisplayName:        collaborator.DisplayName,
			IsSelf:             collaborator.SessionID == client.sessionID,
			CursorAnchor:       browserPresenceAnchorFromProtocol(collaborator.CursorAnchor),
			CursorFocus:        browserPresenceAnchorFromProtocol(collaborator.CursorFocus),
			SelectionDirection: collaborator.SelectionDirection,
			IsCollapsed:        collaborator.IsCollapsed,
			LastSeenAt:         collaborator.LastSeenAt,
		})
	}
	return s.writeJSON(client.conn, browserStateMessage{
		Type:              "state",
		ActorID:           client.actorID,
		DocumentID:        client.documentID,
		Title:             documentMetadata.Title,
		Text:              state.Text,
		ConnectionState:   state.ConnectionState,
		LastError:         state.LastError,
		NextActorCounter:  metadata.NextActorCounter,
		LastSnapshotID:    metadata.LastSnapshotID,
		LastOperationID:   metadata.LastOperationID,
		PendingQueueCount: metadata.PendingQueueCount,
		VisibleElements:   browserVisible,
		Collaborators:     collaborators,
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
	client := s.clients[conn]
	if client == nil {
		s.mu.Unlock()
		return
	}
	delete(s.clients, conn)
	if clients := s.docs[client.documentID]; clients != nil {
		delete(clients, conn)
		if len(clients) == 0 {
			delete(s.docs, client.documentID)
		}
	}
	s.mu.Unlock()

	updates := s.service.DisconnectSession(client.sessionID)
	_ = s.applyPresenceUpdates(client.documentID, updates)
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

func (p browserPresence) toProtocol(actorID model.ActorID, sessionID string) protocol.PresencePayload {
	anchor := browserPresenceAnchor{}
	if p.CursorAnchor != nil {
		anchor = *p.CursorAnchor
	}
	focus := anchor
	if p.CursorFocus != nil {
		focus = *p.CursorFocus
	}
	return protocol.PresencePayload{
		ActorID:            actorID,
		SessionID:          sessionID,
		DisplayName:        strings.TrimSpace(p.DisplayName),
		CursorAnchor:       anchor.toProtocol(),
		CursorFocus:        focus.toProtocol(),
		SelectionDirection: strings.TrimSpace(p.SelectionDirection),
		IsCollapsed:        p.IsCollapsed,
		LastSeenAt:         time.Now().UTC(),
	}
}

func (a browserPresenceAnchor) toProtocol() protocol.PresencePosition {
	return protocol.PresencePosition{
		ElementID:     a.ElementID,
		Offset:        a.Offset,
		FallbackIndex: a.FallbackIndex,
	}
}

func browserPresenceAnchorFromProtocol(position protocol.PresencePosition) browserPresenceAnchor {
	return browserPresenceAnchor{
		ElementID:     position.ElementID,
		Offset:        position.Offset,
		FallbackIndex: position.FallbackIndex,
	}
}

func (o browserOperation) toModelOperation() (model.Operation, error) {
	op := model.Operation{
		DocumentID:   o.DocumentID,
		OperationID:  o.OperationID,
		ActorID:      o.ActorID,
		ActorCounter: o.ActorCounter,
		Type:         o.Type,
	}
	if o.InsertPayload != nil {
		op.InsertPayload = &model.InsertPayload{
			ElementID:     o.InsertPayload.ElementID,
			Value:         o.InsertPayload.Value,
			LeftOriginID:  o.InsertPayload.LeftOriginID,
			RightOriginID: o.InsertPayload.RightOriginID,
		}
	}
	if o.DeletePayload != nil {
		op.DeletePayload = &model.DeletePayload{
			TargetElementID: o.DeletePayload.TargetElementID,
		}
	}
	if err := op.Validate(); err != nil {
		return model.Operation{}, err
	}
	return op, nil
}
