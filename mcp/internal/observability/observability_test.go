package observability

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasPartialAxiomConfig(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		dataset string
		want    bool
	}{
		{name: "neither configured"},
		{name: "both configured", token: "token", dataset: "events"},
		{name: "token only", token: "token", want: true},
		{name: "dataset only", dataset: "events", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(axiomTokenEnv, tt.token)
			t.Setenv(axiomEventsDatasetEnv, tt.dataset)

			assert.Equal(t, tt.want, hasPartialAxiomConfig())
		})
	}
}
