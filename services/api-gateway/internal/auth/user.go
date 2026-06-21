package auth

import "fmt"

// User represents a user in the system.
// In a real application, this would likely be fetched from/stored in the auth-service database.
// I'm keeping it simple here for the gateway's JWT generation.
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"` // Password hash - should not be in JWT or responses
	Role     string `json:"role"`
}

// Mock user store for demonstration purposes.
// In a real setup, I'd interact with the auth-service.
var mockUsers = map[string]*User{
	"user": {
		ID:       "2",
		Username: "user",
		// In a real app, this would be a bcrypt hash.
		// For this example, I'm storing the plain password for easy checking.
		// This is NOT secure for production.
		Password: "user123",
		Role:     "user",
	},
	"admin": {
		ID:       "1",
		Username: "admin",
		Password: "admin123",
		Role:     "admin",
	},
}

// FindUserByUsername simulates finding a user by username.
// Again, this would call the auth-service in reality.
func FindUserByUsername(username string) (*User, bool) {
	user, exists := mockUsers[username]
	return user, exists
}

// AddUser simulates adding a new user.
// This is highly simplified and lacks proper validation, hashing, etc.
func AddUser(newUser *User) error {
	if _, exists := mockUsers[newUser.Username]; exists {
		return fmt.Errorf("username '%s' already exists", newUser.Username)
	}
	// In a real app, I'd hash the password here before storing.
	// Also need to generate a unique ID.
	newUser.ID = fmt.Sprintf("%d", len(mockUsers)+100) // Simple ID generation
	mockUsers[newUser.Username] = newUser
	return nil
}
