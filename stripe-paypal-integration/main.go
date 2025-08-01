package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/customer"
	"github.com/stripe/stripe-go/v74/paymentintent"
	"github.com/stripe/stripe-go/v74/paymentmethod"
	"github.com/stripe/stripe-go/v74/setupintent"
	"go.uber.org/zap"
)

// PaymentService provides payment processing using Stripe and PayPal
type PaymentService struct {
	logger       *zap.Logger
	stripeKey    string
	paypalConfig PayPalConfig
}

// PayPalConfig holds PayPal configuration
type PayPalConfig struct {
	ClientID     string
	ClientSecret string
	Environment  string // sandbox or live
}

// PaymentRequest represents a payment request
type PaymentRequest struct {
	Amount            int64             `json:"amount"`              // Amount in cents
	Currency          string            `json:"currency"`            // USD, EUR, etc.
	PaymentMethod     string            `json:"payment_method"`      // stripe, paypal
	CustomerID        string            `json:"customer_id"`         // Optional existing customer
	CustomerEmail     string            `json:"customer_email"`      // Required for new customers
	Description       string            `json:"description"`         // Payment description
	Metadata          map[string]string `json:"metadata"`            // Additional metadata
	ReturnURL         string            `json:"return_url"`          // For redirects
	CancelURL         string            `json:"cancel_url"`          // For cancellations
	AutoCapture       bool              `json:"auto_capture"`        // Auto capture payment
	SavePaymentMethod bool              `json:"save_payment_method"` // Save for future use
}

// PaymentResponse represents a payment response
type PaymentResponse struct {
	ID             string                 `json:"id"`
	Status         string                 `json:"status"`
	Amount         int64                  `json:"amount"`
	Currency       string                 `json:"currency"`
	PaymentMethod  string                 `json:"payment_method"`
	CustomerID     string                 `json:"customer_id"`
	ClientSecret   string                 `json:"client_secret,omitempty"`
	RedirectURL    string                 `json:"redirect_url,omitempty"`
	RequiresAction bool                   `json:"requires_action"`
	NextAction     map[string]interface{} `json:"next_action,omitempty"`
	Metadata       map[string]string      `json:"metadata"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// CustomerInfo represents customer information
type CustomerInfo struct {
	ID            string            `json:"id"`
	Email         string            `json:"email"`
	Name          string            `json:"name"`
	Phone         string            `json:"phone"`
	DefaultSource string            `json:"default_source"`
	Metadata      map[string]string `json:"metadata"`
	CreatedAt     time.Time         `json:"created_at"`
}

// PaymentMethodInfo represents payment method information
type PaymentMethodInfo struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Card     map[string]interface{} `json:"card,omitempty"`
	Customer string                 `json:"customer"`
	Metadata map[string]string      `json:"metadata"`
}

// SetupIntentRequest represents a setup intent request
type SetupIntentRequest struct {
	CustomerID        string            `json:"customer_id"`
	PaymentMethodType string            `json:"payment_method_type"` // card, sepa_debit, etc.
	Usage             string            `json:"usage"`               // on_session, off_session
	Description       string            `json:"description"`
	Metadata          map[string]string `json:"metadata"`
}

// SetupIntentResponse represents a setup intent response
type SetupIntentResponse struct {
	ID            string                 `json:"id"`
	Status        string                 `json:"status"`
	ClientSecret  string                 `json:"client_secret"`
	Usage         string                 `json:"usage"`
	NextAction    map[string]interface{} `json:"next_action,omitempty"`
	PaymentMethod string                 `json:"payment_method,omitempty"`
	CustomerID    string                 `json:"customer_id"`
}

// NewPaymentService creates a new payment service
func NewPaymentService(stripeKey string, paypalConfig PayPalConfig, logger *zap.Logger) *PaymentService {
	stripe.Key = stripeKey

	// Set app info for Stripe
	stripe.SetAppInfo(&stripe.AppInfo{
		Name:    "DanteGPU",
		URL:     "https://dantegpu.com",
		Version: "1.0.0",
	})

	return &PaymentService{
		logger:       logger,
		stripeKey:    stripeKey,
		paypalConfig: paypalConfig,
	}
}

// CreateCustomer creates a new customer
func (ps *PaymentService) CreateCustomer(c *gin.Context) {
	var req struct {
		Email       string            `json:"email" binding:"required"`
		Name        string            `json:"name"`
		Phone       string            `json:"phone"`
		Description string            `json:"description"`
		Metadata    map[string]string `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	params := &stripe.CustomerParams{
		Email:       stripe.String(req.Email),
		Name:        stripe.String(req.Name),
		Phone:       stripe.String(req.Phone),
		Description: stripe.String(req.Description),
	}

	if req.Metadata != nil {
		for k, v := range req.Metadata {
			params.AddMetadata(k, v)
		}
	}

	cust, err := customer.New(params)
	if err != nil {
		ps.logger.Error("Failed to create customer", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create customer"})
		return
	}

	customerInfo := CustomerInfo{
		ID:            cust.ID,
		Email:         cust.Email,
		Name:          cust.Name,
		Phone:         cust.Phone,
		DefaultSource: cust.DefaultSource.ID,
		Metadata:      cust.Metadata,
		CreatedAt:     time.Unix(cust.Created, 0),
	}

	c.JSON(http.StatusCreated, customerInfo)
}

// CreatePaymentIntent creates a new payment intent
func (ps *PaymentService) CreatePaymentIntent(c *gin.Context) {
	var req PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate amount
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Amount must be greater than 0"})
		return
	}

	// Default currency
	if req.Currency == "" {
		req.Currency = "usd"
	}

	switch req.PaymentMethod {
	case "stripe", "":
		ps.createStripePaymentIntent(c, req)
	case "paypal":
		ps.createPayPalPayment(c, req)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported payment method"})
	}
}

// createStripePaymentIntent creates a Stripe payment intent
func (ps *PaymentService) createStripePaymentIntent(c *gin.Context, req PaymentRequest) {
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(req.Amount),
		Currency: stripe.String(req.Currency),
	}

	if req.Description != "" {
		params.Description = stripe.String(req.Description)
	}

	if req.CustomerID != "" {
		params.Customer = stripe.String(req.CustomerID)
	}

	if req.AutoCapture {
		params.CaptureMethod = stripe.String("automatic")
	} else {
		params.CaptureMethod = stripe.String("manual")
	}

	if req.SavePaymentMethod {
		params.SetupFutureUsage = stripe.String("off_session")
	}

	// Add metadata
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			params.AddMetadata(k, v)
		}
	}

	// Add automatic payment methods
	params.AutomaticPaymentMethods = &stripe.PaymentIntentAutomaticPaymentMethodsParams{
		Enabled: stripe.Bool(true),
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		ps.logger.Error("Failed to create payment intent", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment intent"})
		return
	}

	response := PaymentResponse{
		ID:             pi.ID,
		Status:         string(pi.Status),
		Amount:         pi.Amount,
		Currency:       string(pi.Currency),
		PaymentMethod:  "stripe",
		CustomerID:     pi.Customer.ID,
		ClientSecret:   pi.ClientSecret,
		RequiresAction: pi.Status == stripe.PaymentIntentStatusRequiresAction,
		Metadata:       pi.Metadata,
		CreatedAt:      time.Unix(pi.Created, 0),
		UpdatedAt:      time.Now(),
	}

	if pi.NextAction != nil {
		nextActionData := make(map[string]interface{})
		if pi.NextAction.RedirectToURL != nil {
			nextActionData["redirect_to_url"] = map[string]string{
				"url":        pi.NextAction.RedirectToURL.URL,
				"return_url": pi.NextAction.RedirectToURL.ReturnURL,
			}
		}
		response.NextAction = nextActionData
	}

	c.JSON(http.StatusCreated, response)
}

// createPayPalPayment creates a PayPal payment (simplified implementation)
func (ps *PaymentService) createPayPalPayment(c *gin.Context, req PaymentRequest) {
	// This is a simplified PayPal implementation
	// In a real implementation, you would use PayPal SDK

	paypalOrderID := fmt.Sprintf("PAYPAL_%d", time.Now().Unix())

	response := PaymentResponse{
		ID:             paypalOrderID,
		Status:         "requires_action",
		Amount:         req.Amount,
		Currency:       req.Currency,
		PaymentMethod:  "paypal",
		RequiresAction: true,
		RedirectURL:    fmt.Sprintf("https://www.sandbox.paypal.com/checkoutnow?token=%s", paypalOrderID),
		Metadata:       req.Metadata,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	c.JSON(http.StatusCreated, response)
}

// CreateSetupIntent creates a setup intent for saving payment methods
func (ps *PaymentService) CreateSetupIntent(c *gin.Context) {
	var req SetupIntentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	params := &stripe.SetupIntentParams{
		Customer: stripe.String(req.CustomerID),
		Usage:    stripe.String(req.Usage),
	}

	if req.Description != "" {
		params.Description = stripe.String(req.Description)
	}

	if req.PaymentMethodType != "" {
		params.PaymentMethodTypes = stripe.StringSlice([]string{req.PaymentMethodType})
	}

	if req.Metadata != nil {
		for k, v := range req.Metadata {
			params.AddMetadata(k, v)
		}
	}

	si, err := setupintent.New(params)
	if err != nil {
		ps.logger.Error("Failed to create setup intent", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create setup intent"})
		return
	}

	response := SetupIntentResponse{
		ID:           si.ID,
		Status:       string(si.Status),
		ClientSecret: si.ClientSecret,
		Usage:        string(si.Usage),
		CustomerID:   si.Customer.ID,
	}

	if si.PaymentMethod != nil {
		response.PaymentMethod = si.PaymentMethod.ID
	}

	if si.NextAction != nil {
		nextActionData := make(map[string]interface{})
		if si.NextAction.RedirectToURL != nil {
			nextActionData["redirect_to_url"] = map[string]string{
				"url":        si.NextAction.RedirectToURL.URL,
				"return_url": si.NextAction.RedirectToURL.ReturnURL,
			}
		}
		response.NextAction = nextActionData
	}

	c.JSON(http.StatusCreated, response)
}

// GetPaymentMethods retrieves payment methods for a customer
func (ps *PaymentService) GetPaymentMethods(c *gin.Context) {
	customerID := c.Param("customer_id")
	if customerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Customer ID is required"})
		return
	}

	params := &stripe.PaymentMethodListParams{
		Customer: stripe.String(customerID),
		Type:     stripe.String("card"),
	}

	i := paymentmethod.List(params)
	var paymentMethods []PaymentMethodInfo

	for i.Next() {
		pm := i.PaymentMethod()

		pmInfo := PaymentMethodInfo{
			ID:       pm.ID,
			Type:     string(pm.Type),
			Customer: pm.Customer.ID,
			Metadata: pm.Metadata,
		}

		if pm.Card != nil {
			pmInfo.Card = map[string]interface{}{
				"brand":     pm.Card.Brand,
				"last4":     pm.Card.Last4,
				"exp_month": pm.Card.ExpMonth,
				"exp_year":  pm.Card.ExpYear,
				"funding":   pm.Card.Funding,
			}
		}

		paymentMethods = append(paymentMethods, pmInfo)
	}

	if err := i.Err(); err != nil {
		ps.logger.Error("Failed to list payment methods", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list payment methods"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payment_methods": paymentMethods,
		"has_more":        i.Meta().HasMore,
	})
}

// ConfirmPaymentIntent confirms a payment intent
func (ps *PaymentService) ConfirmPaymentIntent(c *gin.Context) {
	paymentIntentID := c.Param("payment_intent_id")
	if paymentIntentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment Intent ID is required"})
		return
	}

	var req struct {
		PaymentMethodID string `json:"payment_method_id"`
		ReturnURL       string `json:"return_url"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	params := &stripe.PaymentIntentConfirmParams{
		PaymentMethod: stripe.String(req.PaymentMethodID),
		ReturnURL:     stripe.String(req.ReturnURL),
	}

	pi, err := paymentintent.Confirm(paymentIntentID, params)
	if err != nil {
		ps.logger.Error("Failed to confirm payment intent", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm payment intent"})
		return
	}

	response := PaymentResponse{
		ID:             pi.ID,
		Status:         string(pi.Status),
		Amount:         pi.Amount,
		Currency:       string(pi.Currency),
		PaymentMethod:  "stripe",
		RequiresAction: pi.Status == stripe.PaymentIntentStatusRequiresAction,
		Metadata:       pi.Metadata,
		UpdatedAt:      time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// CapturePaymentIntent captures a payment intent
func (ps *PaymentService) CapturePaymentIntent(c *gin.Context) {
	paymentIntentID := c.Param("payment_intent_id")
	if paymentIntentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment Intent ID is required"})
		return
	}

	var req struct {
		AmountToCapture int64 `json:"amount_to_capture,omitempty"`
	}

	c.ShouldBindJSON(&req)

	params := &stripe.PaymentIntentCaptureParams{}
	if req.AmountToCapture > 0 {
		params.AmountToCapture = stripe.Int64(req.AmountToCapture)
	}

	pi, err := paymentintent.Capture(paymentIntentID, params)
	if err != nil {
		ps.logger.Error("Failed to capture payment intent", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to capture payment intent"})
		return
	}

	response := PaymentResponse{
		ID:        pi.ID,
		Status:    string(pi.Status),
		Amount:    pi.Amount,
		Currency:  string(pi.Currency),
		Metadata:  pi.Metadata,
		UpdatedAt: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// GetPaymentIntent retrieves a payment intent
func (ps *PaymentService) GetPaymentIntent(c *gin.Context) {
	paymentIntentID := c.Param("payment_intent_id")
	if paymentIntentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment Intent ID is required"})
		return
	}

	pi, err := paymentintent.Get(paymentIntentID, nil)
	if err != nil {
		ps.logger.Error("Failed to get payment intent", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment intent not found"})
		return
	}

	response := PaymentResponse{
		ID:             pi.ID,
		Status:         string(pi.Status),
		Amount:         pi.Amount,
		Currency:       string(pi.Currency),
		PaymentMethod:  "stripe",
		CustomerID:     pi.Customer.ID,
		ClientSecret:   pi.ClientSecret,
		RequiresAction: pi.Status == stripe.PaymentIntentStatusRequiresAction,
		Metadata:       pi.Metadata,
		CreatedAt:      time.Unix(pi.Created, 0),
		UpdatedAt:      time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// Health check endpoint
func (ps *PaymentService) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "stripe-paypal-integration",
		"stripe":  "connected",
		"paypal":  "configured",
	})
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	stripeKey := getEnv("STRIPE_SECRET_KEY", "sk_test_...")

	paypalConfig := PayPalConfig{
		ClientID:     getEnv("PAYPAL_CLIENT_ID", ""),
		ClientSecret: getEnv("PAYPAL_CLIENT_SECRET", ""),
		Environment:  getEnv("PAYPAL_ENVIRONMENT", "sandbox"),
	}

	paymentService := NewPaymentService(stripeKey, paypalConfig, logger)

	// Setup Gin router
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check
	r.GET("/health", paymentService.HealthCheck)

	// Customer endpoints
	r.POST("/customers", paymentService.CreateCustomer)
	r.GET("/customers/:customer_id/payment-methods", paymentService.GetPaymentMethods)

	// Payment endpoints
	r.POST("/payment-intents", paymentService.CreatePaymentIntent)
	r.GET("/payment-intents/:payment_intent_id", paymentService.GetPaymentIntent)
	r.POST("/payment-intents/:payment_intent_id/confirm", paymentService.ConfirmPaymentIntent)
	r.POST("/payment-intents/:payment_intent_id/capture", paymentService.CapturePaymentIntent)

	// Setup intent endpoints
	r.POST("/setup-intents", paymentService.CreateSetupIntent)

	port := getEnv("PORT", "8096")
	logger.Info("Starting Stripe/PayPal integration service", zap.String("port", port))

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
