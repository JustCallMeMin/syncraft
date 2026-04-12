package engine

import (
	"testing"

	"github.com/JustCallMeMin/syncraft/internal/core/model"
)

func TestApplyInsertProjectsVisibleText(t *testing.T) {
	doc := newTestDocument(t)

	result, err := doc.Apply(insertOp("doc_1", "op_1", "actor_a", 1, "elem_1", "a", nil, nil))
	if err != nil {
		t.Fatalf("Apply() error = %v, want nil", err)
	}
	if result.Status != ApplyStatusApplied {
		t.Fatalf("Apply() status = %s, want %s", result.Status, ApplyStatusApplied)
	}
	if got := doc.VisibleText(); got != "a" {
		t.Fatalf("VisibleText() = %q, want %q", got, "a")
	}
}

func TestApplyDuplicateOperationIsHarmless(t *testing.T) {
	doc := newTestDocument(t)
	op := insertOp("doc_1", "op_1", "actor_a", 1, "elem_1", "a", nil, nil)

	if _, err := doc.Apply(op); err != nil {
		t.Fatalf("first Apply() error = %v, want nil", err)
	}
	result, err := doc.Apply(op)
	if err != nil {
		t.Fatalf("second Apply() error = %v, want nil", err)
	}
	if result.Status != ApplyStatusDuplicate {
		t.Fatalf("second Apply() status = %s, want %s", result.Status, ApplyStatusDuplicate)
	}
	if got := doc.VisibleText(); got != "a" {
		t.Fatalf("VisibleText() = %q, want %q", got, "a")
	}
}

func TestApplyDeleteTombstonesElement(t *testing.T) {
	doc := newTestDocument(t)

	if _, err := doc.Apply(insertOp("doc_1", "op_1", "actor_a", 1, "elem_1", "a", nil, nil)); err != nil {
		t.Fatalf("insert Apply() error = %v, want nil", err)
	}
	result, err := doc.Apply(deleteOp("doc_1", "op_2", "actor_b", 1, "elem_1"))
	if err != nil {
		t.Fatalf("delete Apply() error = %v, want nil", err)
	}
	if result.Status != ApplyStatusApplied {
		t.Fatalf("delete Apply() status = %s, want %s", result.Status, ApplyStatusApplied)
	}
	if got := doc.VisibleText(); got != "" {
		t.Fatalf("VisibleText() = %q, want empty string", got)
	}
}

func TestApplyDeleteBeforeInsertDefersThenResolves(t *testing.T) {
	doc := newTestDocument(t)

	result, err := doc.Apply(deleteOp("doc_1", "op_2", "actor_b", 1, "elem_1"))
	if err != nil {
		t.Fatalf("delete Apply() error = %v, want nil", err)
	}
	if result.Status != ApplyStatusDeferred {
		t.Fatalf("delete Apply() status = %s, want %s", result.Status, ApplyStatusDeferred)
	}
	if doc.PendingDeleteCount() != 1 {
		t.Fatalf("PendingDeleteCount() = %d, want 1", doc.PendingDeleteCount())
	}

	result, err = doc.Apply(insertOp("doc_1", "op_1", "actor_a", 1, "elem_1", "a", nil, nil))
	if err != nil {
		t.Fatalf("insert Apply() error = %v, want nil", err)
	}
	if result.Status != ApplyStatusApplied {
		t.Fatalf("insert Apply() status = %s, want %s", result.Status, ApplyStatusApplied)
	}
	if got := doc.VisibleText(); got != "" {
		t.Fatalf("VisibleText() = %q, want empty string", got)
	}
	if doc.PendingDeleteCount() != 0 {
		t.Fatalf("PendingDeleteCount() = %d, want 0", doc.PendingDeleteCount())
	}
}

func TestApplyInsertWithMissingOriginDefersThenResolves(t *testing.T) {
	doc := newTestDocument(t)
	left := model.ElementID("elem_1")

	result, err := doc.Apply(insertOp("doc_1", "op_2", "actor_b", 1, "elem_2", "b", &left, nil))
	if err != nil {
		t.Fatalf("deferred insert Apply() error = %v, want nil", err)
	}
	if result.Status != ApplyStatusDeferred {
		t.Fatalf("deferred insert Apply() status = %s, want %s", result.Status, ApplyStatusDeferred)
	}
	if doc.PendingInsertCount() != 1 {
		t.Fatalf("PendingInsertCount() = %d, want 1", doc.PendingInsertCount())
	}

	if _, err := doc.Apply(insertOp("doc_1", "op_1", "actor_a", 1, "elem_1", "a", nil, nil)); err != nil {
		t.Fatalf("origin insert Apply() error = %v, want nil", err)
	}
	if got := doc.VisibleText(); got != "ab" {
		t.Fatalf("VisibleText() = %q, want %q", got, "ab")
	}
	if doc.PendingInsertCount() != 0 {
		t.Fatalf("PendingInsertCount() = %d, want 0", doc.PendingInsertCount())
	}
}

func TestConcurrentInsertOrderingIsDeterministic(t *testing.T) {
	doc := newTestDocument(t)
	base := insertOp("doc_1", "op_1", "actor_base", 1, "elem_base", "x", nil, nil)

	if _, err := doc.Apply(base); err != nil {
		t.Fatalf("base Apply() error = %v, want nil", err)
	}

	left := model.ElementID("elem_base")
	a := insertOp("doc_1", "op_2", "actor_a", 1, "elem_a", "a", &left, nil)
	b := insertOp("doc_1", "op_3", "actor_b", 1, "elem_b", "b", &left, nil)

	if _, err := doc.Apply(b); err != nil {
		t.Fatalf("b Apply() error = %v, want nil", err)
	}
	if _, err := doc.Apply(a); err != nil {
		t.Fatalf("a Apply() error = %v, want nil", err)
	}

	if got := doc.VisibleText(); got != "xab" {
		t.Fatalf("VisibleText() = %q, want %q", got, "xab")
	}
}

func TestDuplicateDeferredDeleteRemainsHarmless(t *testing.T) {
	doc := newTestDocument(t)
	delete := deleteOp("doc_1", "op_2", "actor_b", 1, "elem_1")

	if _, err := doc.Apply(delete); err != nil {
		t.Fatalf("first delete Apply() error = %v, want nil", err)
	}
	result, err := doc.Apply(delete)
	if err != nil {
		t.Fatalf("second delete Apply() error = %v, want nil", err)
	}
	if result.Status != ApplyStatusDuplicate {
		t.Fatalf("second delete Apply() status = %s, want %s", result.Status, ApplyStatusDuplicate)
	}
}

func newTestDocument(t *testing.T) *Document {
	t.Helper()
	doc, err := NewDocument("doc_1")
	if err != nil {
		t.Fatalf("NewDocument() error = %v, want nil", err)
	}
	return doc
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
