package live

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/JustCallMeMin/syncraft/internal/backend/persistence"
	"github.com/JustCallMeMin/syncraft/internal/backend/protocol"
	"github.com/JustCallMeMin/syncraft/internal/core/engine"
	"github.com/JustCallMeMin/syncraft/internal/core/model"
)

var (
	// ErrUnknownSession reports that the session has not sent client_hello.
	ErrUnknownSession = errors.New("session is unknown")
)

type sessionState struct {
	actorID       model.ActorID
	subscriptions map[model.DocumentID]bool
	outbox        []any
}

// Service runs the live sync pipeline over validated protocol messages.
type Service struct {
	store    *persistence.FileStore
	mu       sync.Mutex
	sessions map[string]*sessionState
	docs     map[model.DocumentID]*engine.Document
}

// NewService constructs a live sync service over one persistence store.
func NewService(store *persistence.FileStore) *Service {
	return &Service{
		store:    store,
		sessions: make(map[string]*sessionState),
		docs:     make(map[model.DocumentID]*engine.Document),
	}
}

// HandleClientHello validates and registers one live session.
func (s *Service) HandleClientHello(_ context.Context, msg protocol.ClientHelloMessage) (*protocol.ErrorMessage, error) {
	if err := msg.Validate(); err != nil {
		return protocolError(msg.Envelope, protocol.ErrorCodeInvalidMessageType, err), nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[msg.SessionID] = &sessionState{
		actorID:       msg.ActorID,
		subscriptions: make(map[model.DocumentID]bool),
		outbox:        make([]any, 0),
	}
	return nil, nil
}

// HandleSubscribe validates and records a document subscription.
func (s *Service) HandleSubscribe(ctx context.Context, msg protocol.SubscribeDocumentMessage) (*protocol.SubscribeAckMessage, *protocol.ErrorMessage, error) {
	if err := msg.Validate(); err != nil {
		return nil, protocolError(msg.Envelope, protocol.ErrorCodeInvalidDocumentID, err), nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session := s.sessions[msg.SessionID]
	if session == nil {
		return nil, protocolError(msg.Envelope, protocol.ErrorCodeUnauthorized, ErrUnknownSession), nil
	}
	session.subscriptions[msg.DocumentID] = true

	if _, err := s.documentLocked(ctx, msg.DocumentID); err != nil {
		return nil, nil, err
	}

	ack := &protocol.SubscribeAckMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeSubscribeAck,
			DocumentID:      msg.DocumentID,
			SessionID:       msg.SessionID,
			MessageID:       msg.MessageID,
		},
		SubscriptionMode: protocol.SubscriptionModeLiveOnly,
	}
	session.outbox = append(session.outbox, *ack)
	return ack, nil, nil
}

// HandleSubmit validates, persists, applies, and broadcasts one operation.
func (s *Service) HandleSubmit(ctx context.Context, msg protocol.SubmitOperationMessage) ([]protocol.BroadcastOperationMessage, *protocol.ErrorMessage, error) {
	if err := msg.Validate(); err != nil {
		return nil, protocolError(msg.Envelope, protocol.ErrorCodeInvalidOperation, err), nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session := s.sessions[msg.SessionID]
	if session == nil {
		return nil, protocolError(msg.Envelope, protocol.ErrorCodeUnauthorized, ErrUnknownSession), nil
	}
	if session.actorID != msg.Operation.ActorID {
		return nil, protocolError(msg.Envelope, protocol.ErrorCodeInvalidOperation, fmt.Errorf("operation actor_id must match session actor_id")), nil
	}
	if !session.subscriptions[msg.DocumentID] {
		return nil, protocolError(msg.Envelope, protocol.ErrorCodeUnauthorized, fmt.Errorf("session is not subscribed to document")), nil
	}

	doc, err := s.documentLocked(ctx, msg.DocumentID)
	if err != nil {
		return nil, nil, err
	}

	result, err := doc.Apply(msg.Operation)
	if err != nil {
		return nil, protocolError(msg.Envelope, protocol.ErrorCodeInvalidOperation, err), nil
	}
	if result.Status == engine.ApplyStatusDuplicate {
		return nil, nil, nil
	}

	if err := s.store.AppendOperation(ctx, msg.Operation); err != nil {
		return nil, nil, err
	}

	broadcast := protocol.BroadcastOperationMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeBroadcastOp,
			DocumentID:      msg.DocumentID,
			SessionID:       "server",
			MessageID:       msg.MessageID,
		},
		Operation: msg.Operation,
	}

	out := make([]protocol.BroadcastOperationMessage, 0)
	for sessionID, subscriber := range s.sessions {
		if !subscriber.subscriptions[msg.DocumentID] {
			continue
		}
		subscriber.outbox = append(subscriber.outbox, broadcast)
		_ = sessionID
		out = append(out, broadcast)
	}
	return out, nil, nil
}

// SessionOutbox returns a copy of the messages observed by one session.
func (s *Service) SessionOutbox(sessionID string) []any {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := s.sessions[sessionID]
	if session == nil {
		return nil
	}
	out := make([]any, len(session.outbox))
	copy(out, session.outbox)
	return out
}

func (s *Service) documentLocked(ctx context.Context, documentID model.DocumentID) (*engine.Document, error) {
	if doc := s.docs[documentID]; doc != nil {
		return doc, nil
	}

	doc, err := s.store.RebuildDocument(ctx, documentID)
	if err != nil {
		return nil, err
	}
	s.docs[documentID] = doc
	return doc, nil
}

func protocolError(envelope protocol.Envelope, code string, err error) *protocol.ErrorMessage {
	return &protocol.ErrorMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeError,
			DocumentID:      envelope.DocumentID,
			SessionID:       envelope.SessionID,
			MessageID:       envelope.MessageID,
		},
		ErrorCode:    code,
		ErrorMessage: err.Error(),
	}
}
