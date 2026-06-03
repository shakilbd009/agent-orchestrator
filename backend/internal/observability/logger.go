package observability

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Logger is the structured logger for the client portal.
// All client portal log events use this logger with the "client_portal" subsystem.
type Logger struct {
	logger *slog.Logger
	attrs  []slog.Attr
	mu     sync.Mutex
}

// NewLogger creates a new client-portal structured logger.
func NewLogger() *Logger {
	return &Logger{
		logger: slog.Default(),
		attrs:  []slog.Attr{slog.String("subsystem", "client_portal")},
	}
}

// WithFields returns a new Logger with additional fields merged into every subsequent call.
func (l *Logger) WithFields(fields map[string]string) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	attrs := make([]slog.Attr, len(l.attrs), len(l.attrs)+len(fields))
	copy(attrs, l.attrs)
	for k, v := range fields {
		attrs = append(attrs, slog.String(k, v))
	}
	return &Logger{logger: l.logger, attrs: attrs}
}

// portfolioViewed emits client_portal.portfolio.viewed.
func (l *Logger) PortfolioViewed(ctx context.Context, principalID string, projectCount int) {
	l.log(ctx, "client_portal.portfolio.viewed",
		slog.String("principal_id", principalID),
		slog.Int("accessible_project_count", projectCount))
}

// projectViewed emits client_portal.project.viewed.
func (l *Logger) ProjectViewed(ctx context.Context, principalID, projectID string) {
	l.log(ctx, "client_portal.project.viewed",
		slog.String("principal_id", principalID),
		slog.String("project_id", projectID))
}

// approvalSubmitted emits client_portal.approval.submitted.
func (l *Logger) ApprovalSubmitted(ctx context.Context, projectID, itemID, outcome, actorID string) {
	l.log(ctx, "client_portal.approval.submitted",
		slog.String("project_id", projectID),
		slog.String("item_id", itemID),
		slog.String("outcome", outcome),
		slog.String("actor_id", actorID))
}

// approvalNeedMoreInformation emits client_portal.approval.need_more_information.
func (l *Logger) ApprovalNeedMoreInformation(ctx context.Context, itemID, actorID string, timestamp time.Time) {
	l.log(ctx, "client_portal.approval.need_more_information",
		slog.String("item_id", itemID),
		slog.String("actor_id", actorID),
		slog.Time("timestamp", timestamp))
}

// commentCreated emits client_portal.comment.created.
func (l *Logger) CommentCreated(ctx context.Context, projectID, relatedItemID, authorID string) {
	l.log(ctx, "client_portal.comment.created",
		slog.String("project_id", projectID),
		slog.String("related_item_id", relatedItemID),
		slog.String("author_id", authorID))
}

// commentEdited emits client_portal.comment.edited.
func (l *Logger) CommentEdited(ctx context.Context, projectID, commentID, editedBy string, timestamp time.Time) {
	l.log(ctx, "client_portal.comment.edited",
		slog.String("project_id", projectID),
		slog.String("comment_id", commentID),
		slog.String("edited_by", editedBy),
		slog.Time("timestamp", timestamp))
}

// commentDeleted emits client_portal.comment.deleted.
func (l *Logger) CommentDeleted(ctx context.Context, projectID, commentID, deletedBy string, timestamp time.Time) {
	l.log(ctx, "client_portal.comment.deleted",
		slog.String("project_id", projectID),
		slog.String("comment_id", commentID),
		slog.String("deleted_by", deletedBy),
		slog.Time("timestamp", timestamp))
}

// itemPublished emits client_portal.item.published.
func (l *Logger) ItemPublished(ctx context.Context, projectID, itemID, itemType, actorID string, validationResult string) {
	l.log(ctx, "client_portal.item.published",
		slog.String("project_id", projectID),
		slog.String("item_id", itemID),
		slog.String("item_type", itemType),
		slog.String("published_by", actorID),
		slog.String("validation_result", validationResult))
}

// itemUnpublished emits client_portal.item.unpublished.
func (l *Logger) ItemUnpublished(ctx context.Context, projectID, itemID, actorID, reason string) {
	l.log(ctx, "client_portal.item.unpublished",
		slog.String("project_id", projectID),
		slog.String("item_id", itemID),
		slog.String("unpublished_by", actorID),
		slog.String("reason", reason))
}

// publicationValidationFailed emits client_portal.publication_validation.failed.
func (l *Logger) PublicationValidationFailed(ctx context.Context, reasonCategory, projectID string) {
	l.log(ctx, "client_portal.publication_validation.failed",
		slog.String("reason_category", reasonCategory),
		slog.String("project_id", projectID))
}

// accessDenied emits client_portal.access.denied.
func (l *Logger) AccessDenied(ctx context.Context, principalID, resourceType, resourceID string) {
	l.log(ctx, "client_portal.access.denied",
		slog.String("principal_id", principalID),
		slog.String("resource_type", resourceType),
		slog.String("resource_id", resourceID))
}

// sseConnected emits client_portal.sse.connected.
func (l *Logger) SSEConnected(ctx context.Context, projectID string) {
	l.log(ctx, "client_portal.sse.connected",
		slog.String("project_id", projectID))
}

// sseDisconnected emits client_portal.sse.disconnected.
func (l *Logger) SSEDisconnected(ctx context.Context, projectID, reason string) {
	l.log(ctx, "client_portal.sse.disconnected",
		slog.String("project_id", projectID),
		slog.String("reason", reason))
}

// readOnlyModeEntered emits client_portal.read_only_mode.entered.
func (l *Logger) ReadOnlyModeEntered(ctx context.Context, reason string) {
	l.log(ctx, "client_portal.read_only_mode.entered",
		slog.String("reason", reason))
}

// readsUnavailable emits client_portal.reads.unavailable.
func (l *Logger) ReadsUnavailable(ctx context.Context, endpoint string) {
	l.log(ctx, "client_portal.reads.unavailable",
		slog.String("endpoint", endpoint))
}

func (l *Logger) log(ctx context.Context, event string, extra ...slog.Attr) {
	l.mu.Lock()
	allAttrs := make([]any, 0, len(l.attrs)+len(extra)+1)
	allAttrs = append(allAttrs, slog.String("event", event))
	for _, a := range l.attrs {
		allAttrs = append(allAttrs, a)
	}
	for _, a := range extra {
		allAttrs = append(allAttrs, a)
	}
	l.mu.Unlock()
	l.logger.Log(ctx, slog.LevelInfo, event, allAttrs...)
}