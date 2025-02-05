package domain

// RoomType room type for mongo db collection
type RoomType string

const (
	//Invite 1 on 1 collection
	Invite RoomType = "invitations"
	//Group group collection
	Group RoomType = "groups"
)

// Action websocket request action
type Action string

const (
	// CreateRoom websocket action create_room
	CreateRoom Action = "create_room"
	// InvitePrivate websocket action invite_private
	InvitePrivate Action = "invite_private"
	// AcceptInvite websocket action accept_invite
	AcceptInvite Action = "accept_invite"

	// JoinRoom websocket action join_room
	JoinRoom Action = "join_room"
	// ExitRoom websocket action exit_room
	ExitRoom Action = "exit_room"

	// EnterRoom websocket action enter_room
	EnterRoom Action = "enter_room"
	// LeaveRoom websocket action leave_room
	LeaveRoom Action = "leave_room"

	// SendMessage websocket action send_message
	SendMessage Action = "send_message"
	// ReadMessage websocket action read_message
	ReadMessage Action = "read_message"

	// GetUnread websocket action get_unread
	GetUnread Action = "get_unread"

	// GetInvite websocket action get_invite
	GetInvite Action = "get_invite"

	// NotifyMessage websocket action notify_message
	NotifyMessage Action = "notify_message"
)

// WSRequest websocket Request
type WSRequest struct {
	Action    string   `json:"action"`
	RoomType  string   `json:"room_type"`
	RoomName  string   `json:"room_name"`
	Members   []string `json:"members"`
	JoinMode  string   `json:"join_mode"`
	Password  string   `json:"password"`
	RoomID    string   `json:"room_id"`
	// SenderID  string   `json:"sender_id"`
	InviterID string   `json:"inviter_id"`
	InviteeID string   `json:"invitee_id"`
	Content   string   `json:"content"`
	MessageID string   `json:"message_id"`
	IsPrivate bool     `json:"is_private"`
}

// WSResponse websocket Response
type WSResponse struct {
	Action  string                 `json:"action"`
	Success bool                   `json:"success"`
	Payload map[string]interface{} `json:"payload,omitempty"`
	Error   string                 `json:"error,omitempty"`
}
