package repositories

import (
	"context"
	"fmt"
	"sync"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
)

// MemoryAttachmentRepository stores attachments in memory
type MemoryAttachmentRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	data   sync.Map
}

// NewMemoryAttachmentRepository creates a new MemoryAttachmentRepository
func NewMemoryAttachmentRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
) *MemoryAttachmentRepository {
	return &MemoryAttachmentRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &MemoryAttachmentRepository{})),
		tracer: tracer,
	}
}

// Upload stores attachment data at the given path
func (s *MemoryAttachmentRepository) Upload(ctx context.Context, path string, data []byte, _ string) error {
	_, span, ctxLogger := s.tracer.StartWithLogger(ctx, s.logger)
	defer span.End()

	s.data.Store(path, data)
	ctxLogger.Info(fmt.Sprintf("stored attachment at path [%s] with size [%d]", path, len(data)))
	return nil
}

// Download retrieves attachment data from the given path
func (s *MemoryAttachmentRepository) Download(ctx context.Context, path string) ([]byte, error) {
	_, span, _ := s.tracer.StartWithLogger(ctx, s.logger)
	defer span.End()

	value, ok := s.data.Load(path)
	if !ok {
		return nil, s.tracer.WrapErrorSpan(span, stacktrace.NewErrorWithCode(ErrCodeNotFound, fmt.Sprintf("attachment not found at path [%s]", path)))
	}
	return value.([]byte), nil
}

// Delete removes an attachment at the given path
func (s *MemoryAttachmentRepository) Delete(ctx context.Context, path string) error {
	_, span, ctxLogger := s.tracer.StartWithLogger(ctx, s.logger)
	defer span.End()

	s.data.Delete(path)
	ctxLogger.Info(fmt.Sprintf("deleted attachment at path [%s]", path))
	return nil
}
