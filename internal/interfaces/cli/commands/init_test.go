package commands_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/bridge/internal/interfaces/cli/commands"
	"github.com/urfave/cli/v2"
)

func TestInitCommand(t *testing.T) {
	cmd := commands.InitCommand()
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}

	if cmd.Name != "init" {
		t.Errorf("expected name 'init', got %s", cmd.Name)
	}

	if cmd.Usage == "" {
		t.Error("expected non-empty usage")
	}
}

func TestInitCommand_HasFlags(t *testing.T) {
	cmd := commands.InitCommand()

	if len(cmd.Flags) != 2 {
		t.Errorf("expected 2 flags, got %d", len(cmd.Flags))
	}
}

func TestRunInit(t *testing.T) {
	// Create a temp directory
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	app := &cli.App{
		Commands: []*cli.Command{commands.InitCommand()},
	}

	err := app.Run([]string{"test", "init"})
	if err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	// Verify created files
	if _, err := os.Stat(".bridge/config.yaml"); os.IsNotExist(err) {
		t.Error("config.yaml not created")
	}

	if _, err := os.Stat(".bridge/workflows/example.yaml"); os.IsNotExist(err) {
		t.Error("example workflow not created")
	}

	if _, err := os.Stat(".bridge/policies/default.rego"); os.IsNotExist(err) {
		t.Error("default policy not created")
	}
}

func TestRunInit_ForceOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// First init
	app := &cli.App{
		Commands: []*cli.Command{commands.InitCommand()},
	}

	err := app.Run([]string{"test", "init"})
	if err != nil {
		t.Fatalf("first init failed: %v", err)
	}

	// Second init without force should fail
	err = app.Run([]string{"test", "init"})
	if err == nil {
		t.Error("expected error without --force flag")
	}

	// Third init with force should succeed
	err = app.Run([]string{"test", "init", "--force"})
	if err != nil {
		t.Fatalf("init with --force failed: %v", err)
	}
}

func TestRunInit_PRReviewTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	app := &cli.App{
		Commands: []*cli.Command{commands.InitCommand()},
	}

	err := app.Run([]string{"test", "init", "--template", "pr-review"})
	if err != nil {
		t.Fatalf("init with pr-review template failed: %v", err)
	}

	// Verify workflow file was created
	content, err := os.ReadFile(".bridge/workflows/example.yaml")
	if err != nil {
		t.Fatalf("failed to read workflow file: %v", err)
	}

	if len(content) == 0 {
		t.Error("expected non-empty workflow file")
	}
}

func TestValidateCommand(t *testing.T) {
	cmd := commands.ValidateCommand()
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}

	if cmd.Name != "validate" {
		t.Errorf("expected name 'validate', got %s", cmd.Name)
	}
}

func TestRunCommand(t *testing.T) {
	cmd := commands.RunCommand()
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}

	if cmd.Name != "run" {
		t.Errorf("expected name 'run', got %s", cmd.Name)
	}
}

func TestStatusCommand(t *testing.T) {
	cmd := commands.StatusCommand()
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}

	if cmd.Name != "status" {
		t.Errorf("expected name 'status', got %s", cmd.Name)
	}
}

func TestApproveCommand(t *testing.T) {
	cmd := commands.ApproveCommand()
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}

	if cmd.Name != "approve" {
		t.Errorf("expected name 'approve', got %s", cmd.Name)
	}
}

func TestValidateCommand_HasFlags(t *testing.T) {
	cmd := commands.ValidateCommand()

	// Should have workflow flag
	hasWorkflowFlag := false
	for _, flag := range cmd.Flags {
		if flag.Names()[0] == "workflow" {
			hasWorkflowFlag = true
			break
		}
	}

	if !hasWorkflowFlag {
		t.Error("expected workflow flag")
	}
}

func TestRunCommand_HasFlags(t *testing.T) {
	cmd := commands.RunCommand()

	// Should have workflow and dry-run flags
	hasWorkflowFlag := false
	hasDryRunFlag := false
	for _, flag := range cmd.Flags {
		switch flag.Names()[0] {
		case "workflow":
			hasWorkflowFlag = true
		case "dry-run":
			hasDryRunFlag = true
		}
	}

	if !hasWorkflowFlag {
		t.Error("expected workflow flag")
	}
	if !hasDryRunFlag {
		t.Error("expected dry-run flag")
	}
}

func TestConfigTemplateContent(t *testing.T) {
	// Run init to generate templates
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	app := &cli.App{
		Commands: []*cli.Command{commands.InitCommand()},
	}

	err := app.Run([]string{"test", "init"})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Read config
	configContent, err := os.ReadFile(filepath.Join(".bridge", "config.yaml"))
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	// Verify key sections exist
	config := string(configContent)
	sections := []string{"providers:", "anthropic:", "governance:", "audit:"}
	for _, section := range sections {
		if !contains(config, section) {
			t.Errorf("config missing section: %s", section)
		}
	}
}

func TestDefaultPolicyContent(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	app := &cli.App{
		Commands: []*cli.Command{commands.InitCommand()},
	}

	err := app.Run([]string{"test", "init"})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Read policy
	policyContent, err := os.ReadFile(filepath.Join(".bridge", "policies", "default.rego"))
	if err != nil {
		t.Fatalf("failed to read policy: %v", err)
	}

	// Verify key rules exist
	policy := string(policyContent)
	rules := []string{"package bridge.policy", "default allowed = true", "requires_approval"}
	for _, rule := range rules {
		if !contains(policy, rule) {
			t.Errorf("policy missing rule: %s", rule)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
