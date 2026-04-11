package repositories

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
)

// GCSAttachmentStorage stores attachments in Google Cloud Storage
type GCSAttachmentStorage struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	client *storage.Client
	bucket string
}

// NewGCSAttachmentStorage creates a new GCSAttachmentStorage
func NewGCSAttachmentStorage(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	client *storage.Client,
	bucket string,
) *GCSAttachmentStorage {
	return &GCSAttachmentStorage{
		logger: logger.WithService(fmt.Sprintf("%T", &GCSAttachmentStorage{})),
		tracer: tracer,
		client: client,
		bucket: bucket,
	}
}

// Upload stores attachment data at the given path in GCS
func (s *GCSAttachmentStorage) Upload(ctx context.Context, path string, data []byte) error {
	ctx, span := s.tracer.Start(ctx)
	defer span.End()

	writer := s.client.Bucket(s.bucket).Object(path).NewWriter(ctx)
	if _, err := writer.Write(data); err != nil {
		return s.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot write attachment to GCS path [%s]", path)))
	}

	if err := writer.Close(); err != nil {
		return s.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot close GCS writer for path [%s]", path)))
	}

	s.logger.Info(fmt.Sprintf("uploaded attachment to GCS path [%s/%s] with size [%d]", s.bucket, path, len(data)))
	return nil
}

// Download retrieves attachment data from the given path in GCS
func (s *GCSAttachmentStorage) Download(ctx context.Context, path string) ([]byte, error) {
	ctx, span := s.tracer.Start(ctx)
	defer span.End()

	reader, err := s.client.Bucket(s.bucket).Object(path).NewReader(ctx)
	if err != nil {
		return nil, s.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot open GCS reader for path [%s]", path)))
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, s.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot read attachment from GCS path [%s]", path)))
	}

	return data, nil
}

// Delete removes an attachment at the given path in GCS
func (s *GCSAttachmentStorage) Delete(ctx context.Context, path string) error {
	ctx, span := s.tracer.Start(ctx)
	defer span.End()

	if err := s.client.Bucket(s.bucket).Object(path).Delete(ctx); err != nil {
		return s.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot delete GCS object at path [%s]", path)))
	}

	s.logger.Info(fmt.Sprintf("deleted attachment from GCS path [%s/%s]", s.bucket, path))
	return nil
}
