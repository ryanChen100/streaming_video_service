package domain

// ChatRoomType definition chat room type
type ChatRoomType string

const (
	//ChatRoomTypePrivate definition chat room 1 on 1
	ChatRoomTypePrivate ChatRoomType = "private" // 1對1
	//ChatRoomTypeGroup definition chat room group
	ChatRoomTypeGroup ChatRoomType = "group" // 群組
)

// JoinMode 決定加入群組條件
type JoinMode string

const (
	//JoinModeOpen allow all
	JoinModeOpen JoinMode = "open" // 任何人都能加入
	//JoinModePassword need password
	JoinModePassword JoinMode = "password" // 需輸入密碼
	//JoinModeApprove need approve
	JoinModeApprove JoinMode = "approve" // 需群主或管理員同意
)

// ChatRoom definition chat room
type ChatRoom struct {
	ID        string       `bson:"_id,omitempty"`
	RoomType  ChatRoomType `bson:"room_type"`
	Name      string       `bson:"name,omitempty"`
	Members   []string     `bson:"members,omitempty"`
	Admins    []string     `bson:"admins,omitempty"`
	JoinMode  JoinMode     `bson:"join_mode,omitempty"`
	Password  string       `bson:"password,omitempty"`
	IsPrivate bool         `bson:"is_private"`
	IsInvite  bool         `bson:"is_invite"`
	CreatedAt int64        `bson:"created_at,omitempty"` // ID          string         `bson:"_id,omitempty"` // MongoDB 的 _id
	// RoomType    ChatRoomType   `bson:"room_type"`
	// Name        string         `bson:"name,omitempty"`       // 群組名稱 (一對一可忽略)
	// Members     []string       `bson:"members,omitempty"`    // 成員 ID
	// Admins      []string       `bson:"admins,omitempty"`     // 管理員 ID (群組)
	// Invitation  ChatInvitation `bson:"invitation,omitempty"` // 加入聊天室邀請
	// JoinMode    JoinMode       `bson:"join_mode,omitempty"`
	// Password    string         `bson:"password,omitempty"` // 如果 JoinMode=Password
	// CreatedAt   int64          `bson:"created_at"`
	// IsPrivate   bool           `bson:"is_private"` // 群組是否隱密 (公開/隱藏)
	// Description string         `bson:"description,omitempty"`
	// ... 其他資訊 (禁言名單等可擴充)
}

// ChatInvitation invitation chat
type ChatInvitation struct {
	ID        string           `bson:"_id,omitempty"`
	InviterID string           `bson:"inviter_id"`
	InviteeID string           `bson:"invitee_id"`
	RoomID    string           `bson:"room_id,omitempty"` // 還沒同意前可以先不生成
	Status    InvitationStatus `bson:"status"`            // pending, accepted, rejected
	CreatedAt int64            `bson:"created_at"`
}

// InvitationStatus definition invitation status
type InvitationStatus string

const (
	//InvitationPending invitation allow
	InvitationPending InvitationStatus = "pending"
	//InvitationAccepted invitation pending
	InvitationAccepted InvitationStatus = "accepted"
	// InvitationRejected invitation rejected
	InvitationRejected InvitationStatus = "rejected"
)

// PrivateChatInvitation - 1對1邀請
type PrivateChatInvitation struct {
	ID        string           `bson:"_id,omitempty"`
	InviterID string           `bson:"inviter_id"`
	InviteeID string           `bson:"invitee_id"`
	Status    InvitationStatus `bson:"status"`
	CreatedAt int64            `bson:"created_at"`
}
