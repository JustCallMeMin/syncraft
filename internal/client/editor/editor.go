package editor

import (
	"fmt"
	"unicode/utf8"

	"github.com/JustCallMeMin/syncraft/internal/backend/protocol"
	"github.com/JustCallMeMin/syncraft/internal/client/reconnect"
	"github.com/JustCallMeMin/syncraft/internal/core/engine"
	"github.com/JustCallMeMin/syncraft/internal/core/model"
)

// ConnectionState reports the browser-facing collaboration state.
type ConnectionState string

const (
	// ConnectionStateConnecting reports that the editor is not yet ready for live traffic.
	ConnectionStateConnecting ConnectionState = "connecting"
	// ConnectionStateCatchingUp reports that the editor is replaying reconnect state.
	ConnectionStateCatchingUp ConnectionState = "catching_up"
	// ConnectionStateLive reports that the editor is current and ready for collaboration.
	ConnectionStateLive ConnectionState = "live"
	// ConnectionStateError reports that the editor hit a user-visible failure.
	ConnectionStateError ConnectionState = "error"
)

// State is the minimal browser-facing editor view model.
type State struct {
	Text            string
	ConnectionState ConnectionState
	LastError       string
}

// Session is the minimal editor integration over one reconnect-capable replica.
type Session struct {
	documentID  model.DocumentID
	actorID     model.ActorID
	replica     *reconnect.Replica
	nextCounter uint64
	state       State
}

// Metadata captures restart-relevant browser-facing session metadata.
type Metadata struct {
	NextActorCounter  uint64
	LastSnapshotID    *model.SnapshotID
	LastOperationID   *model.OperationID
	PendingQueueCount int
}

// VisibleElement is the browser-facing CRDT element view needed for offline queueing.
type VisibleElement struct {
	ID    model.ElementID
	Value string
}

// NewSession constructs one browser-facing plain-text editor session.
func NewSession(documentID model.DocumentID, actorID model.ActorID) (*Session, error) {
	replica, err := reconnect.NewReplica(documentID, actorID)
	if err != nil {
		return nil, err
	}
	return &Session{
		documentID:  documentID,
		actorID:     actorID,
		replica:     replica,
		nextCounter: 1,
		state: State{
			Text:            "",
			ConnectionState: ConnectionStateConnecting,
		},
	}, nil
}

// ViewState returns the current browser-facing editor state.
func (s *Session) ViewState() State {
	return s.state
}

// ViewMetadata returns restart-relevant metadata for browser persistence.
func (s *Session) ViewMetadata() Metadata {
	lastSnapshotID, lastOperationID := s.replica.StateReference()
	return Metadata{
		NextActorCounter:  s.nextCounter,
		LastSnapshotID:    lastSnapshotID,
		LastOperationID:   lastOperationID,
		PendingQueueCount: 0,
	}
}

// ViewVisibleElements returns the current visible element list in document order.
func (s *Session) ViewVisibleElements() []VisibleElement {
	visible := s.visibleElements()
	out := make([]VisibleElement, 0, len(visible))
	for _, elem := range visible {
		out = append(out, VisibleElement{
			ID:    elem.ID,
			Value: elem.Value,
		})
	}
	return out
}

// SetNextActorCounter overrides the next actor counter for restart continuity.
func (s *Session) SetNextActorCounter(nextCounter uint64) error {
	if nextCounter == 0 {
		return fmt.Errorf("next actor counter must be greater than zero")
	}
	s.nextCounter = nextCounter
	return nil
}

// HandleSubscribeAck updates the editor's connection state from subscription mode.
func (s *Session) HandleSubscribeAck(ack protocol.SubscribeAckMessage) error {
	if err := s.replica.HandleSubscribeAck(ack); err != nil {
		s.setError(err)
		return err
	}
	switch ack.SubscriptionMode {
	case protocol.SubscriptionModeLiveOnly:
		s.state.ConnectionState = ConnectionStateLive
	case protocol.SubscriptionModeSnapshotGap:
		s.state.ConnectionState = ConnectionStateCatchingUp
	}
	s.syncVisibleText()
	return nil
}

// ApplyCatchupSnapshot applies one snapshot baseline into the editor state.
func (s *Session) ApplyCatchupSnapshot(msg protocol.CatchupSnapshotMessage) error {
	if err := s.replica.ApplyCatchupSnapshot(msg); err != nil {
		s.setError(err)
		return err
	}
	s.advanceCounterFromSnapshotWatermark()
	s.syncVisibleText()
	return nil
}

// ApplyCatchupOperations replays one catch-up batch into the editor state.
func (s *Session) ApplyCatchupOperations(msg protocol.CatchupOperationsMessage) error {
	if err := s.replica.ApplyCatchupOperations(msg); err != nil {
		s.setError(err)
		return err
	}
	s.advanceCounterFromOperations(msg.Operations)
	s.syncVisibleText()
	return nil
}

// ApplyCatchupComplete marks the editor session live again after catch-up.
func (s *Session) ApplyCatchupComplete(msg protocol.CatchupCompleteMessage) error {
	if err := s.replica.ApplyCatchupComplete(msg); err != nil {
		s.setError(err)
		return err
	}
	s.state.ConnectionState = ConnectionStateLive
	s.syncVisibleText()
	return nil
}

// ApplyRemoteBroadcast applies one remote operation safely into editor state.
func (s *Session) ApplyRemoteBroadcast(msg protocol.BroadcastOperationMessage) error {
	if err := s.replica.ApplyBroadcastOperation(msg); err != nil {
		s.setError(err)
		return err
	}
	s.advanceCounterFromOperation(msg.Operation)
	s.syncVisibleText()
	return nil
}

// InsertTextAt generates and optimistically applies one or more one-rune insert operations.
func (s *Session) InsertTextAt(index int, value string) ([]model.Operation, error) {
	return s.InsertTextAtWithCounter(index, value, s.nextCounter)
}

// InsertTextAtWithCounter generates and applies one or more insert operations from one explicit counter.
func (s *Session) InsertTextAtWithCounter(index int, value string, startCounter uint64) ([]model.Operation, error) {
	if value == "" {
		return nil, nil
	}
	if err := s.SetNextActorCounter(startCounter); err != nil {
		s.setError(err)
		return nil, err
	}
	runes := []rune(value)
	ops := make([]model.Operation, 0, len(runes))
	currentIndex := index
	for _, r := range runes {
		op, err := s.buildInsertOp(currentIndex, string(r))
		if err != nil {
			s.setError(err)
			return nil, err
		}
		if err := s.replica.ApplyLocalOperation(op); err != nil {
			s.setError(err)
			return nil, err
		}
		ops = append(ops, op)
		currentIndex++
	}
	s.syncVisibleText()
	return ops, nil
}

// DeleteAt generates and optimistically applies one delete operation at one visible index.
func (s *Session) DeleteAt(index int) (model.Operation, error) {
	return s.DeleteAtWithCounter(index, s.nextCounter)
}

// DeleteAtWithCounter generates and applies one delete operation from one explicit actor counter.
func (s *Session) DeleteAtWithCounter(index int, counter uint64) (model.Operation, error) {
	if err := s.SetNextActorCounter(counter); err != nil {
		s.setError(err)
		return model.Operation{}, err
	}
	visible := s.visibleElements()
	if index < 0 || index >= len(visible) {
		err := fmt.Errorf("delete index %d is outside visible text bounds", index)
		s.setError(err)
		return model.Operation{}, err
	}
	counter = s.allocateCounter()
	op := model.Operation{
		DocumentID:   s.documentID,
		OperationID:  model.OperationID(fmt.Sprintf("op_%s_%d", s.actorID, counter)),
		ActorID:      s.actorID,
		ActorCounter: counter,
		Type:         model.OperationTypeDelete,
		DeletePayload: &model.DeletePayload{
			TargetElementID: visible[index].ID,
		},
	}
	if err := s.replica.ApplyLocalOperation(op); err != nil {
		s.setError(err)
		return model.Operation{}, err
	}
	s.syncVisibleText()
	return op, nil
}

func (s *Session) buildInsertOp(index int, value string) (model.Operation, error) {
	if utf8.RuneCountInString(value) != 1 {
		return model.Operation{}, model.ErrInvalidInsertValue
	}
	visible := s.visibleElements()
	if index < 0 || index > len(visible) {
		return model.Operation{}, fmt.Errorf("insert index %d is outside visible text bounds", index)
	}

	var left *model.ElementID
	var right *model.ElementID
	if index > 0 {
		leftID := visible[index-1].ID
		left = &leftID
	}
	if index < len(visible) {
		rightID := visible[index].ID
		right = &rightID
	}

	counter := s.allocateCounter()
	return model.Operation{
		DocumentID:   s.documentID,
		OperationID:  model.OperationID(fmt.Sprintf("op_%s_%d", s.actorID, counter)),
		ActorID:      s.actorID,
		ActorCounter: counter,
		Type:         model.OperationTypeInsert,
		InsertPayload: &model.InsertPayload{
			ElementID:     model.ElementID(fmt.Sprintf("elem_%s_%d", s.actorID, counter)),
			Value:         value,
			LeftOriginID:  left,
			RightOriginID: right,
		},
	}, nil
}

func (s *Session) visibleElements() []engine.ElementSnapshot {
	snapshot, err := s.replica.Snapshot()
	if err != nil {
		return nil
	}
	visible := make([]engine.ElementSnapshot, 0, len(snapshot.OrderedElements))
	for _, elem := range snapshot.OrderedElements {
		if elem.Deleted {
			continue
		}
		visible = append(visible, elem)
	}
	return visible
}

func (s *Session) allocateCounter() uint64 {
	counter := s.nextCounter
	s.nextCounter++
	return counter
}

func (s *Session) advanceCounterFromOperations(operations []model.Operation) {
	for _, op := range operations {
		s.advanceCounterFromOperation(op)
	}
}

func (s *Session) advanceCounterFromOperation(op model.Operation) {
	if op.ActorID != s.actorID {
		return
	}
	nextCounter := op.ActorCounter + 1
	if nextCounter > s.nextCounter {
		s.nextCounter = nextCounter
	}
}

func (s *Session) advanceCounterFromSnapshotWatermark() {
	snapshot, err := s.replica.Snapshot()
	if err != nil {
		return
	}
	s.advanceCounterFromOperations(snapshot.Applied)
}

func (s *Session) syncVisibleText() {
	s.state.Text = s.replica.VisibleText()
	s.state.LastError = ""
	if s.replica.LiveSubscriptionReady() && s.state.ConnectionState != ConnectionStateError {
		s.state.ConnectionState = ConnectionStateLive
	}
}

func (s *Session) setError(err error) {
	s.state.LastError = err.Error()
	s.state.ConnectionState = ConnectionStateError
}
