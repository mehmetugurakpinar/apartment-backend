package repository

import (
	"context"
	"fmt"

	"apartment-backend/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ForumRepository struct {
	pool *pgxpool.Pool
}

func NewForumRepository(pool *pgxpool.Pool) *ForumRepository {
	return &ForumRepository{pool: pool}
}

// GetOrCreateDefaultCategory returns the "General" category for a building, creating it if needed
func (r *ForumRepository) GetOrCreateDefaultCategory(ctx context.Context, buildingID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx,
		`SELECT id FROM forum_categories WHERE building_id = $1 AND slug = 'general'`,
		buildingID,
	).Scan(&id)
	if err == nil {
		return id, nil
	}

	// Create it
	err = r.pool.QueryRow(ctx,
		`INSERT INTO forum_categories (building_id, name, slug, sort_order) VALUES ($1, 'General', 'general', 0) RETURNING id`,
		buildingID,
	).Scan(&id)
	return id, err
}

// Categories

func (r *ForumRepository) GetCategories(ctx context.Context, buildingID uuid.UUID) ([]models.ForumCategory, error) {
	query := `
		SELECT id, building_id, name, slug, sort_order, created_at
		FROM forum_categories WHERE building_id = $1
		ORDER BY sort_order`

	rows, err := r.pool.Query(ctx, query, buildingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.ForumCategory
	for rows.Next() {
		var c models.ForumCategory
		if err := rows.Scan(&c.ID, &c.BuildingID, &c.Name, &c.Slug, &c.SortOrder, &c.CreatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, nil
}

func (r *ForumRepository) CreateCategory(ctx context.Context, cat *models.ForumCategory) error {
	query := `
		INSERT INTO forum_categories (building_id, name, slug, sort_order)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`
	return r.pool.QueryRow(ctx, query, cat.BuildingID, cat.Name, cat.Slug, cat.SortOrder).Scan(&cat.ID, &cat.CreatedAt)
}

// Posts

func (r *ForumRepository) CreatePost(ctx context.Context, post *models.ForumPost) error {
	query := `
		INSERT INTO forum_posts (building_id, category_id, author_id, title, body)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`
	return r.pool.QueryRow(ctx, query,
		post.BuildingID, post.CategoryID, post.AuthorID, post.Title, post.Body,
	).Scan(&post.ID, &post.CreatedAt, &post.UpdatedAt)
}

func (r *ForumRepository) GetPosts(ctx context.Context, buildingID uuid.UUID, categoryID *uuid.UUID, page, limit int) ([]models.ForumPostDetail, int64, error) {
	countQuery := `SELECT COUNT(*) FROM forum_posts WHERE building_id = $1`
	args := []interface{}{buildingID}

	if categoryID != nil {
		countQuery += ` AND category_id = $2`
		args = append(args, *categoryID)
	}

	var total int64
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT fp.id, fp.building_id, fp.category_id, fp.author_id, fp.title, fp.body,
			fp.is_pinned, fp.is_resolved, fp.upvotes, fp.downvotes, fp.comment_count,
			fp.created_at, fp.updated_at,
			u.full_name, u.avatar_url,
			fc.name as category_name
		FROM forum_posts fp
		JOIN users u ON fp.author_id = u.id
		JOIN forum_categories fc ON fp.category_id = fc.id
		WHERE fp.building_id = $1`

	offset := (page - 1) * limit
	queryArgs := []interface{}{buildingID}

	if categoryID != nil {
		query += ` AND fp.category_id = $2 ORDER BY fp.is_pinned DESC, fp.created_at DESC LIMIT $3 OFFSET $4`
		queryArgs = append(queryArgs, *categoryID, limit, offset)
	} else {
		query += ` ORDER BY fp.is_pinned DESC, fp.created_at DESC LIMIT $2 OFFSET $3`
		queryArgs = append(queryArgs, limit, offset)
	}

	rows, err := r.pool.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var posts []models.ForumPostDetail
	for rows.Next() {
		var p models.ForumPostDetail
		if err := rows.Scan(
			&p.ID, &p.BuildingID, &p.CategoryID, &p.AuthorID, &p.Title, &p.Body,
			&p.IsPinned, &p.IsResolved, &p.Upvotes, &p.Downvotes, &p.CommentCount,
			&p.CreatedAt, &p.UpdatedAt,
			&p.AuthorName, &p.AuthorAvatar,
			&p.CategoryName,
		); err != nil {
			return nil, 0, err
		}
		// Load media
		media, _ := r.GetMedia(ctx, p.ID)
		p.Media = media
		posts = append(posts, p)
	}
	return posts, total, nil
}

func (r *ForumRepository) GetPostByID(ctx context.Context, postID uuid.UUID, userID uuid.UUID) (*models.ForumPostDetail, error) {
	query := `
		SELECT fp.id, fp.building_id, fp.category_id, fp.author_id, fp.title, fp.body,
			fp.is_pinned, fp.is_resolved, fp.upvotes, fp.downvotes, fp.comment_count,
			fp.created_at, fp.updated_at,
			u.full_name, u.avatar_url,
			fc.name as category_name
		FROM forum_posts fp
		JOIN users u ON fp.author_id = u.id
		JOIN forum_categories fc ON fp.category_id = fc.id
		WHERE fp.id = $1`

	p := &models.ForumPostDetail{}
	err := r.pool.QueryRow(ctx, query, postID).Scan(
		&p.ID, &p.BuildingID, &p.CategoryID, &p.AuthorID, &p.Title, &p.Body,
		&p.IsPinned, &p.IsResolved, &p.Upvotes, &p.Downvotes, &p.CommentCount,
		&p.CreatedAt, &p.UpdatedAt,
		&p.AuthorName, &p.AuthorAvatar,
		&p.CategoryName,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("forum post not found")
		}
		return nil, err
	}

	// Get user's vote
	var vote int
	err = r.pool.QueryRow(ctx, `SELECT value FROM forum_votes WHERE post_id = $1 AND user_id = $2`, postID, userID).Scan(&vote)
	if err == nil {
		p.UserVote = &vote
	}

	// Get media
	media, _ := r.GetMedia(ctx, postID)
	p.Media = media

	// Get comments
	comments, err := r.GetComments(ctx, postID)
	if err != nil {
		return nil, err
	}
	p.Comments = comments

	return p, nil
}

// Comments

func (r *ForumRepository) CreateComment(ctx context.Context, comment *models.ForumComment) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO forum_comments (post_id, parent_id, author_id, body)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`
	err = tx.QueryRow(ctx, query, comment.PostID, comment.ParentID, comment.AuthorID, comment.Body).
		Scan(&comment.ID, &comment.CreatedAt, &comment.UpdatedAt)
	if err != nil {
		return err
	}

	// Increment comment count
	_, err = tx.Exec(ctx, `UPDATE forum_posts SET comment_count = comment_count + 1 WHERE id = $1`, comment.PostID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *ForumRepository) GetComments(ctx context.Context, postID uuid.UUID) ([]models.CommentDetail, error) {
	query := `
		SELECT fc.id, fc.post_id, fc.parent_id, fc.author_id, fc.body, fc.created_at, fc.updated_at,
			u.full_name, u.avatar_url
		FROM forum_comments fc
		JOIN users u ON fc.author_id = u.id
		WHERE fc.post_id = $1
		ORDER BY fc.created_at`

	rows, err := r.pool.Query(ctx, query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.CommentDetail
	for rows.Next() {
		var c models.CommentDetail
		if err := rows.Scan(
			&c.ID, &c.PostID, &c.ParentID, &c.AuthorID, &c.Body,
			&c.CreatedAt, &c.UpdatedAt,
			&c.AuthorName, &c.AuthorAvatar,
		); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, nil
}

// Media

func (r *ForumRepository) AddMedia(ctx context.Context, postID uuid.UUID, url, mediaType string) (*models.ForumMedia, error) {
	m := &models.ForumMedia{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO forum_media (post_id, url, type) VALUES ($1, $2, $3) RETURNING id, post_id, url, type, created_at`,
		postID, url, mediaType,
	).Scan(&m.ID, &m.PostID, &m.URL, &m.Type, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (r *ForumRepository) GetMedia(ctx context.Context, postID uuid.UUID) ([]models.ForumMedia, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, post_id, url, type, created_at FROM forum_media WHERE post_id = $1 ORDER BY created_at`,
		postID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var media []models.ForumMedia
	for rows.Next() {
		var m models.ForumMedia
		if err := rows.Scan(&m.ID, &m.PostID, &m.URL, &m.Type, &m.CreatedAt); err != nil {
			return nil, err
		}
		media = append(media, m)
	}
	return media, nil
}

// Votes

func (r *ForumRepository) Vote(ctx context.Context, vote *models.ForumVote) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Check existing vote
	var existingValue int
	err = tx.QueryRow(ctx, `SELECT value FROM forum_votes WHERE post_id = $1 AND user_id = $2`,
		vote.PostID, vote.UserID).Scan(&existingValue)

	if err == nil {
		// Update existing vote
		if existingValue == vote.Value {
			// Remove vote
			_, err = tx.Exec(ctx, `DELETE FROM forum_votes WHERE post_id = $1 AND user_id = $2`, vote.PostID, vote.UserID)
			if err != nil {
				return err
			}
			if existingValue == 1 {
				_, err = tx.Exec(ctx, `UPDATE forum_posts SET upvotes = upvotes - 1 WHERE id = $1`, vote.PostID)
			} else {
				_, err = tx.Exec(ctx, `UPDATE forum_posts SET downvotes = downvotes - 1 WHERE id = $1`, vote.PostID)
			}
		} else {
			// Change vote
			_, err = tx.Exec(ctx, `UPDATE forum_votes SET value = $3 WHERE post_id = $1 AND user_id = $2`,
				vote.PostID, vote.UserID, vote.Value)
			if err != nil {
				return err
			}
			if vote.Value == 1 {
				_, err = tx.Exec(ctx, `UPDATE forum_posts SET upvotes = upvotes + 1, downvotes = downvotes - 1 WHERE id = $1`, vote.PostID)
			} else {
				_, err = tx.Exec(ctx, `UPDATE forum_posts SET upvotes = upvotes - 1, downvotes = downvotes + 1 WHERE id = $1`, vote.PostID)
			}
		}
	} else if err == pgx.ErrNoRows {
		// New vote
		_, err = tx.Exec(ctx, `INSERT INTO forum_votes (post_id, user_id, value) VALUES ($1, $2, $3)`,
			vote.PostID, vote.UserID, vote.Value)
		if err != nil {
			return err
		}
		if vote.Value == 1 {
			_, err = tx.Exec(ctx, `UPDATE forum_posts SET upvotes = upvotes + 1 WHERE id = $1`, vote.PostID)
		} else {
			_, err = tx.Exec(ctx, `UPDATE forum_posts SET downvotes = downvotes + 1 WHERE id = $1`, vote.PostID)
		}
	} else {
		return err
	}

	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
