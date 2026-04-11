package model

import (
	"errors"
	"testing"
)

func TestOperationValidateInsert(t *testing.T) {
	left := ElementID("elem_left")
	right := ElementID("elem_right")

	op := Operation{
		DocumentID:   "doc_1",
		OperationID:  "op_1",
		ActorID:      "actor_1",
		ActorCounter: 1,
		Type:         OperationTypeInsert,
		InsertPayload: &InsertPayload{
			ElementID:     "elem_1",
			Value:         "x",
			LeftOriginID:  &left,
			RightOriginID: &right,
		},
	}

	if err := op.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestOperationValidateDelete(t *testing.T) {
	op := Operation{
		DocumentID:   "doc_1",
		OperationID:  "op_2",
		ActorID:      "actor_1",
		ActorCounter: 2,
		Type:         OperationTypeDelete,
		DeletePayload: &DeletePayload{
			TargetElementID: "elem_1",
		},
	}

	if err := op.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestOperationValidateRejectsMultiRuneFreeTextChunks(t *testing.T) {
	op := Operation{
		DocumentID:   "doc_1",
		OperationID:  "op_3",
		ActorID:      "actor_1",
		ActorCounter: 3,
		Type:         OperationTypeInsert,
		InsertPayload: &InsertPayload{
			ElementID: "elem_2",
			Value:     "ab",
		},
	}

	err := op.Validate()
	if !errors.Is(err, ErrInvalidInsertValue) {
		t.Fatalf("Validate() error = %v, want ErrInvalidInsertValue", err)
	}
}

func TestOperationValidateAllowsSingleRuneUnicodeInsert(t *testing.T) {
	op := Operation{
		DocumentID:   "doc_1",
		OperationID:  "op_4",
		ActorID:      "actor_1",
		ActorCounter: 4,
		Type:         OperationTypeInsert,
		InsertPayload: &InsertPayload{
			ElementID: "elem_3",
			Value:     "é",
		},
	}

	if err := op.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestOperationValidateRejectsMissingDeleteTarget(t *testing.T) {
	op := Operation{
		DocumentID:   "doc_1",
		OperationID:  "op_5",
		ActorID:      "actor_1",
		ActorCounter: 5,
		Type:         OperationTypeDelete,
		DeletePayload: &DeletePayload{
			TargetElementID: "",
		},
	}

	err := op.Validate()
	if !errors.Is(err, ErrEmptyElementID) {
		t.Fatalf("Validate() error = %v, want ErrEmptyElementID", err)
	}
}

func TestOperationValidateRejectsMixedPayloads(t *testing.T) {
	op := Operation{
		DocumentID:   "doc_1",
		OperationID:  "op_6",
		ActorID:      "actor_1",
		ActorCounter: 6,
		Type:         OperationTypeDelete,
		InsertPayload: &InsertPayload{
			ElementID: "elem_4",
			Value:     "x",
		},
		DeletePayload: &DeletePayload{
			TargetElementID: "elem_4",
		},
	}

	err := op.Validate()
	if !errors.Is(err, ErrUnexpectedInsertPayload) {
		t.Fatalf("Validate() error = %v, want ErrUnexpectedInsertPayload", err)
	}
}

func TestCompareInsertOrderUsesActorThenCounterThenOperationID(t *testing.T) {
	left := ElementID("left")
	right := ElementID("right")

	first := Operation{
		DocumentID:   "doc_1",
		OperationID:  "op_1",
		ActorID:      "actor_a",
		ActorCounter: 2,
		Type:         OperationTypeInsert,
		InsertPayload: &InsertPayload{
			ElementID:     "elem_1",
			Value:         "a",
			LeftOriginID:  &left,
			RightOriginID: &right,
		},
	}
	second := Operation{
		DocumentID:   "doc_1",
		OperationID:  "op_2",
		ActorID:      "actor_b",
		ActorCounter: 1,
		Type:         OperationTypeInsert,
		InsertPayload: &InsertPayload{
			ElementID:     "elem_2",
			Value:         "b",
			LeftOriginID:  &left,
			RightOriginID: &right,
		},
	}

	got, err := first.CompareInsertOrder(second)
	if err != nil {
		t.Fatalf("CompareInsertOrder() error = %v, want nil", err)
	}
	if got >= 0 {
		t.Fatalf("CompareInsertOrder() = %d, want negative result", got)
	}
}

func TestCompareInsertOrderRejectsDifferentInsertionPoints(t *testing.T) {
	leftA := ElementID("left_a")
	leftB := ElementID("left_b")

	first := Operation{
		DocumentID:   "doc_1",
		OperationID:  "op_1",
		ActorID:      "actor_a",
		ActorCounter: 1,
		Type:         OperationTypeInsert,
		InsertPayload: &InsertPayload{
			ElementID:    "elem_1",
			Value:        "a",
			LeftOriginID: &leftA,
		},
	}
	second := Operation{
		DocumentID:   "doc_1",
		OperationID:  "op_2",
		ActorID:      "actor_b",
		ActorCounter: 1,
		Type:         OperationTypeInsert,
		InsertPayload: &InsertPayload{
			ElementID:    "elem_2",
			Value:        "b",
			LeftOriginID: &leftB,
		},
	}

	if _, err := first.CompareInsertOrder(second); err == nil {
		t.Fatal("CompareInsertOrder() error = nil, want non-nil")
	}
}

func TestPendingDeleteValidate(t *testing.T) {
	pending := PendingDelete{
		DocumentID:      "doc_1",
		OperationID:     "op_10",
		ActorID:         "actor_1",
		ActorCounter:    10,
		TargetElementID: "elem_99",
	}

	if err := pending.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
	if pending.PendingDeleteKey() == "" {
		t.Fatal("PendingDeleteKey() = empty, want non-empty")
	}
}

func TestPendingDeleteValidateRejectsMissingTarget(t *testing.T) {
	pending := PendingDelete{
		DocumentID:   "doc_1",
		OperationID:  "op_10",
		ActorID:      "actor_1",
		ActorCounter: 10,
	}

	err := pending.Validate()
	if !errors.Is(err, ErrEmptyElementID) {
		t.Fatalf("Validate() error = %v, want ErrEmptyElementID", err)
	}
}
