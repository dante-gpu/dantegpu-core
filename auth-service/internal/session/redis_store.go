package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Session represents a user session
type Session struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Email        string    `json:"email"`
	Roles        []string  `json:"roles"`
	RefreshToken string    `json:"refresh_token"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	CreatedAt    time.Time `json:"created_at"`
	LastActivity time.Time `json:"last_activity"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// RedisStore implements session storage using Redis
type RedisStore struct {
	client *redis.Client
	prefix string
}

// NewRedisStore creates a new Redis session store
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{
		client: client,
		prefix: "session:",
	}
}

// Create creates a new session
func (s *RedisStore) Create(ctx context.Context, session *Session) error {
	key := s.sessionKey(session.ID)
	
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}
	
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("session already expired")
	}
	
	// Store session
	if err := s.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store session: %w", err)
	}
	
	// Add to user's session set
	userSessionsKey := s.userSessionsKey(session.UserID)
	if err := s.client.SAdd(ctx, userSessionsKey, session.ID).Err(); err != nil {
		return fmt.Errorf("failed to add session to user set: %w", err)
	}
	
	// Set expiration on user sessions set
	s.client.Expire(ctx, userSessionsKey, ttl)
	
	return nil
}

// Get retrieves a session by ID
func (s *RedisStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	key := s.sessionKey(sessionID)
	
	data, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}
	
	return &session, nil
}

// Update updates a session
func (s *RedisStore) Update(ctx context.Context, session *Session) error {
	// Check if session exists
	exists, err := s.Exists(ctx, session.ID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("session not found")
	}
	
	// Update session
	return s.Create(ctx, session)
}

// Delete deletes a session
func (s *RedisStore) Delete(ctx context.Context, sessionID string) error {
	// Get session to find user ID
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	
	key := s.sessionKey(sessionID)
	
	// Delete session
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	
	// Remove from user's session set
	userSessionsKey := s.userSessionsKey(session.UserID)
	if err := s.client.SRem(ctx, userSessionsKey, sessionID).Err(); err != nil {
		return fmt.Errorf("failed to remove session from user set: %w", err)
	}
	
	return nil
}

// DeleteAllUserSessions deletes all sessions for a user
func (s *RedisStore) DeleteAllUserSessions(ctx context.Context, userID string) error {
	userSessionsKey := s.userSessionsKey(userID)
	
	// Get all session IDs for user
	sessionIDs, err := s.client.SMembers(ctx, userSessionsKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get user sessions: %w", err)
	}
	
	// Delete each session
	for _, sessionID := range sessionIDs {
		key := s.sessionKey(sessionID)
		s.client.Del(ctx, key)
	}
	
	// Delete user sessions set
	if err := s.client.Del(ctx, userSessionsKey).Err(); err != nil {
		return fmt.Errorf("failed to delete user sessions set: %w", err)
	}
	
	return nil
}

// GetUserSessions gets all sessions for a user
func (s *RedisStore) GetUserSessions(ctx context.Context, userID string) ([]*Session, error) {
	userSessionsKey := s.userSessionsKey(userID)
	
	// Get all session IDs for user
	sessionIDs, err := s.client.SMembers(ctx, userSessionsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}
	
	var sessions []*Session
	for _, sessionID := range sessionIDs {
		session, err := s.Get(ctx, sessionID)
		if err != nil {
			// Session might have expired, skip it
			continue
		}
		sessions = append(sessions, session)
	}
	
	return sessions, nil
}

// Exists checks if a session exists
func (s *RedisStore) Exists(ctx context.Context, sessionID string) (bool, error) {
	key := s.sessionKey(sessionID)
	
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check session existence: %w", err)
	}
	
	return exists > 0, nil
}

// UpdateActivity updates the last activity time of a session
func (s *RedisStore) UpdateActivity(ctx context.Context, sessionID string) error {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	
	session.LastActivity = time.Now()
	
	return s.Update(ctx, session)
}

// ExtendExpiration extends the expiration time of a session
func (s *RedisStore) ExtendExpiration(ctx context.Context, sessionID string, duration time.Duration) error {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	
	session.ExpiresAt = time.Now().Add(duration)
	
	return s.Update(ctx, session)
}

// CountUserSessions counts active sessions for a user
func (s *RedisStore) CountUserSessions(ctx context.Context, userID string) (int, error) {
	userSessionsKey := s.userSessionsKey(userID)
	
	count, err := s.client.SCard(ctx, userSessionsKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to count user sessions: %w", err)
	}
	
	return int(count), nil
}

// CleanupExpiredSessions removes expired sessions (called by background job)
func (s *RedisStore) CleanupExpiredSessions(ctx context.Context) (int, error) {
	// Redis automatically removes expired keys, so this is mainly for cleanup
	// of the user sessions sets
	
	// In a production system, you might want to scan for user session sets
	// and clean up any that reference non-existent sessions
	
	return 0, nil
}

// Helper functions

func (s *RedisStore) sessionKey(sessionID string) string {
	return fmt.Sprintf("%s%s", s.prefix, sessionID)
}

func (s *RedisStore) userSessionsKey(userID string) string {
	return fmt.Sprintf("%suser:%s:sessions", s.prefix, userID)
}

// RateLimiter implements rate limiting using Redis
type RateLimiter struct {
	client *redis.Client
	prefix string
}

// NewRateLimiter creates a new Redis rate limiter
func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{
		client: client,
		prefix: "ratelimit:",
	}
}

// Allow checks if a request is allowed based on rate limit
func (r *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	redisKey := r.prefix + key
	
	// Increment counter
	count, err := r.client.Incr(ctx, redisKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to increment rate limit counter: %w", err)
	}
	
	// Set expiration on first request
	if count == 1 {
		r.client.Expire(ctx, redisKey, window)
	}
	
	return count <= int64(limit), nil
}

// GetRemaining gets the remaining requests allowed
func (r *RateLimiter) GetRemaining(ctx context.Context, key string, limit int) (int, error) {
	redisKey := r.prefix + key
	
	count, err := r.client.Get(ctx, redisKey).Int()
	if err == redis.Nil {
		return limit, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get rate limit count: %w", err)
	}
	
	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}
	
	return remaining, nil
}

// Reset resets the rate limit for a key
func (r *RateLimiter) Reset(ctx context.Context, key string) error {
	redisKey := r.prefix + key
	
	if err := r.client.Del(ctx, redisKey).Err(); err != nil {
		return fmt.Errorf("failed to reset rate limit: %w", err)
	}
	
	return nil
}

// TokenBlacklist implements JWT token blacklisting using Redis
type TokenBlacklist struct {
	client *redis.Client
	prefix string
}

// NewTokenBlacklist creates a new token blacklist
func NewTokenBlacklist(client *redis.Client) *TokenBlacklist {
	return &TokenBlacklist{
		client: client,
		prefix: "blacklist:",
	}
}

// Add adds a token to the blacklist
func (t *TokenBlacklist) Add(ctx context.Context, tokenID string, expiresAt time.Time) error {
	key := t.prefix + tokenID
	
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		// Token already expired, no need to blacklist
		return nil
	}
	
	if err := t.client.Set(ctx, key, "1", ttl).Err(); err != nil {
		return fmt.Errorf("failed to blacklist token: %w", err)
	}
	
	return nil
}

// IsBlacklisted checks if a token is blacklisted
func (t *TokenBlacklist) IsBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	key := t.prefix + tokenID
	
	exists, err := t.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check token blacklist: %w", err)
	}
	
	return exists > 0, nil
}

// Remove removes a token from the blacklist
func (t *TokenBlacklist) Remove(ctx context.Context, tokenID string) error {
	key := t.prefix + tokenID
	
	if err := t.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to remove token from blacklist: %w", err)
	}
	
	return nil
}

