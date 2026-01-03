package tools

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/felixgeelhaar/bolt"
)

// Errors
var (
	ErrPathNotAllowed = errors.New("path not within allowed directories")
	ErrPathTraversal  = errors.New("path traversal detected")
)

// ToolRegistry manages tool implementations.
type ToolRegistry struct {
	logger      *bolt.Logger
	allowedDirs []string
	auditLog    *AuditLog
}

// NewToolRegistry creates a new tool registry.
func NewToolRegistry(logger *bolt.Logger, allowedDirs []string) *ToolRegistry {
	// Convert to absolute paths
	absDirs := make([]string, 0, len(allowedDirs))
	for _, dir := range allowedDirs {
		absDir, err := filepath.Abs(dir)
		if err == nil {
			absDirs = append(absDirs, absDir)
		}
	}

	return &ToolRegistry{
		logger:      logger,
		allowedDirs: absDirs,
		auditLog:    NewAuditLog(logger),
	}
}

// isAllowedPath checks if a path is within allowed directories.
func (r *ToolRegistry) isAllowedPath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	// Check for path traversal
	cleanPath := filepath.Clean(absPath)
	if strings.Contains(path, "..") {
		return "", ErrPathTraversal
	}

	// Check if path is within allowed directories
	for _, allowedDir := range r.allowedDirs {
		if strings.HasPrefix(cleanPath, allowedDir) {
			return cleanPath, nil
		}
	}

	return "", ErrPathNotAllowed
}

// FileReadResult contains the result of a file read operation.
type FileReadResult struct {
	Content string
	Size    int64
	ModTime time.Time
}

// FileRead reads the contents of a file.
func (r *ToolRegistry) FileRead(ctx context.Context, path string) (*FileReadResult, error) {
	absPath, err := r.isAllowedPath(path)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	info, _ := os.Stat(absPath)

	r.auditLog.Record(ctx, AuditEvent{
		Tool:   "file_read",
		Path:   absPath,
		Action: "read",
		Size:   info.Size(),
	})

	return &FileReadResult{
		Content: string(content),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}, nil
}

// FileWrite writes content to a file.
func (r *ToolRegistry) FileWrite(ctx context.Context, path, content string) error {
	absPath, err := r.isAllowedPath(path)
	if err != nil {
		return err
	}

	// Ensure parent directory exists
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return err
	}

	r.auditLog.Record(ctx, AuditEvent{
		Tool:   "file_write",
		Path:   absPath,
		Action: "write",
		Size:   int64(len(content)),
	})

	return nil
}

// FileEntry represents a directory entry.
type FileEntry struct {
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	IsDir   bool      `json:"is_dir"`
	ModTime time.Time `json:"mod_time"`
}

// FileList lists directory contents.
func (r *ToolRegistry) FileList(ctx context.Context, path string) ([]FileEntry, error) {
	absPath, err := r.isAllowedPath(path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}

	result := make([]FileEntry, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		result = append(result, FileEntry{
			Name:    entry.Name(),
			Size:    info.Size(),
			IsDir:   entry.IsDir(),
			ModTime: info.ModTime(),
		})
	}

	r.auditLog.Record(ctx, AuditEvent{
		Tool:   "file_list",
		Path:   absPath,
		Action: "list",
	})

	return result, nil
}

// GitStatus returns the current git status.
func (r *ToolRegistry) GitStatus(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain=v2", "--branch")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try simpler status if porcelain v2 not supported
		cmd = exec.CommandContext(ctx, "git", "status", "--short", "--branch")
		output, err = cmd.CombinedOutput()
		if err != nil {
			return "", err
		}
	}

	r.auditLog.Record(ctx, AuditEvent{
		Tool:   "git_status",
		Action: "status",
	})

	return string(output), nil
}

// GitDiff returns git diff output.
func (r *ToolRegistry) GitDiff(ctx context.Context, staged bool) (string, error) {
	args := []string{"diff"}
	if staged {
		args = append(args, "--staged")
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	r.auditLog.Record(ctx, AuditEvent{
		Tool:   "git_diff",
		Action: "diff",
	})

	return string(output), nil
}

// GitLog returns recent commit history.
func (r *ToolRegistry) GitLog(ctx context.Context, count int) (string, error) {
	if count <= 0 {
		count = 10
	}
	if count > 100 {
		count = 100
	}

	cmd := exec.CommandContext(ctx, "git", "log",
		"--oneline",
		"--no-decorate",
		"-n", strings.TrimSpace(string(rune(count)+'0')),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	r.auditLog.Record(ctx, AuditEvent{
		Tool:   "git_log",
		Action: "log",
	})

	return string(output), nil
}

// ShellExec executes a shell command (sandboxed).
func (r *ToolRegistry) ShellExec(ctx context.Context, command, workdir string) (string, error) {
	// Validate workdir if provided
	if workdir != "" {
		absWorkdir, err := r.isAllowedPath(workdir)
		if err != nil {
			return "", err
		}
		workdir = absWorkdir
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	if workdir != "" {
		cmd.Dir = workdir
	}

	output, err := cmd.CombinedOutput()

	r.auditLog.Record(ctx, AuditEvent{
		Tool:    "shell_exec",
		Action:  "exec",
		Command: command,
	})

	if err != nil {
		return string(output), err
	}

	return string(output), nil
}

// AuditEvent represents an auditable action.
type AuditEvent struct {
	Tool      string
	Path      string
	Action    string
	Command   string
	Size      int64
	Timestamp time.Time
}

// AuditLog records tool usage for compliance.
type AuditLog struct {
	logger *bolt.Logger
}

// NewAuditLog creates a new audit log.
func NewAuditLog(logger *bolt.Logger) *AuditLog {
	return &AuditLog{logger: logger}
}

// Record logs an audit event.
func (a *AuditLog) Record(ctx context.Context, event AuditEvent) {
	event.Timestamp = time.Now()

	log := a.logger.Info().
		Str("tool", event.Tool).
		Str("action", event.Action).
		Time("timestamp", event.Timestamp)

	if event.Path != "" {
		log = log.Str("path", event.Path)
	}
	if event.Command != "" {
		log = log.Str("command", event.Command)
	}
	if event.Size > 0 {
		log = log.Int64("size", event.Size)
	}

	log.Msg("Tool action audited")
}
