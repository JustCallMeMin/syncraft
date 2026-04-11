package protocol

import (
	"errors"
	"fmt"

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
	MessageTypeError            = "error"
	SubscriptionModeLiveOnly    = "live_only"
	SubscriptionModeSnapshotGap = "snapshot_then_delta"
)

const (
	ErrorCodeUnsupportedProtocolVersion = "unsupported_protocol_version"
	ErrorCodeInvalidMessageType         = "invalid_message_type"
	ErrorCodeInvalidDocumentID          = "invalid_document_id"
	ErrorCodeInvalidOperation           = "invalid_operation"
	ErrorCodeUnauthorized               = "unauthorized"
	ErrorCodeInternalError              = "internal_error"
)

var (
	ErrUnsupportedProtocolVersion = errors.New("unsupported protocol version")
	ErrInvalidMessageType         = errors.New("invalid message type")
	ErrEmptySessionID             = errors.New("session_id must not be empty")
	ErrEmptyMessageID             = errors.New("message_id must not be empty")
	ErrEmptyActorID               = errors.New("actor_id must not be empty")
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
