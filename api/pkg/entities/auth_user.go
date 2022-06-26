package entities

// AuthUser is the user gotten from an auth request
type AuthUser struct {
	ID    UserID `json:"id"`
	Email string `json:"email"`
}

// IsNoop checks if a user is empty
func (user AuthUser) IsNoop() bool {
	return user.ID == "" || user.Email == ""
}
