package event

import (
	"context"
	"log"
)

// AuditEmitter inserts auth security events directly into the DB.
// Used by middleware that cannot access the EventService.
type AuditEmitter struct {
	insertFn EmitAuthDeniedFunc
}

// EmitAuthDeniedFunc is the function signature for inserting an auth event.
// This decouples the event package from the repository package.
type EmitAuthDeniedFunc func(ctx context.Context, projectID, actorID, actorRole, attemptedAction, deniedReason string) error

// NewAuditEmitterWithFn creates an AuditEmitter using a provided insertion function.
// This avoids circular imports between middleware and repository.
func NewAuditEmitterWithFn(fn EmitAuthDeniedFunc) *AuditEmitter {
	return &AuditEmitter{insertFn: fn}
}

// EmitAuthDenied inserts an auth.mutation.denied event for failed auth attempts.
// This is called by the authorization middleware when a mutation is denied.
// The event is inserted synchronously to guarantee the audit trail (NFR-02-014).
func (e *AuditEmitter) EmitAuthDenied(ctx context.Context, projectID, actorID, actorRole, attemptedAction, deniedReason string) {
	if e.insertFn == nil {
		return
	}
	if err := e.insertFn(ctx, projectID, actorID, actorRole, attemptedAction, deniedReason); err != nil {
		log.Printf("auth.mutation.denied event insert failed: %v", err)
	}
}


