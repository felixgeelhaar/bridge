package tools_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/bolt"
	"github.com/felixgeelhaar/bridge/internal/infrastructure/mcp/tools"
)

func newTestLogger() *bolt.Logger {
	handler := bolt.NewJSONHandler(io.Discard)
	return bolt.New(handler).SetLevel(bolt.ERROR)
}

func TestNewToolRegistry(t *testing.T) {
	logger := newTestLogger()
	registry := tools.NewToolRegistry(logger, []string{"."})

	if registry == nil {
		t.Fatal("expected non-nil registry")
	}
}

func TestNewToolRegistry_WithMultipleDirs(t *testing.T) {
	logger := newTestLogger()
	registry := tools.NewToolRegistry(logger, []string{"/tmp", ".", "/home"})

	if registry == nil {
		t.Fatal("expected non-nil registry")
	}
}

func TestToolRegistry_FileRead(t *testing.T) {
	// Create a temp directory and file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "hello world"

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	logger := newTestLogger()
	registry := tools.NewToolRegistry(logger, []string{tmpDir})

	result, err := registry.FileRead(context.Background(), testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if result.Content != content {
		t.Errorf("expected content %q, got %q", content, result.Content)
	}

	if result.Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), result.Size)
	}
}

func TestToolRegistry_FileRead_NotAllowed(t *testing.T) {
	logger := newTestLogger()
	registry := tools.NewToolRegistry(logger, []string{"/tmp"})

	_, err := registry.FileRead(context.Background(), "/etc/passwd")
	if err == nil {
		t.Error("expected error for path not allowed")
	}

	if err != tools.ErrPathNotAllowed {
		t.Errorf("expected ErrPathNotAllowed, got %v", err)
	}
}

func TestToolRegistry_FileRead_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	logger := newTestLogger()
	registry := tools.NewToolRegistry(logger, []string{tmpDir})

	_, err := registry.FileRead(context.Background(), filepath.Join(tmpDir, "../../../etc/passwd"))
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestToolRegistry_FileWrite(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "subdir", "test.txt")
	content := "new content"

	logger := newTestLogger()
	registry := tools.NewToolRegistry(logger, []string{tmpDir})

	err := registry.FileWrite(context.Background(), testFile, content)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Verify content was written
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}

	if string(data) != content {
		t.Errorf("expected content %q, got %q", content, string(data))
	}
}

func TestToolRegistry_FileWrite_NotAllowed(t *testing.T) {
	logger := newTestLogger()
	registry := tools.NewToolRegistry(logger, []string{"/tmp"})

	err := registry.FileWrite(context.Background(), "/root/test.txt", "content")
	if err == nil {
		t.Error("expected error for path not allowed")
	}
}

func TestToolRegistry_FileList(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some test files
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content2"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	logger := newTestLogger()
	registry := tools.NewToolRegistry(logger, []string{tmpDir})

	entries, err := registry.FileList(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("failed to list directory: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}

	// Check we have the expected entries
	hasFile1 := false
	hasFile2 := false
	hasSubdir := false
	for _, entry := range entries {
		switch entry.Name {
		case "file1.txt":
			hasFile1 = true
			if entry.IsDir {
				t.Error("file1.txt should not be a directory")
			}
		case "file2.txt":
			hasFile2 = true
		case "subdir":
			hasSubdir = true
			if !entry.IsDir {
				t.Error("subdir should be a directory")
			}
		}
	}

	if !hasFile1 || !hasFile2 || !hasSubdir {
		t.Error("missing expected entries")
	}
}

func TestToolRegistry_FileList_NotAllowed(t *testing.T) {
	logger := newTestLogger()
	registry := tools.NewToolRegistry(logger, []string{"/tmp"})

	_, err := registry.FileList(context.Background(), "/root")
	if err == nil {
		t.Error("expected error for path not allowed")
	}
}

func TestFileReadResult(t *testing.T) {
	result := tools.FileReadResult{
		Content: "test content",
		Size:    12,
		ModTime: time.Now(),
	}

	if result.Content != "test content" {
		t.Errorf("expected content 'test content', got %s", result.Content)
	}

	if result.Size != 12 {
		t.Errorf("expected size 12, got %d", result.Size)
	}
}

func TestFileEntry(t *testing.T) {
	now := time.Now()
	entry := tools.FileEntry{
		Name:    "test.txt",
		Size:    100,
		IsDir:   false,
		ModTime: now,
	}

	if entry.Name != "test.txt" {
		t.Errorf("expected name 'test.txt', got %s", entry.Name)
	}

	if entry.Size != 100 {
		t.Errorf("expected size 100, got %d", entry.Size)
	}

	if entry.IsDir {
		t.Error("expected IsDir to be false")
	}
}

func TestAuditEvent(t *testing.T) {
	event := tools.AuditEvent{
		Tool:      "file_read",
		Path:      "/tmp/test.txt",
		Action:    "read",
		Size:      100,
		Timestamp: time.Now(),
	}

	if event.Tool != "file_read" {
		t.Errorf("expected tool 'file_read', got %s", event.Tool)
	}

	if event.Path != "/tmp/test.txt" {
		t.Errorf("expected path '/tmp/test.txt', got %s", event.Path)
	}
}

func TestNewAuditLog(t *testing.T) {
	logger := newTestLogger()
	auditLog := tools.NewAuditLog(logger)

	if auditLog == nil {
		t.Fatal("expected non-nil audit log")
	}
}

func TestAuditLog_Record(t *testing.T) {
	logger := newTestLogger()
	auditLog := tools.NewAuditLog(logger)

	// Test with all fields
	auditLog.Record(context.Background(), tools.AuditEvent{
		Tool:    "shell_exec",
		Path:    "/tmp",
		Action:  "exec",
		Command: "ls -la",
		Size:    0,
	})

	// Test with minimal fields
	auditLog.Record(context.Background(), tools.AuditEvent{
		Tool:   "git_status",
		Action: "status",
	})

	// Test with path only
	auditLog.Record(context.Background(), tools.AuditEvent{
		Tool:   "file_read",
		Action: "read",
		Path:   "/tmp/test.txt",
		Size:   100,
	})
}

func TestErrors(t *testing.T) {
	if tools.ErrPathNotAllowed.Error() != "path not within allowed directories" {
		t.Errorf("unexpected error message: %s", tools.ErrPathNotAllowed.Error())
	}

	if tools.ErrPathTraversal.Error() != "path traversal detected" {
		t.Errorf("unexpected error message: %s", tools.ErrPathTraversal.Error())
	}
}

func BenchmarkFileRead(b *testing.B) {
	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("benchmark content"), 0644)

	logger := newTestLogger()
	registry := tools.NewToolRegistry(logger, []string{tmpDir})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.FileRead(ctx, testFile)
	}
}

func BenchmarkFileList(b *testing.B) {
	tmpDir := b.TempDir()
	for i := 0; i < 10; i++ {
		os.WriteFile(filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt"), []byte("content"), 0644)
	}

	logger := newTestLogger()
	registry := tools.NewToolRegistry(logger, []string{tmpDir})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.FileList(ctx, tmpDir)
	}
}
