package harness

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/JustCallMeMin/syncraft/internal/backend/live"
	"github.com/JustCallMeMin/syncraft/internal/backend/persistence"
	"github.com/JustCallMeMin/syncraft/internal/backend/protocol"
	"github.com/JustCallMeMin/syncraft/internal/client/reconnect"
	"github.com/JustCallMeMin/syncraft/internal/core/engine"
	"github.com/JustCallMeMin/syncraft/internal/core/model"
)

// TestPermutationHarnessPreservesConvergence proves representative delivery permutations converge.
func TestPermutationHarnessPreservesConvergence(t *testing.T) {
	permutedOps := []model.Operation{
		harnessInsertOp("doc_perm", "op_1", "actor_a", 1, "elem_1", "a", nil, nil),
		harnessInsertOp("doc_perm", "op_2", "actor_b", 1, "elem_2", "b", harnessElementID("elem_1"), nil),
		harnessInsertOp("doc_perm", "op_3", "actor_c", 1, "elem_3", "c", harnessElementID("elem_1"), nil),
		harnessDeleteOp("doc_perm", "op_4", "actor_d", 1, "elem_1"),
	}

	permutations := harnessPermutations(permutedOps)
	for index, sequence := range permutations {
		doc := harnessNewDocument(t, "doc_perm")
		for _, op := range sequence {
			if _, err := doc.Apply(op); err != nil {
				t.Fatalf("Apply(%s) error = %v, want nil", op.OperationID, err)
			}
		}
		if got := doc.VisibleText(); got != "bc" {
			t.Fatalf("permutation %d VisibleText() = %q, want %q", index, got, "bc")
		}
		if doc.PendingInsertCount() != 0 {
			t.Fatalf("permutation %d PendingInsertCount() = %d, want 0", index, doc.PendingInsertCount())
		}
		if doc.PendingDeleteCount() != 0 {
			t.Fatalf("permutation %d PendingDeleteCount() = %d, want 0", index, doc.PendingDeleteCount())
		}
	}
}

// TestDuplicateDeliveryHarnessPreservesVisibleState proves duplicate operations stay harmless.
func TestDuplicateDeliveryHarnessPreservesVisibleState(t *testing.T) {
	sequences := [][]model.Operation{
		{
			harnessInsertOp("doc_dup", "op_1", "actor_a", 1, "elem_1", "a", nil, nil),
			harnessInsertOp("doc_dup", "op_1", "actor_a", 1, "elem_1", "a", nil, nil),
			harnessDeleteOp("doc_dup", "op_2", "actor_b", 1, "elem_1"),
			harnessDeleteOp("doc_dup", "op_2", "actor_b", 1, "elem_1"),
		},
		{
			harnessDeleteOp("doc_dup", "op_2", "actor_b", 1, "elem_1"),
			harnessDeleteOp("doc_dup", "op_2", "actor_b", 1, "elem_1"),
			harnessInsertOp("doc_dup", "op_1", "actor_a", 1, "elem_1", "a", nil, nil),
			harnessInsertOp("doc_dup", "op_1", "actor_a", 1, "elem_1", "a", nil, nil),
		},
	}

	for index, sequence := range sequences {
		doc := harnessNewDocument(t, "doc_dup")
		for _, op := range sequence {
			if _, err := doc.Apply(op); err != nil {
				t.Fatalf("sequence %d Apply(%s) error = %v, want nil", index, op.OperationID, err)
			}
		}
		if got := doc.VisibleText(); got != "" {
			t.Fatalf("sequence %d VisibleText() = %q, want empty string", index, got)
		}
		if doc.PendingInsertCount() != 0 {
			t.Fatalf("sequence %d PendingInsertCount() = %d, want 0", index, doc.PendingInsertCount())
		}
		if doc.PendingDeleteCount() != 0 {
			t.Fatalf("sequence %d PendingDeleteCount() = %d, want 0", index, doc.PendingDeleteCount())
		}
	}
}

// TestRestartRecoveryHarnessMatchesContinuousReplica proves rebuild matches continuous apply.
func TestRestartRecoveryHarnessMatchesContinuousReplica(t *testing.T) {
	ctx := context.Background()
	store := harnessNewStore(t)
	continuous := harnessNewDocument(t, "doc_recovery")
	operations := []model.Operation{
		harnessInsertOp("doc_recovery", "op_1", "actor_a", 1, "elem_1", "a", nil, nil),
		harnessInsertOp("doc_recovery", "op_2", "actor_b", 1, "elem_2", "b", harnessElementID("elem_1"), nil),
		harnessDeleteOp("doc_recovery", "op_3", "actor_c", 1, "elem_1"),
	}

	for _, op := range operations {
		if _, err := continuous.Apply(op); err != nil {
			t.Fatalf("continuous Apply(%s) error = %v, want nil", op.OperationID, err)
		}
		if err := store.AppendOperation(ctx, op); err != nil {
			t.Fatalf("AppendOperation(%s) error = %v, want nil", op.OperationID, err)
		}
	}

	base := harnessNewDocument(t, "doc_recovery")
	if _, err := base.Apply(operations[0]); err != nil {
		t.Fatalf("base Apply(%s) error = %v, want nil", operations[0].OperationID, err)
	}
	if err := store.SaveSnapshot(ctx, persistence.SnapshotRecord{
		SnapshotID:              "snap_recovery",
		DocumentID:              "doc_recovery",
		LastIncludedOperationID: operations[0].OperationID,
		CreatedAt:               time.Now().UTC(),
		State:                   base.Snapshot(),
	}); err != nil {
		t.Fatalf("SaveSnapshot() error = %v, want nil", err)
	}

	rebuilt, err := store.RebuildDocument(ctx, "doc_recovery")
	if err != nil {
		t.Fatalf("RebuildDocument() error = %v, want nil", err)
	}
	if got := rebuilt.VisibleText(); got != continuous.VisibleText() {
		t.Fatalf("rebuilt VisibleText() = %q, want %q", got, continuous.VisibleText())
	}
}

// TestReconnectHarnessMatchesContinuouslyConnectedReplica proves reconnect catch-up matches live state.
func TestReconnectHarnessMatchesContinuouslyConnectedReplica(t *testing.T) {
	ctx := context.Background()
	service := live.NewService(harnessNewStore(t))
	continuous := harnessNewReplica(t, "doc_reconnect", "actor_continuous")
	reconnecting := harnessNewReplica(t, "doc_reconnect", "actor_reconnecting")

	harnessMustHello(t, ctx, service, "sess_author", "actor_author")
	harnessMustSubscribe(t, ctx, service, "sess_author", "doc_reconnect", nil)
	harnessMustHello(t, ctx, service, "sess_continuous", "actor_continuous")
	harnessMustSubscribe(t, ctx, service, "sess_continuous", "doc_reconnect", nil)

	applied := []model.Operation{
		harnessInsertOp("doc_reconnect", "op_1", "actor_author", 1, "elem_1", "a", nil, nil),
		harnessInsertOp("doc_reconnect", "op_2", "actor_author", 2, "elem_2", "b", harnessElementID("elem_1"), nil),
	}
	for _, op := range applied {
		harnessMustSubmit(t, ctx, service, "sess_author", op)
		if err := continuous.ApplyBroadcastOperation(harnessLastBroadcast(t, service, "sess_continuous")); err != nil {
			t.Fatalf("continuous ApplyBroadcastOperation(%s) error = %v, want nil", op.OperationID, err)
		}
	}

	harnessMustHello(t, ctx, service, "sess_reconnecting", "actor_reconnecting")
	_, knownLast := reconnecting.StateReference()
	ack := harnessMustSubscribe(t, ctx, service, "sess_reconnecting", "doc_reconnect", knownLast)
	if err := reconnecting.HandleSubscribeAck(*ack); err != nil {
		t.Fatalf("HandleSubscribeAck() error = %v, want nil", err)
	}

	events := harnessMustCatchup(t, ctx, service, "sess_reconnecting", "doc_reconnect", knownLast)
	if err := reconnecting.ApplyCatchupSnapshot(events[0].(protocol.CatchupSnapshotMessage)); err != nil {
		t.Fatalf("ApplyCatchupSnapshot() error = %v, want nil", err)
	}
	if err := reconnecting.ApplyCatchupOperations(events[1].(protocol.CatchupOperationsMessage)); err != nil {
		t.Fatalf("ApplyCatchupOperations() error = %v, want nil", err)
	}
	if err := reconnecting.ApplyCatchupComplete(events[2].(protocol.CatchupCompleteMessage)); err != nil {
		t.Fatalf("ApplyCatchupComplete() error = %v, want nil", err)
	}

	next := harnessInsertOp("doc_reconnect", "op_3", "actor_author", 3, "elem_3", "c", harnessElementID("elem_2"), nil)
	harnessMustSubmit(t, ctx, service, "sess_author", next)
	if err := continuous.ApplyBroadcastOperation(harnessLastBroadcast(t, service, "sess_continuous")); err != nil {
		t.Fatalf("continuous ApplyBroadcastOperation(%s) error = %v, want nil", next.OperationID, err)
	}
	if err := reconnecting.ApplyBroadcastOperation(harnessLastBroadcast(t, service, "sess_reconnecting")); err != nil {
		t.Fatalf("reconnecting ApplyBroadcastOperation(%s) error = %v, want nil", next.OperationID, err)
	}

	if got, want := reconnecting.VisibleText(), continuous.VisibleText(); got != want {
		t.Fatalf("reconnecting VisibleText() = %q, want %q", got, want)
	}
}

// harnessNewDocument constructs one engine document for test scenarios.
func harnessNewDocument(t *testing.T, documentID model.DocumentID) *engine.Document {
	t.Helper()
	doc, err := engine.NewDocument(documentID)
	if err != nil {
		t.Fatalf("NewDocument() error = %v, want nil", err)
	}
	return doc
}

// harnessNewStore constructs one file-backed persistence store for scenario tests.
func harnessNewStore(t *testing.T) *persistence.FileStore {
	t.Helper()
	store, err := persistence.NewFileStore(filepath.Join(t.TempDir(), "store"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v, want nil", err)
	}
	return store
}

// harnessNewReplica constructs one reconnect-capable client replica for scenario tests.
func harnessNewReplica(t *testing.T, documentID model.DocumentID, actorID model.ActorID) *reconnect.Replica {
	t.Helper()
	replica, err := reconnect.NewReplica(documentID, actorID)
	if err != nil {
		t.Fatalf("NewReplica() error = %v, want nil", err)
	}
	return replica
}

// harnessMustHello registers one live session for a scenario test.
func harnessMustHello(t *testing.T, ctx context.Context, service *live.Service, sessionID string, actorID model.ActorID) {
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

// harnessMustSubscribe subscribes one scenario-test session to one document stream.
func harnessMustSubscribe(t *testing.T, ctx context.Context, service *live.Service, sessionID string, documentID model.DocumentID, knownLast *model.OperationID) *protocol.SubscribeAckMessage {
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

// harnessMustCatchup runs one reconnect catch-up transfer for a subscribed session.
func harnessMustCatchup(t *testing.T, ctx context.Context, service *live.Service, sessionID string, documentID model.DocumentID, knownLast *model.OperationID) []any {
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

// harnessMustSubmit sends one accepted operation through the live service.
func harnessMustSubmit(t *testing.T, ctx context.Context, service *live.Service, sessionID string, op model.Operation) {
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

// harnessLastBroadcast returns the latest broadcast operation observed by one session.
func harnessLastBroadcast(t *testing.T, service *live.Service, sessionID string) protocol.BroadcastOperationMessage {
	t.Helper()
	outbox := service.SessionOutbox(sessionID)
	last, ok := outbox[len(outbox)-1].(protocol.BroadcastOperationMessage)
	if !ok {
		t.Fatalf("last outbox message = %T, want BroadcastOperationMessage", outbox[len(outbox)-1])
	}
	return last
}

// harnessInsertOp constructs one insert operation for scenario tests.
func harnessInsertOp(documentID model.DocumentID, operationID model.OperationID, actorID model.ActorID, counter uint64, elementID model.ElementID, value string, left *model.ElementID, right *model.ElementID) model.Operation {
	return model.Operation{
		DocumentID:   documentID,
		OperationID:  operationID,
		ActorID:      actorID,
		ActorCounter: counter,
		Type:         model.OperationTypeInsert,
		InsertPayload: &model.InsertPayload{
			ElementID:     elementID,
			Value:         value,
			LeftOriginID:  left,
			RightOriginID: right,
		},
	}
}

// harnessDeleteOp constructs one delete operation for scenario tests.
func harnessDeleteOp(documentID model.DocumentID, operationID model.OperationID, actorID model.ActorID, counter uint64, target model.ElementID) model.Operation {
	return model.Operation{
		DocumentID:   documentID,
		OperationID:  operationID,
		ActorID:      actorID,
		ActorCounter: counter,
		Type:         model.OperationTypeDelete,
		DeletePayload: &model.DeletePayload{
			TargetElementID: target,
		},
	}
}

// harnessElementID returns one reusable element identifier pointer.
func harnessElementID(id model.ElementID) *model.ElementID {
	return &id
}

// harnessPermutations returns all orderings for one short operation slice.
func harnessPermutations(ops []model.Operation) [][]model.Operation {
	out := make([][]model.Operation, 0)
	current := make([]model.Operation, len(ops))
	copy(current, ops)
	harnessPermuteRecursive(current, 0, &out)
	return out
}

// harnessPermuteRecursive appends recursive swap permutations into the output slice.
func harnessPermuteRecursive(ops []model.Operation, index int, out *[][]model.Operation) {
	if index == len(ops) {
		permutation := make([]model.Operation, len(ops))
		copy(permutation, ops)
		*out = append(*out, permutation)
		return
	}
	for next := index; next < len(ops); next++ {
		ops[index], ops[next] = ops[next], ops[index]
		harnessPermuteRecursive(ops, index+1, out)
		ops[index], ops[next] = ops[next], ops[index]
	}
}
