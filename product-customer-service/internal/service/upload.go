package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"

	api "github.com/linhhuynhcoding/jss-microservices/rpc/gen/product"
	"go.uber.org/zap"
)

func (s *Service) UploadFile(ctx context.Context, req *api.UploadFileRequest) (*api.UploadFileResponse, error) {
	s.logger.With(zap.String("func", "UploadFile"))

	// Generate a unique file ID
	fileID := generateFileID()

	// Create uploads directory if it doesn't exist
	uploadDir := s.cfg.UploadFolder
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %v", err)
	}

	// Create file path with unique ID
	filename := fmt.Sprintf("%s_%s", fileID, req.Filename)
	filePath := filepath.Join(uploadDir, filename)

	// Write file to disk
	if err := os.WriteFile(filePath, req.FileData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %v", err)
	}

	log.Printf("File uploaded successfully: %s, Size: %d bytes, Type: %s",
		req.Filename, req.FileSize, req.ContentType)

	return &api.UploadFileResponse{
		Message:  "File uploaded successfully",
		FileId:   fileID,
		Filename: req.Filename,
		FileSize: req.FileSize,
	}, nil
}

func generateFileID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
