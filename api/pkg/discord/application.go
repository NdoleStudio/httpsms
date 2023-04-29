package discord

// CommandCreateRequest is the request for creating a new command
type CommandCreateRequest struct {
	Name        string                       `json:"name"`
	Type        int                          `json:"type"`
	Description string                       `json:"description"`
	Options     []CommandCreateRequestOption `json:"options"`
}

// CommandCreateRequestOption are options for creating a command
type CommandCreateRequestOption struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        int    `json:"type"`
	Required    bool   `json:"required"`
}

// CommandCreateResponse is the response after creating a command
type CommandCreateResponse struct {
	ID                       string                        `json:"id"`
	ApplicationID            string                        `json:"application_id"`
	Version                  string                        `json:"version"`
	DefaultMemberPermissions any                           `json:"default_member_permissions"`
	Type                     int                           `json:"type"`
	Name                     string                        `json:"name"`
	NameLocalizations        any                           `json:"name_localizations"`
	Description              string                        `json:"description"`
	DescriptionLocalizations any                           `json:"description_localizations"`
	GuildID                  string                        `json:"guild_id"`
	Options                  []CommandCreateResponseOption `json:"options"`
	Nsfw                     bool                          `json:"nsfw"`
}

// CommandCreateResponseOption are options after creating a command
type CommandCreateResponseOption struct {
	Type                     int    `json:"type"`
	Name                     string `json:"name"`
	NameLocalizations        any    `json:"name_localizations"`
	Description              string `json:"description"`
	DescriptionLocalizations any    `json:"description_localizations"`
	Required                 bool   `json:"required"`
}
