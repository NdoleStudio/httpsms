package responses

import "github.com/NdoleStudio/httpsms/pkg/entities"

// DiscordResponse is the payload containing entities.Discord
type DiscordResponse struct {
	response
	Data entities.Discord `json:"data"`
}

// DiscordsResponse is the payload containing []entities.Discord
type DiscordsResponse struct {
	response
	Data []entities.Discord `json:"data"`
}
