package repositories

import (
	"context"
	"errors"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
)

// GoogleCloudStorageAttachmentRepository stores attachments in Google Cloud Storage
type GoogleCloudStorageAttachmentRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	client *storage.Client
	bucket string
}

// NewGoogleCloudStorageAttachmentRepository creates a new GoogleCloudStorageAttachmentRepository
func NewGoogleCloudStorageAttachmentRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	client *storage.Client,
	bucket string,
) *GoogleCloudStorageAttachmentRepository {
	return &GoogleCloudStorageAttachmentRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &GoogleCloudStorageAttachmentRepository{})),
		tracer: tracer,
		client: client,
		bucket: bucket,
	}
}

// Upload stores attachment data at the given path in GCS
func (s *GoogleCloudStorageAttachmentRepository) Upload(ctx context.Context, path string, data []byte, contentType string) error {
	ctx, span, ctxLogger := s.tracer.StartWithLogger(ctx, s.logger)
	defer span.End()

	writer := s.client.Bucket(s.bucket).Object(path).NewWriter(ctx)
	writer.ContentType = contentType

	if _, err := writer.Write(data); err != nil {
		return s.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot write attachment to GCS path [%s]", path)))
	}

	if err := writer.Close(); err != nil {
		return s.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot close GCS writer for path [%s]", path)))
	}

	ctxLogger.Info(fmt.Sprintf("uploaded attachment to GCS path [%s/%s] with size [%d]", s.bucket, path, len(data)))
	return nil
}

// Download retrieves attachment data from the given path in GCS
func (s *GoogleCloudStorageAttachmentRepository) Download(ctx context.Context, path string) ([]byte, error) {
	ctx, span, ctxLogger := s.tracer.StartWithLogger(ctx, s.logger)
	defer span.End()

	reader, err := s.client.Bucket(s.bucket).Object(path).NewReader(ctx)
	if err != nil {
		msg := fmt.Sprintf("cannot open GCS reader for path [%s]", path)
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, s.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, msg))
		}
		return nil, s.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, s.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot read attachment from GCS path [%s]", path)))
	}

	ctxLogger.Info(fmt.Sprintf("downloaded attachment from GCS path [%s/%s] with size [%d]", s.bucket, path, len(data)))
	return data, nil
}

// Delete removes an attachment at the given path in GCS
func (s *GoogleCloudStorageAttachmentRepository) Delete(ctx context.Context, path string) error {
	ctx, span, ctxLogger := s.tracer.StartWithLogger(ctx, s.logger)
	defer span.End()

	if err := s.client.Bucket(s.bucket).Object(path).Delete(ctx); err != nil {
		return s.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot delete GCS object at path [%s]", path)))
	}

	ctxLogger.Info(fmt.Sprintf("deleted attachment from GCS path [%s/%s]", s.bucket, path))
	return nil
}
