package repository

import (
	"context"
	"fmt"
	"time"

	"apartment-backend/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MessagingRepository struct {
	pool *pgxpool.Pool
}

func NewMessagingRepository(pool *pgxpool.Pool) *MessagingRepository {
	return &MessagingRepository{pool: pool}
}

// GetOrCreateDirectConversation finds an existing 1:1 conversation between two users
// or creates a new one.
func (r *MessagingRepository) GetOrCreateDirectConversation(ctx context.Context, userA, userB uuid.UUID) (*models.Conversation, error) {
	// Look for existing 1:1 conversation
	query := `
		SELECT c.id, c.is_group, c.name, c.building_id, c.created_at, c.updated_at
		FROM conversations c
		WHERE c.is_group = false
			AND EXISTS(SELECT 1 FROM conversation_participants WHERE conversation_id = c.id AND user_id = $1)
			AND EXISTS(SELECT 1 FROM conversation_participants WHERE conversation_id = c.id AND user_id = $2)
			AND (SELECT COUNT(*) FROM conversation_participants WHERE conversation_id = c.id) = 2
		LIMIT 1`

	var conv models.Conversation
	err := r.pool.QueryRow(ctx, query, userA, userB).Scan(
		&conv.ID, &conv.IsGroup, &conv.Name, &conv.BuildingID, &conv.CreatedAt, &conv.UpdatedAt,
	)
	if err == nil {
		return &conv, nil
	}
	if err != pgx.ErrNoRows {
		return nil, err
	}

	// Create new conversation
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx,
		`INSERT INTO conversations (is_group) VALUES (false) RETURNING id, is_group, name, building_id, created_at, updated_at`,
	).Scan(&conv.ID, &conv.IsGroup, &conv.Name, &conv.BuildingID, &conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Add both participants
	_, err = tx.Exec(ctx,
		`INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)`,
		conv.ID, userA, userB,
	)
	if err != nil {
		return nil, err
	}

	return &conv, tx.Commit(ctx)
}

// GetConversations returns conversations for a user with last message preview and unread count.
func (r *MessagingRepository) GetConversations(ctx context.Context, userID uuid.UUID, page, limit int) ([]models.ConversationListItem, int64, error) {
	offset := (page - 1) * limit

	// Count total conversations
	var total int64
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM conversation_participants WHERE user_id = $1`, userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT
			c.id, c.is_group, c.name, c.updated_at,
			-- Other user info for 1:1 conversations
			other_u.id as other_user_id,
			other_u.full_name as other_user_name,
			other_u.avatar_url as other_user_avatar,
			-- Last message
			last_msg.content as last_message,
			last_msg.created_at as last_message_at,
			last_msg_user.full_name as last_message_by,
			-- Unread count
			COALESCE((
				SELECT COUNT(*) FROM messages m
				WHERE m.conversation_id = c.id
					AND m.created_at > cp.last_read_at
					AND m.sender_id != $1
					AND m.deleted_at IS NULL
			), 0) as unread_count
		FROM conversations c
		JOIN conversation_participants cp ON cp.conversation_id = c.id AND cp.user_id = $1
		-- Other user for 1:1
		LEFT JOIN LATERAL (
			SELECT cp2.user_id FROM conversation_participants cp2
			WHERE cp2.conversation_id = c.id AND cp2.user_id != $1
			LIMIT 1
		) other_cp ON NOT c.is_group
		LEFT JOIN users other_u ON other_u.id = other_cp.user_id
		-- Last message
		LEFT JOIN LATERAL (
			SELECT m.content, m.created_at, m.sender_id FROM messages m
			WHERE m.conversation_id = c.id AND m.deleted_at IS NULL
			ORDER BY m.created_at DESC
			LIMIT 1
		) last_msg ON true
		LEFT JOIN users last_msg_user ON last_msg_user.id = last_msg.sender_id
		ORDER BY COALESCE(last_msg.created_at, c.updated_at) DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []models.ConversationListItem
	for rows.Next() {
		var item models.ConversationListItem
		if err := rows.Scan(
			&item.ID, &item.IsGroup, &item.Name, &item.UpdatedAt,
			&item.OtherUserID, &item.OtherUserName, &item.OtherUserAvatar,
			&item.LastMessage, &item.LastMessageAt, &item.LastMessageBy,
			&item.UnreadCount,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, nil
}

// GetMessages returns messages for a conversation, paginated with cursor-based approach.
func (r *MessagingRepository) GetMessages(ctx context.Context, conversationID uuid.UUID, before *time.Time, limit int) ([]models.MessageResponse, error) {
	args := []interface{}{conversationID}
	whereClause := "WHERE m.conversation_id = $1 AND m.deleted_at IS NULL"

	if before != nil {
		whereClause += fmt.Sprintf(" AND m.created_at < $%d", len(args)+1)
		args = append(args, *before)
	}

	args = append(args, limit)

	query := fmt.Sprintf(`
		SELECT m.id, m.conversation_id, m.sender_id, u.full_name, u.avatar_url,
			m.content, m.message_type, m.created_at
		FROM messages m
		JOIN users u ON m.sender_id = u.id
		%s
		ORDER BY m.created_at DESC
		LIMIT $%d`, whereClause, len(args))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.MessageResponse
	for rows.Next() {
		var msg models.MessageResponse
		if err := rows.Scan(
			&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.SenderName, &msg.SenderAvatar,
			&msg.Content, &msg.MessageType, &msg.CreatedAt,
		); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

// SendMessage inserts a new message and updates the conversation's updated_at.
func (r *MessagingRepository) SendMessage(ctx context.Context, conversationID, senderID uuid.UUID, content string, msgType models.MessageType) (*models.MessageResponse, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if msgType == "" {
		msgType = models.MessageTypeText
	}

	var msg models.MessageResponse
	err = tx.QueryRow(ctx,
		`INSERT INTO messages (conversation_id, sender_id, content, message_type)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, conversation_id, sender_id, content, message_type, created_at`,
		conversationID, senderID, content, msgType,
	).Scan(&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.Content, &msg.MessageType, &msg.CreatedAt)
	if err != nil {
		return nil, err
	}

	// Get sender info
	err = tx.QueryRow(ctx,
		`SELECT full_name, avatar_url FROM users WHERE id = $1`, senderID,
	).Scan(&msg.SenderName, &msg.SenderAvatar)
	if err != nil {
		return nil, err
	}

	// Update conversation timestamp
	_, err = tx.Exec(ctx,
		`UPDATE conversations SET updated_at = NOW() WHERE id = $1`, conversationID,
	)
	if err != nil {
		return nil, err
	}

	// Update sender's last_read_at
	_, err = tx.Exec(ctx,
		`UPDATE conversation_participants SET last_read_at = NOW()
		 WHERE conversation_id = $1 AND user_id = $2`,
		conversationID, senderID,
	)
	if err != nil {
		return nil, err
	}

	return &msg, tx.Commit(ctx)
}

// MarkAsRead updates the last_read_at for a participant.
func (r *MessagingRepository) MarkAsRead(ctx context.Context, conversationID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE conversation_participants SET last_read_at = NOW()
		 WHERE conversation_id = $1 AND user_id = $2`,
		conversationID, userID,
	)
	return err
}

// IsParticipant checks if a user is a participant of a conversation.
func (r *MessagingRepository) IsParticipant(ctx context.Context, conversationID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM conversation_participants WHERE conversation_id = $1 AND user_id = $2)`,
		conversationID, userID,
	).Scan(&exists)
	return exists, err
}

// GetParticipantIDs returns all user IDs in a conversation.
func (r *MessagingRepository) GetParticipantIDs(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT user_id FROM conversation_participants WHERE conversation_id = $1`,
		conversationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
