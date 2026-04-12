# Syncraft V1 Demo Release Evidence

## Purpose

This document is the local evidence landing page for the Syncraft v1 browser-demo release path.
It maps the agreed demo script and release checklist to concrete automated validation, runnable
entry points, and residual open issues.

Use this page when deciding whether the current repo state supports the v1 product claims defined
in `../01-product/mvp-scope-release-criteria.md`.

## Scope

This evidence page covers the shipped demo entry path:

```bash
go run ./cmd/demo-server
```

It does not expand scope beyond the approved v1 commitments. The evidence here is limited to:

- shared plain-text collaboration in the browser demo shell
- deterministic convergence and duplicate tolerance
- reconnect via snapshot plus delta
- restart recovery from persistence
- operator-visible error handling for the demo path

## Evidence Summary

As of 2026-04-12, the current repo state satisfies the demo-path correctness and recovery claims
required by the v1 release checklist.

- Automated test suite passes with `go test ./...`.
- The browser demo entrypoint responds successfully when started with
  `go run ./cmd/demo-server`.
- Product, planning, and demo docs align with the shipped browser shell.

Final demo-readiness sign-off is not yet fully closed because the practical v1 performance target
is still open in canonical project memory.

## Release Checklist Mapping

| Release item | Status | Evidence |
| --- | --- | --- |
| Shared plain-text editing works with at least two clients | Pass | `internal/demo/browser/server_test.go`: `TestBrowserShellSyncsTwoClients` |
| Concurrent edits converge consistently | Pass | `internal/demo/browser/server_test.go`: `TestBrowserShellSyncsTwoClients`; `internal/qa/harness/harness_test.go`: `TestPermutationHarnessPreservesConvergence` |
| Reconnect flow is verified | Pass | `internal/demo/browser/server_test.go`: `TestBrowserShellReconnectsWithCatchupState`; `internal/qa/harness/harness_test.go`: `TestReconnectHarnessMatchesContinuouslyConnectedReplica` |
| Restart recovery is verified | Pass | `internal/demo/browser/server_test.go`: `TestBrowserShellRecoversAfterServerRestart`; `internal/qa/harness/harness_test.go`: `TestRestartRecoveryHarnessMatchesContinuousReplica` |
| Persistence behavior is verified | Pass | `internal/backend/persistence/store_test.go`; restart-recovery and reconnect tests above depend on persisted snapshots and operation log replay |
| Duplicate delivery tests pass | Pass | `internal/qa/harness/harness_test.go`: `TestDuplicateDeliveryHarnessPreservesVisibleState` |
| Out-of-order delivery tests pass | Pass | `internal/qa/harness/harness_test.go`: `TestPermutationHarnessPreservesConvergence` |
| Demo script is repeatable | Pass | Automated browser-shell coverage above; local smoke run on 2026-04-12 returned HTTP 200 from `http://localhost:8080` after starting `go run ./cmd/demo-server` |
| Scope has not expanded beyond v1 commitments | Pass | `../01-product/prd-business-spec.md`, `../01-product/mvp-scope-release-criteria.md`, `../01-product/user-journeys-demo-script.md` |
| Product and technical docs are aligned | Pass | Product docs above plus `../02-canonical/protocol-spec.md` and `feature-breakdown-milestone-backlog.md` |

## Demo Journey Mapping

### Journey 1: First-Time Shared Editing

- Browser evidence: `TestBrowserShellSyncsTwoClients`
- Claim supported: two clients can edit one shared document and observe the same final text

### Journey 2: Concurrent Insert At The Same Logical Position

- Harness evidence: `TestPermutationHarnessPreservesConvergence`
- Claim supported: concurrent same-position edits converge deterministically

### Journey 3: Reconnect And Catch Up

- Browser evidence: `TestBrowserShellReconnectsWithCatchupState`
- Harness evidence: `TestReconnectHarnessMatchesContinuouslyConnectedReplica`
- Claim supported: reconnect restores the same visible state as a continuously connected replica

### Journey 4: Delivery Irregularities Do Not Corrupt State

- Harness evidence: `TestPermutationHarnessPreservesConvergence`
- Harness evidence: `TestDuplicateDeliveryHarnessPreservesVisibleState`
- Claim supported: duplicate and out-of-order delivery remain harmless to final visible state

### Journey 5: Server Restart Recovery

- Browser evidence: `TestBrowserShellRecoversAfterServerRestart`
- Harness evidence: `TestRestartRecoveryHarnessMatchesContinuousReplica`
- Claim supported: persistence-backed recovery survives backend restart without manual repair

## Operational Evidence

- `go test ./...` passes on the merged implementation branch.
- `go run ./cmd/demo-server` serves the browser shell successfully on `http://localhost:8080`.
- The browser shell exposes connection state and operator-visible errors, with browser-level
  coverage in `TestBrowserShellSurfacesInvalidCommandError`.

## Residual Open Issues

- The practical v1 performance target is still open in canonical memory and should be resolved
  before final demo-readiness sign-off is treated as settled.
- Final reviewer-security release review for validation, logging, and persistence boundaries
  should be recorded explicitly as the next gating task.

## Next Task

After this evidence page is accepted, the next release-hardening tasks are:

- record final reviewer-security sign-off for validation, logging, and persistence boundaries
- resolve or explicitly defer the practical v1 performance target in canonical memory
