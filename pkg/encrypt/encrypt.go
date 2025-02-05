package encrypt

import (
	"errors"
	"fmt"
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

// 定義密碼加密的強度，bcrypt.DefaultCost = 10
const bcryptCost = bcrypt.DefaultCost

// 定義錯誤信息
var (
	ErrWeakPassword     = errors.New("password does not meet strength requirements")
	ErrPasswordMismatch = errors.New("password does not match")
)

// ValidatePasswordStrength 驗證密碼強度
// 可以根據需求調整規則，例如最小長度、是否包含數字或特殊字符
func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	// 至少包含一個大寫字母
	if matched, _ := regexp.MatchString(`[A-Z]`, password); !matched {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}

	// 至少包含一個數字
	if matched, _ := regexp.MatchString(`[0-9]`, password); !matched {
		return fmt.Errorf("password must contain at least one digit")
	}

	// 至少包含一個特殊字符
	if matched, _ := regexp.MatchString(`[!@#\$%\^&\*]`, password); !matched {
		return fmt.Errorf("password must contain at least one special character (!@#$%^&*)")
	}

	return nil
}

// HashPassword 將密碼進行加密
func HashPassword(password string) (string, error) {
	// 檢查密碼強度
	if err := ValidatePasswordStrength(password); err != nil {
		return "", fmt.Errorf("weak password: %w", err)
	}

	// 使用 bcrypt 加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hashedPassword), nil
}

// CheckPassword 驗證密碼是否匹配
func CheckPassword(hashedPassword, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return ErrPasswordMismatch
	}
	return nil
}
