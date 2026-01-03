package mcp

import (
	"github.com/felixgeelhaar/bolt"
	"github.com/felixgeelhaar/bridge/internal/infrastructure/mcp/tools"
)

// ToolRegistry wraps the tools package registry.
type ToolRegistry = tools.ToolRegistry

// NewToolRegistry creates a new tool registry.
func NewToolRegistry(logger *bolt.Logger, allowedDirs []string) *ToolRegistry {
	return tools.NewToolRegistry(logger, allowedDirs)
}

// Re-export types
type (
	FileReadResult = tools.FileReadResult
	FileEntry      = tools.FileEntry
	AuditEvent     = tools.AuditEvent
	AuditLog       = tools.AuditLog
)
