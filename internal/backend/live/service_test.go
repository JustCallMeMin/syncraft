package live

import (
	"bytes"
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JustCallMeMin/syncraft/internal/backend/persistence"
	"github.com/JustCallMeMin/syncraft/internal/backend/protocol"
	"github.com/JustCallMeMin/syncraft/internal/core/model"
)

func TestHandleClientHelloAndSubscribe(t *testing.T) {
	service := newService(t)
	ctx := context.Background()

	errMsg, err := service.HandleClientHello(ctx, protocol.ClientHelloMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeClientHello,
			SessionID:       "sess_1",
			MessageID:       "msg_1",
		},
		ActorID: "actor_a",
	})
	if err != nil {
		t.Fatalf("HandleClientHello() error = %v, want nil", err)
	}
	if errMsg != nil {
		t.Fatalf("HandleClientHello() error message = %+v, want nil", errMsg)
	}

	ack, errMsg, err := service.HandleSubscribe(ctx, protocol.SubscribeDocumentMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeSubscribeDoc,
			DocumentID:      "doc_1",
			SessionID:       "sess_1",
			MessageID:       "msg_2",
		},
	})
	if err != nil {
		t.Fatalf("HandleSubscribe() error = %v, want nil", err)
	}
	if errMsg != nil {
		t.Fatalf("HandleSubscribe() error message = %+v, want nil", errMsg)
	}
	if ack == nil || ack.SubscriptionMode != protocol.SubscriptionModeLiveOnly {
		t.Fatalf("HandleSubscribe() ack = %+v, want live_only ack", ack)
	}
}

func TestHandleSubmitBroadcastsToSubscribersAndPersists(t *testing.T) {
	service := newService(t)
	ctx := context.Background()

	mustHello(t, ctx, service, "sess_1", "actor_a")
	mustHello(t, ctx, service, "sess_2", "actor_b")
	mustSubscribe(t, ctx, service, "sess_1", "doc_1")
	mustSubscribe(t, ctx, service, "sess_2", "doc_1")

	broadcasts, errMsg, err := service.HandleSubmit(ctx, protocol.SubmitOperationMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeSubmitOperation,
			DocumentID:      "doc_1",
			SessionID:       "sess_1",
			MessageID:       "msg_3",
		},
		Operation: insertOp("doc_1", "op_1", "actor_a", 1, "elem_1", "a"),
	})
	if err != nil {
		t.Fatalf("HandleSubmit() error = %v, want nil", err)
	}
	if errMsg != nil {
		t.Fatalf("HandleSubmit() error message = %+v, want nil", errMsg)
	}
	if len(broadcasts) != 2 {
		t.Fatalf("broadcast count = %d, want 2", len(broadcasts))
	}

	outbox1 := service.SessionOutbox("sess_1")
	outbox2 := service.SessionOutbox("sess_2")
	if len(outbox1) != 2 || len(outbox2) != 2 {
		t.Fatalf("outbox lengths = %d and %d, want 2 and 2", len(outbox1), len(outbox2))
	}
}

func TestHandleSubmitRejectsUnknownSession(t *testing.T) {
	service := newService(t)
	ctx := context.Background()

	_, errMsg, err := service.HandleSubmit(ctx, protocol.SubmitOperationMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeSubmitOperation,
			DocumentID:      "doc_1",
			SessionID:       "sess_missing",
			MessageID:       "msg_1",
		},
		Operation: insertOp("doc_1", "op_1", "actor_a", 1, "elem_1", "a"),
	})
	if err != nil {
		t.Fatalf("HandleSubmit() error = %v, want nil", err)
	}
	if errMsg == nil || errMsg.ErrorCode != protocol.ErrorCodeUnauthorized {
		t.Fatalf("HandleSubmit() error message = %+v, want unauthorized error", errMsg)
	}
}

func TestHandleSubmitRejectsActorMismatch(t *testing.T) {
	service := newService(t)
	ctx := context.Background()

	mustHello(t, ctx, service, "sess_1", "actor_a")
	mustSubscribe(t, ctx, service, "sess_1", "doc_1")

	_, errMsg, err := service.HandleSubmit(ctx, protocol.SubmitOperationMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeSubmitOperation,
			DocumentID:      "doc_1",
			SessionID:       "sess_1",
			MessageID:       "msg_1",
		},
		Operation: insertOp("doc_1", "op_1", "actor_b", 1, "elem_1", "a"),
	})
	if err != nil {
		t.Fatalf("HandleSubmit() error = %v, want nil", err)
	}
	if errMsg == nil || errMsg.ErrorCode != protocol.ErrorCodeInvalidOperation {
		t.Fatalf("HandleSubmit() error message = %+v, want invalid_operation", errMsg)
	}
}

func TestHandleSubmitDuplicateDoesNotBroadcastTwice(t *testing.T) {
	service := newService(t)
	ctx := context.Background()

	mustHello(t, ctx, service, "sess_1", "actor_a")
	mustSubscribe(t, ctx, service, "sess_1", "doc_1")

	msg := protocol.SubmitOperationMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeSubmitOperation,
			DocumentID:      "doc_1",
			SessionID:       "sess_1",
			MessageID:       "msg_1",
		},
		Operation: insertOp("doc_1", "op_1", "actor_a", 1, "elem_1", "a"),
	}
	if _, errMsg, err := service.HandleSubmit(ctx, msg); err != nil || errMsg != nil {
		t.Fatalf("first HandleSubmit() err = %v, errMsg = %+v, want nil", err, errMsg)
	}
	broadcasts, errMsg, err := service.HandleSubmit(ctx, msg)
	if err != nil {
		t.Fatalf("second HandleSubmit() error = %v, want nil", err)
	}
	if errMsg != nil {
		t.Fatalf("second HandleSubmit() errMsg = %+v, want nil", errMsg)
	}
	if len(broadcasts) != 0 {
		t.Fatalf("broadcast count = %d, want 0", len(broadcasts))
	}
}

func TestHandleSubscribeReturnsSnapshotThenDeltaWhenClientIsBehind(t *testing.T) {
	service := newService(t)
	ctx := context.Background()

	mustHello(t, ctx, service, "sess_1", "actor_a")
	mustSubscribeAndExpectMode(t, ctx, service, "sess_1", "doc_1", protocol.SubscriptionModeLiveOnly)

	msg := protocol.SubmitOperationMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeSubmitOperation,
			DocumentID:      "doc_1",
			SessionID:       "sess_1",
			MessageID:       "msg_1",
		},
		Operation: insertOp("doc_1", "op_1", "actor_a", 1, "elem_1", "a"),
	}
	if _, errMsg, err := service.HandleSubmit(ctx, msg); err != nil || errMsg != nil {
		t.Fatalf("HandleSubmit() err = %v, errMsg = %+v, want nil", err, errMsg)
	}

	mustHello(t, ctx, service, "sess_2", "actor_b")
	knownLast := model.OperationID("op_0")
	ack, errMsg, err := service.HandleSubscribe(ctx, protocol.SubscribeDocumentMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeSubscribeDoc,
			DocumentID:      "doc_1",
			SessionID:       "sess_2",
			MessageID:       "msg_2",
		},
		KnownLastOperationID: &knownLast,
	})
	if err != nil || errMsg != nil {
		t.Fatalf("HandleSubscribe() err = %v, errMsg = %+v, want nil", err, errMsg)
	}
	if ack.SubscriptionMode != protocol.SubscriptionModeSnapshotGap {
		t.Fatalf("SubscriptionMode = %s, want %s", ack.SubscriptionMode, protocol.SubscriptionModeSnapshotGap)
	}
}

func TestHandleRequestCatchupReturnsSnapshotOperationsAndComplete(t *testing.T) {
	service := newService(t)
	ctx := context.Background()

	mustHello(t, ctx, service, "sess_1", "actor_a")
	mustSubscribeAndExpectMode(t, ctx, service, "sess_1", "doc_1", protocol.SubscriptionModeLiveOnly)

	for i, op := range []model.Operation{
		insertOp("doc_1", "op_1", "actor_a", 1, "elem_1", "a"),
		insertOp("doc_1", "op_2", "actor_a", 2, "elem_2", "b"),
	} {
		msg := protocol.SubmitOperationMessage{
			Envelope: protocol.Envelope{
				ProtocolVersion: protocol.VersionV1,
				MessageType:     protocol.MessageTypeSubmitOperation,
				DocumentID:      "doc_1",
				SessionID:       "sess_1",
				MessageID:       "msg_submit",
			},
			Operation: op,
		}
		if _, errMsg, err := service.HandleSubmit(ctx, msg); err != nil || errMsg != nil {
			t.Fatalf("submit %d err = %v, errMsg = %+v, want nil", i, err, errMsg)
		}
	}

	mustHello(t, ctx, service, "sess_2", "actor_b")
	oldWatermark := model.OperationID("op_0")
	mustSubscribeAndExpectModeWithKnown(t, ctx, service, "sess_2", "doc_1", &oldWatermark, protocol.SubscriptionModeSnapshotGap)

	events, errMsg, err := service.HandleRequestCatchup(ctx, protocol.RequestCatchupMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeRequestCatchup,
			DocumentID:      "doc_1",
			SessionID:       "sess_2",
			MessageID:       "msg_catchup",
		},
		KnownLastOperationID: &oldWatermark,
	})
	if err != nil || errMsg != nil {
		t.Fatalf("HandleRequestCatchup() err = %v, errMsg = %+v, want nil", err, errMsg)
	}
	if len(events) != 3 {
		t.Fatalf("event count = %d, want 3", len(events))
	}

	if _, ok := events[0].(protocol.CatchupSnapshotMessage); !ok {
		t.Fatalf("events[0] = %T, want CatchupSnapshotMessage", events[0])
	}
	ops, ok := events[1].(protocol.CatchupOperationsMessage)
	if !ok {
		t.Fatalf("events[1] = %T, want CatchupOperationsMessage", events[1])
	}
	if len(ops.Operations) != 2 {
		t.Fatalf("operations count = %d, want 2", len(ops.Operations))
	}
	if _, ok := events[2].(protocol.CatchupCompleteMessage); !ok {
		t.Fatalf("events[2] = %T, want CatchupCompleteMessage", events[2])
	}
}

func TestHandleSubmitLogsAcceptedAndRejectedActions(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, nil))
	service := newService(t)
	service.SetLogger(logger)
	ctx := context.Background()

	_, errMsg, err := service.HandleSubmit(ctx, protocol.SubmitOperationMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeSubmitOperation,
			DocumentID:      "doc_1",
			SessionID:       "sess_missing",
			MessageID:       "msg_reject",
		},
		Operation: insertOp("doc_1", "op_1", "actor_a", 1, "elem_1", "a"),
	})
	if err != nil {
		t.Fatalf("HandleSubmit() reject error = %v, want nil", err)
	}
	if errMsg == nil {
		t.Fatal("HandleSubmit() reject errMsg = nil, want non-nil")
	}

	mustHello(t, ctx, service, "sess_1", "actor_a")
	mustSubscribe(t, ctx, service, "sess_1", "doc_1")
	_, errMsg, err = service.HandleSubmit(ctx, protocol.SubmitOperationMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeSubmitOperation,
			DocumentID:      "doc_1",
			SessionID:       "sess_1",
			MessageID:       "msg_accept",
		},
		Operation: insertOp("doc_1", "op_2", "actor_a", 2, "elem_2", "b"),
	})
	if err != nil || errMsg != nil {
		t.Fatalf("HandleSubmit() accept err = %v, errMsg = %+v, want nil", err, errMsg)
	}

	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "reject protocol action") {
		t.Fatalf("log output = %q, want reject protocol action entry", logOutput)
	}
	if !strings.Contains(logOutput, "accept operation") {
		t.Fatalf("log output = %q, want accept operation entry", logOutput)
	}
	if strings.Contains(logOutput, "operation_value") {
		t.Fatalf("log output = %q, want no raw text payload values", logOutput)
	}
}

func TestHandleUpdateTitlePersistsAndBroadcasts(t *testing.T) {
	service := newService(t)
	ctx := context.Background()

	mustHello(t, ctx, service, "sess_1", "actor_a")
	mustHello(t, ctx, service, "sess_2", "actor_b")
	mustSubscribe(t, ctx, service, "sess_1", "doc_1")
	mustSubscribe(t, ctx, service, "sess_2", "doc_1")

	changes, errMsg, err := service.HandleUpdateTitle(ctx, protocol.UpdateDocumentTitleMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeUpdateTitle,
			DocumentID:      "doc_1",
			SessionID:       "sess_1",
			MessageID:       "msg_title",
		},
		Title: "Planning Notes",
	})
	if err != nil || errMsg != nil {
		t.Fatalf("HandleUpdateTitle() err = %v, errMsg = %+v, want nil", err, errMsg)
	}
	if len(changes) != 2 {
		t.Fatalf("change count = %d, want 2", len(changes))
	}

	record, err := service.DocumentMetadata(ctx, "doc_1")
	if err != nil {
		t.Fatalf("DocumentMetadata() error = %v, want nil", err)
	}
	if record.Title != "Planning Notes" {
		t.Fatalf("Title = %q, want %q", record.Title, "Planning Notes")
	}
	if record.LastEditorActorID != "actor_a" {
		t.Fatalf("LastEditorActorID = %q, want %q", record.LastEditorActorID, "actor_a")
	}
}

func TestHandlePresenceUpdateBroadcastsAndExpires(t *testing.T) {
	service := newService(t)
	service.SetPresenceTTL(20 * time.Millisecond)
	ctx := context.Background()

	mustHello(t, ctx, service, "sess_1", "actor_a")
	mustHello(t, ctx, service, "sess_2", "actor_b")
	mustSubscribe(t, ctx, service, "sess_1", "doc_1")
	mustSubscribe(t, ctx, service, "sess_2", "doc_1")

	updates, errMsg, err := service.HandlePresenceUpdate(ctx, protocol.PresenceUpdateMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypePresenceUpdate,
			DocumentID:      "doc_1",
			SessionID:       "sess_1",
			MessageID:       "msg_presence",
		},
		Presence: protocol.PresencePayload{
			ActorID:    "actor_a",
			SessionID:  "sess_1",
			DisplayName: "A",
			CursorAnchor: protocol.PresencePosition{FallbackIndex: 0},
			CursorFocus:  protocol.PresencePosition{FallbackIndex: 0},
			IsCollapsed: true,
			LastSeenAt:  time.Now().UTC(),
		},
	})
	if err != nil || errMsg != nil {
		t.Fatalf("HandlePresenceUpdate() err = %v, errMsg = %+v, want nil", err, errMsg)
	}
	if len(updates) != 2 {
		t.Fatalf("update count = %d, want 2", len(updates))
	}
	if snapshot := service.PresenceSnapshot("doc_1"); len(snapshot) != 1 {
		t.Fatalf("PresenceSnapshot len = %d, want 1", len(snapshot))
	}

	time.Sleep(30 * time.Millisecond)
	if snapshot := service.PresenceSnapshot("doc_1"); len(snapshot) != 0 {
		t.Fatalf("PresenceSnapshot len after expiry = %d, want 0", len(snapshot))
	}
}

func TestDisconnectSessionRemovesPresence(t *testing.T) {
	service := newService(t)
	ctx := context.Background()

	mustHello(t, ctx, service, "sess_1", "actor_a")
	mustHello(t, ctx, service, "sess_2", "actor_b")
	mustSubscribe(t, ctx, service, "sess_1", "doc_1")
	mustSubscribe(t, ctx, service, "sess_2", "doc_1")

	_, errMsg, err := service.HandlePresenceUpdate(ctx, protocol.PresenceUpdateMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypePresenceUpdate,
			DocumentID:      "doc_1",
			SessionID:       "sess_1",
			MessageID:       "msg_presence",
		},
		Presence: protocol.PresencePayload{
			ActorID:    "actor_a",
			SessionID:  "sess_1",
			CursorAnchor: protocol.PresencePosition{FallbackIndex: 0},
			CursorFocus:  protocol.PresencePosition{FallbackIndex: 0},
			IsCollapsed: true,
			LastSeenAt:  time.Now().UTC(),
		},
	})
	if err != nil || errMsg != nil {
		t.Fatalf("HandlePresenceUpdate() err = %v, errMsg = %+v, want nil", err, errMsg)
	}

	removals := service.DisconnectSession("sess_1")
	if len(removals) != 1 {
		t.Fatalf("DisconnectSession removal count = %d, want 1", len(removals))
	}
	if !removals[0].Removed {
		t.Fatal("DisconnectSession removal flag = false, want true")
	}
	if snapshot := service.PresenceSnapshot("doc_1"); len(snapshot) != 0 {
		t.Fatalf("PresenceSnapshot len after disconnect = %d, want 0", len(snapshot))
	}
}

func newService(t *testing.T) *Service {
	t.Helper()
	store, err := persistence.NewFileStore(filepath.Join(t.TempDir(), "store"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v, want nil", err)
	}
	return NewService(store)
}

func mustHello(t *testing.T, ctx context.Context, service *Service, sessionID string, actorID model.ActorID) {
	t.Helper()
	errMsg, err := service.HandleClientHello(ctx, protocol.ClientHelloMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeClientHello,
			SessionID:       sessionID,
			MessageID:       "msg_" + sessionID,
		},
		ActorID: actorID,
	})
	if err != nil || errMsg != nil {
		t.Fatalf("HandleClientHello() err = %v, errMsg = %+v, want nil", err, errMsg)
	}
}

func mustSubscribe(t *testing.T, ctx context.Context, service *Service, sessionID string, documentID model.DocumentID) {
	t.Helper()
	mustSubscribeAndExpectMode(t, ctx, service, sessionID, documentID, protocol.SubscriptionModeLiveOnly)
}

func mustSubscribeAndExpectMode(t *testing.T, ctx context.Context, service *Service, sessionID string, documentID model.DocumentID, mode string) {
	t.Helper()
	mustSubscribeAndExpectModeWithKnown(t, ctx, service, sessionID, documentID, nil, mode)
}

func mustSubscribeAndExpectModeWithKnown(t *testing.T, ctx context.Context, service *Service, sessionID string, documentID model.DocumentID, knownLast *model.OperationID, mode string) {
	t.Helper()
	ack, errMsg, err := service.HandleSubscribe(ctx, protocol.SubscribeDocumentMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeSubscribeDoc,
			DocumentID:      documentID,
			SessionID:       sessionID,
			MessageID:       "sub_" + sessionID,
		},
		KnownLastOperationID: knownLast,
	})
	if err != nil || errMsg != nil {
		t.Fatalf("HandleSubscribe() err = %v, errMsg = %+v, want nil", err, errMsg)
	}
	if ack == nil || ack.SubscriptionMode != mode {
		t.Fatalf("SubscriptionMode = %+v, want %s", ack, mode)
	}
}

func insertOp(documentID model.DocumentID, operationID model.OperationID, actorID model.ActorID, counter uint64, elementID model.ElementID, value string) model.Operation {
	return model.Operation{
		DocumentID:   documentID,
		OperationID:  operationID,
		ActorID:      actorID,
		ActorCounter: counter,
		Type:         model.OperationTypeInsert,
		InsertPayload: &model.InsertPayload{
			ElementID: elementID,
			Value:     value,
		},
	}
}
