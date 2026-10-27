package services

import (
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/stretchr/testify/assert"
)

func TestPhoneUpdateArchiveNewOutgoingMessageThreads(t *testing.T) {
	enabled := true
	disabled := false
	tests := []struct {
		name    string
		initial bool
		setting *bool
		want    bool
	}{
		{name: "enable", setting: &enabled, want: true},
		{name: "disable", initial: true, setting: &disabled, want: false},
		{name: "omitted preserves enabled", initial: true, want: true},
		{name: "omitted defaults off", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			phone := &entities.Phone{ArchiveNewOutgoingMessageThreads: test.initial}
			params := PhoneUpsertParams{ArchiveNewOutgoingMessageThreads: test.setting}

			updated := (&PhoneService{}).update(phone, &params)

			assert.Equal(t, test.want, updated.ArchiveNewOutgoingMessageThreads)
		})
	}
}
