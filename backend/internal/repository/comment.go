package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/agent-orchestrator/backend/internal/models"
)

// CommentRepository handles comment CRUD with project/item scope.
type CommentRepository struct {
	db *Pool
}

// NewCommentRepository creates a comment repository.
func NewCommentRepository(db *Pool) *CommentRepository {
	return &CommentRepository{db: db}
}

// ListByProjectAndItem returns all comments for a given project and item.
// itemID may equal projectID to fetch project-level comments.
func (r *CommentRepository) ListByProjectAndItem(ctx context.Context, projectID, itemID string) ([]models.ClientComment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, author_name, created_at, updated_at, project_id, related_item_id, body
		FROM comments
		WHERE project_id=$1 AND related_item_id=$2
		ORDER BY created_at ASC`, projectID, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.ClientComment
	for rows.Next() {
		var c models.ClientComment
		var updatedAt *time.Time
		if err := rows.Scan(&c.ID, &c.AuthorName, &c.CreatedAt, &updatedAt, &c.ProjectID, &c.RelatedItemID, &c.Body); err != nil {
			return nil, err
		}
		c.UpdatedAt = updatedAt
		c.Edited = updatedAt != nil
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

// Insert adds a new comment.
func (r *CommentRepository) Insert(ctx context.Context, c *models.ClientComment) error {
	if c.ID == "" {
		c.ID = "cmt_" + uuid.New().String()[:12]
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO comments (id, project_id, item_id, author_name, body, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		c.ID, c.ProjectID, c.RelatedItemID, c.AuthorName, c.Body, c.CreatedAt, c.UpdatedAt)
	return err
}