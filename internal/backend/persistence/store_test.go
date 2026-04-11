package persistence

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/JustCallMeMin/syncraft/internal/core/engine"
	"github.com/JustCallMeMin/syncraft/internal/core/model"
)

func TestAppendOperationAndLoadOperationsAfter(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	op1 := insertOp("doc_1", "op_1", "actor_a", 1, "elem_1", "a", nil, nil)
	op2 := insertOp("doc_1", "op_2", "actor_b", 1, "elem_2", "b", nil, nil)

	mustAppend(t, store, ctx, op1)
	mustAppend(t, store, ctx, op2)

	ops, err := store.LoadOperationsAfter(ctx, "doc_1", "op_1")
	if err != nil {
		t.Fatalf("LoadOperationsAfter() error = %v, want nil", err)
	}
	if len(ops) != 1 || ops[0].OperationID != "op_2" {
		t.Fatalf("LoadOperationsAfter() = %+v, want only op_2", ops)
	}
}

func TestSaveSnapshotAndLoadLatestSnapshot(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	doc := newEngineDoc(t, "doc_1")

	if _, err := doc.Apply(insertOp("doc_1", "op_1", "actor_a", 1, "elem_1", "a", nil, nil)); err != nil {
		t.Fatalf("Apply() error = %v, want nil", err)
	}

	record := SnapshotRecord{
		SnapshotID:              "snap_1",
		DocumentID:              "doc_1",
		LastIncludedOperationID: "op_1",
		CreatedAt:               time.Now().UTC(),
		State:                   doc.Snapshot(),
	}

	if err := store.SaveSnapshot(ctx, record); err != nil {
		t.Fatalf("SaveSnapshot() error = %v, want nil", err)
	}

	got, err := store.LoadLatestSnapshot(ctx, "doc_1")
	if err != nil {
		t.Fatalf("LoadLatestSnapshot() error = %v, want nil", err)
	}
	if got.SnapshotID != "snap_1" {
		t.Fatalf("SnapshotID = %s, want snap_1", got.SnapshotID)
	}
	if got.State.DocumentID != "doc_1" {
		t.Fatalf("snapshot state document_id = %s, want doc_1", got.State.DocumentID)
	}
}

func TestRebuildDocumentFromSnapshotPlusLaterOperations(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	op1 := insertOp("doc_1", "op_1", "actor_a", 1, "elem_1", "a", nil, nil)
	op2 := insertOp("doc_1", "op_2", "actor_b", 1, "elem_2", "b", nil, nil)
	op3 := deleteOp("doc_1", "op_3", "actor_c", 1, "elem_1")

	doc := newEngineDoc(t, "doc_1")
	for _, op := range []model.Operation{op1, op2, op3} {
		if _, err := doc.Apply(op); err != nil {
			t.Fatalf("Apply(%s) error = %v, want nil", op.OperationID, err)
		}
	}

	mustAppend(t, store, ctx, op1)
	mustAppend(t, store, ctx, op2)
	mustAppend(t, store, ctx, op3)

	base := newEngineDoc(t, "doc_1")
	if _, err := base.Apply(op1); err != nil {
		t.Fatalf("Apply(op1) error = %v, want nil", err)
	}
	if err := store.SaveSnapshot(ctx, SnapshotRecord{
		SnapshotID:              "snap_1",
		DocumentID:              "doc_1",
		LastIncludedOperationID: "op_1",
		CreatedAt:               time.Now().UTC(),
		State:                   base.Snapshot(),
	}); err != nil {
		t.Fatalf("SaveSnapshot() error = %v, want nil", err)
	}

	rebuilt, err := store.RebuildDocument(ctx, "doc_1")
	if err != nil {
		t.Fatalf("RebuildDocument() error = %v, want nil", err)
	}
	if got := rebuilt.VisibleText(); got != doc.VisibleText() {
		t.Fatalf("VisibleText() = %q, want %q", got, doc.VisibleText())
	}
}

func newTestStore(t *testing.T) *FileStore {
	t.Helper()
	root := filepath.Join(t.TempDir(), "store")
	store, err := NewFileStore(root)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v, want nil", err)
	}
	return store
}

func newEngineDoc(t *testing.T, documentID model.DocumentID) *engine.Document {
	t.Helper()
	doc, err := engine.NewDocument(documentID)
	if err != nil {
		t.Fatalf("NewDocument() error = %v, want nil", err)
	}
	return doc
}

func mustAppend(t *testing.T, store *FileStore, ctx context.Context, op model.Operation) {
	t.Helper()
	if err := store.AppendOperation(ctx, op); err != nil {
		t.Fatalf("AppendOperation(%s) error = %v, want nil", op.OperationID, err)
	}
}

func insertOp(
	documentID model.DocumentID,
	operationID model.OperationID,
	actorID model.ActorID,
	counter uint64,
	elementID model.ElementID,
	value string,
	left *model.ElementID,
	right *model.ElementID,
) model.Operation {
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

func deleteOp(
	documentID model.DocumentID,
	operationID model.OperationID,
	actorID model.ActorID,
	counter uint64,
	target model.ElementID,
) model.Operation {
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
