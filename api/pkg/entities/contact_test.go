package entities

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContactProperties_ValueScanRoundTrip(t *testing.T) {
	cases := []ContactProperties{
		nil,
		{},
		{"company": "Acme", "role": "CTO"},
	}

	for _, original := range cases {
		value, err := original.Value()
		assert.Nil(t, err)

		var scanned ContactProperties
		assert.Nil(t, scanned.Scan(value))

		if len(original) == 0 {
			assert.Equal(t, 0, len(scanned))
			continue
		}
		assert.Equal(t, original, scanned)
	}
}

func TestContactProperties_ScanFromString(t *testing.T) {
	var scanned ContactProperties
	assert.Nil(t, scanned.Scan(`{"k":"v"}`))
	assert.Equal(t, ContactProperties{"k": "v"}, scanned)
}

func TestContactProperties_ScanNil(t *testing.T) {
	var scanned ContactProperties
	assert.Nil(t, scanned.Scan(nil))
	assert.Equal(t, 0, len(scanned))
}

func TestContactProperties_ScanUnsupportedType(t *testing.T) {
	var scanned ContactProperties

	err := scanned.Scan(123)

	assert.Error(t, err)
	assert.Equal(t, "*stacktrace.stacktrace", reflect.TypeOf(err).String())
	assert.Contains(t, err.Error(), "unsupported type [int] for ContactProperties")
}
