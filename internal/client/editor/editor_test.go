package editor

import (
	"testing"

	"github.com/JustCallMeMin/syncraft/internal/backend/protocol"
	"github.com/JustCallMeMin/syncraft/internal/core/engine"
	"github.com/JustCallMeMin/syncraft/internal/core/model"
)

func TestInsertTextAtOptimisticallyAppliesVisibleText(t *testing.T) {
	session := newTestSession(t)

	ops, err := session.InsertTextAt(0, "ab")
	if err != nil {
		t.Fatalf("InsertTextAt() error = %v, want nil", err)
	}
	if len(ops) != 2 {
		t.Fatalf("len(ops) = %d, want 2", len(ops))
	}
	if got := session.ViewState().Text; got != "ab" {
		t.Fatalf("ViewState().Text = %q, want %q", got, "ab")
	}
}

func TestDeleteAtOptimisticallyRemovesVisibleRune(t *testing.T) {
	session := newTestSession(t)
	if _, err := session.InsertTextAt(0, "ab"); err != nil {
		t.Fatalf("InsertTextAt() error = %v, want nil", err)
	}

	op, err := session.DeleteAt(0)
	if err != nil {
		t.Fatalf("DeleteAt() error = %v, want nil", err)
	}
	if op.Type != model.OperationTypeDelete {
		t.Fatalf("DeleteAt() type = %s, want %s", op.Type, model.OperationTypeDelete)
	}
	if got := session.ViewState().Text; got != "b" {
		t.Fatalf("ViewState().Text = %q, want %q", got, "b")
	}
}

func TestApplyRemoteBroadcastSafelyUpdatesVisibleText(t *testing.T) {
	session := newTestSession(t)
	if _, err := session.InsertTextAt(0, "a"); err != nil {
		t.Fatalf("InsertTextAt() error = %v, want nil", err)
	}

	base := model.ElementID("elem_actor_local_1")
	remote := protocol.BroadcastOperationMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeBroadcastOp,
			DocumentID:      "doc_1",
			SessionID:       "server",
			MessageID:       "broadcast_remote",
		},
		Operation: model.Operation{
			DocumentID:   "doc_1",
			OperationID:  "op_remote_1",
			ActorID:      "actor_remote",
			ActorCounter: 1,
			Type:         model.OperationTypeInsert,
			InsertPayload: &model.InsertPayload{
				ElementID:    "elem_remote_1",
				Value:        "b",
				LeftOriginID: &base,
			},
		},
	}
	if err := session.ApplyRemoteBroadcast(remote); err != nil {
		t.Fatalf("ApplyRemoteBroadcast() error = %v, want nil", err)
	}
	if got := session.ViewState().Text; got != "ab" {
		t.Fatalf("ViewState().Text = %q, want %q", got, "ab")
	}
}

func TestHandleSubscribeAckTracksReconnectState(t *testing.T) {
	session := newTestSession(t)

	if err := session.HandleSubscribeAck(protocol.SubscribeAckMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeSubscribeAck,
			DocumentID:      "doc_1",
			SessionID:       "sess_1",
			MessageID:       "ack_1",
		},
		SubscriptionMode: protocol.SubscriptionModeSnapshotGap,
	}); err != nil {
		t.Fatalf("HandleSubscribeAck(snapshot_then_delta) error = %v, want nil", err)
	}
	if got := session.ViewState().ConnectionState; got != ConnectionStateCatchingUp {
		t.Fatalf("ConnectionState = %s, want %s", got, ConnectionStateCatchingUp)
	}

	if err := session.ApplyCatchupSnapshot(protocol.CatchupSnapshotMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeCatchupSnapshot,
			DocumentID:      "doc_1",
			SessionID:       "server",
			MessageID:       "snap_1",
		},
		Snapshot: protocol.CatchupSnapshotPayload{
			SnapshotID:              "snap_1",
			LastIncludedOperationID: "",
			State:                   newEmptySnapshot(),
		},
	}); err != nil {
		t.Fatalf("ApplyCatchupSnapshot() error = %v, want nil", err)
	}
	if err := session.ApplyCatchupComplete(protocol.CatchupCompleteMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeCatchupComplete,
			DocumentID:      "doc_1",
			SessionID:       "server",
			MessageID:       "complete_1",
		},
	}); err != nil {
		t.Fatalf("ApplyCatchupComplete() error = %v, want nil", err)
	}
	if got := session.ViewState().ConnectionState; got != ConnectionStateLive {
		t.Fatalf("ConnectionState = %s, want %s", got, ConnectionStateLive)
	}
}

func TestDuplicateRemoteBroadcastIsHarmless(t *testing.T) {
	session := newTestSession(t)
	ops, err := session.InsertTextAt(0, "a")
	if err != nil {
		t.Fatalf("InsertTextAt() error = %v, want nil", err)
	}
	duplicate := protocol.BroadcastOperationMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeBroadcastOp,
			DocumentID:      "doc_1",
			SessionID:       "server",
			MessageID:       "broadcast_duplicate",
		},
		Operation: ops[0],
	}
	if err := session.ApplyRemoteBroadcast(duplicate); err != nil {
		t.Fatalf("ApplyRemoteBroadcast() error = %v, want nil", err)
	}
	if got := session.ViewState().Text; got != "a" {
		t.Fatalf("ViewState().Text = %q, want %q", got, "a")
	}
}

func newTestSession(t *testing.T) *Session {
	t.Helper()
	session, err := NewSession("doc_1", "actor_local")
	if err != nil {
		t.Fatalf("NewSession() error = %v, want nil", err)
	}
	return session
}

func newEmptySnapshot() any {
	return engine.Snapshot{
		DocumentID:      "doc_1",
		OrderedElements: []engine.ElementSnapshot{},
		Applied:         []model.Operation{},
		PendingInserts:  []model.Operation{},
		PendingDeletes:  []model.PendingDelete{},
	}
}
