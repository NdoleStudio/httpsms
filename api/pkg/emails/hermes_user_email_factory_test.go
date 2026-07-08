package emails

import (
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/stretchr/testify/assert"
)

func testUserEmailFactory() UserEmailFactory {
	return NewHermesUserEmailFactory(&HermesGeneratorConfig{
		AppURL:     "https://httpsms.com",
		AppName:    "httpSMS",
		AppLogoURL: "https://httpsms.com/logo.png",
	})
}

func TestFormatBillingDate_RendersInProvidedTimezone(t *testing.T) {
	// 2026-06-19 02:00 UTC
	timestamp := time.Date(2026, 6, 19, 2, 0, 0, 0, time.UTC)

	// A timezone five hours behind UTC rolls back to the previous day.
	behind := time.FixedZone("UTC-5", -5*60*60)
	assert.Equal(t, "18 June 2026", formatBillingDate(timestamp, behind))

	// A timezone ahead of UTC stays on the same day.
	ahead := time.FixedZone("UTC+10", 10*60*60)
	assert.Equal(t, "19 June 2026", formatBillingDate(timestamp, ahead))

	// UTC renders the underlying date as-is.
	assert.Equal(t, "19 June 2026", formatBillingDate(timestamp, time.UTC))
}

func TestUsageLimitExceeded_IncludesBreakdownAndBillingPeriod(t *testing.T) {
	factory := testUserEmailFactory()
	user := &entities.User{
		Email:            "name@email.com",
		Timezone:         "UTC",
		SubscriptionName: entities.SubscriptionNameProMonthly,
	}
	usage := &entities.BillingUsage{
		SentMessages:     3000,
		ReceivedMessages: 2000,
		StartTimestamp:   time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC),
		EndTimestamp:     time.Date(2026, 7, 18, 23, 59, 59, 0, time.UTC),
	}

	email, err := factory.UsageLimitExceeded(user, usage)

	assert.NoError(t, err)
	assert.Equal(t, "name@email.com", email.ToEmail)
	assert.Equal(t, "⚠️ You have exceeded your plan limit", email.Subject)
	assert.Contains(t, email.Text, "limit of 5000 messages")
	assert.Contains(t, email.Text, "Between 19 June 2026 and 18 July 2026")
	assert.Contains(t, email.Text, "you sent 3000 messages and received 2000")
	assert.Contains(t, email.Text, "for a total of 5000")
}

func TestUsageLimitAlert_IncludesPercentBreakdownAndLimit(t *testing.T) {
	factory := testUserEmailFactory()
	user := &entities.User{
		Email:            "name@email.com",
		Timezone:         "UTC",
		SubscriptionName: entities.SubscriptionNameProMonthly,
	}
	usage := &entities.BillingUsage{
		SentMessages:     2500,
		ReceivedMessages: 1500,
		StartTimestamp:   time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC),
		EndTimestamp:     time.Date(2026, 7, 18, 23, 59, 59, 0, time.UTC),
	}

	email, err := factory.UsageLimitAlert(user, usage)

	assert.NoError(t, err)
	assert.Equal(t, "name@email.com", email.ToEmail)
	assert.Equal(t, "⚠️ 80% Usage Limit Alert", email.Subject)
	assert.Contains(t, email.Text, "used 80% of your monthly SMS limit")
	assert.Contains(t, email.Text, "Between 19 June 2026 and 18 July 2026")
	assert.Contains(t, email.Text, "you sent 2500 messages and received 1500")
	assert.Contains(t, email.Text, "for a total of 4000 out of your 5000 message limit")
}
