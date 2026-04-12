package model

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// OperationType identifies the semantic kind of CRDT mutation.
type OperationType string

const (
	// OperationTypeInsert inserts exactly one rune at a stable context.
	OperationTypeInsert OperationType = "insert"
	// OperationTypeDelete tombstones one previously inserted element.
	OperationTypeDelete OperationType = "delete"
)

// ActorID identifies one logical operation producer.
type ActorID string

// DocumentID identifies one shared plain-text document.
type DocumentID string

// ElementID identifies one inserted CRDT element.
type ElementID string

// OperationID identifies one semantic CRDT mutation for dedupe purposes.
type OperationID string

// SnapshotID identifies one derived persisted snapshot.
type SnapshotID string

var (
	// ErrEmptyDocumentID reports that a document identifier is missing.
	ErrEmptyDocumentID = errors.New("document_id must not be empty")
	// ErrEmptyActorID reports that an actor identifier is missing.
	ErrEmptyActorID = errors.New("actor_id must not be empty")
	// ErrEmptyOperationID reports that an operation identifier is missing.
	ErrEmptyOperationID = errors.New("operation_id must not be empty")
	// ErrEmptyElementID reports that an element identifier is missing.
	ErrEmptyElementID = errors.New("element_id must not be empty")
	// ErrInvalidActorCounter reports that actor counters must start at 1.
	ErrInvalidActorCounter = errors.New("actor_counter must be greater than zero")
	// ErrUnknownOperationType reports an unsupported operation kind.
	ErrUnknownOperationType = errors.New("operation type must be insert or delete")
	// ErrMissingInsertPayload reports that an insert payload is required.
	ErrMissingInsertPayload = errors.New("insert operation requires insert payload")
	// ErrMissingDeletePayload reports that a delete payload is required.
	ErrMissingDeletePayload = errors.New("delete operation requires delete payload")
	// ErrUnexpectedInsertPayload reports that an insert payload appears on a non-insert operation.
	ErrUnexpectedInsertPayload = errors.New("insert payload is only valid for insert operations")
	// ErrUnexpectedDeletePayload reports that a delete payload appears on a non-delete operation.
	ErrUnexpectedDeletePayload = errors.New("delete payload is only valid for delete operations")
	// ErrInvalidInsertValue reports that insert values must contain exactly one rune.
	ErrInvalidInsertValue = errors.New("insert value must contain exactly one rune")
)

// InsertPayload holds the semantic content required for one-rune insertion.
type InsertPayload struct {
	ElementID     ElementID
	Value         string
	LeftOriginID  *ElementID
	RightOriginID *ElementID
}

// Validate reports whether the insert payload satisfies the v1 protocol contract.
func (p InsertPayload) Validate() error {
	if p.ElementID == "" {
		return fieldError("payload.element_id", ErrEmptyElementID)
	}
	if utf8.RuneCountInString(p.Value) != 1 {
		return fieldError("payload.value", ErrInvalidInsertValue)
	}

	if p.LeftOriginID != nil && *p.LeftOriginID == "" {
		return fieldError("payload.left_origin_id", ErrEmptyElementID)
	}
	if p.RightOriginID != nil && *p.RightOriginID == "" {
		return fieldError("payload.right_origin_id", ErrEmptyElementID)
	}

	return nil
}

// SameInsertionPoint reports whether two inserts target the same stable origin region.
func (p InsertPayload) SameInsertionPoint(other InsertPayload) bool {
	return equalElementPtr(p.LeftOriginID, other.LeftOriginID) &&
		equalElementPtr(p.RightOriginID, other.RightOriginID)
}

// DeletePayload holds the semantic content required for element tombstoning.
type DeletePayload struct {
	TargetElementID ElementID
}

// Validate reports whether the delete payload satisfies the v1 protocol contract.
func (p DeletePayload) Validate() error {
	if p.TargetElementID == "" {
		return fieldError("payload.target_element_id", ErrEmptyElementID)
	}
	return nil
}

// Operation is one semantic CRDT mutation scoped to a document and actor.
type Operation struct {
	DocumentID    DocumentID
	OperationID   OperationID
	ActorID       ActorID
	ActorCounter  uint64
	Type          OperationType
	InsertPayload *InsertPayload
	DeletePayload *DeletePayload
}

// Validate reports whether the operation satisfies the frozen v1 model contract.
func (o Operation) Validate() error {
	if o.DocumentID == "" {
		return fieldError("document_id", ErrEmptyDocumentID)
	}
	if o.OperationID == "" {
		return fieldError("operation_id", ErrEmptyOperationID)
	}
	if o.ActorID == "" {
		return fieldError("actor_id", ErrEmptyActorID)
	}
	if o.ActorCounter == 0 {
		return fieldError("actor_counter", ErrInvalidActorCounter)
	}

	switch o.Type {
	case OperationTypeInsert:
		if o.InsertPayload == nil {
			return fieldError("payload", ErrMissingInsertPayload)
		}
		if o.DeletePayload != nil {
			return fieldError("payload", ErrUnexpectedDeletePayload)
		}
		return o.InsertPayload.Validate()
	case OperationTypeDelete:
		if o.DeletePayload == nil {
			return fieldError("payload", ErrMissingDeletePayload)
		}
		if o.InsertPayload != nil {
			return fieldError("payload", ErrUnexpectedInsertPayload)
		}
		return o.DeletePayload.Validate()
	default:
		return fieldError("type", ErrUnknownOperationType)
	}
}

// CompareInsertOrder reports deterministic order for concurrent inserts in the same region.
//
// Ordering is defined by actor identity first, actor counter second, and
// operation identity third as a final stable tiebreaker. Callers must only use
// this for insert operations that target the same insertion point.
func (o Operation) CompareInsertOrder(other Operation) (int, error) {
	if err := o.validateComparableInsert(other); err != nil {
		return 0, err
	}

	if cmp := strings.Compare(string(o.ActorID), string(other.ActorID)); cmp != 0 {
		return cmp, nil
	}
	if o.ActorCounter < other.ActorCounter {
		return -1, nil
	}
	if o.ActorCounter > other.ActorCounter {
		return 1, nil
	}
	return strings.Compare(string(o.OperationID), string(other.OperationID)), nil
}

// PendingDelete is one deferred delete staged until its target element is known.
type PendingDelete struct {
	DocumentID      DocumentID
	OperationID     OperationID
	ActorID         ActorID
	ActorCounter    uint64
	TargetElementID ElementID
}

// Validate reports whether the pending-delete record is well-formed.
func (p PendingDelete) Validate() error {
	if p.DocumentID == "" {
		return fieldError("document_id", ErrEmptyDocumentID)
	}
	if p.OperationID == "" {
		return fieldError("operation_id", ErrEmptyOperationID)
	}
	if p.ActorID == "" {
		return fieldError("actor_id", ErrEmptyActorID)
	}
	if p.ActorCounter == 0 {
		return fieldError("actor_counter", ErrInvalidActorCounter)
	}
	if p.TargetElementID == "" {
		return fieldError("target_element_id", ErrEmptyElementID)
	}
	return nil
}

// PendingDeleteKey returns the deterministic staging key for deferred deletes.
func (p PendingDelete) PendingDeleteKey() string {
	return fmt.Sprintf("%s/%s/%d/%s", p.ActorID, p.OperationID, p.ActorCounter, p.TargetElementID)
}

// ValidationError identifies one violated model precondition.
type ValidationError struct {
	Field   string
	Message string
	Cause   error
}

// Error formats one human-readable validation failure.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Unwrap exposes the underlying failure mode for callers using errors.Is.
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

func equalElementPtr(left, right *ElementID) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return *left == *right
	}
}

func (o Operation) validateComparableInsert(other Operation) error {
	if err := o.Validate(); err != nil {
		return err
	}
	if err := other.Validate(); err != nil {
		return err
	}
	if o.Type != OperationTypeInsert || other.Type != OperationTypeInsert {
		return fieldError("type", ErrUnknownOperationType)
	}
	if !o.InsertPayload.SameInsertionPoint(*other.InsertPayload) {
		return &ValidationError{
			Field:   "payload",
			Message: "insert comparison requires identical origin context",
			Cause:   errors.New("mismatched insertion point"),
		}
	}
	return nil
}
