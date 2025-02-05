package unit

import (
	"testing"
	"time"

	"streaming_video_service/internal/member/domain"

	"github.com/stretchr/testify/assert"
)

func TestUserPasswordMatch(t *testing.T) {
	user := domain.Member{
		ID:       1,
		Email:    "user@example.com",
		Password: "pass1234",
	}

	assert.True(t, user.IsPasswordMatch("pass1234") == nil, "should match correct password")
	assert.False(t, user.IsPasswordMatch("wrongpass") == nil, "should not match incorrect password")
}

func TestUserSessionExpiration(t *testing.T) {
	session := domain.MemberSession{
		Token:        "abcd1234",
		MemberID:     "1",
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		ExpiredAt:    time.Now().Add(-1 * time.Minute), // 已經過期
	}

	assert.True(t, session.IsExpired(), "session should be expired")
}
