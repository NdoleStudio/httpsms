package repositories

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// AttachmentRepository is the interface for storing and retrieving message attachments
type AttachmentRepository interface {
	// Upload stores attachment data at the given path with the specified content type
	Upload(ctx context.Context, path string, data []byte, contentType string) error
	// Download retrieves attachment data from the given path
	Download(ctx context.Context, path string) ([]byte, error)
	// Delete removes an attachment at the given path
	Delete(ctx context.Context, path string) error
}

// contentTypeExtensions maps MIME types to file extensions
var contentTypeExtensions = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/gif":       ".gif",
	"image/webp":      ".webp",
	"image/bmp":       ".bmp",
	"video/mp4":       ".mp4",
	"video/3gpp":      ".3gp",
	"audio/mpeg":      ".mp3",
	"audio/ogg":       ".ogg",
	"audio/amr":       ".amr",
	"application/pdf": ".pdf",
	"text/vcard":      ".vcf",
	"text/x-vcard":    ".vcf",
}

// extensionContentTypes is the reverse map from file extensions to canonical MIME types
var extensionContentTypes = map[string]string{
	".jpg":  "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".bmp":  "image/bmp",
	".mp4":  "video/mp4",
	".3gp":  "video/3gpp",
	".mp3":  "audio/mpeg",
	".ogg":  "audio/ogg",
	".amr":  "audio/amr",
	".pdf":  "application/pdf",
	".vcf":  "text/vcard",
}

// AllowedContentTypes returns the set of allowed MIME types for attachments
func AllowedContentTypes() map[string]bool {
	allowed := make(map[string]bool, len(contentTypeExtensions))
	for ct := range contentTypeExtensions {
		allowed[ct] = true
	}
	return allowed
}

// ExtensionFromContentType returns the file extension for a MIME content type.
// Returns ".bin" if the content type is not recognized.
func ExtensionFromContentType(contentType string) string {
	if ext, ok := contentTypeExtensions[contentType]; ok {
		return ext
	}
	return ".bin"
}

// ContentTypeFromExtension returns the MIME content type for a file extension.
// Returns "application/octet-stream" if the extension is not recognized.
func ContentTypeFromExtension(ext string) string {
	if ct, ok := extensionContentTypes[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}

// SanitizeFilename removes path separators and traversal sequences from a filename.
// Returns "attachment-{index}" if the sanitized name is empty.
func SanitizeFilename(name string, index int) string {
	name = strings.TrimSuffix(name, filepath.Ext(name))

	var builder strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			builder.WriteRune(r)
		} else if r == ' ' {
			builder.WriteRune('-')
		}
	}
	name = strings.Trim(builder.String(), "-")

	if name == "" {
		return fmt.Sprintf("attachment-%d", index)
	}
	return name
}
