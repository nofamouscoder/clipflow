package utils

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	MaxFileSize = 100 * 1024 * 1024 // 100MB
)

var (
	AllowedVideoFormats = []string{".mp4", ".avi", ".mov", ".wmv", ".flv", ".webm"}
	AllowedAudioFormats = []string{".mp3", ".wav", ".aac", ".ogg", ".flac"}
)

// FileInfo represents information about an uploaded file
type FileInfo struct {
	OriginalName string
	StoredName   string
	FilePath     string
	FileType     string
	Size         int64
}

// ValidateFile checks if a file is valid for upload
func ValidateFile(file *multipart.FileHeader, allowedFormats []string) error {
	// Check file size
	if file.Size > MaxFileSize {
		return fmt.Errorf("file size exceeds maximum allowed size of %d bytes", MaxFileSize)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowed := false
	for _, format := range allowedFormats {
		if ext == format {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("file format not allowed. Allowed formats: %v", allowedFormats)
	}

	return nil
}

// SaveFile saves an uploaded file to the specified directory
func SaveFile(file *multipart.FileHeader, uploadDir string) (*FileInfo, error) {
	// Generate unique filename
	ext := filepath.Ext(file.Filename)
	uniqueName := fmt.Sprintf("%s_%s%s",
		time.Now().Format("20060102_150405"),
		uuid.New().String()[:8],
		ext)

	filePath := filepath.Join(uploadDir, uniqueName)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %v", err)
	}

	// Save the file
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %v", err)
	}

	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %v", err)
	}
	defer src.Close()

	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %v", err)
	}
	defer dst.Close()

	// Copy file content
	if _, err := dst.ReadFrom(src); err != nil {
		return nil, fmt.Errorf("failed to save file: %v", err)
	}

	return &FileInfo{
		OriginalName: file.Filename,
		StoredName:   uniqueName,
		FilePath:     filePath,
		FileType:     ext,
		Size:         file.Size,
	}, nil
}

// CleanupFile removes a file from the filesystem
func CleanupFile(filePath string) error {
	return os.Remove(filePath)
}

// GetFileType determines if a file is video or audio based on extension
func GetFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	for _, format := range AllowedVideoFormats {
		if ext == format {
			return "video"
		}
	}

	for _, format := range AllowedAudioFormats {
		if ext == format {
			return "audio"
		}
	}

	return "unknown"
}

// ValidateVideoFile validates a video file
func ValidateVideoFile(file *multipart.FileHeader) error {
	return ValidateFile(file, AllowedVideoFormats)
}

// ValidateAudioFile validates an audio file
func ValidateAudioFile(file *multipart.FileHeader) error {
	return ValidateFile(file, AllowedAudioFormats)
}
