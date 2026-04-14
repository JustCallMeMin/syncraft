package protocol

import (
	"errors"
	"testing"
	"time"

	"github.com/JustCallMeMin/syncraft/internal/core/model"
)

func TestClientHelloValidate(t *testing.T) {
	msg := ClientHelloMessage{
		Envelope: Envelope{
			ProtocolVersion: VersionV1,
			MessageType:     MessageTypeClientHello,
			SessionID:       "sess_1",
			MessageID:       "msg_1",
		},
		ActorID: "actor_1",
	}

	if err := msg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestClientHelloValidateRejectsMissingActorID(t *testing.T) {
	msg := ClientHelloMessage{
		Envelope: Envelope{
			ProtocolVersion: VersionV1,
			MessageType:     MessageTypeClientHello,
			SessionID:       "sess_1",
			MessageID:       "msg_1",
		},
	}

	err := msg.Validate()
	if !errors.Is(err, ErrEmptyActorID) {
		t.Fatalf("Validate() error = %v, want ErrEmptyActorID", err)
	}
}

func TestSubmitOperationValidateRejectsDocumentMismatch(t *testing.T) {
	msg := SubmitOperationMessage{
		Envelope: Envelope{
			ProtocolVersion: VersionV1,
			MessageType:     MessageTypeSubmitOperation,
			DocumentID:      "doc_1",
			SessionID:       "sess_1",
			MessageID:       "msg_1",
		},
		Operation: model.Operation{
			DocumentID:   "doc_2",
			OperationID:  "op_1",
			ActorID:      "actor_1",
			ActorCounter: 1,
			Type:         model.OperationTypeInsert,
			InsertPayload: &model.InsertPayload{
				ElementID: "elem_1",
				Value:     "a",
			},
		},
	}

	if err := msg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want non-nil")
	}
}

func TestRequestCatchupValidate(t *testing.T) {
	msg := RequestCatchupMessage{
		Envelope: Envelope{
			ProtocolVersion: VersionV1,
			MessageType:     MessageTypeRequestCatchup,
			DocumentID:      "doc_1",
			SessionID:       "sess_1",
			MessageID:       "msg_1",
		},
	}

	if err := msg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestUpdateDocumentTitleValidate(t *testing.T) {
	msg := UpdateDocumentTitleMessage{
		Envelope: Envelope{
			ProtocolVersion: VersionV1,
			MessageType:     MessageTypeUpdateTitle,
			DocumentID:      "doc_1",
			SessionID:       "sess_1",
			MessageID:       "msg_1",
		},
		Title: "Shared Notes",
	}
	if err := msg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestPresenceUpdateValidate(t *testing.T) {
	msg := PresenceUpdateMessage{
		Envelope: Envelope{
			ProtocolVersion: VersionV1,
			MessageType:     MessageTypePresenceUpdate,
			DocumentID:      "doc_1",
			SessionID:       "sess_1",
			MessageID:       "msg_1",
		},
		Presence: PresencePayload{
			ActorID:     "actor_1",
			SessionID:   "sess_1",
			CursorAnchor: PresencePosition{FallbackIndex: 0},
			CursorFocus:  PresencePosition{FallbackIndex: 0},
			IsCollapsed: true,
			LastSeenAt:  time.Now().UTC(),
		},
	}
	if err := msg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}
