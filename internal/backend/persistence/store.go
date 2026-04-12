package persistence

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/JustCallMeMin/syncraft/internal/core/engine"
	"github.com/JustCallMeMin/syncraft/internal/core/model"
)

var (
	// ErrSnapshotNotFound reports that no snapshot is available for a document.
	ErrSnapshotNotFound = errors.New("snapshot not found")
)

// SnapshotRecord captures one persisted snapshot plus its replay watermark.
type SnapshotRecord struct {
	SnapshotID              model.SnapshotID `json:"snapshot_id"`
	DocumentID              model.DocumentID `json:"document_id"`
	LastIncludedOperationID model.OperationID `json:"last_included_operation_id"`
	CreatedAt               time.Time         `json:"created_at"`
	State                   engine.Snapshot   `json:"state"`
}

// FileStore persists operation logs and snapshots on the local filesystem.
type FileStore struct {
	root   string
	logger *slog.Logger
	mu     sync.Mutex
}

// NewFileStore validates the root path and prepares the persistence directory.
func NewFileStore(root string) (*FileStore, error) {
	if root == "" {
		return nil, fmt.Errorf("root path must not be empty")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create persistence root: %w", err)
	}
	return &FileStore{
		root:   root,
		logger: slog.Default(),
	}, nil
}

// SetLogger overrides the audit logger used by the persistence store.
func (s *FileStore) SetLogger(logger *slog.Logger) {
	if logger == nil {
		s.logger = slog.Default()
		return
	}
	s.logger = logger
}

// AppendOperation stores one accepted operation in append-only order.
func (s *FileStore) AppendOperation(ctx context.Context, op model.Operation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := op.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	docDir := s.documentDir(op.DocumentID)
	if err := os.MkdirAll(docDir, 0o755); err != nil {
		return fmt.Errorf("create document persistence directory: %w", err)
	}

	file, err := os.OpenFile(s.operationsPath(op.DocumentID), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open operation log: %w", err)
	}
	defer file.Close()

	data, err := json.Marshal(op)
	if err != nil {
		return fmt.Errorf("marshal operation: %w", err)
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("append operation: %w", err)
	}
	s.logger.InfoContext(ctx, "append operation",
		"document_id", op.DocumentID,
		"operation_id", op.OperationID,
		"actor_id", op.ActorID,
		"operation_type", op.Type,
	)
	return nil
}

// SaveSnapshot stores one derived snapshot and updates the latest snapshot pointer.
func (s *FileStore) SaveSnapshot(ctx context.Context, snapshot SnapshotRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if snapshot.DocumentID == "" {
		return model.ErrEmptyDocumentID
	}
	if snapshot.SnapshotID == "" {
		return fmt.Errorf("snapshot_id must not be empty")
	}
	if snapshot.State.DocumentID != snapshot.DocumentID {
		return engine.ErrSnapshotDocumentMismatch
	}
	if snapshot.CreatedAt.IsZero() {
		snapshot.CreatedAt = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	docDir := s.documentDir(snapshot.DocumentID)
	snapshotDir := filepath.Join(docDir, "snapshots")
	if err := os.MkdirAll(snapshotDir, 0o755); err != nil {
		return fmt.Errorf("create snapshot directory: %w", err)
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	if err := os.WriteFile(s.snapshotPath(snapshot.DocumentID, snapshot.SnapshotID), data, 0o644); err != nil {
		return fmt.Errorf("write snapshot: %w", err)
	}
	if err := os.WriteFile(s.latestSnapshotPath(snapshot.DocumentID), data, 0o644); err != nil {
		return fmt.Errorf("write latest snapshot: %w", err)
	}
	s.logger.InfoContext(ctx, "save snapshot",
		"document_id", snapshot.DocumentID,
		"snapshot_id", snapshot.SnapshotID,
		"last_included_operation_id", snapshot.LastIncludedOperationID,
	)
	return nil
}

// LoadLatestSnapshot retrieves the latest stored snapshot for one document.
func (s *FileStore) LoadLatestSnapshot(ctx context.Context, documentID model.DocumentID) (*SnapshotRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if documentID == "" {
		return nil, model.ErrEmptyDocumentID
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.latestSnapshotPath(documentID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrSnapshotNotFound
		}
		return nil, fmt.Errorf("read latest snapshot: %w", err)
	}

	var snapshot SnapshotRecord
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("decode latest snapshot: %w", err)
	}
	return &snapshot, nil
}

// LoadOperationsAfter returns all logged operations after the given watermark.
func (s *FileStore) LoadOperationsAfter(ctx context.Context, documentID model.DocumentID, after model.OperationID) ([]model.Operation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if documentID == "" {
		return nil, model.ErrEmptyDocumentID
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(s.operationsPath(documentID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("open operation log: %w", err)
	}
	defer file.Close()

	ops := make([]model.Operation, 0)
	allOps := make([]model.Operation, 0)
	scanner := bufio.NewScanner(file)
	seenWatermark := after == ""
	watermarkFound := after == ""
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		var op model.Operation
		if err := json.Unmarshal(scanner.Bytes(), &op); err != nil {
			return nil, fmt.Errorf("decode operation log: %w", err)
		}
		allOps = append(allOps, op)
		if !seenWatermark {
			if op.OperationID == after {
				seenWatermark = true
				watermarkFound = true
			}
			continue
		}
		if after != "" && op.OperationID == after {
			continue
		}
		ops = append(ops, op)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan operation log: %w", err)
	}
	if after != "" && !watermarkFound {
		return allOps, nil
	}
	return ops, nil
}

// RebuildDocument reconstructs one document from the latest snapshot plus later operations.
func (s *FileStore) RebuildDocument(ctx context.Context, documentID model.DocumentID) (*engine.Document, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if documentID == "" {
		return nil, model.ErrEmptyDocumentID
	}

	snapshot, err := s.LoadLatestSnapshot(ctx, documentID)
	if err != nil && !errors.Is(err, ErrSnapshotNotFound) {
		return nil, err
	}

	var doc *engine.Document
	var after model.OperationID
	if snapshot != nil {
		doc, err = engine.NewDocumentFromSnapshot(snapshot.State)
		if err != nil {
			return nil, fmt.Errorf("restore snapshot: %w", err)
		}
		after = snapshot.LastIncludedOperationID
	} else {
		doc, err = engine.NewDocument(documentID)
		if err != nil {
			return nil, err
		}
	}

	ops, err := s.LoadOperationsAfter(ctx, documentID, after)
	if err != nil {
		return nil, err
	}
	for _, op := range ops {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if _, err := doc.Apply(op); err != nil {
			return nil, fmt.Errorf("rebuild apply %s: %w", op.OperationID, err)
		}
	}
	return doc, nil
}

func (s *FileStore) documentDir(documentID model.DocumentID) string {
	return filepath.Join(s.root, string(documentID))
}

func (s *FileStore) operationsPath(documentID model.DocumentID) string {
	return filepath.Join(s.documentDir(documentID), "operations.jsonl")
}

func (s *FileStore) latestSnapshotPath(documentID model.DocumentID) string {
	return filepath.Join(s.documentDir(documentID), "latest_snapshot.json")
}

func (s *FileStore) snapshotPath(documentID model.DocumentID, snapshotID model.SnapshotID) string {
	return filepath.Join(s.documentDir(documentID), "snapshots", fmt.Sprintf("%s.json", snapshotID))
}
