package validators

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"strings"

	"github.com/xuri/excelize/v2"

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

	messages, result := v.parseFile(ctxLogger, userID, header)
	if len(result) != 0 {
		return messages, result
	}

	if len(messages) == 0 {
		result.Add("document", "The uploaded file doesn't contain any valid records. Make sure you are using the official httpSMS template.")
		return messages, result
	}

	if len(messages) > 100 {
		result.Add("document", "The uploaded file must contain less than 100 records.")
		return messages, result
	}

	for index, message := range messages {
		messages[index] = message.Sanitize()
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

func (v *BulkMessageHandlerValidator) parseFile(ctxLogger telemetry.Logger, userID entities.UserID, header *multipart.FileHeader) ([]*requests.BulkMessage, url.Values) {
	if header.Header.Get("Content-Type") == "text/csv" || strings.HasSuffix(header.Filename, ".csv") {
		return v.parseCSV(ctxLogger, userID, header)
	}
	if header.Header.Get("Content-Type") == "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" || strings.HasSuffix(header.Filename, ".xlsx") {
		return v.parseXlsx(ctxLogger, userID, header)
	}

	ctxLogger.Error(stacktrace.NewError(fmt.Sprintf("cannot parse file [%s] for user [%s] with content type [%s]", header.Filename, userID, header.Header.Get("Content-Type"))))

	result := url.Values{}
	result.Add("document", fmt.Sprintf("The file [%s] is not a valid CSV or Excel file.", header.Filename))
	return nil, result
}

func (v *BulkMessageHandlerValidator) parseXlsx(ctxLogger telemetry.Logger, userID entities.UserID, header *multipart.FileHeader) ([]*requests.BulkMessage, url.Values) {
	content, result := v.parseBytes(ctxLogger, userID, header)
	if len(result) != 0 {
		return nil, result
	}

	excel, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot generate excel file from [%s] for user [%s]", header.Filename, userID)))
		result.Add("document", fmt.Sprintf("Cannot parse the uploaded excel file with name [%s].", header.Filename))
		return nil, result
	}

	rows, err := excel.GetRows(excel.GetSheetName(0))
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot get rows from excel file [%s] for user [%s]", header.Filename, userID)))
		result.Add("document", fmt.Sprintf("Cannot parse the uploaded excel file with name [%s].", header.Filename))
		return nil, result
	}

	var messages []*requests.BulkMessage
	for index, row := range rows {
		if len(row) < 3 || strings.TrimSpace(row[0]) == "" || index == 0 {
			continue
		}
		messages = append(messages, &requests.BulkMessage{
			FromPhoneNumber: strings.TrimSpace(row[0]),
			ToPhoneNumber:   strings.TrimSpace(row[1]),
			Content:         row[2],
		})
	}

	return messages, nil
}

func (v *BulkMessageHandlerValidator) parseBytes(ctxLogger telemetry.Logger, userID entities.UserID, header *multipart.FileHeader) ([]byte, url.Values) {
	result := url.Values{}

	if header.Size >= 5000000 {
		result.Add("document", fmt.Sprintf("The CSV file must be less than 500 KB the file you uploaded is [%s].", humanize.Bytes(uint64(header.Size))))
		return nil, result
	}

	file, err := header.Open()
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot open file [%s] for reading for user [%s]", header.Filename, userID)))
		result.Add("document", fmt.Sprintf("Cannot open the uploaded file with name [%s].", header.Filename))
		return nil, result
	}
	defer func() {
		if e := file.Close(); e != nil {
			ctxLogger.Error(stacktrace.Propagate(e, fmt.Sprintf("cannot close file [%s] for user [%s]", header.Filename, userID)))
		}
	}()

	b := new(bytes.Buffer)
	if _, err = io.Copy(b, file); err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot copy file [%s] to buffer for user [%s]", header.Filename, userID)))
		result.Add("document", fmt.Sprintf("Cannot read the conents of the uploaded file [%s].", header.Filename))
		return nil, result
	}

	return b.Bytes(), result
}

func (v *BulkMessageHandlerValidator) parseCSV(ctxLogger telemetry.Logger, userID entities.UserID, header *multipart.FileHeader) ([]*requests.BulkMessage, url.Values) {
	content, result := v.parseBytes(ctxLogger, userID, header)
	if len(result) != 0 {
		return nil, result
	}

	var messages []*requests.BulkMessage
	if err := csvutil.Unmarshal(content, &messages); err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot unmarshall contents [%s] into type [%T] for file [%s] and user [%s]", content, messages, header.Filename, userID)))
		result.Add("document", fmt.Sprintf("Cannot read the conents of the uploaded file [%s].", header.Filename))
		return nil, result
	}

	return messages, nil
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
