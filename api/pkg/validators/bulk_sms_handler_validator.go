package validators

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/dustin/go-humanize"
	"github.com/jszwec/csvutil"
	"github.com/nyaruka/phonenumbers"
	"github.com/palantir/stacktrace"
)

// BulkMessageHandlerValidator validates models used in handlers.BillingHandler
type BulkMessageHandlerValidator struct {
	validator
	phoneService *services.PhoneService
	logger       telemetry.Logger
	tracer       telemetry.Tracer
}

// NewBulkMessageHandlerValidator creates a new handlers.BulkMessageHandlerValidator validator
func NewBulkMessageHandlerValidator(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	phoneService *services.PhoneService,
) (v *BulkMessageHandlerValidator) {
	return &BulkMessageHandlerValidator{
		logger:       logger.WithService(fmt.Sprintf("%T", v)),
		tracer:       tracer,
		phoneService: phoneService,
	}
}

// ValidateStore validates the requests.BillingUsageHistory request
func (v *BulkMessageHandlerValidator) ValidateStore(ctx context.Context, userID entities.UserID, header *multipart.FileHeader) ([]*requests.BulkMessage, url.Values) {
	ctx, span, ctxLogger := v.tracer.StartWithLogger(ctx, v.logger)
	defer span.End()

	result := url.Values{}

	if header.Size >= 5000000 {
		result.Add("document", fmt.Sprintf("The CSV file must be less than 500 KB the file you uploaded is [%s].", humanize.Bytes(uint64(header.Size))))
		return nil, result
	}

	file, err := header.Open()
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot open file [%s] for reading", header.Filename)))
		result.Add("document", fmt.Sprintf("Cannot open the uploaded file with name [%s].", header.Filename))
		return nil, result
	}
	defer func() {
		if e := file.Close(); e != nil {
			ctxLogger.Error(stacktrace.Propagate(e, fmt.Sprintf("cannot close file [%s]", header.Filename)))
		}
	}()

	b := new(bytes.Buffer)
	if _, err = io.Copy(b, file); err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot copy file [%s] to buffer", header.Filename)))
		result.Add("document", fmt.Sprintf("Cannot read the conents of the uploaded file [%s].", header.Filename))
		return nil, result
	}

	var messages []*requests.BulkMessage
	if err := csvutil.Unmarshal(b.Bytes(), &messages); err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot unmarshall contents [%s] into type [%T] for file [%s]", b.Bytes(), messages, header.Filename)))
		result.Add("document", fmt.Sprintf("Cannot read the conents of the uploaded file [%s].", header.Filename))
		return nil, result
	}

	if len(messages) == 0 {
		result.Add("document", "The CSV file doesn't contain any valid records. Make sure you are using the official httpSMS template.")
		return messages, result
	}

	if len(messages) > 100 {
		result.Add("document", "The CSV file must contain less than 100 records.")
		return messages, result
	}

	result = v.validateMessages(messages)
	if len(result) != 0 {
		return messages, result
	}

	result = v.validateOwners(ctx, userID, messages)
	if len(result) != 0 {
		return messages, result
	}

	return messages, result
}

func (v *BulkMessageHandlerValidator) validateMessages(messages []*requests.BulkMessage) url.Values {
	result := url.Values{}
	for index, message := range messages {
		if _, err := phonenumbers.Parse(message.FromPhoneNumber, phonenumbers.UNKNOWN_REGION); err != nil {
			result.Add("document", fmt.Sprintf("Row [%d]: The FromPhoneNumber [%s] is not a valid E.164 phone number", index+2, message.FromPhoneNumber))
		}

		if _, err := phonenumbers.Parse(message.ToPhoneNumber, phonenumbers.UNKNOWN_REGION); err != nil {
			result.Add("document", fmt.Sprintf("Row [%d]: The ToPhoneNumber [%s] is not a valid E.164 phone number", index+2, message.ToPhoneNumber))
		}

		if len(message.Content) > 1024 {
			result.Add("document", fmt.Sprintf("Row [%d]: The message content must be less than 1024 characters.", index+2))
		}
	}
	return result
}

func (v *BulkMessageHandlerValidator) validateOwners(ctx context.Context, userID entities.UserID, messages []*requests.BulkMessage) url.Values {
	numbers := map[string][]int{}
	for index, message := range messages {
		numbers[message.FromPhoneNumber] = append(numbers[message.FromPhoneNumber], index+2)
	}

	result := url.Values{}
	for number, rows := range numbers {
		_, err := v.phoneService.Load(ctx, userID, strings.TrimSpace(number))
		if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
			result.Add("document", fmt.Sprintf("Rows [%s]: The FromPhoneNumber [%s] is not registered on your account", v.toString(rows), number))
		}
	}
	return result
}

func (v *BulkMessageHandlerValidator) toString(value []int) string {
	result := strings.Builder{}
	for index, row := range value {
		if index != 0 {
			result.WriteString(", ")
		}
		result.WriteString(fmt.Sprintf("%d", row))
	}
	return result.String()
}
