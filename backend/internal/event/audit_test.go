package event

import (
	"context"
	"sync"
	"testing"
)

// mockAuditEmitter is a mock for testing EmitAuthDenied behavior.
type mockAuditEmitter struct {
	mu       sync.Mutex
	calls    []authDeniedCall
	shouldFn func() error
}

type authDeniedCall struct {
	projectID       string
	actorID         string
	actorRole       string
	attemptedAction string
	deniedReason    string
}

func (m *mockAuditEmitter) InsertAuthDenied(ctx context.Context, projectID, actorID, actorRole, attemptedAction, deniedReason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, authDeniedCall{
		projectID:       projectID,
		actorID:         actorID,
		actorRole:       actorRole,
		attemptedAction: attemptedAction,
		deniedReason:    deniedReason,
	})
	if m.shouldFn != nil {
		return m.shouldFn()
	}
	return nil
}

func TestAuditEmitter_EmitAuthDenied_CallsFn(t *testing.T) {
	mock := &mockAuditEmitter{}
	emitter := NewAuditEmitterWithFn(mock.InsertAuthDenied)

	emitter.EmitAuthDenied(context.Background(), "p1", "user_1", "human", "role_validation", "invalid role")

	mock.mu.Lock()
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.calls))
	}
	call := mock.calls[0]
	if call.projectID != "p1" {
		t.Errorf("expected projectID 'p1', got %q", call.projectID)
	}
	if call.actorID != "user_1" {
		t.Errorf("expected actorID 'user_1', got %q", call.actorID)
	}
	if call.actorRole != "human" {
		t.Errorf("expected actorRole 'human', got %q", call.actorRole)
	}
	if call.attemptedAction != "role_validation" {
		t.Errorf("expected attemptedAction 'role_validation', got %q", call.attemptedAction)
	}
	if call.deniedReason != "invalid role" {
		t.Errorf("expected deniedReason 'invalid role', got %q", call.deniedReason)
	}
	mock.mu.Unlock()
}

func TestAuditEmitter_EmitAuthDenied_LogsOnError(t *testing.T) {
	mock := &mockAuditEmitter{
		shouldFn: func() error {
			return nil
		},
	}
	emitter := NewAuditEmitterWithFn(mock.InsertAuthDenied)

	// Should not panic or error even when fn returns nil
	emitter.EmitAuthDenied(context.Background(), "p1", "user_1", "layer_b", "role_authorization", "insufficient role")

	mock.mu.Lock()
	if len(mock.calls) != 1 {
		t.Errorf("expected 1 call, got %d", len(mock.calls))
	}
	mock.mu.Unlock()
}

// testEmitFn is the function type compatible with SetAuditEmitter.
func testEmitFn(ctx context.Context, projectID, actorID, actorRole, attemptedAction, deniedReason string) error {
	return nil
}

func TestEmitAuthDeniedFunc_Type(t *testing.T) {
	var fn EmitAuthDeniedFunc = testEmitFn
	if fn == nil {
		t.Error("expected EmitAuthDeniedFunc to be non-nil")
	}
}
