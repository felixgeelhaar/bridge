package mcp_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/felixgeelhaar/bolt"
	"github.com/felixgeelhaar/bridge/internal/infrastructure/mcp"
)

func newTestLogger() *bolt.Logger {
	handler := bolt.NewJSONHandler(io.Discard)
	return bolt.New(handler).SetLevel(bolt.ERROR)
}

func TestDefaultConfig(t *testing.T) {
	cfg := mcp.DefaultConfig()

	if cfg.Name != "bridge-tools" {
		t.Errorf("expected name 'bridge-tools', got %s", cfg.Name)
	}

	if cfg.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %s", cfg.Version)
	}

	if len(cfg.AllowedDirs) != 1 || cfg.AllowedDirs[0] != "." {
		t.Errorf("expected AllowedDirs [.], got %v", cfg.AllowedDirs)
	}

	if cfg.AllowShell {
		t.Error("expected AllowShell to be false by default")
	}
}

func TestNewServer(t *testing.T) {
	logger := newTestLogger()
	cfg := mcp.DefaultConfig()

	server := mcp.NewServer(logger, cfg)
	if server == nil {
		t.Fatal("expected non-nil server")
	}

	mcpSrv := server.GetMCPServer()
	if mcpSrv == nil {
		t.Error("expected non-nil MCP server")
	}
}

func TestNewServer_WithShellEnabled(t *testing.T) {
	logger := newTestLogger()
	cfg := mcp.Config{
		Name:        "test-server",
		Version:     "0.1.0",
		AllowedDirs: []string{"/tmp"},
		AllowShell:  true,
	}

	server := mcp.NewServer(logger, cfg)
	if server == nil {
		t.Fatal("expected non-nil server")
	}
}

func TestToolResult(t *testing.T) {
	result := mcp.ToolResult{
		Success: true,
		Data:    "test data",
	}

	if !result.Success {
		t.Error("expected success to be true")
	}

	if result.Data != "test data" {
		t.Errorf("expected data 'test data', got %v", result.Data)
	}

	if result.Error != "" {
		t.Errorf("expected empty error, got %s", result.Error)
	}
}

func TestToolResult_WithError(t *testing.T) {
	result := mcp.ToolResult{
		Success: false,
		Error:   "something went wrong",
	}

	if result.Success {
		t.Error("expected success to be false")
	}

	if result.Error != "something went wrong" {
		t.Errorf("expected error 'something went wrong', got %s", result.Error)
	}
}

func TestFileReadInput(t *testing.T) {
	input := mcp.FileReadInput{
		Path: "/path/to/file.txt",
	}

	if input.Path != "/path/to/file.txt" {
		t.Errorf("expected path '/path/to/file.txt', got %s", input.Path)
	}
}

func TestFileWriteInput(t *testing.T) {
	input := mcp.FileWriteInput{
		Path:    "/path/to/file.txt",
		Content: "test content",
	}

	if input.Path != "/path/to/file.txt" {
		t.Errorf("expected path '/path/to/file.txt', got %s", input.Path)
	}

	if input.Content != "test content" {
		t.Errorf("expected content 'test content', got %s", input.Content)
	}
}

func TestFileListInput(t *testing.T) {
	input := mcp.FileListInput{
		Path: "/path/to/dir",
	}

	if input.Path != "/path/to/dir" {
		t.Errorf("expected path '/path/to/dir', got %s", input.Path)
	}

	// Test empty path
	emptyInput := mcp.FileListInput{}
	if emptyInput.Path != "" {
		t.Errorf("expected empty path, got %s", emptyInput.Path)
	}
}

func TestGitDiffInput(t *testing.T) {
	input := mcp.GitDiffInput{
		Staged: true,
	}

	if !input.Staged {
		t.Error("expected Staged to be true")
	}
}

func TestGitLogInput(t *testing.T) {
	input := mcp.GitLogInput{
		Count: 5,
	}

	if input.Count != 5 {
		t.Errorf("expected count 5, got %d", input.Count)
	}
}

func TestShellExecInput(t *testing.T) {
	input := mcp.ShellExecInput{
		Command: "echo hello",
		Workdir: "/tmp",
	}

	if input.Command != "echo hello" {
		t.Errorf("expected command 'echo hello', got %s", input.Command)
	}

	if input.Workdir != "/tmp" {
		t.Errorf("expected workdir '/tmp', got %s", input.Workdir)
	}
}

func TestEmptyInput(t *testing.T) {
	_ = mcp.EmptyInput{}
}

func TestConfig_Fields(t *testing.T) {
	cfg := mcp.Config{
		Name:        "custom-server",
		Version:     "2.0.0",
		AllowedDirs: []string{"/home", "/tmp"},
		AllowShell:  true,
	}

	if cfg.Name != "custom-server" {
		t.Errorf("expected name 'custom-server', got %s", cfg.Name)
	}

	if cfg.Version != "2.0.0" {
		t.Errorf("expected version '2.0.0', got %s", cfg.Version)
	}

	if len(cfg.AllowedDirs) != 2 {
		t.Errorf("expected 2 allowed dirs, got %d", len(cfg.AllowedDirs))
	}

	if !cfg.AllowShell {
		t.Error("expected AllowShell to be true")
	}
}

func BenchmarkNewServer(b *testing.B) {
	handler := bolt.NewJSONHandler(&bytes.Buffer{})
	logger := bolt.New(handler).SetLevel(bolt.ERROR)
	cfg := mcp.DefaultConfig()

	for i := 0; i < b.N; i++ {
		mcp.NewServer(logger, cfg)
	}
}
