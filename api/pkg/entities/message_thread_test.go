package entities

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageThreadReadFieldsHaveBackwardCompatibleDefaults(t *testing.T) {
	threadType := reflect.TypeOf(MessageThread{})

	isRead, ok := threadType.FieldByName("IsRead")
	require.True(t, ok)
	assert.Contains(t, isRead.Tag.Get("gorm"), "not null")
	assert.Contains(t, isRead.Tag.Get("gorm"), "default:true")
	assert.Equal(t, "is_read", isRead.Tag.Get("json"))

	lastReadAt, ok := threadType.FieldByName("LastReadAt")
	require.True(t, ok)
	assert.Contains(t, lastReadAt.Tag.Get("gorm"), "not null")
	assert.Contains(t, lastReadAt.Tag.Get("gorm"), "default:CURRENT_TIMESTAMP")
	assert.Equal(t, "-", lastReadAt.Tag.Get("json"))
}
