package reconnect

import (
	"errors"
	"fmt"

	"github.com/JustCallMeMin/syncraft/internal/backend/protocol"
	"github.com/JustCallMeMin/syncraft/internal/core/engine"
	"github.com/JustCallMeMin/syncraft/internal/core/model"
)

var (
	// ErrDocumentNotInitialized reports that the local replica has not been bootstrapped yet.
	ErrDocumentNotInitialized = errors.New("document is not initialized")
	// ErrInvalidSnapshotPayload reports an unexpected snapshot state payload shape.
	ErrInvalidSnapshotPayload = errors.New("catchup snapshot state has invalid type")
	// ErrLiveModeDoesNotNeedCatchup reports that catch-up was requested while the session is already current.
	ErrLiveModeDoesNotNeedCatchup = errors.New("live_only subscription does not require catchup")
)

// Replica is one client-side reconnect bootstrap state machine for one document.
type Replica struct {
	documentID            model.DocumentID
	actorID               model.ActorID
	doc                   *engine.Document
	lastSnapshotID        *model.SnapshotID
	lastOperationID       *model.OperationID
	awaitingCatchup       bool
	liveSubscriptionReady bool
}

// NewReplica constructs one client-side document replica.
func NewReplica(documentID model.DocumentID, actorID model.ActorID) (*Replica, error) {
	if documentID == "" {
		return nil, model.ErrEmptyDocumentID
	}
	if actorID == "" {
		return nil, model.ErrEmptyActorID
	}

	doc, err := engine.NewDocument(documentID)
	if err != nil {
		return nil, err
	}
	return &Replica{
		documentID: documentID,
		actorID:    actorID,
		doc:        doc,
	}, nil
}

// StateReference returns the client's current reconnect reference point.
func (r *Replica) StateReference() (*model.SnapshotID, *model.OperationID) {
	return r.lastSnapshotID, r.lastOperationID
}

// HandleSubscribeAck updates reconnect state based on server subscription mode.
func (r *Replica) HandleSubscribeAck(ack protocol.SubscribeAckMessage) error {
	if ack.DocumentID != r.documentID {
		return fmt.Errorf("subscribe ack document_id %q does not match replica document_id %q", ack.DocumentID, r.documentID)
	}

	switch ack.SubscriptionMode {
	case protocol.SubscriptionModeLiveOnly:
		r.awaitingCatchup = false
		r.liveSubscriptionReady = true
		return nil
	case protocol.SubscriptionModeSnapshotGap:
		r.awaitingCatchup = true
		r.liveSubscriptionReady = false
		return nil
	default:
		return fmt.Errorf("unknown subscription mode %q", ack.SubscriptionMode)
	}
}

// ApplyCatchupSnapshot replaces local replica state with one snapshot baseline.
func (r *Replica) ApplyCatchupSnapshot(msg protocol.CatchupSnapshotMessage) error {
	if !r.awaitingCatchup {
		return ErrLiveModeDoesNotNeedCatchup
	}

	snapshot, ok := msg.Snapshot.State.(engine.Snapshot)
	if !ok {
		return ErrInvalidSnapshotPayload
	}
	doc, err := engine.NewDocumentFromSnapshot(snapshot)
	if err != nil {
		return err
	}
	r.doc = doc
	snapshotID := msg.Snapshot.SnapshotID
	lastIncluded := msg.Snapshot.LastIncludedOperationID
	r.lastSnapshotID = &snapshotID
	r.lastOperationID = &lastIncluded
	return nil
}

// ApplyCatchupOperations replays delta operations after the snapshot watermark.
func (r *Replica) ApplyCatchupOperations(msg protocol.CatchupOperationsMessage) error {
	if r.doc == nil {
		return ErrDocumentNotInitialized
	}
	for _, op := range msg.Operations {
		if _, err := r.doc.Apply(op); err != nil {
			return err
		}
		last := op.OperationID
		r.lastOperationID = &last
	}
	return nil
}

// ApplyCatchupComplete marks the replica ready for resumed live sync.
func (r *Replica) ApplyCatchupComplete(msg protocol.CatchupCompleteMessage) error {
	if !r.awaitingCatchup {
		return ErrLiveModeDoesNotNeedCatchup
	}
	r.awaitingCatchup = false
	r.liveSubscriptionReady = true
	if msg.LastOperationID != "" {
		last := msg.LastOperationID
		r.lastOperationID = &last
	}
	return nil
}

// ApplyBroadcastOperation applies one live broadcast after the subscription is current.
func (r *Replica) ApplyBroadcastOperation(msg protocol.BroadcastOperationMessage) error {
	if r.doc == nil {
		return ErrDocumentNotInitialized
	}
	if _, err := r.doc.Apply(msg.Operation); err != nil {
		return err
	}
	last := msg.Operation.OperationID
	r.lastOperationID = &last
	return nil
}

// VisibleText returns the replica's visible plain-text state.
func (r *Replica) VisibleText() string {
	if r.doc == nil {
		return ""
	}
	return r.doc.VisibleText()
}

// LiveSubscriptionReady reports whether the replica can resume normal live traffic.
func (r *Replica) LiveSubscriptionReady() bool {
	return r.liveSubscriptionReady
}
