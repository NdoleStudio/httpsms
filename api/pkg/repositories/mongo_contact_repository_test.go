package repositories

import (
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestMongoContactFilter_ScopesByUserAndSearchesContactFields(t *testing.T) {
	filter := mongoContactFilter(entities.UserID("user-1"), "Alice")

	assert.Equal(t, bson.D{
		{Key: "user_id", Value: "user-1"},
		{Key: "$or", Value: bson.A{
			bson.D{{Key: "name", Value: bson.Regex{Pattern: "Alice", Options: "i"}}},
			bson.D{{Key: "emails", Value: bson.Regex{Pattern: "Alice", Options: "i"}}},
			bson.D{{Key: "phone_numbers", Value: bson.Regex{Pattern: "Alice", Options: "i"}}},
		}},
	}, filter)
}

func TestMongoContactFilter_WithoutSearchOnlyScopesByUser(t *testing.T) {
	filter := mongoContactFilter(entities.UserID("user-1"), "")

	assert.Equal(t, bson.D{{Key: "user_id", Value: "user-1"}}, filter)
}

func TestMongoContactFilter_TreatsSearchAsLiteralText(t *testing.T) {
	filter := mongoContactFilter(entities.UserID("user-1"), "Alice.*")

	expectedExpression := bson.Regex{Pattern: `Alice\.\*`, Options: "i"}
	assert.Equal(t, bson.D{
		{Key: "user_id", Value: "user-1"},
		{Key: "$or", Value: bson.A{
			bson.D{{Key: "name", Value: expectedExpression}},
			bson.D{{Key: "emails", Value: expectedExpression}},
			bson.D{{Key: "phone_numbers", Value: expectedExpression}},
		}},
	}, filter)
}

func TestMongoContactSort_UsesStableRequestedOrdering(t *testing.T) {
	tests := []struct {
		name     string
		params   IndexParams
		expected bson.D
	}{
		{
			name:     "default",
			params:   IndexParams{},
			expected: bson.D{{Key: "updated_at", Value: -1}, {Key: "_id", Value: -1}},
		},
		{
			name:     "name ascending",
			params:   IndexParams{SortBy: "name"},
			expected: bson.D{{Key: "name", Value: 1}, {Key: "_id", Value: 1}},
		},
		{
			name:     "name descending",
			params:   IndexParams{SortBy: "name", SortDescending: true},
			expected: bson.D{{Key: "name", Value: -1}, {Key: "_id", Value: -1}},
		},
		{
			name:     "updated ascending",
			params:   IndexParams{SortBy: "updated_at"},
			expected: bson.D{{Key: "updated_at", Value: 1}, {Key: "_id", Value: 1}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, mongoContactSort(tt.params))
		})
	}
}

func TestMongoContactPhoneNumbersFilter_ScopesByUserAndRequestedNumbers(t *testing.T) {
	filter := mongoContactPhoneNumbersFilter(
		entities.UserID("user-1"),
		[]string{"+18005550199", "+18005550100"},
	)

	assert.Equal(t, bson.D{
		{Key: "user_id", Value: "user-1"},
		{Key: "phone_numbers", Value: bson.D{{Key: "$in", Value: []string{"+18005550199", "+18005550100"}}}},
	}, filter)
}

func TestMongoContactIDFilter_ScopesByUserAndContactID(t *testing.T) {
	contactID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	filter := mongoContactIDFilter(entities.UserID("user-1"), contactID)

	assert.Equal(t, bson.D{
		{Key: "user_id", Value: "user-1"},
		{Key: "_id", Value: contactID.String()},
	}, filter)
}

func TestContactMongoIndexModels_CoverListingAndPhoneLookup(t *testing.T) {
	indexes := contactMongoIndexModels()

	require.Len(t, indexes, 3)
	assert.Equal(t, bson.D{
		{Key: "user_id", Value: 1},
		{Key: "updated_at", Value: -1},
		{Key: "_id", Value: -1},
	}, indexes[0].Keys)
	assert.Equal(t, bson.D{
		{Key: "user_id", Value: 1},
		{Key: "name", Value: 1},
		{Key: "_id", Value: 1},
	}, indexes[1].Keys)
	assert.Equal(t, bson.D{
		{Key: "user_id", Value: 1},
		{Key: "phone_numbers", Value: 1},
		{Key: "updated_at", Value: 1},
		{Key: "_id", Value: 1},
	}, indexes[2].Keys)
}
