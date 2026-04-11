package repositories

import (
	"context"
	"fmt"
	"sync"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
)

// MemoryAttachmentStorage stores attachments in memory
type MemoryAttachmentStorage struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	data   sync.Map
}

// NewMemoryAttachmentStorage creates a new MemoryAttachmentStorage
func NewMemoryAttachmentStorage(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
) *MemoryAttachmentStorage {
	return &MemoryAttachmentStorage{
		logger: logger.WithService(fmt.Sprintf("%T", &MemoryAttachmentStorage{})),
		tracer: tracer,
	}
}

// Upload stores attachment data at the given path
func (s *MemoryAttachmentStorage) Upload(ctx context.Context, path string, data []byte) error {
	_, span := s.tracer.Start(ctx)
	defer span.End()

	s.data.Store(path, data)
	s.logger.Info(fmt.Sprintf("stored attachment at path [%s] with size [%d]", path, len(data)))
	return nil
}

// Download retrieves attachment data from the given path
func (s *MemoryAttachmentStorage) Download(ctx context.Context, path string) ([]byte, error) {
	_, span := s.tracer.Start(ctx)
	defer span.End()

	value, ok := s.data.Load(path)
	if !ok {
		return nil, ErrAttachmentNotFound
	}
	return value.([]byte), nil
}

// Delete removes an attachment at the given path
func (s *MemoryAttachmentStorage) Delete(ctx context.Context, path string) error {
	_, span := s.tracer.Start(ctx)
	defer span.End()

	s.data.Delete(path)
	s.logger.Info(fmt.Sprintf("deleted attachment at path [%s]", path))
	return nil
}
