package repositories

import "testing"

func TestExtensionFromContentType(t *testing.T) {
	tests := []struct {
		contentType string
		expected    string
	}{
		{"image/jpeg", ".jpg"},
		{"image/png", ".png"},
		{"image/gif", ".gif"},
		{"image/webp", ".webp"},
		{"image/bmp", ".bmp"},
		{"video/mp4", ".mp4"},
		{"video/3gpp", ".3gp"},
		{"audio/mpeg", ".mp3"},
		{"audio/ogg", ".ogg"},
		{"audio/amr", ".amr"},
		{"application/pdf", ".pdf"},
		{"text/vcard", ".vcf"},
		{"text/x-vcard", ".vcf"},
		{"application/octet-stream", ".bin"},
		{"unknown/type", ".bin"},
		{"", ".bin"},
	}
	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			got := ExtensionFromContentType(tt.contentType)
			if got != tt.expected {
				t.Errorf("ExtensionFromContentType(%q) = %q, want %q", tt.contentType, got, tt.expected)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		index    int
		expected string
	}{
		{"photo.jpg", 0, "photo"},
		{"../../etc/passwd", 0, "etcpasswd"},
		{"hello/world\\test", 0, "helloworldtest"},
		{"normal_file", 0, "normal_file"},
		{"", 0, "attachment-0"},
		{"   ", 0, "attachment-0"},
		{"...", 1, "attachment-1"},
		{"My Photo", 0, "My-Photo"},
		{"file name with spaces.png", 0, "file-name-with-spaces"},
		{"UPPER_CASE", 0, "UPPER_CASE"},
		{"special!@#chars", 0, "specialchars"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeFilename(tt.name, tt.index)
			if got != tt.expected {
				t.Errorf("SanitizeFilename(%q, %d) = %q, want %q", tt.name, tt.index, got, tt.expected)
			}
		})
	}
}
