package executor_test

import (
	"strings"
	"testing"

	"github.com/GoCodeAlone/workflow-plugin-tofu/internal/executor"
)

func TestNewTofuExecutor_MissingBinary(t *testing.T) {
	_, err := executor.NewTofuExecutor("/nonexistent/tofu")
	if err == nil {
		t.Fatal("expected error for nonexistent binary path, got nil")
	}
}

func TestNewTofuExecutor_NotInPath(t *testing.T) {
	// Override PATH to guarantee tofu is not found.
	t.Setenv("PATH", "")
	_, err := executor.NewTofuExecutor("")
	if err == nil {
		t.Skip("tofu binary found; skipping not-in-path test")
	}
	if !strings.Contains(err.Error(), "tofu binary not found") {
		t.Errorf("expected 'tofu binary not found' in error, got: %v", err)
	}
}

func TestRunResult_ResourceIDs(t *testing.T) {
	stdout := `
Terraform will perform the following actions:

  # aws_db_instance.mydb will be created
  + resource "aws_db_instance" "mydb" {

Plan: 1 to add, 0 to change, 0 to destroy.
aws_db_instance.mydb: Creating...
aws_db_instance.mydb: Still creating... [30s elapsed]
aws_db_instance.mydb: Creation complete after 3m2s [id=mydb-abc123]

Apply complete! Resources: 1 added, 0 changed, 0 destroyed.
`
	result := &executor.RunResult{Stdout: stdout}
	ids := result.ResourceIDs()

	if len(ids) == 0 {
		t.Fatal("expected at least one resource ID to be parsed")
	}
	var found bool
	for k, v := range ids {
		if strings.Contains(k, "mydb") && v == "mydb-abc123" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected mydb-abc123 in resource IDs, got: %v", ids)
	}
}

func TestRunResult_ResourceIDs_Empty(t *testing.T) {
	result := &executor.RunResult{Stdout: "No changes. Infrastructure is up-to-date."}
	ids := result.ResourceIDs()
	if len(ids) != 0 {
		t.Errorf("expected empty resource IDs, got: %v", ids)
	}
}

func TestRunResult_ResourceIDs_Multiple(t *testing.T) {
	stdout := `
aws_vpc.network: Creation complete after 2s [id=vpc-abc]
aws_db_instance.db: Creation complete after 3m [id=db-xyz]
`
	result := &executor.RunResult{Stdout: stdout}
	ids := result.ResourceIDs()
	if len(ids) < 2 {
		t.Errorf("expected 2 resource IDs, got %d: %v", len(ids), ids)
	}
}

// MockExecutor is a test double for executor tests that don't need a real binary.
type MockExecutor struct {
	InitFn   func() (*executor.RunResult, error)
	PlanFn   func() (*executor.RunResult, error)
	ApplyFn  func() (*executor.RunResult, error)
	DestroyFn func() (*executor.RunResult, error)
}

func (m *MockExecutor) successResult(cmd string) *executor.RunResult {
	return &executor.RunResult{
		Command:  cmd,
		Stdout:   "Apply complete! Resources: 1 added.",
		ExitCode: 0,
	}
}

func TestMockExecutor_Interface(t *testing.T) {
	// Verify RunResult fields are accessible.
	r := &executor.RunResult{
		Command:  "tofu plan",
		Stdout:   "output",
		Stderr:   "",
		ExitCode: 0,
	}
	if r.Command == "" {
		t.Error("Command field should not be empty")
	}
	if r.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", r.ExitCode)
	}
}
