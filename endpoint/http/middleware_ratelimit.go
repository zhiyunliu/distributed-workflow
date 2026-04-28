package http

import (
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// ─────────────────────────────────────────────────────────────────────────────
// TokenBucket 令牌桶（单客户端）
// ─────────────────────────────────────────────────────────────────────────────

// TokenBucket 基于令牌桶算法的单客户端限流器
type TokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // 每秒补充的令牌数
	lastRefill time.Time
}

// NewTokenBucket 创建令牌桶
func NewTokenBucket(maxTokens, refillRate float64) *TokenBucket {
	return &TokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow 尝试消耗一个令牌，成功返回 true，否则返回 false
func (b *TokenBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens = min(b.maxTokens, b.tokens+elapsed*b.refillRate)
	b.lastRefill = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// lastSeen 返回上次补充时间（用于清理判断）
func (b *TokenBucket) lastSeen() time.Time {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.lastRefill
}

// ─────────────────────────────────────────────────────────────────────────────
// RateLimiter 多客户端限流器（按客户端标识）
// ─────────────────────────────────────────────────────────────────────────────

// RateLimiter 管理多个客户端的令牌桶，定期清理不活跃的桶
type RateLimiter struct {
	mu         sync.RWMutex
	buckets    map[string]*TokenBucket
	maxTokens  float64
	refillRate float64
}

// NewRateLimiter 创建多客户端限流器，并启动后台清理 goroutine
func NewRateLimiter(maxTokens, refillRate float64) *RateLimiter {
	l := &RateLimiter{
		buckets:    make(map[string]*TokenBucket),
		maxTokens:  maxTokens,
		refillRate: refillRate,
	}
	go l.cleanup()
	return l
}

// Allow 检查指定客户端是否允许本次请求
func (l *RateLimiter) Allow(clientID string) bool {
	l.mu.RLock()
	bucket, ok := l.buckets[clientID]
	l.mu.RUnlock()

	if !ok {
		l.mu.Lock()
		// double-check after acquiring write lock
		if bucket, ok = l.buckets[clientID]; !ok {
			bucket = NewTokenBucket(l.maxTokens, l.refillRate)
			l.buckets[clientID] = bucket
		}
		l.mu.Unlock()
	}
	return bucket.Allow()
}

// cleanup 每 10 分钟清理超过 10 分钟未活跃的令牌桶，防止内存泄漏
func (l *RateLimiter) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		threshold := time.Now().Add(-10 * time.Minute)
		l.mu.Lock()
		for id, bucket := range l.buckets {
			if bucket.lastSeen().Before(threshold) {
				delete(l.buckets, id)
			}
		}
		l.mu.Unlock()
		log.Debug().Int("remaining_buckets", l.bucketCount()).Msg("rate limiter cleanup done")
	}
}

func (l *RateLimiter) bucketCount() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.buckets)
}

// ─────────────────────────────────────────────────────────────────────────────
// Gin 中间件
// ─────────────────────────────────────────────────────────────────────────────

// RateLimitMiddleware 返回基于令牌桶的 gin 限流中间件，按客户端 IP 进行限流
// 超出限流时返回 HTTP 429 Too Many Requests
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if !limiter.Allow(clientIP) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "请求过于频繁，请稍后重试",
			})
			return
		}
		c.Next()
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 默认限流器（从环境变量读取配置）
// ─────────────────────────────────────────────────────────────────────────────

// newDefaultRateLimiter 从环境变量创建默认限流器
//
// 环境变量：
//   - RATE_LIMIT_MAX_TOKENS  令牌桶容量（默认 100）
//   - RATE_LIMIT_REFILL_RATE 每秒补充令牌数（默认 20）
func newDefaultRateLimiter() *RateLimiter {
	maxTokens := envFloat("RATE_LIMIT_MAX_TOKENS", 100)
	refillRate := envFloat("RATE_LIMIT_REFILL_RATE", 20)
	log.Info().
		Float64("max_tokens", maxTokens).
		Float64("refill_rate", refillRate).
		Msg("rate limiter initialized")
	return NewRateLimiter(maxTokens, refillRate)
}

// envFloat 从环境变量读取 float64，失败时返回 defaultVal
func envFloat(key string, defaultVal float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			return f
		}
	}
	return defaultVal
}
