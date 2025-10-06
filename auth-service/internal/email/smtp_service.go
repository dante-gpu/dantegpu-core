package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"os"
	"time"
)

// SMTPConfig holds SMTP configuration
type SMTPConfig struct {
	Host         string
	Port         int
	Username     string
	Password     string
	FromEmail    string
	FromName     string
	UseTLS       bool
	FrontendURL  string
}

// SMTPService implements email sending via SMTP
type SMTPService struct {
	config    *SMTPConfig
	templates *template.Template
}

// NewSMTPService creates a new SMTP email service
func NewSMTPService(config *SMTPConfig) (*SMTPService, error) {
	// Load email templates
	templates, err := loadEmailTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to load email templates: %w", err)
	}

	return &SMTPService{
		config:    config,
		templates: templates,
	}, nil
}

// SendVerificationEmail sends an email verification link
func (s *SMTPService) SendVerificationEmail(ctx context.Context, email, token, firstName string) error {
	verificationURL := fmt.Sprintf("%s/verify-email?token=%s", s.config.FrontendURL, token)

	data := map[string]interface{}{
		"FirstName":       firstName,
		"VerificationURL": verificationURL,
		"Year":            time.Now().Year(),
	}

	subject := "Verify Your DanteGPU Account"
	body, err := s.renderTemplate("verification", data)
	if err != nil {
		return err
	}

	return s.sendEmail(email, subject, body)
}

// SendWelcomeEmail sends a welcome email after successful verification
func (s *SMTPService) SendWelcomeEmail(ctx context.Context, email, firstName string) error {
	data := map[string]interface{}{
		"FirstName":   firstName,
		"DashboardURL": fmt.Sprintf("%s/dashboard", s.config.FrontendURL),
		"Year":        time.Now().Year(),
	}

	subject := "Welcome to DanteGPU!"
	body, err := s.renderTemplate("welcome", data)
	if err != nil {
		return err
	}

	return s.sendEmail(email, subject, body)
}

// SendPasswordResetEmail sends a password reset link
func (s *SMTPService) SendPasswordResetEmail(ctx context.Context, email, token, firstName string) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.config.FrontendURL, token)

	data := map[string]interface{}{
		"FirstName": firstName,
		"ResetURL":  resetURL,
		"ExpiresIn": "1 hour",
		"Year":      time.Now().Year(),
	}

	subject := "Reset Your DanteGPU Password"
	body, err := s.renderTemplate("password_reset", data)
	if err != nil {
		return err
	}

	return s.sendEmail(email, subject, body)
}

// SendPasswordChangedEmail sends a notification when password is changed
func (s *SMTPService) SendPasswordChangedEmail(ctx context.Context, email, firstName string) error {
	data := map[string]interface{}{
		"FirstName":  firstName,
		"SupportURL": fmt.Sprintf("%s/support", s.config.FrontendURL),
		"Year":       time.Now().Year(),
	}

	subject := "Your DanteGPU Password Has Been Changed"
	body, err := s.renderTemplate("password_changed", data)
	if err != nil {
		return err
	}

	return s.sendEmail(email, subject, body)
}

// Send2FACodeEmail sends a 2FA verification code
func (s *SMTPService) Send2FACodeEmail(ctx context.Context, email, code, firstName string) error {
	data := map[string]interface{}{
		"FirstName": firstName,
		"Code":      code,
		"ExpiresIn": "10 minutes",
		"Year":      time.Now().Year(),
	}

	subject := "Your DanteGPU 2FA Code"
	body, err := s.renderTemplate("2fa_code", data)
	if err != nil {
		return err
	}

	return s.sendEmail(email, subject, body)
}

// SendLoginAlertEmail sends an alert for new login
func (s *SMTPService) SendLoginAlertEmail(ctx context.Context, email, firstName, ipAddress, location string) error {
	data := map[string]interface{}{
		"FirstName":  firstName,
		"IPAddress":  ipAddress,
		"Location":   location,
		"Time":       time.Now().Format("January 2, 2006 at 3:04 PM MST"),
		"SupportURL": fmt.Sprintf("%s/support", s.config.FrontendURL),
		"Year":       time.Now().Year(),
	}

	subject := "New Login to Your DanteGPU Account"
	body, err := s.renderTemplate("login_alert", data)
	if err != nil {
		return err
	}

	return s.sendEmail(email, subject, body)
}

// sendEmail sends an email via SMTP
func (s *SMTPService) sendEmail(to, subject, body string) error {
	from := fmt.Sprintf("%s <%s>", s.config.FromName, s.config.FromEmail)
	
	// Build email message
	msg := s.buildEmailMessage(from, to, subject, body)

	// SMTP authentication
	auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)

	// Server address
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// Send email
	if s.config.UseTLS {
		return s.sendEmailTLS(addr, auth, s.config.FromEmail, []string{to}, msg)
	}

	return smtp.SendMail(addr, auth, s.config.FromEmail, []string{to}, msg)
}

// sendEmailTLS sends email with TLS
func (s *SMTPService) sendEmailTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	// TLS config
	tlsConfig := &tls.Config{
		ServerName: s.config.Host,
	}

	// Connect to server
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return err
	}
	defer client.Quit()

	// Authenticate
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}

	// Set sender
	if err := client.Mail(from); err != nil {
		return err
	}

	// Set recipients
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}

	// Send message
	w, err := client.Data()
	if err != nil {
		return err
	}

	_, err = w.Write(msg)
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	return client.Quit()
}

// buildEmailMessage builds the email message with headers
func (s *SMTPService) buildEmailMessage(from, to, subject, body string) []byte {
	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"
	headers["Date"] = time.Now().Format(time.RFC1123Z)

	var msg bytes.Buffer
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(body)

	return msg.Bytes()
}

// renderTemplate renders an email template
func (s *SMTPService) renderTemplate(name string, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := s.templates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// loadEmailTemplates loads all email templates
func loadEmailTemplates() (*template.Template, error) {
	// In production, load from files
	// For now, we'll define templates inline
	
	templates := template.New("")

	// Verification email template
	_, err := templates.New("verification").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Verify Your Email</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h1 style="color: #4F46E5;">Welcome to DanteGPU!</h1>
        <p>Hi {{.FirstName}},</p>
        <p>Thank you for registering with DanteGPU. Please verify your email address by clicking the button below:</p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.VerificationURL}}" style="background-color: #4F46E5; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Verify Email</a>
        </div>
        <p>Or copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #4F46E5;">{{.VerificationURL}}</p>
        <p>This link will expire in 24 hours.</p>
        <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
        <p style="font-size: 12px; color: #666;">© {{.Year}} DanteGPU. All rights reserved.</p>
    </div>
</body>
</html>
`)
	if err != nil {
		return nil, err
	}

	// Welcome email template
	_, err = templates.New("welcome").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Welcome to DanteGPU</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h1 style="color: #4F46E5;">Welcome to DanteGPU!</h1>
        <p>Hi {{.FirstName}},</p>
        <p>Your email has been verified successfully. You're all set to start using DanteGPU!</p>
        <p>Here's what you can do next:</p>
        <ul>
            <li>Browse available GPUs</li>
            <li>Submit your first job</li>
            <li>Add funds to your wallet</li>
            <li>Explore our documentation</li>
        </ul>
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.DashboardURL}}" style="background-color: #4F46E5; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Go to Dashboard</a>
        </div>
        <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
        <p style="font-size: 12px; color: #666;">© {{.Year}} DanteGPU. All rights reserved.</p>
    </div>
</body>
</html>
`)
	if err != nil {
		return nil, err
	}

	// Password reset template
	_, err = templates.New("password_reset").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Reset Your Password</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h1 style="color: #4F46E5;">Reset Your Password</h1>
        <p>Hi {{.FirstName}},</p>
        <p>We received a request to reset your password. Click the button below to create a new password:</p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.ResetURL}}" style="background-color: #4F46E5; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Reset Password</a>
        </div>
        <p>Or copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #4F46E5;">{{.ResetURL}}</p>
        <p>This link will expire in {{.ExpiresIn}}.</p>
        <p>If you didn't request this, please ignore this email.</p>
        <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
        <p style="font-size: 12px; color: #666;">© {{.Year}} DanteGPU. All rights reserved.</p>
    </div>
</body>
</html>
`)
	if err != nil {
		return nil, err
	}

	// Password changed template
	_, err = templates.New("password_changed").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Password Changed</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h1 style="color: #4F46E5;">Password Changed</h1>
        <p>Hi {{.FirstName}},</p>
        <p>Your password has been changed successfully.</p>
        <p>If you didn't make this change, please contact our support team immediately:</p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.SupportURL}}" style="background-color: #DC2626; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Contact Support</a>
        </div>
        <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
        <p style="font-size: 12px; color: #666;">© {{.Year}} DanteGPU. All rights reserved.</p>
    </div>
</body>
</html>
`)
	if err != nil {
		return nil, err
	}

	// 2FA code template
	_, err = templates.New("2fa_code").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Your 2FA Code</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h1 style="color: #4F46E5;">Your 2FA Code</h1>
        <p>Hi {{.FirstName}},</p>
        <p>Your two-factor authentication code is:</p>
        <div style="text-align: center; margin: 30px 0;">
            <div style="background-color: #F3F4F6; padding: 20px; border-radius: 5px; font-size: 32px; font-weight: bold; letter-spacing: 5px;">{{.Code}}</div>
        </div>
        <p>This code will expire in {{.ExpiresIn}}.</p>
        <p>If you didn't request this code, please ignore this email.</p>
        <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
        <p style="font-size: 12px; color: #666;">© {{.Year}} DanteGPU. All rights reserved.</p>
    </div>
</body>
</html>
`)
	if err != nil {
		return nil, err
	}

	// Login alert template
	_, err = templates.New("login_alert").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>New Login Alert</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h1 style="color: #4F46E5;">New Login Detected</h1>
        <p>Hi {{.FirstName}},</p>
        <p>We detected a new login to your DanteGPU account:</p>
        <ul>
            <li><strong>Time:</strong> {{.Time}}</li>
            <li><strong>IP Address:</strong> {{.IPAddress}}</li>
            <li><strong>Location:</strong> {{.Location}}</li>
        </ul>
        <p>If this was you, you can safely ignore this email.</p>
        <p>If you don't recognize this login, please secure your account immediately:</p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.SupportURL}}" style="background-color: #DC2626; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Secure My Account</a>
        </div>
        <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
        <p style="font-size: 12px; color: #666;">© {{.Year}} DanteGPU. All rights reserved.</p>
    </div>
</body>
</html>
`)
	if err != nil {
		return nil, err
	}

	return templates, nil
}

// LoadFromEnv loads SMTP configuration from environment variables
func LoadFromEnv() *SMTPConfig {
	port := 587
	if p := os.Getenv("SMTP_PORT"); p != "" {
		fmt.Sscanf(p, "%d", &port)
	}

	return &SMTPConfig{
		Host:        os.Getenv("SMTP_HOST"),
		Port:        port,
		Username:    os.Getenv("SMTP_USER"),
		Password:    os.Getenv("SMTP_PASSWORD"),
		FromEmail:   os.Getenv("EMAIL_FROM"),
		FromName:    os.Getenv("EMAIL_FROM_NAME"),
		UseTLS:      os.Getenv("SMTP_SECURE") == "true",
		FrontendURL: os.Getenv("FRONTEND_URL"),
	}
}

