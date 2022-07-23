package responses

import "github.com/NdoleStudio/httpsms/pkg/entities"

// UserResponse is the payload containing entities.User
type UserResponse struct {
	response
	Data entities.User `json:"data"`
}
