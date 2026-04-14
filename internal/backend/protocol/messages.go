package protocol

import (
	"errors"
	"fmt"
	"time"

	"github.com/JustCallMeMin/syncraft/internal/core/model"
)

const (
	// VersionV1 is the only supported protocol version for Syncraft v1.
	VersionV1 = "syncraft.v1"

	MessageTypeClientHello      = "client_hello"
	MessageTypeSubscribeDoc     = "subscribe_document"
	MessageTypeSubscribeAck     = "subscribe_ack"
	MessageTypeSubmitOperation  = "submit_operation"
	MessageTypeBroadcastOp      = "broadcast_operation"
	MessageTypeRequestCatchup   = "request_catchup"
	MessageTypeCatchupSnapshot  = "catchup_snapshot"
	MessageTypeCatchupOps       = "catchup_operations"
	MessageTypeCatchupComplete  = "catchup_complete"
	MessageTypeUpdateTitle      = "update_document_title"
	MessageTypeTitleChanged     = "document_title_changed"
	MessageTypePresenceUpdate   = "presence_update"
	MessageTypePresenceSnapshot = "presence_snapshot"
	MessageTypePresenceBroadcast = "presence_broadcast"
	MessageTypeError            = "error"
	SubscriptionModeLiveOnly    = "live_only"
	SubscriptionModeSnapshotGap = "snapshot_then_delta"
)

const (
	ErrorCodeUnsupportedProtocolVersion = "unsupported_protocol_version"
	ErrorCodeInvalidMessageType         = "invalid_message_type"
	ErrorCodeInvalidDocumentID          = "invalid_document_id"
	ErrorCodeInvalidOperation           = "invalid_operation"
	ErrorCodeInvalidTitle               = "invalid_title"
	ErrorCodeInvalidPresence            = "invalid_presence"
	ErrorCodeUnauthorized               = "unauthorized"
	ErrorCodeInternalError              = "internal_error"
)

var (
	ErrUnsupportedProtocolVersion = errors.New("unsupported protocol version")
	ErrInvalidMessageType         = errors.New("invalid message type")
	ErrEmptySessionID             = errors.New("session_id must not be empty")
	ErrEmptyMessageID             = errors.New("message_id must not be empty")
	ErrEmptyActorID               = errors.New("actor_id must not be empty")
	ErrEmptyTitle                 = errors.New("title must not be empty")
)

// Envelope is the common transport shell for protocol messages.
type Envelope struct {
	ProtocolVersion string           `json:"protocol_version"`
	MessageType     string           `json:"message_type"`
	DocumentID      model.DocumentID `json:"document_id,omitempty"`
	SessionID       string           `json:"session_id"`
	MessageID       string           `json:"message_id"`
}

// Validate validates generic envelope expectations.
func (e Envelope) Validate(messageType string, requireDocument bool) error {
	if e.ProtocolVersion != VersionV1 {
		return fieldError("protocol_version", ErrUnsupportedProtocolVersion)
	}
	if e.MessageType != messageType {
		return fieldError("message_type", ErrInvalidMessageType)
	}
	if e.SessionID == "" {
		return fieldError("session_id", ErrEmptySessionID)
	}
	if e.MessageID == "" {
		return fieldError("message_id", ErrEmptyMessageID)
	}
	if requireDocument && e.DocumentID == "" {
		return fieldError("document_id", model.ErrEmptyDocumentID)
	}
	return nil
}

// ClientHelloMessage starts one live session.
type ClientHelloMessage struct {
	Envelope
	ActorID model.ActorID `json:"actor_id"`
}

// Validate validates the hello message.
func (m ClientHelloMessage) Validate() error {
	if err := m.Envelope.Validate(MessageTypeClientHello, false); err != nil {
		return err
	}
	if m.ActorID == "" {
		return fieldError("actor_id", ErrEmptyActorID)
	}
	return nil
}

// SubscribeDocumentMessage subscribes one session to one document stream.
type SubscribeDocumentMessage struct {
	Envelope
	KnownSnapshotID      *model.SnapshotID  `json:"known_snapshot_id,omitempty"`
	KnownLastOperationID *model.OperationID `json:"known_last_operation_id,omitempty"`
}

// Validate validates the subscribe request.
func (m SubscribeDocumentMessage) Validate() error {
	if err := m.Envelope.Validate(MessageTypeSubscribeDoc, true); err != nil {
		return err
	}
	return nil
}

// SubscribeAckMessage confirms subscription mode for one document stream.
type SubscribeAckMessage struct {
	Envelope
	SubscriptionMode string `json:"subscription_mode"`
}

// RequestCatchupMessage asks the server for snapshot-plus-delta state transfer.
type RequestCatchupMessage struct {
	Envelope
	KnownSnapshotID      *model.SnapshotID  `json:"known_snapshot_id,omitempty"`
	KnownLastOperationID *model.OperationID `json:"known_last_operation_id,omitempty"`
}

// Validate validates the catch-up request envelope.
func (m RequestCatchupMessage) Validate() error {
	return m.Envelope.Validate(MessageTypeRequestCatchup, true)
}

// CatchupSnapshotMessage transfers one snapshot baseline.
type CatchupSnapshotMessage struct {
	Envelope
	Snapshot CatchupSnapshotPayload `json:"snapshot"`
}

// CatchupSnapshotPayload carries derived snapshot state and replay watermark.
type CatchupSnapshotPayload struct {
	SnapshotID              model.SnapshotID `json:"snapshot_id"`
	LastIncludedOperationID model.OperationID `json:"last_included_operation_id"`
	State                   any               `json:"state"`
}

// CatchupOperationsMessage transfers one batch of later operations.
type CatchupOperationsMessage struct {
	Envelope
	CatchupID              string            `json:"catchup_id"`
	BatchIndex             int               `json:"batch_index"`
	HasMore                bool              `json:"has_more"`
	LastOperationIDInBatch model.OperationID `json:"last_operation_id_in_batch,omitempty"`
	Operations             []model.Operation `json:"operations"`
}

// CatchupCompleteMessage confirms the end of one catch-up transfer.
type CatchupCompleteMessage struct {
	Envelope
	LastOperationID model.OperationID `json:"last_operation_id,omitempty"`
}

// UpdateDocumentTitleMessage updates lightweight document metadata outside the CRDT model.
type UpdateDocumentTitleMessage struct {
	Envelope
	Title string `json:"title"`
}

// Validate validates one title update request.
func (m UpdateDocumentTitleMessage) Validate() error {
	if err := m.Envelope.Validate(MessageTypeUpdateTitle, true); err != nil {
		return err
	}
	if m.Title == "" {
		return fieldError("title", ErrEmptyTitle)
	}
	return nil
}

// DocumentTitleChangedMessage reports one accepted title change.
type DocumentTitleChangedMessage struct {
	Envelope
	Title             string        `json:"title"`
	LastEditorActorID model.ActorID `json:"last_editor_actor_id,omitempty"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

// PresencePosition anchors one caret or selection endpoint to a visible element identity.
type PresencePosition struct {
	ElementID     model.ElementID `json:"element_id,omitempty"`
	Offset        int             `json:"offset,omitempty"`
	FallbackIndex int             `json:"fallback_index,omitempty"`
}

// PresencePayload captures ephemeral browser presence state.
type PresencePayload struct {
	ActorID             model.ActorID    `json:"actor_id"`
	SessionID           string           `json:"session_id"`
	DisplayName         string           `json:"display_name,omitempty"`
	CursorAnchor        PresencePosition `json:"cursor_anchor"`
	CursorFocus         PresencePosition `json:"cursor_focus"`
	SelectionDirection  string           `json:"selection_direction,omitempty"`
	IsCollapsed         bool             `json:"is_collapsed"`
	LastSeenAt          time.Time        `json:"last_seen_at"`
}

// Validate validates one presence payload.
func (p PresencePayload) Validate() error {
	if p.ActorID == "" {
		return fieldError("actor_id", ErrEmptyActorID)
	}
	if p.SessionID == "" {
		return fieldError("session_id", ErrEmptySessionID)
	}
	if err := p.CursorAnchor.Validate(); err != nil {
		return fieldError("cursor_anchor", err)
	}
	if err := p.CursorFocus.Validate(); err != nil {
		return fieldError("cursor_focus", err)
	}
	if p.LastSeenAt.IsZero() {
		return fieldError("last_seen_at", errors.New("last_seen_at must not be zero"))
	}
	return nil
}

// Validate validates one presence position.
func (p PresencePosition) Validate() error {
	if p.ElementID == "" && p.FallbackIndex < 0 {
		return errors.New("presence position must include element_id or fallback_index")
	}
	if p.Offset < 0 {
		return errors.New("presence position offset must not be negative")
	}
	if p.FallbackIndex < 0 && p.ElementID == "" {
		return errors.New("presence position fallback_index must not be negative")
	}
	return nil
}

// PresenceUpdateMessage sends one ephemeral presence update.
type PresenceUpdateMessage struct {
	Envelope
	Presence PresencePayload `json:"presence"`
}

// Validate validates one presence update request.
func (m PresenceUpdateMessage) Validate() error {
	if err := m.Envelope.Validate(MessageTypePresenceUpdate, true); err != nil {
		return err
	}
	if err := m.Presence.Validate(); err != nil {
		return err
	}
	if m.Presence.ActorID == "" || m.Presence.SessionID == "" {
		return fieldError("presence", errors.New("presence actor_id and session_id must not be empty"))
	}
	return nil
}

// PresenceSnapshotMessage reports the current ephemeral collaborator set for one document.
type PresenceSnapshotMessage struct {
	Envelope
	Collaborators []PresencePayload `json:"collaborators"`
}

// PresenceBroadcastMessage reports one accepted presence update or removal.
type PresenceBroadcastMessage struct {
	Envelope
	Presence PresencePayload `json:"presence"`
	Removed  bool            `json:"removed,omitempty"`
}

// BroadcastOperationMessage delivers one accepted operation to subscribers.
type BroadcastOperationMessage struct {
	Envelope
	Operation model.Operation `json:"operation"`
}

// ErrorMessage reports one rejected or failed protocol action.
type ErrorMessage struct {
	Envelope
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

// SubmitOperationMessage submits one local CRDT operation.
type SubmitOperationMessage struct {
	Envelope
	Operation model.Operation `json:"operation"`
}

// Validate validates the submit-operation message.
func (m SubmitOperationMessage) Validate() error {
	if err := m.Envelope.Validate(MessageTypeSubmitOperation, true); err != nil {
		return err
	}
	if err := m.Operation.Validate(); err != nil {
		return fieldError("operation", err)
	}
	if m.Operation.DocumentID != m.DocumentID {
		return fieldError("operation.document_id", fmt.Errorf("operation document_id must match envelope document_id"))
	}
	return nil
}

// ValidationError identifies one protocol validation failure.
type ValidationError struct {
	Field   string
	Message string
	Cause   error
}

// Error returns a human-readable validation failure.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Unwrap exposes the underlying failure mode.
func (e *ValidationError) Unwrap() error {
	return e.Cause
}

func fieldError(field string, err error) error {
	return &ValidationError{
		Field:   field,
		Message: err.Error(),
		Cause:   err,
	}
}
