package entities

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBillingUsage_TotalMessages(t *testing.T) {
	usage := BillingUsage{SentMessages: 321, ReceivedMessages: 465}
	assert.Equal(t, uint(786), usage.TotalMessages())
}

func TestBillingUsage_IsEntitled_BelowLimit(t *testing.T) {
	usage := BillingUsage{SentMessages: 100, ReceivedMessages: 100}
	assert.True(t, usage.IsEntitled(1, 500))
}

func TestBillingUsage_IsEntitled_ReachingExactlyLimitIsEntitled(t *testing.T) {
	// total is one below the limit, sending one more brings the total to
	// exactly the limit, which should still be allowed.
	usage := BillingUsage{SentMessages: 300, ReceivedMessages: 199}
	assert.True(t, usage.IsEntitled(1, 500))
}

func TestBillingUsage_IsEntitled_ExceedingLimitIsNotEntitled(t *testing.T) {
	// total already equals the limit, sending one more would exceed it.
	usage := BillingUsage{SentMessages: 300, ReceivedMessages: 200}
	assert.False(t, usage.IsEntitled(1, 500))
}

func TestBillingUsage_IsEntitled_BulkCountFittingExactly(t *testing.T) {
	usage := BillingUsage{SentMessages: 250, ReceivedMessages: 248}
	assert.True(t, usage.IsEntitled(2, 500))
}

func TestBillingUsage_IsEntitled_BulkCountExceeding(t *testing.T) {
	usage := BillingUsage{SentMessages: 250, ReceivedMessages: 248}
	assert.False(t, usage.IsEntitled(3, 500))
}
