package performance

import (
	"context"
	"fmt"
	"io"
	"log/slog"
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

const (
	targetDocumentBytes      = 12 * 1024
	targetSnapshotBytes      = 8 * 1024
	targetBootstrapThreshold = 5 * time.Second
	targetReconnectThreshold = 5 * time.Second
)

// TestV1TargetScaleBootstrapAndReconnect stays within the practical v1 local-demo baseline.
func TestV1TargetScaleBootstrapAndReconnect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	store := performanceNewStore(t)
	store.SetLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))
	ops := performanceInsertOps("doc_perf", targetDocumentBytes)
	baseSnapshot, snapshotWatermark, laterOps := performanceSnapshotSplit(t, ops, targetSnapshotBytes)

	for _, op := range ops {
		if err := store.AppendOperation(ctx, op); err != nil {
			t.Fatalf("AppendOperation(%s) error = %v, want nil", op.OperationID, err)
		}
	}
	if err := store.SaveSnapshot(ctx, persistence.SnapshotRecord{
		SnapshotID:              "snap_perf",
		DocumentID:              "doc_perf",
		LastIncludedOperationID: snapshotWatermark,
		CreatedAt:               time.Now().UTC(),
		State:                   baseSnapshot,
	}); err != nil {
		t.Fatalf("SaveSnapshot() error = %v, want nil", err)
	}

	rebuildStarted := time.Now()
	rebuilt, err := store.RebuildDocument(ctx, "doc_perf")
	if err != nil {
		t.Fatalf("RebuildDocument() error = %v, want nil", err)
	}
	rebuildDuration := time.Since(rebuildStarted)
	if rebuildDuration > targetBootstrapThreshold {
		t.Fatalf("RebuildDocument() duration = %s, want <= %s", rebuildDuration, targetBootstrapThreshold)
	}
	if got := rebuilt.VisibleText(); len(got) != targetDocumentBytes {
		t.Fatalf("rebuilt text length = %d, want %d", len(got), targetDocumentBytes)
	}

	service := live.NewService(store)
	service.SetLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))
	replica, err := reconnect.NewReplica("doc_perf", "actor_reader")
	if err != nil {
		t.Fatalf("NewReplica() error = %v, want nil", err)
	}
	if err := performanceHello(ctx, service, "sess_reconnect", "actor_reader"); err != nil {
		t.Fatalf("performanceHello() error = %v, want nil", err)
	}

	reconnectStarted := time.Now()
	ack, errMsg, err := service.HandleSubscribe(ctx, protocol.SubscribeDocumentMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeSubscribeDoc,
			DocumentID:      "doc_perf",
			SessionID:       "sess_reconnect",
			MessageID:       "msg_sub_reconnect",
		},
		KnownLastOperationID: modelOperationIDPtr(snapshotWatermark),
	})
	if err != nil || errMsg != nil {
		t.Fatalf("HandleSubscribe() err = %v, errMsg = %+v, want nil", err, errMsg)
	}
	if err := replica.HandleSubscribeAck(*ack); err != nil {
		t.Fatalf("HandleSubscribeAck() error = %v, want nil", err)
	}
	events, errMsg, err := service.HandleRequestCatchup(ctx, protocol.RequestCatchupMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeRequestCatchup,
			DocumentID:      "doc_perf",
			SessionID:       "sess_reconnect",
			MessageID:       "msg_catchup_reconnect",
		},
		KnownLastOperationID: modelOperationIDPtr(snapshotWatermark),
	})
	if err != nil || errMsg != nil {
		t.Fatalf("HandleRequestCatchup() err = %v, errMsg = %+v, want nil", err, errMsg)
	}
	if len(events) != 3 {
		t.Fatalf("catchup event count = %d, want 3", len(events))
	}
	if err := replica.ApplyCatchupSnapshot(events[0].(protocol.CatchupSnapshotMessage)); err != nil {
		t.Fatalf("ApplyCatchupSnapshot() error = %v, want nil", err)
	}
	if err := replica.ApplyCatchupOperations(events[1].(protocol.CatchupOperationsMessage)); err != nil {
		t.Fatalf("ApplyCatchupOperations() error = %v, want nil", err)
	}
	if err := replica.ApplyCatchupComplete(events[2].(protocol.CatchupCompleteMessage)); err != nil {
		t.Fatalf("ApplyCatchupComplete() error = %v, want nil", err)
	}
	reconnectDuration := time.Since(reconnectStarted)
	if reconnectDuration > targetReconnectThreshold {
		t.Fatalf("catchup duration = %s, want <= %s", reconnectDuration, targetReconnectThreshold)
	}
	if got := replica.VisibleText(); len(got) != targetDocumentBytes {
		t.Fatalf("replica text length = %d, want %d", len(got), targetDocumentBytes)
	}
	if !replica.LiveSubscriptionReady() {
		t.Fatal("LiveSubscriptionReady() = false, want true")
	}
	if len(laterOps) == 0 {
		t.Fatal("laterOps length = 0, want non-zero delta for reconnect scenario")
	}

	t.Logf("v1 target-scale rebuild completed in %s for %d bytes", rebuildDuration, targetDocumentBytes)
	t.Logf("v1 target-scale reconnect completed in %s with %d delta operations", reconnectDuration, len(laterOps))
}

// performanceNewStore constructs one file-backed store for performance smoke tests.
func performanceNewStore(t *testing.T) *persistence.FileStore {
	t.Helper()
	store, err := persistence.NewFileStore(filepath.Join(t.TempDir(), "store"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v, want nil", err)
	}
	return store
}

// performanceInsertOps builds one deterministic insert sequence for the target document size.
func performanceInsertOps(documentID model.DocumentID, count int) []model.Operation {
	ops := make([]model.Operation, 0, count)
	for index := 0; index < count; index++ {
		var left *model.ElementID
		if index > 0 {
			left = modelElementIDPtr(model.ElementID(fmt.Sprintf("elem_%05d", index-1)))
		}
		ops = append(ops, model.Operation{
			DocumentID:   documentID,
			OperationID:  model.OperationID(fmt.Sprintf("op_%05d", index)),
			ActorID:      "actor_author",
			ActorCounter: uint64(index + 1),
			Type:         model.OperationTypeInsert,
			InsertPayload: &model.InsertPayload{
				ElementID:    model.ElementID(fmt.Sprintf("elem_%05d", index)),
				Value:        string(rune('a' + (index % 26))),
				LeftOriginID: left,
			},
		})
	}
	return ops
}

// performanceSnapshotSplit returns one snapshot baseline, its watermark, and the later operation tail.
func performanceSnapshotSplit(t *testing.T, ops []model.Operation, split int) (engine.Snapshot, model.OperationID, []model.Operation) {
	t.Helper()
	doc, err := engine.NewDocument("doc_perf")
	if err != nil {
		t.Fatalf("NewDocument() error = %v, want nil", err)
	}
	for _, op := range ops[:split] {
		if _, err := doc.Apply(op); err != nil {
			t.Fatalf("Apply(%s) error = %v, want nil", op.OperationID, err)
		}
	}
	return doc.Snapshot(), ops[split-1].OperationID, ops[split:]
}

// performanceHello registers one session against the live service.
func performanceHello(ctx context.Context, service *live.Service, sessionID string, actorID model.ActorID) error {
	errMsg, err := service.HandleClientHello(ctx, protocol.ClientHelloMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeClientHello,
			SessionID:       sessionID,
			MessageID:       "hello_" + sessionID,
		},
		ActorID: actorID,
	})
	if err != nil {
		return err
	}
	if errMsg != nil {
		return fmt.Errorf("client hello rejected: %s", errMsg.ErrorMessage)
	}
	return nil
}

// performanceSubscribe registers one document subscription against the live service.
func performanceSubscribe(ctx context.Context, service *live.Service, sessionID string, documentID model.DocumentID, knownLast *model.OperationID) error {
	_, errMsg, err := service.HandleSubscribe(ctx, protocol.SubscribeDocumentMessage{
		Envelope: protocol.Envelope{
			ProtocolVersion: protocol.VersionV1,
			MessageType:     protocol.MessageTypeSubscribeDoc,
			DocumentID:      documentID,
			SessionID:       sessionID,
			MessageID:       "sub_" + sessionID,
		},
		KnownLastOperationID: knownLast,
	})
	if err != nil {
		return err
	}
	if errMsg != nil {
		return fmt.Errorf("subscribe rejected: %s", errMsg.ErrorMessage)
	}
	return nil
}

// modelElementIDPtr returns one reusable element-id pointer.
func modelElementIDPtr(id model.ElementID) *model.ElementID {
	return &id
}

// modelOperationIDPtr returns one reusable operation-id pointer.
func modelOperationIDPtr(id model.OperationID) *model.OperationID {
	return &id
}
