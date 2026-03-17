package models

import (
	"time"

	"github.com/google/uuid"
)

type MessageType string

const (
	MessageTypeText   MessageType = "text"
	MessageTypeImage  MessageType = "image"
	MessageTypeSystem MessageType = "system"
)

type Conversation struct {
	ID         uuid.UUID  `json:"id"`
	IsGroup    bool       `json:"is_group"`
	Name       *string    `json:"name,omitempty"`
	BuildingID *uuid.UUID `json:"building_id,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type ConversationParticipant struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	UserID         uuid.UUID `json:"user_id"`
	JoinedAt       time.Time `json:"joined_at"`
	LastReadAt     time.Time `json:"last_read_at"`
}

type Message struct {
	ID             uuid.UUID   `json:"id"`
	ConversationID uuid.UUID   `json:"conversation_id"`
	SenderID       uuid.UUID   `json:"sender_id"`
	Content        string      `json:"content"`
	MessageType    MessageType `json:"message_type"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	DeletedAt      *time.Time  `json:"deleted_at,omitempty"`
}

// Request DTOs

type CreateConversationRequest struct {
	UserID string `json:"user_id" validate:"required"`
}

type SendMessageRequest struct {
	Content     string      `json:"content" validate:"required"`
	MessageType MessageType `json:"message_type,omitempty"`
}

// Response DTOs

type ConversationListItem struct {
	ID              uuid.UUID  `json:"id"`
	IsGroup         bool       `json:"is_group"`
	Name            *string    `json:"name,omitempty"`
	OtherUserID     *uuid.UUID `json:"other_user_id,omitempty"`
	OtherUserName   *string    `json:"other_user_name,omitempty"`
	OtherUserAvatar *string    `json:"other_user_avatar,omitempty"`
	LastMessage     *string    `json:"last_message,omitempty"`
	LastMessageAt   *time.Time `json:"last_message_at,omitempty"`
	LastMessageBy   *string    `json:"last_message_by,omitempty"`
	UnreadCount     int        `json:"unread_count"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type MessageResponse struct {
	ID             uuid.UUID   `json:"id"`
	ConversationID uuid.UUID   `json:"conversation_id"`
	SenderID       uuid.UUID   `json:"sender_id"`
	SenderName     string      `json:"sender_name"`
	SenderAvatar   *string     `json:"sender_avatar,omitempty"`
	Content        string      `json:"content"`
	MessageType    MessageType `json:"message_type"`
	CreatedAt      time.Time   `json:"created_at"`
}
