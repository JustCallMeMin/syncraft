package live

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

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
	logger   *slog.Logger
	mu       sync.Mutex
	sessions map[string]*sessionState
	docs     map[model.DocumentID]*engine.Document
}

// NewService constructs a live sync service over one persistence store.
func NewService(store *persistence.FileStore) *Service {
	return &Service{
		store:    store,
		logger:   slog.Default(),
		sessions: make(map[string]*sessionState),
		docs:     make(map[model.DocumentID]*engine.Document),
	}
}

// SetLogger overrides the audit logger used by the live sync service.
func (s *Service) SetLogger(logger *slog.Logger) {
	if logger == nil {
		s.logger = slog.Default()
		return
	}
	s.logger = logger
}

// HandleClientHello validates and registers one live session.
func (s *Service) HandleClientHello(_ context.Context, msg protocol.ClientHelloMessage) (*protocol.ErrorMessage, error) {
	if err := msg.Validate(); err != nil {
		s.logProtocolReject(context.Background(), msg.Envelope, protocol.ErrorCodeInvalidMessageType, err)
		return protocolError(msg.Envelope, protocol.ErrorCodeInvalidMessageType, err), nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[msg.SessionID] = &sessionState{
		actorID:       msg.ActorID,
		subscriptions: make(map[model.DocumentID]bool),
		outbox:        make([]any, 0),
	}
	s.logger.Info("register session",
		"session_id", msg.SessionID,
		"actor_id", msg.ActorID,
	)
	return nil, nil
}

// HandleSubscribe validates and records a document subscription.
func (s *Service) HandleSubscribe(ctx context.Context, msg protocol.SubscribeDocumentMessage) (*protocol.SubscribeAckMessage, *protocol.ErrorMessage, error) {
	if err := msg.Validate(); err != nil {
		s.logProtocolReject(ctx, msg.Envelope, protocol.ErrorCodeInvalidDocumentID, err)
		return nil, protocolError(msg.Envelope, protocol.ErrorCodeInvalidDocumentID, err), nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session := s.sessions[msg.SessionID]
	if session == nil {
		s.logProtocolReject(ctx, msg.Envelope, protocol.ErrorCodeUnauthorized, ErrUnknownSession)
		return nil, protocolError(msg.Envelope, protocol.ErrorCodeUnauthorized, ErrUnknownSession), nil
	}
	session.subscriptions[msg.DocumentID] = true

	if _, err := s.documentLocked(ctx, msg.DocumentID); err != nil {
		return nil, nil, err
	}

	subscriptionMode, err := s.subscriptionModeLocked(ctx, msg.DocumentID, msg.KnownLastOperationID)
	if err != nil {
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
		SubscriptionMode: subscriptionMode,
	}
	session.outbox = append(session.outbox, *ack)
	s.logger.InfoContext(ctx, "subscribe document",
		"session_id", msg.SessionID,
		"actor_id", session.actorID,
		"document_id", msg.DocumentID,
		"subscription_mode", subscriptionMode,
	)
	return ack, nil, nil
}

// HandleRequestCatchup serves snapshot-plus-delta state transfer for one subscribed session.
func (s *Service) HandleRequestCatchup(ctx context.Context, msg protocol.RequestCatchupMessage) ([]any, *protocol.ErrorMessage, error) {
	if err := msg.Validate(); err != nil {
		s.logProtocolReject(ctx, msg.Envelope, protocol.ErrorCodeInvalidDocumentID, err)
		return nil, protocolError(msg.Envelope, protocol.ErrorCodeInvalidDocumentID, err), nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session := s.sessions[msg.SessionID]
	if session == nil {
		s.logProtocolReject(ctx, msg.Envelope, protocol.ErrorCodeUnauthorized, ErrUnknownSession)
		return nil, protocolError(msg.Envelope, protocol.ErrorCodeUnauthorized, ErrUnknownSession), nil
	}
	if !session.subscriptions[msg.DocumentID] {
		err := fmt.Errorf("session is not subscribed to document")
		s.logProtocolReject(ctx, msg.Envelope, protocol.ErrorCodeUnauthorized, err)
		return nil, protocolError(msg.Envelope, protocol.ErrorCodeUnauthorized, err), nil
	}

	snapshotRecord, laterOps, err := s.catchupStateLocked(ctx, msg.DocumentID, msg.KnownLastOperationID)
	if err != nil {
		return nil, nil, err
	}

	catchupID := fmt.Sprintf("catchup_%s_%d", msg.SessionID, time.Now().UTC().UnixNano())
	out := make([]any, 0, 3)

	snapshotMsg := protocol.CatchupSnapshotMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeCatchupSnapshot,
			DocumentID:      msg.DocumentID,
			SessionID:       "server",
			MessageID:       msg.MessageID,
		},
		Snapshot: protocol.CatchupSnapshotPayload{
			SnapshotID:              snapshotRecord.SnapshotID,
			LastIncludedOperationID: snapshotRecord.LastIncludedOperationID,
			State:                   snapshotRecord.State,
		},
	}
	out = append(out, snapshotMsg)
	session.outbox = append(session.outbox, snapshotMsg)

	opsMsg := protocol.CatchupOperationsMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeCatchupOps,
			DocumentID:      msg.DocumentID,
			SessionID:       "server",
			MessageID:       msg.MessageID,
		},
		CatchupID:  catchupID,
		BatchIndex: 1,
		HasMore:    false,
		Operations: laterOps,
	}
	if len(laterOps) > 0 {
		opsMsg.LastOperationIDInBatch = laterOps[len(laterOps)-1].OperationID
	}
	out = append(out, opsMsg)
	session.outbox = append(session.outbox, opsMsg)

	completeMsg := protocol.CatchupCompleteMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeCatchupComplete,
			DocumentID:      msg.DocumentID,
			SessionID:       "server",
			MessageID:       msg.MessageID,
		},
	}
	if len(laterOps) > 0 {
		completeMsg.LastOperationID = laterOps[len(laterOps)-1].OperationID
	} else {
		completeMsg.LastOperationID = snapshotRecord.LastIncludedOperationID
	}
	out = append(out, completeMsg)
	session.outbox = append(session.outbox, completeMsg)
	s.logger.InfoContext(ctx, "serve catchup",
		"session_id", msg.SessionID,
		"actor_id", session.actorID,
		"document_id", msg.DocumentID,
		"snapshot_id", snapshotRecord.SnapshotID,
		"snapshot_watermark", snapshotRecord.LastIncludedOperationID,
		"operation_count", len(laterOps),
	)

	return out, nil, nil
}

// HandleSubmit validates, persists, applies, and broadcasts one operation.
func (s *Service) HandleSubmit(ctx context.Context, msg protocol.SubmitOperationMessage) ([]protocol.BroadcastOperationMessage, *protocol.ErrorMessage, error) {
	if err := msg.Validate(); err != nil {
		s.logProtocolReject(ctx, msg.Envelope, protocol.ErrorCodeInvalidOperation, err)
		return nil, protocolError(msg.Envelope, protocol.ErrorCodeInvalidOperation, err), nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session := s.sessions[msg.SessionID]
	if session == nil {
		s.logProtocolReject(ctx, msg.Envelope, protocol.ErrorCodeUnauthorized, ErrUnknownSession)
		return nil, protocolError(msg.Envelope, protocol.ErrorCodeUnauthorized, ErrUnknownSession), nil
	}
	if session.actorID != msg.Operation.ActorID {
		err := fmt.Errorf("operation actor_id must match session actor_id")
		s.logProtocolReject(ctx, msg.Envelope, protocol.ErrorCodeInvalidOperation, err)
		return nil, protocolError(msg.Envelope, protocol.ErrorCodeInvalidOperation, err), nil
	}
	if !session.subscriptions[msg.DocumentID] {
		err := fmt.Errorf("session is not subscribed to document")
		s.logProtocolReject(ctx, msg.Envelope, protocol.ErrorCodeUnauthorized, err)
		return nil, protocolError(msg.Envelope, protocol.ErrorCodeUnauthorized, err), nil
	}

	doc, err := s.documentLocked(ctx, msg.DocumentID)
	if err != nil {
		return nil, nil, err
	}

	result, err := doc.Apply(msg.Operation)
	if err != nil {
		s.logProtocolReject(ctx, msg.Envelope, protocol.ErrorCodeInvalidOperation, err)
		return nil, protocolError(msg.Envelope, protocol.ErrorCodeInvalidOperation, err), nil
	}
	if result.Status == engine.ApplyStatusDuplicate {
		s.logger.InfoContext(ctx, "dedupe operation",
			"session_id", msg.SessionID,
			"actor_id", session.actorID,
			"document_id", msg.DocumentID,
			"operation_id", msg.Operation.OperationID,
			"operation_type", msg.Operation.Type,
		)
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
	s.logger.InfoContext(ctx, "accept operation",
		"session_id", msg.SessionID,
		"actor_id", session.actorID,
		"document_id", msg.DocumentID,
		"operation_id", msg.Operation.OperationID,
		"operation_type", msg.Operation.Type,
		"subscriber_count", len(out),
	)
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

func (s *Service) subscriptionModeLocked(ctx context.Context, documentID model.DocumentID, knownLast *model.OperationID) (string, error) {
	ops, err := s.store.LoadOperationsAfter(ctx, documentID, "")
	if err != nil {
		return "", err
	}
	if len(ops) == 0 {
		return protocol.SubscriptionModeLiveOnly, nil
	}
	if knownLast != nil && *knownLast == ops[len(ops)-1].OperationID {
		return protocol.SubscriptionModeLiveOnly, nil
	}
	return protocol.SubscriptionModeSnapshotGap, nil
}

func (s *Service) catchupStateLocked(ctx context.Context, documentID model.DocumentID, knownLast *model.OperationID) (persistence.SnapshotRecord, []model.Operation, error) {
	snapshot, err := s.store.LoadLatestSnapshot(ctx, documentID)
	if err != nil && !errors.Is(err, persistence.ErrSnapshotNotFound) {
		return persistence.SnapshotRecord{}, nil, err
	}

	doc, err := s.documentLocked(ctx, documentID)
	if err != nil {
		return persistence.SnapshotRecord{}, nil, err
	}

	if snapshot == nil {
		ops, err := s.store.LoadOperationsAfter(ctx, documentID, "")
		if err != nil {
			return persistence.SnapshotRecord{}, nil, err
		}
		var watermark model.OperationID
		if len(ops) > 0 {
			watermark = ops[len(ops)-1].OperationID
		}
		snapshot = &persistence.SnapshotRecord{
			SnapshotID:              model.SnapshotID(fmt.Sprintf("snap_%d", time.Now().UTC().UnixNano())),
			DocumentID:              documentID,
			LastIncludedOperationID: watermark,
			CreatedAt:               time.Now().UTC(),
			State:                   doc.Snapshot(),
		}
	}

	after := snapshot.LastIncludedOperationID
	if knownLast != nil && *knownLast != "" && *knownLast != snapshot.LastIncludedOperationID {
		after = *knownLast
	}
	laterOps, err := s.store.LoadOperationsAfter(ctx, documentID, after)
	if err != nil {
		return persistence.SnapshotRecord{}, nil, err
	}
	return *snapshot, laterOps, nil
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

func (s *Service) logProtocolReject(ctx context.Context, envelope protocol.Envelope, code string, err error) {
	s.logger.WarnContext(ctx, "reject protocol action",
		"session_id", envelope.SessionID,
		"document_id", envelope.DocumentID,
		"message_id", envelope.MessageID,
		"message_type", envelope.MessageType,
		"error_code", code,
		"reason", err.Error(),
	)
}
