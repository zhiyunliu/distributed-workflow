package builtin

import (
	"errors"
	"fmt"
	"sync"
	"time"
	"unicode"
)

// PasswordPolicy 密码策略配置
type PasswordPolicy struct {
	MinLength      int  // 最小长度，默认8
	RequireUpper   bool // 需要大写字母
	RequireLower   bool // 需要小写字母
	RequireDigit   bool // 需要数字
	RequireSpecial bool // 需要特殊字符
}

// DefaultPasswordPolicy 默认密码策略
var DefaultPasswordPolicy = PasswordPolicy{
	MinLength:      8,
	RequireUpper:   true,
	RequireLower:   true,
	RequireDigit:   true,
	RequireSpecial: false,
}

// ValidatePassword 校验密码强度
// 返回中文错误信息，nil 表示校验通过
func ValidatePassword(password string, policy PasswordPolicy) error {
	if len(password) < policy.MinLength {
		return fmt.Errorf("密码长度不能少于%d位", policy.MinLength)
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}
	if policy.RequireUpper && !hasUpper {
		return errors.New("密码必须包含大写字母")
	}
	if policy.RequireLower && !hasLower {
		return errors.New("密码必须包含小写字母")
	}
	if policy.RequireDigit && !hasDigit {
		return errors.New("密码必须包含数字")
	}
	if policy.RequireSpecial && !hasSpecial {
		return errors.New("密码必须包含特殊字符")
	}
	return nil
}

// LoginAttemptTracker 登录尝试追踪（内存实现，简单版）
// 生产环境应替换为 Redis 实现以支持分布式场景和持久化
type LoginAttemptTracker struct {
	mu       sync.Mutex
	attempts map[string]*loginRecord // key: username
}

type loginRecord struct {
	count    int
	lastTry  time.Time
	lockedAt *time.Time
}

const (
	// MaxLoginAttempts 最大连续登录失败次数，超出后锁定账号
	MaxLoginAttempts = 5
	// LockoutDuration 账号锁定时长
	LockoutDuration = 15 * time.Minute
)

// GlobalLoginTracker 全局登录尝试追踪器
var GlobalLoginTracker = NewLoginAttemptTracker()

// NewLoginAttemptTracker 创建登录尝试追踪器
func NewLoginAttemptTracker() *LoginAttemptTracker {
	return &LoginAttemptTracker{
		attempts: make(map[string]*loginRecord),
	}
}

// IsLocked 检查账号是否处于锁定期内
func (t *LoginAttemptTracker) IsLocked(username string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	rec, ok := t.attempts[username]
	if !ok || rec.lockedAt == nil {
		return false
	}
	if time.Since(*rec.lockedAt) >= LockoutDuration {
		// 锁定期已过，自动解锁并清除记录
		delete(t.attempts, username)
		return false
	}
	return true
}

// RecordFailedAttempt 记录登录失败，达到 MaxLoginAttempts 则锁定账号
func (t *LoginAttemptTracker) RecordFailedAttempt(username string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	rec, ok := t.attempts[username]
	if !ok {
		rec = &loginRecord{}
		t.attempts[username] = rec
	}
	rec.count++
	rec.lastTry = time.Now()
	if rec.count >= MaxLoginAttempts && rec.lockedAt == nil {
		now := time.Now()
		rec.lockedAt = &now
	}
}

// RecordSuccessAttempt 记录登录成功，清除失败计数
func (t *LoginAttemptTracker) RecordSuccessAttempt(username string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.attempts, username)
}
