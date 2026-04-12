package engine

import (
	"errors"
	"fmt"
	"strings"

	"github.com/JustCallMeMin/syncraft/internal/core/model"
)

var (
	// ErrInsertOriginsOutOfOrder reports a malformed insertion region.
	ErrInsertOriginsOutOfOrder = errors.New("insert origins must define a forward region")
	// ErrInsertOriginConflict reports an impossible insertion context.
	ErrInsertOriginConflict = errors.New("insert origins conflict with current order")
	// ErrSnapshotDocumentMismatch reports that a snapshot belongs to another document.
	ErrSnapshotDocumentMismatch = errors.New("snapshot document does not match requested document")
)

// ApplyStatus describes what happened when one operation was handled.
type ApplyStatus string

const (
	// ApplyStatusApplied means the operation changed engine state.
	ApplyStatusApplied ApplyStatus = "applied"
	// ApplyStatusDuplicate means the operation was already known and ignored safely.
	ApplyStatusDuplicate ApplyStatus = "duplicate"
	// ApplyStatusDeferred means the operation was staged until missing context arrives.
	ApplyStatusDeferred ApplyStatus = "deferred"
)

// ApplyResult summarizes one Apply call.
type ApplyResult struct {
	Status ApplyStatus
}

type element struct {
	id            model.ElementID
	value         string
	leftOriginID  *model.ElementID
	rightOriginID *model.ElementID
	deleted       bool
}

// ElementSnapshot captures one persisted element state.
type ElementSnapshot struct {
	ID            model.ElementID  `json:"id"`
	Value         string           `json:"value"`
	LeftOriginID  *model.ElementID `json:"left_origin_id,omitempty"`
	RightOriginID *model.ElementID `json:"right_origin_id,omitempty"`
	Deleted       bool             `json:"deleted"`
}

// Snapshot captures enough engine state for deterministic rebuild.
type Snapshot struct {
	DocumentID     model.DocumentID                `json:"document_id"`
	OrderedElements []ElementSnapshot              `json:"ordered_elements"`
	Applied        []model.Operation               `json:"applied_operations"`
	PendingInserts []model.Operation               `json:"pending_inserts"`
	PendingDeletes []model.PendingDelete           `json:"pending_deletes"`
}

// Document is the in-memory CRDT engine for one Syncraft document replica.
type Document struct {
	id             model.DocumentID
	order          []*element
	elementsByID   map[model.ElementID]*element
	applied        map[model.OperationID]model.Operation
	pendingInserts map[model.OperationID]model.Operation
	pendingDeletes map[model.OperationID]model.PendingDelete
}

// NewDocument constructs one empty CRDT document engine.
func NewDocument(id model.DocumentID) (*Document, error) {
	if id == "" {
		return nil, model.ErrEmptyDocumentID
	}

	return &Document{
		id:             id,
		order:          make([]*element, 0),
		elementsByID:   make(map[model.ElementID]*element),
		applied:        make(map[model.OperationID]model.Operation),
		pendingInserts: make(map[model.OperationID]model.Operation),
		pendingDeletes: make(map[model.OperationID]model.PendingDelete),
	}, nil
}

// NewDocumentFromSnapshot reconstructs one document replica from persisted state.
func NewDocumentFromSnapshot(snapshot Snapshot) (*Document, error) {
	if snapshot.DocumentID == "" {
		return nil, model.ErrEmptyDocumentID
	}

	doc, err := NewDocument(snapshot.DocumentID)
	if err != nil {
		return nil, err
	}

	for _, entry := range snapshot.OrderedElements {
		if entry.ID == "" {
			return nil, model.ErrEmptyElementID
		}
		docElem := &element{
			id:            entry.ID,
			value:         entry.Value,
			leftOriginID:  entry.LeftOriginID,
			rightOriginID: entry.RightOriginID,
			deleted:       entry.Deleted,
		}
		doc.order = append(doc.order, docElem)
		doc.elementsByID[entry.ID] = docElem
	}

	for _, op := range snapshot.Applied {
		if err := op.Validate(); err != nil {
			return nil, err
		}
		doc.applied[op.OperationID] = op
	}
	for _, op := range snapshot.PendingInserts {
		if err := op.Validate(); err != nil {
			return nil, err
		}
		doc.pendingInserts[op.OperationID] = op
	}
	for _, pending := range snapshot.PendingDeletes {
		if err := pending.Validate(); err != nil {
			return nil, err
		}
		doc.pendingDeletes[pending.OperationID] = pending
	}

	return doc, nil
}

// Apply validates and handles one semantic operation deterministically.
func (d *Document) Apply(op model.Operation) (ApplyResult, error) {
	if err := op.Validate(); err != nil {
		return ApplyResult{}, err
	}
	if op.DocumentID != d.id {
		return ApplyResult{}, &model.ValidationError{
			Field:   "document_id",
			Message: fmt.Sprintf("operation document_id %q does not match engine document_id %q", op.DocumentID, d.id),
			Cause:   model.ErrEmptyDocumentID,
		}
	}

	if _, ok := d.applied[op.OperationID]; ok {
		return ApplyResult{Status: ApplyStatusDuplicate}, nil
	}
	if _, ok := d.pendingInserts[op.OperationID]; ok {
		return ApplyResult{Status: ApplyStatusDuplicate}, nil
	}
	if _, ok := d.pendingDeletes[op.OperationID]; ok {
		return ApplyResult{Status: ApplyStatusDuplicate}, nil
	}

	switch op.Type {
	case model.OperationTypeInsert:
		applied, err := d.applyInsert(op)
		if err != nil {
			return ApplyResult{}, err
		}
		if !applied {
			d.pendingInserts[op.OperationID] = op
			return ApplyResult{Status: ApplyStatusDeferred}, nil
		}
		d.applied[op.OperationID] = op
		d.resolvePending()
		return ApplyResult{Status: ApplyStatusApplied}, nil
	case model.OperationTypeDelete:
		applied, err := d.applyDelete(op)
		if err != nil {
			return ApplyResult{}, err
		}
		if !applied {
			pending := model.PendingDelete{
				DocumentID:      op.DocumentID,
				OperationID:     op.OperationID,
				ActorID:         op.ActorID,
				ActorCounter:    op.ActorCounter,
				TargetElementID: op.DeletePayload.TargetElementID,
			}
			if err := pending.Validate(); err != nil {
				return ApplyResult{}, err
			}
			d.pendingDeletes[op.OperationID] = pending
			return ApplyResult{Status: ApplyStatusDeferred}, nil
		}
		d.applied[op.OperationID] = op
		return ApplyResult{Status: ApplyStatusApplied}, nil
	default:
		return ApplyResult{}, model.ErrUnknownOperationType
	}
}

// VisibleText projects the current visible plain-text content.
func (d *Document) VisibleText() string {
	var out strings.Builder
	for _, elem := range d.order {
		if elem.deleted {
			continue
		}
		out.WriteString(elem.value)
	}
	return out.String()
}

// AppliedCount reports how many unique operations are fully applied.
func (d *Document) AppliedCount() int {
	return len(d.applied)
}

// PendingInsertCount reports staged inserts waiting for missing context.
func (d *Document) PendingInsertCount() int {
	return len(d.pendingInserts)
}

// PendingDeleteCount reports staged deletes waiting for target materialization.
func (d *Document) PendingDeleteCount() int {
	return len(d.pendingDeletes)
}

// Snapshot exports the current engine state for persistence or rebuild.
func (d *Document) Snapshot() Snapshot {
	snapshot := Snapshot{
		DocumentID:      d.id,
		OrderedElements: make([]ElementSnapshot, 0, len(d.order)),
		Applied:         make([]model.Operation, 0, len(d.applied)),
		PendingInserts:  make([]model.Operation, 0, len(d.pendingInserts)),
		PendingDeletes:  make([]model.PendingDelete, 0, len(d.pendingDeletes)),
	}

	for _, elem := range d.order {
		snapshot.OrderedElements = append(snapshot.OrderedElements, ElementSnapshot{
			ID:            elem.id,
			Value:         elem.value,
			LeftOriginID:  elem.leftOriginID,
			RightOriginID: elem.rightOriginID,
			Deleted:       elem.deleted,
		})
	}
	for _, op := range d.applied {
		snapshot.Applied = append(snapshot.Applied, op)
	}
	for _, op := range d.pendingInserts {
		snapshot.PendingInserts = append(snapshot.PendingInserts, op)
	}
	for _, pending := range d.pendingDeletes {
		snapshot.PendingDeletes = append(snapshot.PendingDeletes, pending)
	}

	return snapshot
}

func (d *Document) applyInsert(op model.Operation) (bool, error) {
	payload := op.InsertPayload
	if !d.originsReady(payload.LeftOriginID, payload.RightOriginID) {
		return false, nil
	}

	insertIndex, err := d.computeInsertIndex(op)
	if err != nil {
		return false, err
	}
	elem := &element{
		id:            payload.ElementID,
		value:         payload.Value,
		leftOriginID:  payload.LeftOriginID,
		rightOriginID: payload.RightOriginID,
	}
	d.order = insertAt(d.order, insertIndex, elem)
	d.elementsByID[elem.id] = elem
	d.applyPendingDeletesFor(elem.id)
	return true, nil
}

func (d *Document) applyDelete(op model.Operation) (bool, error) {
	target := d.elementsByID[op.DeletePayload.TargetElementID]
	if target == nil {
		return false, nil
	}
	target.deleted = true
	return true, nil
}

func (d *Document) originsReady(left, right *model.ElementID) bool {
	if left != nil {
		if _, ok := d.elementsByID[*left]; !ok {
			return false
		}
	}
	if right != nil {
		if _, ok := d.elementsByID[*right]; !ok {
			return false
		}
	}
	return true
}

func (d *Document) computeInsertIndex(op model.Operation) (int, error) {
	leftIndex := -1
	rightIndex := len(d.order)

	if op.InsertPayload.LeftOriginID != nil {
		leftElem := d.elementsByID[*op.InsertPayload.LeftOriginID]
		if leftElem == nil {
			return 0, ErrInsertOriginConflict
		}
		leftIndex = d.indexOf(leftElem.id)
	}
	if op.InsertPayload.RightOriginID != nil {
		rightElem := d.elementsByID[*op.InsertPayload.RightOriginID]
		if rightElem == nil {
			return 0, ErrInsertOriginConflict
		}
		rightIndex = d.indexOf(rightElem.id)
	}
	if leftIndex >= rightIndex {
		return 0, ErrInsertOriginsOutOfOrder
	}

	insertIndex := leftIndex + 1
	for insertIndex < rightIndex {
		current := d.order[insertIndex]
		if !sameInsertionPoint(current, op.InsertPayload) {
			break
		}
		currentOp := d.findAppliedInsertByElementID(current.id)
		if currentOp == nil {
			break
		}
		cmp, err := op.CompareInsertOrder(*currentOp)
		if err != nil {
			return 0, err
		}
		if cmp < 0 {
			break
		}
		insertIndex++
	}

	return insertIndex, nil
}

func (d *Document) resolvePending() {
	for {
		progress := false

		for operationID, op := range d.pendingInserts {
			if !d.originsReady(op.InsertPayload.LeftOriginID, op.InsertPayload.RightOriginID) {
				continue
			}

			applied, err := d.applyInsert(op)
			if err != nil || !applied {
				continue
			}
			delete(d.pendingInserts, operationID)
			d.applied[operationID] = op
			progress = true
		}

		for operationID, pending := range d.pendingDeletes {
			target := d.elementsByID[pending.TargetElementID]
			if target == nil {
				continue
			}
			target.deleted = true
			delete(d.pendingDeletes, operationID)
			d.applied[operationID] = model.Operation{
				DocumentID:   pending.DocumentID,
				OperationID:  pending.OperationID,
				ActorID:      pending.ActorID,
				ActorCounter: pending.ActorCounter,
				Type:         model.OperationTypeDelete,
				DeletePayload: &model.DeletePayload{
					TargetElementID: pending.TargetElementID,
				},
			}
			progress = true
		}

		if !progress {
			return
		}
	}
}

func (d *Document) applyPendingDeletesFor(elementID model.ElementID) {
	for operationID, pending := range d.pendingDeletes {
		if pending.TargetElementID != elementID {
			continue
		}
		elem := d.elementsByID[elementID]
		if elem != nil {
			elem.deleted = true
		}
		delete(d.pendingDeletes, operationID)
		d.applied[operationID] = model.Operation{
			DocumentID:   pending.DocumentID,
			OperationID:  pending.OperationID,
			ActorID:      pending.ActorID,
			ActorCounter: pending.ActorCounter,
			Type:         model.OperationTypeDelete,
			DeletePayload: &model.DeletePayload{
				TargetElementID: pending.TargetElementID,
			},
		}
	}
}

func (d *Document) indexOf(elementID model.ElementID) int {
	for i, elem := range d.order {
		if elem.id == elementID {
			return i
		}
	}
	return -1
}

func (d *Document) findAppliedInsertByElementID(elementID model.ElementID) *model.Operation {
	for _, op := range d.applied {
		if op.Type != model.OperationTypeInsert || op.InsertPayload == nil {
			continue
		}
		if op.InsertPayload.ElementID == elementID {
			copy := op
			return &copy
		}
	}
	return nil
}

func insertAt(items []*element, index int, elem *element) []*element {
	items = append(items, nil)
	copy(items[index+1:], items[index:])
	items[index] = elem
	return items
}

func sameInsertionPoint(elem *element, payload *model.InsertPayload) bool {
	return equalElementIDs(elem.leftOriginID, payload.LeftOriginID) &&
		equalElementIDs(elem.rightOriginID, payload.RightOriginID)
}

func equalElementIDs(left, right *model.ElementID) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return *left == *right
	}
}
