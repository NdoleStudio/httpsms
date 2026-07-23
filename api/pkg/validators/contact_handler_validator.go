package validators

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"mime/multipart"
	"net/mail"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/nyaruka/phonenumbers"
	"github.com/thedevsaddam/govalidator"
)

const (
	maxContactBatch          = 1000
	maxContactUploadBytes    = 500 * 1024
	contactUploadDocumentKey = "document"
	contactUploadContactsKey = "contacts"
	contactCSVColumnName     = "Name"
	contactCSVColumnEmails   = "Emails"
	contactCSVColumnPhones   = "PhoneNumbers"
	contactCSVContentType    = "text/csv"
	contactCSVContentTypeAlt = "application/csv"
	contactCSVContentTypeBin = "application/octet-stream"
	contactCSVContentTypeXls = "application/vnd.ms-excel"
)

// ContactHandlerValidator validates models used in handlers.ContactHandler
type ContactHandlerValidator struct {
	validator
	logger telemetry.Logger
	tracer telemetry.Tracer
}

// NewContactHandlerValidator creates a new handlers.ContactHandler validator
func NewContactHandlerValidator(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
) (v *ContactHandlerValidator) {
	return &ContactHandlerValidator{
		logger: logger.WithService(fmt.Sprintf("%T", v)),
		tracer: tracer,
	}
}

// ValidateStore validates a contact create request.
func (validator *ContactHandlerValidator) ValidateStore(_ context.Context, request requests.ContactStoreRequest) url.Values {
	result := url.Values{}

	if len(request.Contacts) == 0 {
		result.Add(contactUploadContactsKey, "You must provide at least one contact.")
		return result
	}

	if len(request.Contacts) > maxContactBatch {
		result.Add(contactUploadContactsKey, fmt.Sprintf("You cannot create more than %d contacts in one request.", maxContactBatch))
		return result
	}

	for index, item := range request.Contacts {
		validator.validateItem(result, contactUploadContactsKey, "Contact", index+1, item)
	}

	return result
}

// ValidateUpdate validates a contact update request.
func (validator *ContactHandlerValidator) ValidateUpdate(_ context.Context, request requests.ContactUpdateRequest) url.Values {
	result := url.Values{}
	validator.validateItem(result, contactUploadContactsKey, "Contact", 1, requests.ContactItem{
		Name:         request.Name,
		Emails:       request.Emails,
		PhoneNumbers: request.PhoneNumbers,
		Properties:   request.Properties,
	})
	return result
}

// ValidateIndex validates a contact index request.
func (validator *ContactHandlerValidator) ValidateIndex(_ context.Context, request requests.ContactIndex) url.Values {
	result := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"limit": []string{"required", "numeric"},
			"skip":  []string{"required", "numeric"},
			"query": []string{"max:100"},
		},
	}).ValidateStruct()

	if limit, err := strconv.Atoi(request.Limit); request.Limit != "" && err == nil {
		if limit < 1 || limit > 100 {
			result.Add("limit", "The limit must be between 1 and 100.")
		}
	}

	if skip, err := strconv.Atoi(request.Skip); request.Skip != "" && err == nil {
		if skip < 0 {
			result.Add("skip", "The skip must be greater than or equal to 0.")
		}
	}

	return result
}

// ValidateUpload parses and validates a CSV contacts upload.
func (validator *ContactHandlerValidator) ValidateUpload(ctx context.Context, userID entities.UserID, header *multipart.FileHeader) ([]requests.ContactItem, url.Values) {
	_, span, ctxLogger := validator.tracer.StartWithLogger(ctx, validator.logger)
	defer span.End()

	result := url.Values{}
	if header == nil {
		result.Add(contactUploadDocumentKey, "The CSV file is required.")
		return nil, result
	}

	if !isContactCSVFile(header) {
		ctxLogger.Error(stacktrace.NewErrorf("cannot parse file [%s] for user [%s] with content type [%s]", header.Filename, userID, header.Header.Get("Content-Type")))
		result.Add(contactUploadDocumentKey, fmt.Sprintf("The file [%s] is not a valid CSV file. Only CSV files are supported.", header.Filename))
		return nil, result
	}

	content, errors := validator.parseContactUploadBytes(ctxLogger, userID, header)
	if len(errors) != 0 {
		return nil, errors
	}

	rows, errors := validator.parseContactCSV(ctxLogger, userID, header.Filename, content)
	if len(errors) != 0 {
		return nil, errors
	}

	if len(rows) > maxContactBatch {
		result.Add(contactUploadDocumentKey, fmt.Sprintf("The uploaded file must contain no more than %d records.", maxContactBatch))
		return nil, result
	}

	items := make([]requests.ContactItem, 0, len(rows))
	for _, row := range rows {
		// Sanitize each row exactly like the JSON create path (SanitizeContactItem)
		// before validating it, so both entry points accept and normalize the same
		// phone/email formats. Row-indexed error messages are preserved because the
		// row number is still passed to validateItem.
		item := requests.SanitizeContactItem(requests.ContactItem{
			Name:         row.values[contactCSVColumnName],
			Emails:       splitContactMultiValue(row.values[contactCSVColumnEmails]),
			PhoneNumbers: splitContactMultiValue(row.values[contactCSVColumnPhones]),
		})
		items = append(items, item)
		validator.validateItem(result, contactUploadDocumentKey, "Row", row.number, item)
	}

	if len(items) == 0 {
		result.Add(contactUploadDocumentKey, "The uploaded file must contain at least one contact.")
	}

	return items, result
}

func (validator *ContactHandlerValidator) validateItem(result url.Values, key string, label string, index int, item requests.ContactItem) {
	prefix := fmt.Sprintf("%s [%d]", label, index)

	if strings.TrimSpace(item.Name) == "" {
		result.Add(key, fmt.Sprintf("%s: The name is required.", prefix))
	}

	if !hasNonEmptyValue(item.PhoneNumbers) {
		result.Add(key, fmt.Sprintf("%s: At least one phone number is required.", prefix))
	}

	for _, number := range item.PhoneNumbers {
		cleanNumber := strings.TrimSpace(number)
		if cleanNumber == "" {
			result.Add(key, fmt.Sprintf("%s: The phone number is required.", prefix))
			continue
		}

		parsed, err := phonenumbers.Parse(cleanNumber, phonenumbers.UNKNOWN_REGION)
		if err != nil || !phonenumbers.IsValidNumber(parsed) {
			result.Add(key, fmt.Sprintf("%s: The phone number [%s] is not a valid E.164 phone number.", prefix, cleanNumber))
		}
	}

	for _, email := range item.Emails {
		cleanEmail := strings.TrimSpace(email)
		if cleanEmail == "" {
			result.Add(key, fmt.Sprintf("%s: The email is not a valid email address.", prefix))
			continue
		}

		address, err := mail.ParseAddress(cleanEmail)
		if err != nil || address.Address != cleanEmail {
			result.Add(key, fmt.Sprintf("%s: The email [%s] is not a valid email address.", prefix, cleanEmail))
		}
	}
}

func (validator *ContactHandlerValidator) parseContactUploadBytes(ctxLogger telemetry.Logger, userID entities.UserID, header *multipart.FileHeader) ([]byte, url.Values) {
	result := url.Values{}

	if header.Size > maxContactUploadBytes {
		result.Add(contactUploadDocumentKey, "The CSV file must be less than or equal to 500 KB.")
		return nil, result
	}

	file, err := header.Open()
	if err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot open file [%s] for reading for user [%s]", header.Filename, userID))
		result.Add(contactUploadDocumentKey, fmt.Sprintf("Cannot open the uploaded file [%s].", header.Filename))
		return nil, result
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			ctxLogger.Error(stacktrace.Propagatef(closeErr, "cannot close file [%s] for user [%s]", header.Filename, userID))
		}
	}()

	content, err := io.ReadAll(io.LimitReader(file, maxContactUploadBytes+1))
	if err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot read file [%s] for user [%s]", header.Filename, userID))
		result.Add(contactUploadDocumentKey, fmt.Sprintf("Cannot read the uploaded file [%s].", header.Filename))
		return nil, result
	}

	if len(content) > maxContactUploadBytes {
		result.Add(contactUploadDocumentKey, "The CSV file must be less than or equal to 500 KB.")
		return nil, result
	}

	return content, result
}

type contactCSVRow struct {
	number int
	values map[string]string
}

func (validator *ContactHandlerValidator) parseContactCSV(ctxLogger telemetry.Logger, userID entities.UserID, filename string, content []byte) ([]contactCSVRow, url.Values) {
	result := url.Values{}
	reader := csv.NewReader(bytes.NewReader(content))
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err == io.EOF {
		result.Add(contactUploadDocumentKey, fmt.Sprintf("Cannot parse the uploaded CSV file [%s]. The file is empty.", filename))
		return nil, result
	}
	if err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot read CSV header from file [%s] for user [%s]", filename, userID))
		result.Add(contactUploadDocumentKey, fmt.Sprintf("Cannot parse the uploaded CSV file [%s]. Use the official httpSMS contacts template.", filename))
		return nil, result
	}

	columnIndexes, ok := contactCSVColumnIndexes(headers)
	if !ok {
		result.Add(contactUploadDocumentKey, "The uploaded CSV file must contain the columns [Name, Emails, PhoneNumbers].")
		return nil, result
	}

	reader.FieldsPerRecord = len(headers)

	var rows []contactCSVRow
	for rowNumber := 2; ; rowNumber++ {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			ctxLogger.Error(stacktrace.Propagatef(readErr, "cannot read CSV row [%d] from file [%s] for user [%s]", rowNumber, filename, userID))
			result.Add(contactUploadDocumentKey, fmt.Sprintf("Cannot parse the uploaded CSV file [%s]. Use the official httpSMS contacts template.", filename))
			return nil, result
		}

		row := contactCSVRow{
			number: rowNumber,
			values: map[string]string{
				contactCSVColumnName:   record[columnIndexes[contactCSVColumnName]],
				contactCSVColumnEmails: record[columnIndexes[contactCSVColumnEmails]],
				contactCSVColumnPhones: record[columnIndexes[contactCSVColumnPhones]],
			},
		}
		rows = append(rows, row)
		if len(rows) > maxContactBatch {
			return rows, result
		}
	}

	return rows, result
}

func contactCSVColumnIndexes(headers []string) (map[string]int, bool) {
	required := []string{contactCSVColumnName, contactCSVColumnEmails, contactCSVColumnPhones}
	indexes := map[string]int{}
	for index, header := range headers {
		indexes[strings.TrimSpace(header)] = index
	}

	for _, column := range required {
		if _, ok := indexes[column]; !ok {
			return nil, false
		}
	}

	return indexes, true
}

func isContactCSVFile(header *multipart.FileHeader) bool {
	if strings.ToLower(filepath.Ext(header.Filename)) != ".csv" {
		return false
	}

	contentType := strings.ToLower(strings.TrimSpace(header.Header.Get("Content-Type")))
	if index := strings.Index(contentType, ";"); index >= 0 {
		contentType = strings.TrimSpace(contentType[:index])
	}

	return contentType == "" ||
		contentType == contactCSVContentType ||
		contentType == contactCSVContentTypeAlt ||
		contentType == contactCSVContentTypeBin ||
		contentType == contactCSVContentTypeXls
}

func splitContactMultiValue(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	var result []string
	for _, item := range strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';'
	}) {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}

	return result
}

func hasNonEmptyValue(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}
