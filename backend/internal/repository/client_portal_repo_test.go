package repository

import (
	"testing"

	"github.com/agent-orchestrator/backend/internal/service"
)

// Compile-time interface compliance checks.
// These verify that concrete repository types satisfy the service layer interfaces
// without requiring a live database connection.
func TestClientPortalRepos_InterfaceCompliance(t *testing.T) {
	// ClientPortalProjectRepository must implement service.ClientPortalProjectRepo
	var _ service.ClientPortalProjectRepo = (*ClientPortalProjectRepository)(nil)

	// ClientPortalApprovalRepository must implement service.ClientPortalApprovalRepo
	var _ service.ClientPortalApprovalRepo = (*ClientPortalApprovalRepository)(nil)

	// CommentRepository must implement service.ClientPortalCommentRepo
	var _ service.ClientPortalCommentRepo = (*CommentRepository)(nil)
}

// Verify that NewClientPortalProjectRepository and NewClientPortalApprovalRepository
// return the correct concrete types.
func TestClientPortalRepos_FactoryReturnsCorrectTypes(t *testing.T) {
	// These would panic at compile time if the return types were wrong.
	// We test that the constructors exist and are callable with a nil Pool.
	var pool *Pool

	cpProj := NewClientPortalProjectRepository(pool)
	if cpProj == nil {
		t.Error("NewClientPortalProjectRepository should not return nil")
	}

	cpApprove := NewClientPortalApprovalRepository(pool)
	if cpApprove == nil {
		t.Error("NewClientPortalApprovalRepository should not return nil")
	}

	commentRepo := NewCommentRepository(pool)
	if commentRepo == nil {
		t.Error("NewCommentRepository should not return nil")
	}

	// Verify type assertions compile (interface compliance is checked at compile time)
	var _ service.ClientPortalProjectRepo = cpProj
	var _ service.ClientPortalApprovalRepo = cpApprove
	var _ service.ClientPortalCommentRepo = commentRepo
}