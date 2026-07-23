package requests

import (
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/stretchr/testify/assert"
)

func TestMessageThreadIndex_ToGetParams_WithContactsTrue(t *testing.T) {
	input := (&MessageThreadIndex{Owner: "+18005550199", Contacts: " true "}).Sanitize()
	params := input.ToGetParams(entities.UserID("user-id"))

	assert.Equal(t, "true", input.Contacts)
	assert.True(t, params.WithContacts)
}

func TestMessageThreadIndex_ToGetParams_WithContactsFalse(t *testing.T) {
	input := (&MessageThreadIndex{Owner: "+18005550199", Contacts: "0"}).Sanitize()
	params := input.ToGetParams(entities.UserID("user-id"))

	assert.Equal(t, "false", input.Contacts)
	assert.False(t, params.WithContacts)
}

func TestMessageThreadIndex_ToGetParams_WithContactsDefaultsFalse(t *testing.T) {
	input := (&MessageThreadIndex{Owner: "+18005550199"}).Sanitize()
	params := input.ToGetParams(entities.UserID("user-id"))

	assert.Equal(t, "false", input.Contacts)
	assert.False(t, params.WithContacts)
}

func TestMessageThreadIndex_ToGetParams_WithContactsInvalidNormalizesFalse(t *testing.T) {
	input := (&MessageThreadIndex{Owner: "+18005550199", Contacts: "definitely"}).Sanitize()
	params := input.ToGetParams(entities.UserID("user-id"))

	assert.Equal(t, "false", input.Contacts)
	assert.False(t, params.WithContacts)
}
