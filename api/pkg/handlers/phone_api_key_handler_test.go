package handlers

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/responses"
	"github.com/jaswdr/faker/v2"
	"github.com/stretchr/testify/assert"
)

func TestPhoneAPIKeyHandler_store(t *testing.T) {
	// Arrange
	fake := faker.New()
	payload := requests.PhoneAPIKeyStoreRequest{
		Name: fake.RandomStringWithLength(20),
	}
	response := new(responses.PhoneAPIKeyResponse)

	// Act
	err := testClient().
		Post().
		Path("/v1/phone-api-keys").
		BodyJSON(payload).
		ToJSON(response).
		Fetch(context.Background())

	// Assert
	assert.Nil(t, err)
	assert.NotEmpty(t, response.Data.ID)
	assert.True(t, strings.HasPrefix(response.Data.APIKey, "pk_"))
	assert.Equal(t, payload.Name, response.Data.Name)
	assert.True(t, len(response.Data.PhoneNumbers) == 0)
	assert.True(t, len(response.Data.PhoneIDs) == 0)

	// Teardown
	_ = testClient().
		Delete().
		Path("/v1/phone-api-keys/" + response.Data.ID.String()).
		Fetch(context.Background())
}

func TestPhoneAPIKeyHandler_delete(t *testing.T) {
	// Arrange
	fake := faker.New()
	payload := requests.PhoneAPIKeyStoreRequest{
		Name: fake.RandomStringWithLength(20),
	}

	// Act
	response := new(responses.PhoneAPIKeyResponse)
	_ = testClient().
		Post().
		Path("/v1/phone-api-keys").
		BodyJSON(payload).
		ToJSON(response).
		Fetch(context.Background())

	err := testClient().
		Delete().
		Path("/v1/phone-api-keys/" + response.Data.ID.String()).
		Fetch(context.Background())

	// Assert
	assert.Nil(t, err)

	keys := new(responses.PhoneAPIKeysResponse)
	_ = testClient().
		Path("/v1/phone-api-keys").
		ToJSON(response).
		Fetch(context.Background())

	assert.Equal(t, -1, slices.IndexFunc(keys.Data, func(key *entities.PhoneAPIKey) bool {
		return key.ID == response.Data.ID
	}))
}

func TestPhoneAPIKeyHandler_index(t *testing.T) {
	// Arrange
	fake := faker.New()
	createResponse := new(responses.PhoneAPIKeyResponse)
	response := new(responses.PhoneAPIKeysResponse)
	payload := requests.PhoneAPIKeyStoreRequest{
		Name: fake.RandomStringWithLength(20),
	}

	// Act
	_ = testClient().
		Post().
		Path("/v1/phone-api-keys").
		BodyJSON(payload).
		ToJSON(createResponse).
		Fetch(context.Background())

	err := testClient().
		Path("/v1/phone-api-keys").
		ToJSON(response).
		Fetch(context.Background())

	// Assert
	assert.Nil(t, err)
	assert.NotEmpty(t, response.Data)
	assert.NotEqual(t, -1, slices.IndexFunc(response.Data, func(key *entities.PhoneAPIKey) bool {
		return key.ID == createResponse.Data.ID
	}))

	// Teardown
	_ = testClient().
		Delete().
		Path("/v1/phone-api-keys/" + createResponse.Data.ID.String()).
		Fetch(context.Background())
}
