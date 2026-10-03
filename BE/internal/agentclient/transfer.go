package agentclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

const MaxUpload = 10 << 20
const MaxTransfer = 64 << 20

type uploadInput struct {
	Filename string `json:"filename" validate:"required"`
	Base64   string `json:"base64" validate:"required"`
}

func (c *Client) Upload(ctx context.Context, filename string, reader io.Reader, max int64) (any, error) {
	if filepath.Base(filename) != filename || filename == "." || filename == "" {
		return nil, Err("INVALID_INPUT", "A filename without a path is required", 0)
	}
	raw, err := io.ReadAll(io.LimitReader(reader, max+1))
	if err != nil {
		return nil, Err("IO_ERROR", "Cannot read upload", 0)
	}
	if int64(len(raw)) > max {
		return nil, Err("INPUT_TOO_LARGE", "Upload exceeds supported size", 0)
	}
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, err := form.CreateFormFile("file", filename)
	if err != nil {
		return nil, Err("INVALID_INPUT", "Invalid upload filename", 0)
	}
	_, _ = part.Write(raw)
	_ = form.Close()
	u := *c.base
	u.Path = "/api/workspaces/" + c.workspace + "/upload"
	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), &body)
	if err != nil {
		return nil, Err("INVALID_INPUT", "Invalid upload request", 0)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", form.FormDataContentType())
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, Err("TRANSPORT_ERROR", "Upload request failed", 0)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, httpError(resp.StatusCode)
	}
	resultRaw, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponse+1))
	if err != nil || len(resultRaw) > MaxResponse {
		return nil, Err("RESPONSE_TOO_LARGE", "Invalid upload response size", 0)
	}
	var result any
	decoder := json.NewDecoder(bytes.NewReader(resultRaw))
	decoder.UseNumber()
	if decoder.Decode(&result) != nil || decoder.Decode(new(any)) != io.EOF {
		return nil, Err("INVALID_RESPONSE", "Invalid upload response", 0)
	}
	return project(result, true, nil, c.token), nil
}
func (c *Client) uploadBase64(ctx context.Context, input map[string]any) (any, error) {
	body := input["body"].(map[string]any)
	raw, err := base64.StdEncoding.DecodeString(body["base64"].(string))
	if err != nil {
		return nil, Err("INVALID_INPUT", "Invalid base64 file", 0)
	}
	return c.Upload(ctx, body["filename"].(string), bytes.NewReader(raw), 1<<20)
}

// Export writes a bounded archive to a new file. Partial failures remove that file.
func (c *Client) Export(ctx context.Context, path string) (any, error) {
	if path == "" {
		return nil, Err("INVALID_INPUT", "An output file is required", 0)
	}
	u := *c.base
	u.Path = "/api/workspaces/" + c.workspace + "/export"
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, Err("INVALID_INPUT", "Invalid export request", 0)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/zip")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, Err("TRANSPORT_ERROR", "Export request failed", 0)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, httpError(resp.StatusCode)
	}
	if resp.Header.Get("Content-Type") != "application/zip" {
		return nil, Err("INVALID_RESPONSE", "Export must return a ZIP archive", 0)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, Err("IO_ERROR", "Cannot create output file; existing files are not replaced", 0)
	}
	complete := false
	defer func() {
		_ = file.Close()
		if !complete {
			_ = os.Remove(path)
		}
	}()
	count, err := io.Copy(file, io.LimitReader(resp.Body, MaxTransfer+1))
	if err != nil {
		return nil, Err("TRANSPORT_ERROR", "Export download failed", 0)
	}
	if count > MaxTransfer {
		return nil, Err("RESPONSE_TOO_LARGE", "Export exceeds 64 MiB; use the web transfer flow", 0)
	}
	if err := file.Close(); err != nil {
		return nil, Err("IO_ERROR", "Cannot complete output file", 0)
	}
	complete = true
	return map[string]any{"file": path, "bytes": count, "content_type": "application/zip"}, nil
}
