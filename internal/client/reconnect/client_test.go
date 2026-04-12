package reconnect

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/JustCallMeMin/syncraft/internal/backend/live"
	"github.com/JustCallMeMin/syncraft/internal/backend/persistence"
	"github.com/JustCallMeMin/syncraft/internal/backend/protocol"
	"github.com/JustCallMeMin/syncraft/internal/core/model"
)

func TestReplicaReconnectBootstrapAndLiveResume(t *testing.T) {
	ctx := context.Background()
	service := newService(t)

	author := mustReplica(t, "doc_1", "actor_author")
	reader := mustReplica(t, "doc_1", "actor_reader")

	mustHello(t, ctx, service, "sess_author", "actor_author")
	mustSubscribe(t, ctx, service, "sess_author", "doc_1", nil)

	mustSubmit(t, ctx, service, "sess_author", insertOp("doc_1", "op_1", "actor_author", 1, "elem_1", "a"))
	mustSubmit(t, ctx, service, "sess_author", insertOp("doc_1", "op_2", "actor_author", 2, "elem_2", "b"))

	mustHello(t, ctx, service, "sess_reader", "actor_reader")
	_, knownLast := reader.StateReference()
	ack := mustSubscribe(t, ctx, service, "sess_reader", "doc_1", knownLast)
	if ack.SubscriptionMode != protocol.SubscriptionModeSnapshotGap {
		t.Fatalf("SubscriptionMode = %s, want %s", ack.SubscriptionMode, protocol.SubscriptionModeSnapshotGap)
	}
	if err := reader.HandleSubscribeAck(*ack); err != nil {
		t.Fatalf("HandleSubscribeAck() error = %v, want nil", err)
	}

	events := mustCatchup(t, ctx, service, "sess_reader", "doc_1", knownLast)
	if err := reader.ApplyCatchupSnapshot(events[0].(protocol.CatchupSnapshotMessage)); err != nil {
		t.Fatalf("ApplyCatchupSnapshot() error = %v, want nil", err)
	}
	if err := reader.ApplyCatchupOperations(events[1].(protocol.CatchupOperationsMessage)); err != nil {
		t.Fatalf("ApplyCatchupOperations() error = %v, want nil", err)
	}
	if err := reader.ApplyCatchupComplete(events[2].(protocol.CatchupCompleteMessage)); err != nil {
		t.Fatalf("ApplyCatchupComplete() error = %v, want nil", err)
	}
	if !reader.LiveSubscriptionReady() {
		t.Fatal("LiveSubscriptionReady() = false, want true")
	}
	if got := reader.VisibleText(); got != "ab" {
		t.Fatalf("VisibleText() after catchup = %q, want %q", got, "ab")
	}

	mustSubmit(t, ctx, service, "sess_author", insertOp("doc_1", "op_3", "actor_author", 3, "elem_3", "c"))
	outbox := service.SessionOutbox("sess_reader")
	last, ok := outbox[len(outbox)-1].(protocol.BroadcastOperationMessage)
	if !ok {
		t.Fatalf("last outbox message = %T, want BroadcastOperationMessage", outbox[len(outbox)-1])
	}
	if err := reader.ApplyBroadcastOperation(last); err != nil {
		t.Fatalf("ApplyBroadcastOperation() error = %v, want nil", err)
	}
	if got := reader.VisibleText(); got != "abc" {
		t.Fatalf("VisibleText() after live resume = %q, want %q", got, "abc")
	}

	_ = author
}

func mustReplica(t *testing.T, documentID model.DocumentID, actorID model.ActorID) *Replica {
	t.Helper()
	replica, err := NewReplica(documentID, actorID)
	if err != nil {
		t.Fatalf("NewReplica() error = %v, want nil", err)
	}
	return replica
}

func newService(t *testing.T) *live.Service {
	t.Helper()
	store, err := persistence.NewFileStore(filepath.Join(t.TempDir(), "store"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v, want nil", err)
	}
	return live.NewService(store)
}

func mustHello(t *testing.T, ctx context.Context, service *live.Service, sessionID string, actorID model.ActorID) {
	t.Helper()
	errMsg, err := service.HandleClientHello(ctx, protocol.ClientHelloMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeClientHello,
			SessionID:       sessionID,
			MessageID:       "hello_" + sessionID,
		},
		ActorID: actorID,
	})
	if err != nil || errMsg != nil {
		t.Fatalf("HandleClientHello() err = %v, errMsg = %+v, want nil", err, errMsg)
	}
}

func mustSubscribe(t *testing.T, ctx context.Context, service *live.Service, sessionID string, documentID model.DocumentID, knownLast *model.OperationID) *protocol.SubscribeAckMessage {
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
	return ack
}

func mustCatchup(t *testing.T, ctx context.Context, service *live.Service, sessionID string, documentID model.DocumentID, knownLast *model.OperationID) []any {
	t.Helper()
	events, errMsg, err := service.HandleRequestCatchup(ctx, protocol.RequestCatchupMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeRequestCatchup,
			DocumentID:      documentID,
			SessionID:       sessionID,
			MessageID:       "catchup_" + sessionID,
		},
		KnownLastOperationID: knownLast,
	})
	if err != nil || errMsg != nil {
		t.Fatalf("HandleRequestCatchup() err = %v, errMsg = %+v, want nil", err, errMsg)
	}
	return events
}

func mustSubmit(t *testing.T, ctx context.Context, service *live.Service, sessionID string, op model.Operation) {
	t.Helper()
	_, errMsg, err := service.HandleSubmit(ctx, protocol.SubmitOperationMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeSubmitOperation,
			DocumentID:      op.DocumentID,
			SessionID:       sessionID,
			MessageID:       "submit_" + string(op.OperationID),
		},
		Operation: op,
	})
	if err != nil || errMsg != nil {
		t.Fatalf("HandleSubmit() err = %v, errMsg = %+v, want nil", err, errMsg)
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
