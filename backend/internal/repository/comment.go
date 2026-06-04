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

// CreateComment inserts a new comment.
func (r *CommentRepository) CreateComment(ctx context.Context, c *models.ClientComment) error {
	if c.ID == "" {
		c.ID = "cmt_" + uuid.New().String()[:12]
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO comments (id, project_id, related_item_id, author_name, body, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		c.ID, c.ProjectID, c.RelatedItemID, c.AuthorName, c.Body, c.CreatedAt, c.UpdatedAt)
	return err
}

// EditComment edits an existing comment body. Only the comment author may edit.
func (r *CommentRepository) EditComment(ctx context.Context, commentID, authorID, newBody string) (*models.ClientComment, error) {
	now := time.Now()
	res, err := r.db.Exec(ctx, `
		UPDATE comments SET body=$1, updated_at=$2
		WHERE id=$3 AND author_name=$4`,
		newBody, now, commentID, authorID)
	if err != nil {
		return nil, err
	}
	affected := res.RowsAffected()
	if affected == 0 {
		return nil, nil // not found or not owner
	}
	// Fetch the updated comment
	row := r.db.QueryRow(ctx, `
		SELECT id, author_name, created_at, updated_at, project_id, related_item_id, body
		FROM comments WHERE id=$1`, commentID)
	var c models.ClientComment
	var updatedAt *time.Time
	if err := row.Scan(&c.ID, &c.AuthorName, &c.CreatedAt, &updatedAt, &c.ProjectID, &c.RelatedItemID, &c.Body); err != nil {
		return nil, err
	}
	c.UpdatedAt = updatedAt
	c.Edited = updatedAt != nil
	return &c, nil
}

// Delete removes a comment by ID and author. Returns true if deleted.
func (r *CommentRepository) Delete(ctx context.Context, commentID, authorID string) (bool, error) {
	res, err := r.db.Exec(ctx, `
		DELETE FROM comments WHERE id=$1 AND author_name=$2`,
		commentID, authorID)
	if err != nil {
		return false, err
	}
	affected := res.RowsAffected()
	return affected > 0, nil
}

// DeleteComment is an alias for Delete to match the service interface.
func (r *CommentRepository) DeleteComment(ctx context.Context, commentID, authorID string) (bool, error) {
	return r.Delete(ctx, commentID, authorID)
}
