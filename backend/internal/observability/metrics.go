package observability

import (
	"context"
	"sync"
	"time"
)

// Metrics is the client portal metrics emitter.
// It provides all counters, gauges, and histograms defined in BRD-03.
//
// All methods accept a context for future OpenTelemetry trace propagation.
// Counters accept an optional label map for multi-dimensional metrics.
type Metrics struct {
	mu sync.Mutex
	// counters
	portfolioViewTotal       int
	projectViewTotal        int
	publicationFailedTotal   int
	commentCreatedTotal      int
	commentEditedTotal       int
	commentDeletedTotal      int
	sseDisconnectTotal       int
	manualRefreshTotal       int
	submissionFailedTotal    int
	accessDeniedTotal        int
	decisionOutcomeTotal     map[string]int // labeled by outcome

	// gauges
	pendingApprovalsCurrent    int
	overdueDecisionsCurrent    int
	needMoreInformationCurrent int
	requestedChangesCurrent    int
	blockedProjectsCurrent     int
	atRiskProjectsCurrent       int
	oldestPendingDecisionAgeMs int64

	// histograms (stored as slice of observed values for test verification)
	portfolioLoadDurations []int64 // ms
	projectLoadDurations    []int64 // ms
	decisionTurnaroundMs    []int64 // ms
}

// NewMetrics creates a new client portal metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{
		decisionOutcomeTotal: make(map[string]int),
	}
}

// RecordPortfolioView increments the portfolio view counter.
func (m *Metrics) RecordPortfolioView(ctx context.Context) {
	m.mu.Lock()
	m.portfolioViewTotal++
	m.mu.Unlock()
}

// RecordProjectView increments the project view counter.
func (m *Metrics) RecordProjectView(ctx context.Context, projectID string) {
	m.mu.Lock()
	m.projectViewTotal++
	m.mu.Unlock()
}

// RecordPortfolioLoadDuration records a portfolio load duration sample in ms.
func (m *Metrics) RecordPortfolioLoadDuration(ctx context.Context, durationMs int64) {
	m.mu.Lock()
	m.portfolioLoadDurations = append(m.portfolioLoadDurations, durationMs)
	m.mu.Unlock()
}

// RecordProjectLoadDuration records a project load duration sample in ms.
func (m *Metrics) RecordProjectLoadDuration(ctx context.Context, durationMs int64) {
	m.mu.Lock()
	m.projectLoadDurations = append(m.projectLoadDurations, durationMs)
	m.mu.Unlock()
}

// RecordPendingApprovalsGauge updates the pending approvals gauge.
func (m *Metrics) RecordPendingApprovalsGauge(ctx context.Context, count int) {
	m.mu.Lock()
	m.pendingApprovalsCurrent = count
	m.mu.Unlock()
}

// RecordOverdueDecisionsGauge updates the overdue decisions gauge.
func (m *Metrics) RecordOverdueDecisionsGauge(ctx context.Context, count int) {
	m.mu.Lock()
	m.overdueDecisionsCurrent = count
	m.mu.Unlock()
}

// RecordOldestPendingDecisionAge updates the oldest pending decision age gauge in ms.
func (m *Metrics) RecordOldestPendingDecisionAge(ctx context.Context, ageMs int64) {
	m.mu.Lock()
	m.oldestPendingDecisionAgeMs = ageMs
	m.mu.Unlock()
}

// RecordDecisionTurnaround records a decision turnaround time sample in ms.
func (m *Metrics) RecordDecisionTurnaround(ctx context.Context, turnaroundMs int64) {
	m.mu.Lock()
	m.decisionTurnaroundMs = append(m.decisionTurnaroundMs, turnaroundMs)
	m.mu.Unlock()
}

// RecordDecisionOutcome increments the decision outcome counter with outcome label.
func (m *Metrics) RecordDecisionOutcome(ctx context.Context, outcome string) {
	m.mu.Lock()
	m.decisionOutcomeTotal[outcome]++
	m.mu.Unlock()
}

// RecordNeedMoreInformationGauge updates the need-more-information gauge.
func (m *Metrics) RecordNeedMoreInformationGauge(ctx context.Context, count int) {
	m.mu.Lock()
	m.needMoreInformationCurrent = count
	m.mu.Unlock()
}

// RecordRequestedChangesGauge updates the requested-changes gauge.
func (m *Metrics) RecordRequestedChangesGauge(ctx context.Context, count int) {
	m.mu.Lock()
	m.requestedChangesCurrent = count
	m.mu.Unlock()
}

// RecordBlockedProjectsGauge updates the blocked projects gauge.
func (m *Metrics) RecordBlockedProjectsGauge(ctx context.Context, count int) {
	m.mu.Lock()
	m.blockedProjectsCurrent = count
	m.mu.Unlock()
}

// RecordAtRiskProjectsGauge updates the at-risk projects gauge.
func (m *Metrics) RecordAtRiskProjectsGauge(ctx context.Context, count int) {
	m.mu.Lock()
	m.atRiskProjectsCurrent = count
	m.mu.Unlock()
}

// RecordPublicationValidationFailed increments the publication validation failed counter.
func (m *Metrics) RecordPublicationValidationFailed(ctx context.Context, reasonCategory string) {
	m.mu.Lock()
	m.publicationFailedTotal++
	m.mu.Unlock()
}

// RecordCommentCreated increments the comment created counter.
func (m *Metrics) RecordCommentCreated(ctx context.Context) {
	m.mu.Lock()
	m.commentCreatedTotal++
	m.mu.Unlock()
}

// RecordCommentEdited increments the comment edited counter.
func (m *Metrics) RecordCommentEdited(ctx context.Context) {
	m.mu.Lock()
	m.commentEditedTotal++
	m.mu.Unlock()
}

// RecordCommentDeleted increments the comment deleted counter.
func (m *Metrics) RecordCommentDeleted(ctx context.Context) {
	m.mu.Lock()
	m.commentDeletedTotal++
	m.mu.Unlock()
}

// RecordSSEDisconnect increments the SSE disconnect counter.
func (m *Metrics) RecordSSEDisconnect(ctx context.Context) {
	m.mu.Lock()
	m.sseDisconnectTotal++
	m.mu.Unlock()
}

// RecordManualRefresh increments the manual refresh counter with context label.
func (m *Metrics) RecordManualRefresh(ctx context.Context, contextLabel string) {
	m.mu.Lock()
	m.manualRefreshTotal++
	m.mu.Unlock()
}

// RecordSubmissionFailed increments the submission failed counter.
func (m *Metrics) RecordSubmissionFailed(ctx context.Context) {
	m.mu.Lock()
	m.submissionFailedTotal++
	m.mu.Unlock()
}

// RecordAccessDenied increments the access denied counter.
func (m *Metrics) RecordAccessDenied(ctx context.Context) {
	m.mu.Lock()
	m.accessDeniedTotal++
	m.mu.Unlock()
}

// --- Snapshot methods for testing ---

// PortfolioViewTotal returns the current portfolio view counter value.
func (m *Metrics) PortfolioViewTotal() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.portfolioViewTotal
}

// ProjectViewTotal returns the current project view counter value.
func (m *Metrics) ProjectViewTotal() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.projectViewTotal
}

// PublicationFailedTotal returns the current publication failed counter.
func (m *Metrics) PublicationFailedTotal() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.publicationFailedTotal
}

// CommentCreatedTotal returns the current comment created counter.
func (m *Metrics) CommentCreatedTotal() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.commentCreatedTotal
}

// CommentEditedTotal returns the current comment edited counter.
func (m *Metrics) CommentEditedTotal() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.commentEditedTotal
}

// CommentDeletedTotal returns the current comment deleted counter.
func (m *Metrics) CommentDeletedTotal() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.commentDeletedTotal
}

// SSEDisconnectTotal returns the current SSE disconnect counter.
func (m *Metrics) SSEDisconnectTotal() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sseDisconnectTotal
}

// ManualRefreshTotal returns the current manual refresh counter.
func (m *Metrics) ManualRefreshTotal() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.manualRefreshTotal
}

// SubmissionFailedTotal returns the current submission failed counter.
func (m *Metrics) SubmissionFailedTotal() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.submissionFailedTotal
}

// AccessDeniedTotal returns the current access denied counter.
func (m *Metrics) AccessDeniedTotal() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.accessDeniedTotal
}

// DecisionOutcome returns the count for a specific decision outcome label.
func (m *Metrics) DecisionOutcome(outcome string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.decisionOutcomeTotal[outcome]
}

// PendingApprovalsCurrent returns the current pending approvals gauge value.
func (m *Metrics) PendingApprovalsCurrent() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pendingApprovalsCurrent
}

// OverdueDecisionsCurrent returns the current overdue decisions gauge value.
func (m *Metrics) OverdueDecisionsCurrent() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.overdueDecisionsCurrent
}

// NeedMoreInformationCurrent returns the current need-more-information gauge.
func (m *Metrics) NeedMoreInformationCurrent() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.needMoreInformationCurrent
}

// RequestedChangesCurrent returns the current requested-changes gauge.
func (m *Metrics) RequestedChangesCurrent() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.requestedChangesCurrent
}

// BlockedProjectsCurrent returns the current blocked projects gauge.
func (m *Metrics) BlockedProjectsCurrent() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.blockedProjectsCurrent
}

// AtRiskProjectsCurrent returns the current at-risk projects gauge.
func (m *Metrics) AtRiskProjectsCurrent() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.atRiskProjectsCurrent
}

// OldestPendingDecisionAgeMs returns the current oldest pending decision age gauge in ms.
func (m *Metrics) OldestPendingDecisionAgeMs() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.oldestPendingDecisionAgeMs
}

// PortfolioLoadDurations returns all recorded portfolio load duration samples in ms.
func (m *Metrics) PortfolioLoadDurations() []int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.portfolioLoadDurations
}

// ProjectLoadDurations returns all recorded project load duration samples in ms.
func (m *Metrics) ProjectLoadDurations() []int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.projectLoadDurations
}

// DecisionTurnaroundMs returns all recorded decision turnaround samples in ms.
func (m *Metrics) DecisionTurnaroundMs() []int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.decisionTurnaroundMs
}

// ObserveManualRefreshMetrics is a test helper that captures snapshot values
// after simulating a portfolio load for metrics verification.
func ObserveManualRefreshMetrics(ctx context.Context, m *Metrics, label string) {
	m.RecordManualRefresh(ctx, label)
}

// Snapshot returns a point-in-time copy of gauge values for test assertions.
type MetricsSnapshot struct {
	PortfolioViewTotal            int
	ProjectViewTotal              int
	PublicationFailedTotal        int
	CommentCreatedTotal           int
	CommentEditedTotal            int
	CommentDeletedTotal           int
	SSEDisconnectTotal            int
	ManualRefreshTotal            int
	SubmissionFailedTotal         int
	AccessDeniedTotal             int
	PendingApprovalsCurrent        int
	OverdueDecisionsCurrent        int
	NeedMoreInformationCurrent     int
	RequestedChangesCurrent        int
	BlockedProjectsCurrent         int
	AtRiskProjectsCurrent           int
	OldestPendingDecisionAgeMs     int64
	DecisionOutcome               map[string]int
	PortfolioLoadDurations        []int64
	ProjectLoadDurations           []int64
	DecisionTurnaroundMs           []int64
}

// Snapshot captures a point-in-time copy of all metrics.
func (m *Metrics) Snapshot() MetricsSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	decisionOutcome := make(map[string]int, len(m.decisionOutcomeTotal))
	for k, v := range m.decisionOutcomeTotal {
		decisionOutcome[k] = v
	}
	portfolioDurations := make([]int64, len(m.portfolioLoadDurations))
	copy(portfolioDurations, m.portfolioLoadDurations)
	projectDurations := make([]int64, len(m.projectLoadDurations))
	copy(projectDurations, m.projectLoadDurations)
	turnaround := make([]int64, len(m.decisionTurnaroundMs))
	copy(turnaround, m.decisionTurnaroundMs)
	return MetricsSnapshot{
		PortfolioViewTotal:            m.portfolioViewTotal,
		ProjectViewTotal:              m.projectViewTotal,
		PublicationFailedTotal:        m.publicationFailedTotal,
		CommentCreatedTotal:           m.commentCreatedTotal,
		CommentEditedTotal:            m.commentEditedTotal,
		CommentDeletedTotal:           m.commentDeletedTotal,
		SSEDisconnectTotal:            m.sseDisconnectTotal,
		ManualRefreshTotal:            m.manualRefreshTotal,
		SubmissionFailedTotal:         m.submissionFailedTotal,
		AccessDeniedTotal:             m.accessDeniedTotal,
		PendingApprovalsCurrent:        m.pendingApprovalsCurrent,
		OverdueDecisionsCurrent:        m.overdueDecisionsCurrent,
		NeedMoreInformationCurrent:     m.needMoreInformationCurrent,
		RequestedChangesCurrent:        m.requestedChangesCurrent,
		BlockedProjectsCurrent:         m.blockedProjectsCurrent,
		AtRiskProjectsCurrent:          m.atRiskProjectsCurrent,
		OldestPendingDecisionAgeMs:     m.oldestPendingDecisionAgeMs,
		DecisionOutcome:               decisionOutcome,
		PortfolioLoadDurations:        portfolioDurations,
		ProjectLoadDurations:          projectDurations,
		DecisionTurnaroundMs:          turnaround,
	}
}

// ElapsedMs is a test helper that returns time.Since(start) in milliseconds.
func ElapsedMs(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}