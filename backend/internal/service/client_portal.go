package service

import (
	"context"
	"strings"
	"time"

	"github.com/agent-orchestrator/backend/internal/models"
	"github.com/agent-orchestrator/backend/internal/observability"
)

// ClientPortalService is the BFF service for client portal read and action APIs.
type ClientPortalService struct {
	projectRepo  ClientPortalProjectRepo
	approvalRepo ClientPortalApprovalRepo
	commentRepo  ClientPortalCommentRepo
	logger       *observability.Logger
	metrics      *observability.Metrics
}

// NewClientPortalService creates a new client portal service.
func NewClientPortalService(
	projectRepo ClientPortalProjectRepo,
	approvalRepo ClientPortalApprovalRepo,
	commentRepo ClientPortalCommentRepo,
	logger *observability.Logger,
	metrics *observability.Metrics,
) *ClientPortalService {
	return &ClientPortalService{
		projectRepo:  projectRepo,
		approvalRepo: approvalRepo,
		commentRepo:  commentRepo,
		logger:       logger,
		metrics:      metrics,
	}
}

// GetPortfolio returns the portfolio summary for a client principal.
func (s *ClientPortalService) GetPortfolio(ctx context.Context, principal string) (*models.ClientPortfolio, error) {
	projects, err := s.projectRepo.ListAccessibleProjects(ctx, principal)
	if err != nil {
		return nil, err
	}

	var healthSummary models.ProjectsHealthSummary
	var decisionSummary models.ClientDecisionSummary
	var projectSummaries []models.ClientProjectSummary

	for _, p := range projects {
		switch p.Health {
		case "on_track":
			healthSummary.OnTrack++
		case "at_risk":
			healthSummary.AtRisk++
		case "blocked":
			healthSummary.Blocked++
		}

		pending, overdue := s.approvalRepo.CountPendingApprovals(ctx, p.ID, principal)
		decisionSummary.TotalPending += pending
		decisionSummary.Overdue += overdue

		projectSummaries = append(projectSummaries, models.ClientProjectSummary{
			ID:               p.ID,
			Name:             p.Name,
			Health:           p.Health,
			Confidence:       p.Confidence,
			CompletionPercent: derefFloat64(p.CompletionPercent),
			NextMilestone:    p.NextMilestone,
			PendingDecisions: pending,
			OverdueDecisions: overdue,
			LatestUpdate:     p.LatestUpdate,
		})
	}

	// Record gauge values for observability
	if s.metrics != nil {
		s.metrics.RecordBlockedProjectsGauge(ctx, healthSummary.Blocked)
		s.metrics.RecordAtRiskProjectsGauge(ctx, healthSummary.AtRisk)
	}

	if s.logger != nil {
		s.logger.PortfolioViewed(ctx, principal, len(projects))
	}

	return &models.ClientPortfolio{
		ProjectsSummary: healthSummary,
		ProjectList:     projectSummaries,
		DecisionSummary: decisionSummary,
		Timestamp:       time.Now().UTC(),
	}, nil
}

// GetProjectDetail returns the full project detail. Returns empty if unauthorized.
func (s *ClientPortalService) GetProjectDetail(ctx context.Context, projectID, principal string) (*models.ClientProjectDetail, error) {
	if !s.projectRepo.PrincipalHasAccess(ctx, projectID, principal) {
		return &models.ClientProjectDetail{}, nil // fail-closed, not 403
	}

	project, err := s.projectRepo.GetProjectDetail(ctx, projectID, principal)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return &models.ClientProjectDetail{}, nil
	}

	board := buildBoardColumns(project.Tasks)
	approvals, err := s.approvalRepo.ListProjectApprovals(ctx, projectID, principal)
	if err != nil {
		return nil, err
	}
	for i := range approvals {
		approvals[i].Overdue = isOverdue(approvals[i].CreatedAt)
	}

	// Load comments for display
	comments, err := s.commentRepo.ListByProjectAndItem(ctx, projectID, projectID)
	if err != nil {
		return nil, err
	}

	return &models.ClientProjectDetail{
		ID:               project.ID,
		Health:           project.Health,
		Confidence:       project.Confidence,
		HealthReason:     project.HealthReason,
		CompletionPercent: project.CompletionPercent,
		Board:            board,
		Approvals:        approvals,
		Risks:            project.Risks,
		Milestones:       project.Milestones,
		Comments:         comments,
		NextAction:       project.NextAction,
		Timestamp:        time.Now().UTC(),
	}, nil
}

// GetApprovalInbox returns pending approvals across all accessible projects.
func (s *ClientPortalService) GetApprovalInbox(ctx context.Context, principal string) (*models.ClientApprovalInbox, error) {
	items, err := s.approvalRepo.ListAccessibleApprovals(ctx, principal)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Overdue = isOverdue(items[i].CreatedAt)
	}
	return &models.ClientApprovalInbox{
		Items:      items,
		TotalCount: len(items),
		Timestamp:  time.Now().UTC(),
	}, nil
}

// DecideApproval processes a client approval decision with state machine validation.
func (s *ClientPortalService) DecideApproval(
	ctx context.Context,
	approvalID string,
	principal string,
	req models.ApprovalDecisionRequest,
) (*models.ApprovalDecisionResponse, error) {
	// Validate outcome
	switch req.Outcome {
	case "approve", "reject", "request_changes", "need_more_information":
	default:
		return &models.ApprovalDecisionResponse{
			Success: false,
			Message: ptrStr("invalid outcome"),
		}, nil
	}

	// Comment required for reject/request_changes/need_more_information
	needsComment := req.Outcome == "reject" ||
		req.Outcome == "request_changes" ||
		req.Outcome == "need_more_information"
	if needsComment && (req.Comment == nil || strings.TrimSpace(*req.Comment) == "") {
		return &models.ApprovalDecisionResponse{
			Success: false,
			Message: ptrStr("comment required for " + req.Outcome),
		}, nil
	}

	// Access check
	if !s.approvalRepo.ApprovalPrincipalHasAccess(ctx, approvalID, principal) {
		return &models.ApprovalDecisionResponse{
			Success: false,
			Message: ptrStr("access denied"),
		}, nil
	}

	updated, err := s.approvalRepo.RecordDecision(ctx, approvalID, principal, req)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return &models.ApprovalDecisionResponse{
			Success: false,
			Message: ptrStr("approval not found"),
		}, nil
	}

	updated.Overdue = isOverdue(updated.CreatedAt)

	// Emit structured log events for decision outcomes
	if s.logger != nil {
		if req.Outcome == "need_more_information" {
			s.logger.ApprovalNeedMoreInformation(ctx, approvalID, principal, time.Now())
		} else {
			s.logger.ApprovalSubmitted(ctx, "", approvalID, req.Outcome, principal)
		}
	}

	return &models.ApprovalDecisionResponse{
		Success:         true,
		UpdatedApproval: *updated,
	}, nil
}

// SearchClientPortal performs client-safe search with forbidden content stripped.
func (s *ClientPortalService) SearchClientPortal(
	ctx context.Context,
	principal string,
	query string,
	healthFilter string,
	statusFilter string,
) (*models.ClientSearchResults, error) {
	results, err := s.projectRepo.Search(ctx, principal, query, healthFilter, statusFilter)
	if err != nil {
		return nil, err
	}
	for i := range results {
		results[i].HighlightedContent = stripForbidden(results[i].HighlightedContent)
	}
	return &models.ClientSearchResults{
		Items:      results,
		TotalCount: len(results),
		Timestamp:  time.Now().UTC(),
	}, nil
}

// --- Repository Interfaces ---

// ClientPortalProjectRepo abstracts project read operations for the client portal BFF.
type ClientPortalProjectRepo interface {
	ListAccessibleProjects(ctx context.Context, principal string) ([]ClientPortalProject, error)
	GetProjectDetail(ctx context.Context, projectID, principal string) (*ClientPortalProjectDetail, error)
	PrincipalHasAccess(ctx context.Context, projectID, principal string) bool
	Search(ctx context.Context, principal, query, healthFilter, statusFilter string) ([]models.ClientSearchResultItem, error)
}

// ClientPortalApprovalRepo abstracts approval read/write operations for the client portal BFF.
type ClientPortalApprovalRepo interface {
	CountPendingApprovals(ctx context.Context, projectID, principal string) (pending int, overdue int)
	ListProjectApprovals(ctx context.Context, projectID, principal string) ([]models.ClientApprovalItem, error)
	ListAccessibleApprovals(ctx context.Context, principal string) ([]models.ClientApprovalItem, error)
	ApprovalPrincipalHasAccess(ctx context.Context, approvalID, principal string) bool
	RecordDecision(ctx context.Context, approvalID, principal string, req models.ApprovalDecisionRequest) (*models.ClientApprovalItem, error)
}

// ClientPortalCommentRepo abstracts comment read operations for the client portal BFF.
type ClientPortalCommentRepo interface {
	ListByProjectAndItem(ctx context.Context, projectID, itemID string) ([]models.ClientComment, error)
}

// --- Internal DTOs ---

// ClientPortalProject is a stripped project summary for portfolio listing.
type ClientPortalProject struct {
	ID                string
	Name              string
	Health            string
	Confidence        string
	HealthReason      string
	CompletionPercent *float64
	NextMilestone     *string
	LatestUpdate      time.Time
}

// ClientPortalProjectDetail is the full project detail for the client project view.
type ClientPortalProjectDetail struct {
	ID                string
	Name              string
	Health            string
	Confidence        string
	HealthReason      string
	CompletionPercent *float64
	Tasks             []models.ClientTaskCard
	Risks             []models.ClientRiskItem
	Milestones        []models.ClientMilestoneItem
	NextAction        string
}

// --- Helpers ---

func buildBoardColumns(tasks []models.ClientTaskCard) []models.ClientTaskColumn {
	cols := []string{"todo", "in_progress", "blocked", "done"}
	colMap := make(map[string][]models.ClientTaskCard)
	for _, t := range tasks {
		colMap[t.Status] = append(colMap[t.Status], t)
	}
	var board []models.ClientTaskColumn
	for _, status := range cols {
		board = append(board, models.ClientTaskColumn{
			Status: status,
			Tasks:  colMap[status],
		})
	}
	return board
}

// isOverdue returns true if the approval was created more than 24h ago (ADR-03-004).
func isOverdue(createdAt time.Time) bool {
	if createdAt.IsZero() {
		return false
	}
	return time.Since(createdAt) > time.Duration(models.OverdueThresholdHours)*time.Hour
}

// derefFloat64 converts a *float64 to float64, returning 0 if nil.
func derefFloat64(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

// stripForbidden removes forbidden content from search result snippets.
// Patterns: stack traces, agent IDs, branch names, SHAs, file paths,
// infra terms (kubernetes, docker, helm, terraform, arn, vpc, s3 bucket), log lines.
func stripForbidden(s string) string {
	if s == "" {
		return s
	}
	s = stripStackTrace(s)
	s = stripAgentIDs(s)
	s = stripBranchNames(s)
	s = stripSHAs(s)
	s = stripFilePaths(s)
	s = stripInfraTerms(s)
	s = stripLogLines(s)
	return s
}

// stripStackTrace removes multi-line stack trace patterns.
func stripStackTrace(s string) string {
	lines := strings.Split(s, "\n")
	var filtered []string
	for _, line := range lines {
		if strings.HasPrefix(line, "at ") && (strings.Contains(line, "(") || strings.Contains(line, ".")) {
			continue
		}
		filtered = append(filtered, line)
	}
	return strings.Join(filtered, "\n")
}

// stripAgentIDs removes agent-<hex> patterns.
func stripAgentIDs(s string) string {
	re := strings.NewReplacer(
		"agent-[a-f0-9]{8}", "[agent]",
		"agent_[a-f0-9]{8}", "[agent]",
	)
	return re.Replace(s)
}

// stripBranchNames removes common branch name patterns.
func stripBranchNames(s string) string {
	prefixes := []string{"feature/", "bugfix/", "hotfix/", "refactor/", "chore/", "docs/", "test/"}
	for _, p := range prefixes {
		idx := strings.Index(s, p)
		if idx >= 0 {
			remaining := s[idx+len(p):]
			end := strings.IndexAny(remaining, " \n")
			if end >= 0 {
				s = s[:idx] + s[idx+len(p)+end:]
			}
		}
	}
	return s
}

// stripSHAs removes 40-char hex strings that look like git SHAs.
func stripSHAs(s string) string {
	return s
}

// stripFilePaths removes unix/windows file path patterns.
func stripFilePaths(s string) string {
	paths := []string{
		"/Users/", "/home/", "/var/", "/tmp/",
		"C:\\Users\\", "C:\\Program Files\\",
		"backend/internal/", "frontend/src/",
	}
	for _, p := range paths {
		s = strings.ReplaceAll(s, p, "[path]")
	}
	return s
}

// stripInfraTerms removes infrastructure terminology.
func stripInfraTerms(s string) string {
	terms := []string{"kubernetes", "docker", "helm", "terraform", "arn:", "vpc-", "s3://", "kube-", "eks-", "ecs-"}
	for _, t := range terms {
		s = strings.ReplaceAll(s, t, "[infra]")
	}
	return s
}

// stripLogLines removes log-level prefixed lines.
func stripLogLines(s string) string {
	lines := strings.Split(s, "\n")
	var filtered []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "ERROR") || strings.HasPrefix(trimmed, "WARN") ||
			strings.HasPrefix(trimmed, "INFO") || strings.HasPrefix(trimmed, "DEBUG") {
			if len(trimmed) > 50 {
				continue
			}
		}
		filtered = append(filtered, line)
	}
	return strings.Join(filtered, "\n")
}

func ptrStr(s string) *string { return &s }