package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"strings"
	"time"
)

type SeaweedFS struct {
	filerURL string
	client   *http.Client
}

func NewSeaweedFS(filerURL string) *SeaweedFS {
	return &SeaweedFS{
		filerURL: strings.TrimRight(filerURL, "/"),
		client:   &http.Client{Timeout: 5 * time.Minute},
	}
}

func (s *SeaweedFS) Upload(ctx context.Context, filePath string, r io.Reader, contentType string) error {
	url := s.filerURL + "/" + strings.TrimLeft(filePath, "/")

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", path.Base(filePath))
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, r); err != nil {
		return fmt.Errorf("copy file data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("upload to seaweedfs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("seaweedfs upload failed: status=%d body=%s", resp.StatusCode, string(respBody))
	}
	return nil
}

func (s *SeaweedFS) Download(ctx context.Context, filePath string) (io.ReadCloser, error) {
	url := s.filerURL + "/" + strings.TrimLeft(filePath, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download from seaweedfs: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, fmt.Errorf("file not found: %s", filePath)
	}
	if resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("seaweedfs download failed: status=%d", resp.StatusCode)
	}
	return resp.Body, nil
}

func (s *SeaweedFS) Delete(ctx context.Context, filePath string) error {
	url := s.filerURL + "/" + strings.TrimLeft(filePath, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("create delete request: %w", err)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("delete from seaweedfs: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("seaweedfs delete failed: status=%d", resp.StatusCode)
	}
	return nil
}
