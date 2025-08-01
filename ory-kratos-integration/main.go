package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	ory "github.com/ory/kratos-client-go"
	"go.uber.org/zap"
)

// OryKratosService provides authentication and authorization using Ory Kratos
type OryKratosService struct {
	kratosClient *ory.APIClient
	redisClient  *redis.Client
	logger       *zap.Logger
	ctx          context.Context
}

// KratosConfig holds Ory Kratos configuration
type KratosConfig struct {
	PublicURL string
	AdminURL  string
	RedisURL  string
}

// SessionInfo represents user session information
type SessionInfo struct {
	ID              string       `json:"id"`
	Active          bool         `json:"active"`
	ExpiresAt       string       `json:"expires_at"`
	AuthenticatedAt string       `json:"authenticated_at"`
	Identity        IdentityInfo `json:"identity"`
	AAL             string       `json:"authenticator_assurance_level"`
	AMR             []string     `json:"authentication_methods_reference"`
	Devices         []DeviceInfo `json:"devices"`
}

// IdentityInfo represents user identity information
type IdentityInfo struct {
	ID                  string                 `json:"id"`
	SchemaID            string                 `json:"schema_id"`
	SchemaURL           string                 `json:"schema_url"`
	State               string                 `json:"state"`
	StateChangedAt      string                 `json:"state_changed_at"`
	Traits              map[string]interface{} `json:"traits"`
	VerifiableAddresses []VerifiableAddress    `json:"verifiable_addresses"`
	RecoveryAddresses   []RecoveryAddress      `json:"recovery_addresses"`
	MetadataPublic      map[string]interface{} `json:"metadata_public"`
	MetadataAdmin       map[string]interface{} `json:"metadata_admin"`
	CreatedAt           string                 `json:"created_at"`
	UpdatedAt           string                 `json:"updated_at"`
}

// VerifiableAddress represents a verifiable address
type VerifiableAddress struct {
	ID         string `json:"id"`
	Value      string `json:"value"`
	Verified   bool   `json:"verified"`
	Via        string `json:"via"`
	Status     string `json:"status"`
	VerifiedAt string `json:"verified_at"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// RecoveryAddress represents a recovery address
type RecoveryAddress struct {
	ID        string `json:"id"`
	Value     string `json:"value"`
	Via       string `json:"via"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// DeviceInfo represents device information
type DeviceInfo struct {
	ID        string `json:"id"`
	IPAddress string `json:"ip_address"`
	UserAgent string `json:"user_agent"`
	Location  string `json:"location"`
}

// FlowInfo represents authentication flow information
type FlowInfo struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	ExpiresAt  string                 `json:"expires_at"`
	IssuedAt   string                 `json:"issued_at"`
	RequestURL string                 `json:"request_url"`
	Active     string                 `json:"active"`
	UI         map[string]interface{} `json:"ui"`
	State      string                 `json:"state"`
}

// NewOryKratosService creates a new Ory Kratos service
func NewOryKratosService(config KratosConfig, logger *zap.Logger) *OryKratosService {
	// Configure Kratos client
	kratosConfig := ory.NewConfiguration()
	kratosConfig.Servers = []ory.ServerConfiguration{
		{
			URL: config.PublicURL,
		},
	}

	// Configure Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: config.RedisURL,
	})

	return &OryKratosService{
		kratosClient: ory.NewAPIClient(kratosConfig),
		redisClient:  redisClient,
		logger:       logger,
		ctx:          context.Background(),
	}
}

// SessionMiddleware validates user sessions using Ory Kratos
func (oks *OryKratosService) SessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, err := oks.validateSession(c.Request)
		if err != nil {
			oks.logger.Warn("Session validation failed", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":     "unauthorized",
				"message":   "Invalid or expired session",
				"login_url": fmt.Sprintf("%s/self-service/login/browser", oks.kratosClient.GetConfig().Servers[0].URL),
			})
			c.Abort()
			return
		}

		if !session.Active {
			oks.logger.Warn("Inactive session detected")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":     "session_inactive",
				"message":   "Session is not active",
				"login_url": fmt.Sprintf("%s/self-service/login/browser", oks.kratosClient.GetConfig().Servers[0].URL),
			})
			c.Abort()
			return
		}

		// Cache session information
		oks.cacheSession(session)

		// Add session to context
		c.Set("session", session)
		c.Set("user_id", session.Identity.ID)
		c.Set("user_email", oks.extractEmail(session.Identity.Traits))
		c.Set("user_role", oks.extractRole(session.Identity.Traits))

		c.Next()
	}
}

// OptionalSessionMiddleware validates sessions but doesn't require authentication
func (oks *OryKratosService) OptionalSessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, err := oks.validateSession(c.Request)
		if err == nil && session.Active {
			// Cache session information
			oks.cacheSession(session)

			// Add session to context
			c.Set("session", session)
			c.Set("user_id", session.Identity.ID)
			c.Set("user_email", oks.extractEmail(session.Identity.Traits))
			c.Set("user_role", oks.extractRole(session.Identity.Traits))
		}

		c.Next()
	}
}

// validateSession validates the session using Ory Kratos
func (oks *OryKratosService) validateSession(r *http.Request) (*SessionInfo, error) {
	// Get session cookie
	cookie, err := r.Cookie("ory_kratos_session")
	if err != nil {
		return nil, fmt.Errorf("no session cookie found: %w", err)
	}

	// Check cache first
	if cachedSession := oks.getCachedSession(cookie.Value); cachedSession != nil {
		return cachedSession, nil
	}

	// Validate with Kratos
	resp, _, err := oks.kratosClient.FrontendApi.ToSession(oks.ctx).Cookie(cookie.String()).Execute()
	if err != nil {
		return nil, fmt.Errorf("session validation failed: %w", err)
	}

	// Convert to our SessionInfo structure
	session := &SessionInfo{
		ID:              resp.GetId(),
		Active:          resp.GetActive(),
		ExpiresAt:       resp.GetExpiresAt().String(),
		AuthenticatedAt: resp.GetAuthenticatedAt().String(),
		AAL:             string(resp.GetAuthenticatorAssuranceLevel()),
		AMR:             oks.convertAMR(resp.GetAuthenticationMethods()),
	}

	// Convert identity
	identity := resp.GetIdentity()

	// Type assert traits to map[string]interface{}
	var traits map[string]interface{}
	if t := identity.GetTraits(); t != nil {
		if traitsMap, ok := t.(map[string]interface{}); ok {
			traits = traitsMap
		} else {
			traits = make(map[string]interface{})
		}
	} else {
		traits = make(map[string]interface{})
	}

	session.Identity = IdentityInfo{
		ID:        identity.GetId(),
		SchemaID:  identity.GetSchemaId(),
		SchemaURL: identity.GetSchemaUrl(),
		State:     string(identity.GetState()),
		Traits:    traits,
	}

	// Convert verifiable addresses
	for _, addr := range identity.GetVerifiableAddresses() {
		session.Identity.VerifiableAddresses = append(session.Identity.VerifiableAddresses, VerifiableAddress{
			ID:       addr.GetId(),
			Value:    addr.GetValue(),
			Verified: addr.GetVerified(),
			Via:      string(addr.GetVia()),
			Status:   string(addr.GetStatus()),
		})
	}

	// Convert recovery addresses
	for _, addr := range identity.GetRecoveryAddresses() {
		session.Identity.RecoveryAddresses = append(session.Identity.RecoveryAddresses, RecoveryAddress{
			ID:    addr.GetId(),
			Value: addr.GetValue(),
			Via:   string(addr.GetVia()),
		})
	}

	return session, nil
}

// cacheSession caches session information in Redis
func (oks *OryKratosService) cacheSession(session *SessionInfo) {
	sessionData, err := json.Marshal(session)
	if err != nil {
		oks.logger.Error("Failed to marshal session for caching", zap.Error(err))
		return
	}

	cacheKey := fmt.Sprintf("kratos:session:%s", session.ID)
	err = oks.redisClient.Set(oks.ctx, cacheKey, sessionData, 0).Err()
	if err != nil {
		oks.logger.Error("Failed to cache session", zap.Error(err))
	}
}

// getCachedSession retrieves cached session information
func (oks *OryKratosService) getCachedSession(sessionToken string) *SessionInfo {
	// Extract session ID from token (simplified)
	sessionID := oks.extractSessionID(sessionToken)
	if sessionID == "" {
		return nil
	}

	cacheKey := fmt.Sprintf("kratos:session:%s", sessionID)
	sessionData, err := oks.redisClient.Get(oks.ctx, cacheKey).Result()
	if err != nil {
		return nil
	}

	var session SessionInfo
	err = json.Unmarshal([]byte(sessionData), &session)
	if err != nil {
		oks.logger.Error("Failed to unmarshal cached session", zap.Error(err))
		return nil
	}

	return &session
}

// extractSessionID extracts session ID from session token (simplified implementation)
func (oks *OryKratosService) extractSessionID(token string) string {
	// This is a simplified implementation
	// In reality, you'd need to decode the session token properly
	parts := strings.Split(token, "|")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// extractEmail extracts email from user traits
func (oks *OryKratosService) extractEmail(traits map[string]interface{}) string {
	if email, ok := traits["email"].(string); ok {
		return email
	}
	return ""
}

// extractRole extracts role from user traits
func (oks *OryKratosService) extractRole(traits map[string]interface{}) string {
	if role, ok := traits["role"].(string); ok {
		return role
	}
	return "user" // default role
}

// convertAMR converts authentication methods reference
func (oks *OryKratosService) convertAMR(methods []ory.SessionAuthenticationMethod) []string {
	var amr []string
	for _, method := range methods {
		amr = append(amr, string(method.GetMethod()))
	}
	return amr
}

// GetUserInfo returns user information from session
func (oks *OryKratosService) GetUserInfo(c *gin.Context) {
	session, exists := c.Get("session")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no session found"})
		return
	}

	sessionInfo := session.(*SessionInfo)
	c.JSON(http.StatusOK, gin.H{
		"user": sessionInfo.Identity,
		"session": map[string]interface{}{
			"id":               sessionInfo.ID,
			"active":           sessionInfo.Active,
			"expires_at":       sessionInfo.ExpiresAt,
			"authenticated_at": sessionInfo.AuthenticatedAt,
			"aal":              sessionInfo.AAL,
			"amr":              sessionInfo.AMR,
		},
	})
}

// CreateLoginFlow creates a new login flow
func (oks *OryKratosService) CreateLoginFlow(c *gin.Context) {
	returnTo := c.Query("return_to")

	req := oks.kratosClient.FrontendApi.CreateBrowserLoginFlow(oks.ctx)
	if returnTo != "" {
		req = req.ReturnTo(returnTo)
	}

	flow, _, err := req.Execute()
	if err != nil {
		oks.logger.Error("Failed to create login flow", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create login flow"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"flow_id":    flow.GetId(),
		"ui":         flow.GetUi(),
		"expires_at": flow.GetExpiresAt(),
	})
}

// CreateRegistrationFlow creates a new registration flow
func (oks *OryKratosService) CreateRegistrationFlow(c *gin.Context) {
	returnTo := c.Query("return_to")

	req := oks.kratosClient.FrontendApi.CreateBrowserRegistrationFlow(oks.ctx)
	if returnTo != "" {
		req = req.ReturnTo(returnTo)
	}

	flow, _, err := req.Execute()
	if err != nil {
		oks.logger.Error("Failed to create registration flow", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create registration flow"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"flow_id":    flow.GetId(),
		"ui":         flow.GetUi(),
		"expires_at": flow.GetExpiresAt(),
	})
}

// CreateLogoutFlow creates a new logout flow
func (oks *OryKratosService) CreateLogoutFlow(c *gin.Context) {
	returnTo := c.Query("return_to")

	req := oks.kratosClient.FrontendApi.CreateBrowserLogoutFlow(oks.ctx)
	if returnTo != "" {
		req = req.ReturnTo(returnTo)
	}

	flow, _, err := req.Execute()
	if err != nil {
		oks.logger.Error("Failed to create logout flow", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create logout flow"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logout_url":   flow.GetLogoutUrl(),
		"logout_token": flow.GetLogoutToken(),
	})
}

// CreateSettingsFlow creates a new settings flow
func (oks *OryKratosService) CreateSettingsFlow(c *gin.Context) {
	returnTo := c.Query("return_to")

	req := oks.kratosClient.FrontendApi.CreateBrowserSettingsFlow(oks.ctx)
	if returnTo != "" {
		req = req.ReturnTo(returnTo)
	}

	flow, _, err := req.Execute()
	if err != nil {
		oks.logger.Error("Failed to create settings flow", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create settings flow"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"flow_id":    flow.GetId(),
		"ui":         flow.GetUi(),
		"expires_at": flow.GetExpiresAt(),
	})
}

// Health check endpoint
func (oks *OryKratosService) HealthCheck(c *gin.Context) {
	// Check Kratos connectivity
	_, _, err := oks.kratosClient.MetadataApi.GetVersion(oks.ctx).Execute()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"kratos": "disconnected",
			"error":  err.Error(),
		})
		return
	}

	// Check Redis connectivity
	_, err = oks.redisClient.Ping(oks.ctx).Result()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"redis":  "disconnected",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"kratos": "connected",
		"redis":  "connected",
	})
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	config := KratosConfig{
		PublicURL: getEnv("KRATOS_PUBLIC_URL", "http://localhost:4433"),
		AdminURL:  getEnv("KRATOS_ADMIN_URL", "http://localhost:4434"),
		RedisURL:  getEnv("REDIS_URL", "localhost:6379"),
	}

	oryService := NewOryKratosService(config, logger)

	// Setup Gin router
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, Cookie")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Public endpoints
	r.GET("/health", oryService.HealthCheck)
	r.POST("/auth/login/flow", oryService.CreateLoginFlow)
	r.POST("/auth/registration/flow", oryService.CreateRegistrationFlow)
	r.POST("/auth/logout/flow", oryService.CreateLogoutFlow)

	// Protected endpoints
	protected := r.Group("/api")
	protected.Use(oryService.SessionMiddleware())
	{
		protected.GET("/user", oryService.GetUserInfo)
		protected.POST("/settings/flow", oryService.CreateSettingsFlow)
	}

	// Optional auth endpoints
	optional := r.Group("/public")
	optional.Use(oryService.OptionalSessionMiddleware())
	{
		optional.GET("/status", func(c *gin.Context) {
			userID, exists := c.Get("user_id")
			if exists {
				c.JSON(http.StatusOK, gin.H{
					"authenticated": true,
					"user_id":       userID,
				})
			} else {
				c.JSON(http.StatusOK, gin.H{
					"authenticated": false,
				})
			}
		})
	}

	port := getEnv("PORT", "8095")
	logger.Info("Starting Ory Kratos integration service", zap.String("port", port))

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
