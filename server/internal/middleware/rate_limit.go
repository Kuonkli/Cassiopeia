package middleware

import (
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimitMiddleware создает middleware для ограничения запросов
func RateLimitMiddleware(limiter *rate.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Пропускаем health-check
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/api/v1/health" {
			c.Next()
			return
		}

		clientIP := c.ClientIP()

		// Используем IP для ключа кэша или логирования
		// Например, можно вести статистику по IP

		// Проверяем лимит
		if !limiter.Allow() {
			// Логируем блокировку с IP
			log.Printf("Rate limit blocked IP: %s for path: %s",
				clientIP, c.Request.URL.Path)

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate limit exceeded",
				"message": "please try again later",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// IPBasedRateLimitMiddleware - более продвинутая версия с разделением по IP
type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  sync.RWMutex
	r   rate.Limit
	b   int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		r:   r,
		b:   b,
	}
}

func (i *IPRateLimiter) AddIP(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter := rate.NewLimiter(i.r, i.b)
	i.ips[ip] = limiter

	return limiter
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.ips[ip]
	if !exists {
		return i.AddIP(ip)
	}

	return limiter
}

func IPRateLimitMiddleware(ipLimiter *IPRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		limiter := ipLimiter.GetLimiter(clientIP)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate limit exceeded for your IP",
				"message": "please try again in a few seconds",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
