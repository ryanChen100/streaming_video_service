package domain

import (
	"streaming_video_service/pkg/encrypt"
	"time"
)

// MemberStatus 用來表示使用者狀態
type MemberStatus int

// 状态: 0=offline, 1=online, 2=ban ,3=delete
const (
	// MemberStatusOffLine 用來表示使用者狀態為啟用
	MemberStatusOffLine MemberStatus = iota
	// MemberStatusOnLine 用來表示使用者狀態為停用
	MemberStatusOnLine
	// MemberStatusBan 用來表示使用者狀態為封鎖
	MemberStatusBan
	// MemberStatusDelete 用來表示使用者狀態為刪除
	MemberStatusDelete
)

// Member 用來表示使用者
type Member struct {
	ID       int64
	MemberID string
	Email    string
	Password string
	Status   MemberStatus
	// ... 其他欄位(如 Nickname, CreatedAt 等)
}

// MemberSession 用來表示使用者的 Session
type MemberSession struct {
	Token        string    `json:"Token"`
	MemberID     string    `json:"MemberID"`
	CreatedAt    time.Time `json:"CreatedAt"`
	LastActivity time.Time `json:"LastActivity"`
	ExpiredAt    time.Time `json:"ExpiredAt"`
	// 其他屬性: 是否強制登出, 是否已斷線重連 等
}

// IsPasswordMatch 密碼驗證 內還可以有一些方法
func (m *Member) IsPasswordMatch(inputPwd string) error {
	err := encrypt.CheckPassword(m.Password, inputPwd)
	return err
}

// IsExpired 檢查 Session 是否已過期
func (s *MemberSession) IsExpired() bool {
	return time.Now().After(s.ExpiredAt)
}

// MemberQuery join conditions are used to query members
type MemberQuery struct {
	ID       *int64  `db:"id"`
	MemberID *string `db:"member_id"`
	Email    *string `db:"email"`
	// Password *string `db:"password"`
}
