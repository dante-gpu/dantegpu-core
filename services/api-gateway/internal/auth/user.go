package auth

// User represents the identity carried in a JWT. The authoritative user store
// (PostgreSQL + bcrypt) lives in the auth-service; the gateway never holds
// credentials. This type only describes the claims subject.
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}
