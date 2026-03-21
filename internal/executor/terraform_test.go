package executor_test

import (
	"strings"
	"testing"

	"github.com/GoCodeAlone/workflow-plugin-tofu/internal/executor"
)

func TestNewTerraformExecutor_MissingBinary(t *testing.T) {
	_, err := executor.NewTerraformExecutor("/nonexistent/terraform")
	if err == nil {
		t.Fatal("expected error for nonexistent binary path, got nil")
	}
}

func TestNewTerraformExecutor_NotInPath(t *testing.T) {
	t.Setenv("PATH", "")
	_, err := executor.NewTerraformExecutor("")
	if err == nil {
		t.Skip("terraform binary found; skipping not-in-path test")
	}
	if !strings.Contains(err.Error(), "terraform binary not found") {
		t.Errorf("expected 'terraform binary not found' in error, got: %v", err)
	}
}

func TestTerraformExecutor_ExplicitPath_NotFound(t *testing.T) {
	_, err := executor.NewTerraformExecutor("/absolutely/not/there/terraform")
	if err == nil {
		t.Fatal("expected error when explicit binary path does not exist")
	}
}
