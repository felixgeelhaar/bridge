package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/felixgeelhaar/bolt"
	mcplib "github.com/felixgeelhaar/mcp-go"
)

// Server wraps the MCP server for Bridge tool bindings.
type Server struct {
	mcpServer *mcplib.Server
	logger    *bolt.Logger
	tools     *ToolRegistry
	config    Config
}

// Config contains MCP server configuration.
type Config struct {
	Name        string
	Version     string
	AllowedDirs []string // Directories allowed for file operations
	AllowShell  bool     // Whether to allow shell execution
}

// DefaultConfig returns the default MCP server configuration.
func DefaultConfig() Config {
	return Config{
		Name:        "bridge-tools",
		Version:     "1.0.0",
		AllowedDirs: []string{"."},
		AllowShell:  false,
	}
}

// Input types for tools

// FileReadInput is the input for file_read tool.
type FileReadInput struct {
	Path string `json:"path" jsonschema:"required,description=Path to the file to read"`
}

// FileWriteInput is the input for file_write tool.
type FileWriteInput struct {
	Path    string `json:"path" jsonschema:"required,description=Path to the file to write"`
	Content string `json:"content" jsonschema:"required,description=Content to write to the file"`
}

// FileListInput is the input for file_list tool.
type FileListInput struct {
	Path string `json:"path,omitempty" jsonschema:"description=Directory path to list (defaults to current directory)"`
}

// GitDiffInput is the input for git_diff tool.
type GitDiffInput struct {
	Staged bool `json:"staged,omitempty" jsonschema:"description=Show staged changes only"`
}

// GitLogInput is the input for git_log tool.
type GitLogInput struct {
	Count int `json:"count,omitempty" jsonschema:"description=Number of commits to show (default 10)"`
}

// ShellExecInput is the input for shell_exec tool.
type ShellExecInput struct {
	Command string `json:"command" jsonschema:"required,description=Shell command to execute"`
	Workdir string `json:"workdir,omitempty" jsonschema:"description=Working directory for command execution"`
}

// EmptyInput is used for tools with no input.
type EmptyInput struct{}

// ToolResult is a generic result structure.
type ToolResult struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// NewServer creates a new MCP server.
func NewServer(logger *bolt.Logger, cfg Config) *Server {
	mcpServer := mcplib.NewServer(mcplib.ServerInfo{
		Name:    cfg.Name,
		Version: cfg.Version,
		Capabilities: mcplib.Capabilities{
			Tools:     true,
			Resources: true,
		},
	}, mcplib.WithInstructions(`
Bridge provides tools for AI-assisted software development workflows.

Available tools:
- file_read: Read file contents
- file_write: Write content to a file
- file_list: List directory contents
- git_status: Get git repository status
- git_diff: Get git diff output
- git_log: Get recent commit history

All file operations are sandboxed to allowed directories.
All operations are audited for compliance.
`))

	srv := &Server{
		mcpServer: mcpServer,
		logger:    logger,
		tools:     NewToolRegistry(logger, cfg.AllowedDirs),
		config:    cfg,
	}

	srv.registerTools()
	srv.registerResources()

	return srv
}

// registerTools registers all available tools.
func (s *Server) registerTools() {
	// File operations
	s.mcpServer.Tool("file_read").
		Description("Read the contents of a file").
		Handler(func(ctx context.Context, input FileReadInput) (ToolResult, error) {
			result, err := s.tools.FileRead(ctx, input.Path)
			if err != nil {
				return ToolResult{Success: false, Error: err.Error()}, nil
			}
			return ToolResult{Success: true, Data: result.Content}, nil
		})

	s.mcpServer.Tool("file_write").
		Description("Write content to a file").
		Handler(func(ctx context.Context, input FileWriteInput) (ToolResult, error) {
			if err := s.tools.FileWrite(ctx, input.Path, input.Content); err != nil {
				return ToolResult{Success: false, Error: err.Error()}, nil
			}
			return ToolResult{
				Success: true,
				Data:    fmt.Sprintf("Successfully wrote %d bytes to %s", len(input.Content), input.Path),
			}, nil
		})

	s.mcpServer.Tool("file_list").
		Description("List directory contents").
		Handler(func(ctx context.Context, input FileListInput) (ToolResult, error) {
			path := input.Path
			if path == "" {
				path = "."
			}
			result, err := s.tools.FileList(ctx, path)
			if err != nil {
				return ToolResult{Success: false, Error: err.Error()}, nil
			}
			return ToolResult{Success: true, Data: result}, nil
		})

	// Git operations
	s.mcpServer.Tool("git_status").
		Description("Get the current git repository status").
		Handler(func(ctx context.Context, input EmptyInput) (ToolResult, error) {
			result, err := s.tools.GitStatus(ctx)
			if err != nil {
				return ToolResult{Success: false, Error: err.Error()}, nil
			}
			return ToolResult{Success: true, Data: result}, nil
		})

	s.mcpServer.Tool("git_diff").
		Description("Get git diff for staged or unstaged changes").
		Handler(func(ctx context.Context, input GitDiffInput) (ToolResult, error) {
			result, err := s.tools.GitDiff(ctx, input.Staged)
			if err != nil {
				return ToolResult{Success: false, Error: err.Error()}, nil
			}
			return ToolResult{Success: true, Data: result}, nil
		})

	s.mcpServer.Tool("git_log").
		Description("Get recent commit history").
		Handler(func(ctx context.Context, input GitLogInput) (ToolResult, error) {
			count := input.Count
			if count <= 0 {
				count = 10
			}
			result, err := s.tools.GitLog(ctx, count)
			if err != nil {
				return ToolResult{Success: false, Error: err.Error()}, nil
			}
			return ToolResult{Success: true, Data: result}, nil
		})

	// Shell execution (if enabled)
	if s.config.AllowShell {
		s.mcpServer.Tool("shell_exec").
			Description("Execute a shell command (sandboxed)").
			Handler(func(ctx context.Context, input ShellExecInput) (ToolResult, error) {
				result, err := s.tools.ShellExec(ctx, input.Command, input.Workdir)
				if err != nil {
					return ToolResult{Success: false, Error: err.Error()}, nil
				}
				return ToolResult{Success: true, Data: result}, nil
			})
	}
}

// registerResources registers available resources.
func (s *Server) registerResources() {
	// Workflow context resource
	s.mcpServer.Resource("bridge://workflow/context").
		Name("Workflow Context").
		Description("Current workflow execution context").
		MimeType("application/json").
		Handler(func(ctx context.Context, uri string, params map[string]string) (*mcplib.ResourceContent, error) {
			// Return current workflow context (placeholder)
			contextData := map[string]any{
				"workflow": "current",
				"step":     "active",
			}
			data, _ := json.Marshal(contextData)
			return &mcplib.ResourceContent{
				URI:      uri,
				MimeType: "application/json",
				Text:     string(data),
			}, nil
		})
}

// Start starts the MCP server with stdio transport.
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info().
		Str("name", s.config.Name).
		Str("version", s.config.Version).
		Msg("Starting MCP server")

	return mcplib.ServeStdio(ctx, s.mcpServer)
}

// GetMCPServer returns the underlying MCP server.
func (s *Server) GetMCPServer() *mcplib.Server {
	return s.mcpServer
}
