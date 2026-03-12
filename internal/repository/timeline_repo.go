package repository

import (
	"context"
	"fmt"

	"apartment-backend/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TimelineRepository struct {
	pool *pgxpool.Pool
}

func NewTimelineRepository(pool *pgxpool.Pool) *TimelineRepository {
	return &TimelineRepository{pool: pool}
}

func (r *TimelineRepository) Create(ctx context.Context, post *models.TimelinePost) error {
	query := `
		INSERT INTO timeline_posts (author_id, building_id, content, type, visibility, location_lat, location_lng, original_post_id, is_repost)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`
	return r.pool.QueryRow(ctx, query,
		post.AuthorID, post.BuildingID, post.Content, post.Type,
		post.Visibility, post.LocationLat, post.LocationLng,
		post.OriginalPostID, post.IsRepost,
	).Scan(&post.ID, &post.CreatedAt, &post.UpdatedAt)
}

func (r *TimelineRepository) Repost(ctx context.Context, originalPostID, userID uuid.UUID) (*models.TimelinePost, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Check if already reposted
	var exists bool
	err = tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM timeline_posts WHERE original_post_id = $1 AND author_id = $2 AND is_repost = true)`,
		originalPostID, userID,
	).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("already reposted")
	}

	// Get original post
	var origBuildingID uuid.UUID
	var origContent *string
	var origType models.TimelinePostType
	var origVisibility models.VisibilityType
	err = tx.QueryRow(ctx,
		`SELECT building_id, content, type, visibility FROM timeline_posts WHERE id = $1`,
		originalPostID,
	).Scan(&origBuildingID, &origContent, &origType, &origVisibility)
	if err != nil {
		return nil, fmt.Errorf("original post not found")
	}

	// Create repost
	repost := &models.TimelinePost{
		AuthorID:       userID,
		BuildingID:     origBuildingID,
		Content:        origContent,
		Type:           origType,
		Visibility:     origVisibility,
		OriginalPostID: &originalPostID,
		IsRepost:       true,
	}

	err = tx.QueryRow(ctx,
		`INSERT INTO timeline_posts (author_id, building_id, content, type, visibility, original_post_id, is_repost)
		 VALUES ($1, $2, $3, $4, $5, $6, true)
		 RETURNING id, created_at, updated_at`,
		repost.AuthorID, repost.BuildingID, repost.Content, repost.Type, repost.Visibility, repost.OriginalPostID,
	).Scan(&repost.ID, &repost.CreatedAt, &repost.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Increment repost count on original
	_, err = tx.Exec(ctx,
		`UPDATE timeline_posts SET repost_count = repost_count + 1 WHERE id = $1`,
		originalPostID,
	)
	if err != nil {
		return nil, err
	}

	return repost, tx.Commit(ctx)
}

func (r *TimelineRepository) Unrepost(ctx context.Context, originalPostID, userID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	result, err := tx.Exec(ctx,
		`DELETE FROM timeline_posts WHERE original_post_id = $1 AND author_id = $2 AND is_repost = true`,
		originalPostID, userID,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return nil
	}

	_, err = tx.Exec(ctx,
		`UPDATE timeline_posts SET repost_count = GREATEST(repost_count - 1, 0) WHERE id = $1`,
		originalPostID,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *TimelineRepository) GetFeed(ctx context.Context, buildingIDs []uuid.UUID, followedUserIDs []uuid.UUID, userID uuid.UUID, page, limit int) ([]models.TimelinePostDetail, int64, error) {
	// Build WHERE clause based on available data
	whereClause := "WHERE false"
	args := []interface{}{}
	argIdx := 1

	if len(buildingIDs) > 0 {
		whereClause = fmt.Sprintf("WHERE (tp.building_id = ANY($%d))", argIdx)
		args = append(args, buildingIDs)
		argIdx++
	}

	if len(followedUserIDs) > 0 {
		if whereClause == "WHERE false" {
			whereClause = fmt.Sprintf("WHERE (tp.author_id = ANY($%d) AND tp.visibility IN ('public','neighborhood'))", argIdx)
		} else {
			whereClause += fmt.Sprintf(" OR (tp.author_id = ANY($%d) AND tp.visibility IN ('public','neighborhood'))", argIdx)
		}
		args = append(args, followedUserIDs)
		argIdx++
	}

	if len(buildingIDs) == 0 && len(followedUserIDs) == 0 {
		return nil, 0, nil
	}

	// Count
	var total int64
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM timeline_posts tp %s`, whereClause)
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Add userID for is_liked subquery
	userIDIdx := argIdx
	args = append(args, userID)
	argIdx++

	limitIdx := argIdx
	args = append(args, limit)
	argIdx++

	offsetIdx := argIdx
	offset := (page - 1) * limit
	args = append(args, offset)

	query := fmt.Sprintf(`
		SELECT tp.id, tp.author_id, tp.building_id, tp.content, tp.type, tp.visibility,
			tp.location_lat, tp.location_lng, tp.like_count, tp.comment_count,
			tp.repost_count, tp.original_post_id, tp.is_repost,
			tp.created_at, tp.updated_at,
			u.full_name, u.avatar_url,
			EXISTS(SELECT 1 FROM timeline_likes tl WHERE tl.post_id = tp.id AND tl.user_id = $%d) as is_liked,
			EXISTS(SELECT 1 FROM timeline_posts rp WHERE rp.original_post_id = tp.id AND rp.author_id = $%d AND rp.is_repost = true) as is_reposted,
			orig_u.full_name as original_author_name
		FROM timeline_posts tp
		JOIN users u ON tp.author_id = u.id
		LEFT JOIN timeline_posts orig ON tp.original_post_id = orig.id
		LEFT JOIN users orig_u ON orig.author_id = orig_u.id
		%s
		ORDER BY tp.created_at DESC
		LIMIT $%d OFFSET $%d`,
		userIDIdx, userIDIdx, whereClause, limitIdx, offsetIdx)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var posts []models.TimelinePostDetail
	for rows.Next() {
		var p models.TimelinePostDetail
		if err := rows.Scan(
			&p.ID, &p.AuthorID, &p.BuildingID, &p.Content, &p.Type, &p.Visibility,
			&p.LocationLat, &p.LocationLng, &p.LikeCount, &p.CommentCount,
			&p.RepostCount, &p.OriginalPostID, &p.IsRepost,
			&p.CreatedAt, &p.UpdatedAt,
			&p.AuthorName, &p.AuthorAvatar,
			&p.IsLiked, &p.IsReposted,
			&p.OriginalAuthorName,
		); err != nil {
			return nil, 0, err
		}

		// Get media
		media, _ := r.GetMedia(ctx, p.ID)
		p.Media = media

		// Get poll if type is poll
		if p.Type == models.PostTypePoll {
			poll, _ := r.GetPollByPost(ctx, p.ID, userID)
			p.Poll = poll
		}

		posts = append(posts, p)
	}
	return posts, total, nil
}

func (r *TimelineRepository) GetByID(ctx context.Context, postID, userID uuid.UUID) (*models.TimelinePostDetail, error) {
	query := `
		SELECT tp.id, tp.author_id, tp.building_id, tp.content, tp.type, tp.visibility,
			tp.location_lat, tp.location_lng, tp.like_count, tp.comment_count,
			tp.repost_count, tp.original_post_id, tp.is_repost,
			tp.created_at, tp.updated_at,
			u.full_name, u.avatar_url,
			EXISTS(SELECT 1 FROM timeline_likes tl WHERE tl.post_id = tp.id AND tl.user_id = $2) as is_liked,
			EXISTS(SELECT 1 FROM timeline_posts rp WHERE rp.original_post_id = tp.id AND rp.author_id = $2 AND rp.is_repost = true) as is_reposted,
			orig_u.full_name as original_author_name
		FROM timeline_posts tp
		JOIN users u ON tp.author_id = u.id
		LEFT JOIN timeline_posts orig ON tp.original_post_id = orig.id
		LEFT JOIN users orig_u ON orig.author_id = orig_u.id
		WHERE tp.id = $1`

	p := &models.TimelinePostDetail{}
	err := r.pool.QueryRow(ctx, query, postID, userID).Scan(
		&p.ID, &p.AuthorID, &p.BuildingID, &p.Content, &p.Type, &p.Visibility,
		&p.LocationLat, &p.LocationLng, &p.LikeCount, &p.CommentCount,
		&p.RepostCount, &p.OriginalPostID, &p.IsRepost,
		&p.CreatedAt, &p.UpdatedAt,
		&p.AuthorName, &p.AuthorAvatar,
		&p.IsLiked, &p.IsReposted,
		&p.OriginalAuthorName,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("post not found")
		}
		return nil, err
	}

	media, _ := r.GetMedia(ctx, p.ID)
	p.Media = media

	if p.Type == models.PostTypePoll {
		poll, _ := r.GetPollByPost(ctx, p.ID, userID)
		p.Poll = poll
	}

	return p, nil
}

// Like

func (r *TimelineRepository) ToggleLike(ctx context.Context, postID, userID uuid.UUID) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	var exists bool
	err = tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM timeline_likes WHERE post_id = $1 AND user_id = $2)`,
		postID, userID,
	).Scan(&exists)
	if err != nil {
		return false, err
	}

	if exists {
		_, err = tx.Exec(ctx, `DELETE FROM timeline_likes WHERE post_id = $1 AND user_id = $2`, postID, userID)
		if err != nil {
			return false, err
		}
		_, err = tx.Exec(ctx, `UPDATE timeline_posts SET like_count = like_count - 1 WHERE id = $1`, postID)
	} else {
		_, err = tx.Exec(ctx, `INSERT INTO timeline_likes (post_id, user_id) VALUES ($1, $2)`, postID, userID)
		if err != nil {
			return false, err
		}
		_, err = tx.Exec(ctx, `UPDATE timeline_posts SET like_count = like_count + 1 WHERE id = $1`, postID)
	}
	if err != nil {
		return false, err
	}

	return !exists, tx.Commit(ctx)
}

// Comments

func (r *TimelineRepository) CreateComment(ctx context.Context, comment *models.TimelineComment) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO timeline_comments (post_id, parent_id, author_id, body)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`
	err = tx.QueryRow(ctx, query, comment.PostID, comment.ParentID, comment.AuthorID, comment.Body).
		Scan(&comment.ID, &comment.CreatedAt)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `UPDATE timeline_posts SET comment_count = comment_count + 1 WHERE id = $1`, comment.PostID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *TimelineRepository) GetComments(ctx context.Context, postID uuid.UUID) ([]models.TimelineCommentDetail, error) {
	query := `
		SELECT tc.id, tc.post_id, tc.parent_id, tc.author_id, tc.body, tc.created_at,
			u.full_name, u.avatar_url
		FROM timeline_comments tc
		JOIN users u ON tc.author_id = u.id
		WHERE tc.post_id = $1
		ORDER BY tc.created_at`

	rows, err := r.pool.Query(ctx, query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.TimelineCommentDetail
	for rows.Next() {
		var c models.TimelineCommentDetail
		if err := rows.Scan(
			&c.ID, &c.PostID, &c.ParentID, &c.AuthorID, &c.Body, &c.CreatedAt,
			&c.AuthorName, &c.AuthorAvatar,
		); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, nil
}

// Media

func (r *TimelineRepository) AddMedia(ctx context.Context, media *models.TimelineMedia) error {
	query := `INSERT INTO timeline_media (post_id, url, type) VALUES ($1, $2, $3) RETURNING id, created_at`
	return r.pool.QueryRow(ctx, query, media.PostID, media.URL, media.Type).Scan(&media.ID, &media.CreatedAt)
}

func (r *TimelineRepository) GetMedia(ctx context.Context, postID uuid.UUID) ([]models.TimelineMedia, error) {
	query := `SELECT id, post_id, url, type, created_at FROM timeline_media WHERE post_id = $1`
	rows, err := r.pool.Query(ctx, query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var media []models.TimelineMedia
	for rows.Next() {
		var m models.TimelineMedia
		if err := rows.Scan(&m.ID, &m.PostID, &m.URL, &m.Type, &m.CreatedAt); err != nil {
			return nil, err
		}
		media = append(media, m)
	}
	return media, nil
}

// Polls

func (r *TimelineRepository) CreatePoll(ctx context.Context, poll *models.Poll, options []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `INSERT INTO polls (post_id, question, ends_at) VALUES ($1, $2, $3) RETURNING id, created_at`
	err = tx.QueryRow(ctx, query, poll.PostID, poll.Question, poll.EndsAt).Scan(&poll.ID, &poll.CreatedAt)
	if err != nil {
		return err
	}

	for i, optText := range options {
		_, err = tx.Exec(ctx,
			`INSERT INTO poll_options (poll_id, text, sort_order) VALUES ($1, $2, $3)`,
			poll.ID, optText, i,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *TimelineRepository) GetPollByPost(ctx context.Context, postID, userID uuid.UUID) (*models.PollDetail, error) {
	query := `SELECT id, post_id, question, ends_at, total_votes, created_at FROM polls WHERE post_id = $1`
	pd := &models.PollDetail{}
	err := r.pool.QueryRow(ctx, query, postID).Scan(
		&pd.ID, &pd.PostID, &pd.Question, &pd.EndsAt, &pd.TotalVotes, &pd.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Get options
	optRows, err := r.pool.Query(ctx,
		`SELECT id, poll_id, text, vote_count, sort_order FROM poll_options WHERE poll_id = $1 ORDER BY sort_order`,
		pd.ID)
	if err != nil {
		return nil, err
	}
	defer optRows.Close()

	for optRows.Next() {
		var o models.PollOption
		if err := optRows.Scan(&o.ID, &o.PollID, &o.Text, &o.VoteCount, &o.SortOrder); err != nil {
			return nil, err
		}
		pd.Options = append(pd.Options, o)
	}

	// Get user's vote
	var votedOption uuid.UUID
	err = r.pool.QueryRow(ctx,
		`SELECT option_id FROM poll_votes WHERE poll_id = $1 AND user_id = $2`,
		pd.ID, userID).Scan(&votedOption)
	if err == nil {
		pd.UserVote = &votedOption
	}

	return pd, nil
}

func (r *TimelineRepository) VotePoll(ctx context.Context, pollID, optionID, userID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Check if already voted
	var exists bool
	err = tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM poll_votes WHERE poll_id = $1 AND user_id = $2)`,
		pollID, userID,
	).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("already voted")
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO poll_votes (poll_id, option_id, user_id) VALUES ($1, $2, $3)`,
		pollID, optionID, userID,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `UPDATE poll_options SET vote_count = vote_count + 1 WHERE id = $1`, optionID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `UPDATE polls SET total_votes = total_votes + 1 WHERE id = $1`, pollID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Nearby posts
func (r *TimelineRepository) GetNearby(ctx context.Context, lat, lng, radiusKm float64, userID uuid.UUID, limit int) ([]models.TimelinePostDetail, error) {
	// Using Haversine formula for distance calculation
	query := `
		SELECT tp.id, tp.author_id, tp.building_id, tp.content, tp.type, tp.visibility,
			tp.location_lat, tp.location_lng, tp.like_count, tp.comment_count,
			tp.repost_count, tp.original_post_id, tp.is_repost,
			tp.created_at, tp.updated_at,
			u.full_name, u.avatar_url,
			EXISTS(SELECT 1 FROM timeline_likes tl WHERE tl.post_id = tp.id AND tl.user_id = $4) as is_liked,
			EXISTS(SELECT 1 FROM timeline_posts rp WHERE rp.original_post_id = tp.id AND rp.author_id = $4 AND rp.is_repost = true) as is_reposted,
			orig_u.full_name as original_author_name
		FROM timeline_posts tp
		JOIN users u ON tp.author_id = u.id
		LEFT JOIN timeline_posts orig ON tp.original_post_id = orig.id
		LEFT JOIN users orig_u ON orig.author_id = orig_u.id
		WHERE tp.location_lat IS NOT NULL AND tp.location_lng IS NOT NULL
			AND tp.visibility IN ('neighborhood', 'public')
			AND (
				6371 * acos(
					cos(radians($1)) * cos(radians(tp.location_lat)) *
					cos(radians(tp.location_lng) - radians($2)) +
					sin(radians($1)) * sin(radians(tp.location_lat))
				)
			) <= $3
		ORDER BY tp.created_at DESC
		LIMIT $5`

	rows, err := r.pool.Query(ctx, query, lat, lng, radiusKm, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.TimelinePostDetail
	for rows.Next() {
		var p models.TimelinePostDetail
		if err := rows.Scan(
			&p.ID, &p.AuthorID, &p.BuildingID, &p.Content, &p.Type, &p.Visibility,
			&p.LocationLat, &p.LocationLng, &p.LikeCount, &p.CommentCount,
			&p.RepostCount, &p.OriginalPostID, &p.IsRepost,
			&p.CreatedAt, &p.UpdatedAt,
			&p.AuthorName, &p.AuthorAvatar,
			&p.IsLiked, &p.IsReposted,
			&p.OriginalAuthorName,
		); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, nil
}
