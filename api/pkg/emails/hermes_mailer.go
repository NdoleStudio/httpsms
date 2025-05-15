package emails

import (
	"fmt"
	"strconv"
	"time"

	"github.com/matcornic/hermes"
)

// HermesGeneratorConfig contains details for the generator
type HermesGeneratorConfig struct {
	AppURL     string
	AppName    string
	AppLogoURL string
}

// Generator creates hermes.Hermes from HermesGeneratorConfig
func (config *HermesGeneratorConfig) Generator() hermes.Hermes {
	return hermes.Hermes{
		Theme: newHermesTheme(),
		Product: hermes.Product{
			// Appears in header & footer of e-mails
			Name: fmt.Sprintf("The %s Team", config.AppName),
			Link: config.AppURL,
			// Optional product logo
			Copyright: fmt.Sprintf("© %s %s. All rights reserved.", strconv.Itoa(time.Now().Year()), config.AppName),
			Logo:      config.AppLogoURL,
		},
	}
}
