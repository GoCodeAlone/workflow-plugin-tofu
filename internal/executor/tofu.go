// Package executor shells out to the tofu or terraform binary.
package executor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// TofuExecutor runs OpenTofu commands in a working directory.
type TofuExecutor struct {
	BinaryPath string // path to the tofu binary; empty = auto-discover from PATH
}

// NewTofuExecutor creates an executor for the tofu binary.
// It resolves the binary path from PATH if not explicitly set.
// If an explicit path is given, it verifies the binary exists.
func NewTofuExecutor(binaryPath string) (*TofuExecutor, error) {
	if binaryPath == "" {
		path, err := exec.LookPath("tofu")
		if err != nil {
			return nil, fmt.Errorf("tofu binary not found in PATH: %w (install OpenTofu from https://opentofu.org)", err)
		}
		binaryPath = path
	} else {
		if _, err := exec.LookPath(binaryPath); err != nil {
			return nil, fmt.Errorf("tofu binary not found at %q: %w", binaryPath, err)
		}
	}
	return &TofuExecutor{BinaryPath: binaryPath}, nil
}

// Init runs `tofu init` in workDir.
func (e *TofuExecutor) Init(ctx context.Context, workDir string) (*RunResult, error) {
	return e.run(ctx, workDir, "init", "-input=false", "-no-color")
}

// Plan runs `tofu plan -out=plan.tfplan` and returns the result.
// varFile is optional; pass empty string to skip.
func (e *TofuExecutor) Plan(ctx context.Context, workDir, varFile string) (*RunResult, error) {
	args := []string{"plan", "-out=plan.tfplan", "-input=false", "-no-color"}
	if varFile != "" {
		args = append(args, "-var-file="+varFile)
	}
	return e.run(ctx, workDir, args...)
}

// Apply runs `tofu apply plan.tfplan` (or with a supplied planFile path).
func (e *TofuExecutor) Apply(ctx context.Context, workDir, planFile string) (*RunResult, error) {
	if planFile == "" {
		planFile = "plan.tfplan"
	}
	return e.run(ctx, workDir, "apply", "-input=false", "-no-color", planFile)
}

// Destroy runs `tofu destroy -auto-approve`.
func (e *TofuExecutor) Destroy(ctx context.Context, workDir string) (*RunResult, error) {
	return e.run(ctx, workDir, "destroy", "-auto-approve", "-input=false", "-no-color")
}

// run executes the tofu binary with given args in workDir.
func (e *TofuExecutor) run(ctx context.Context, workDir string, args ...string) (*RunResult, error) {
	cmd := exec.CommandContext(ctx, e.BinaryPath, args...)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := &RunResult{
		Command: e.BinaryPath + " " + strings.Join(args, " "),
		Stdout:  stdout.String(),
		Stderr:  stderr.String(),
		ExitCode: 0,
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		result.Error = err.Error()
		return result, fmt.Errorf("tofu %s: %w\n%s", args[0], err, stderr.String())
	}
	return result, nil
}

// RunResult is the output of a tofu/terraform command invocation.
type RunResult struct {
	Command  string
	Stdout   string
	Stderr   string
	ExitCode int
	Error    string
}

// ResourceIDs parses the stdout of a `tofu apply` run and extracts created resource IDs.
// This is a best-effort extraction based on the standard tofu output format.
func (r *RunResult) ResourceIDs() map[string]string {
	ids := make(map[string]string)
	for _, line := range strings.Split(r.Stdout, "\n") {
		// Lines like: "  + resource "aws_db_instance" "mydb" {"
		// or completed lines like: "aws_db_instance.mydb: Creation complete after 3m [id=mydb-xxxx]"
		if strings.Contains(line, ": Creation complete") && strings.Contains(line, "[id=") {
			parts := strings.SplitN(line, "[id=", 2)
			if len(parts) == 2 {
				resourceAddr := strings.TrimSpace(parts[0])
				resourceAddr = strings.TrimSuffix(resourceAddr, ": Creation complete after ")
				if idx := strings.Index(resourceAddr, ": Creation complete"); idx != -1 {
					resourceAddr = resourceAddr[:idx]
				}
				id := strings.TrimSuffix(parts[1], "]")
				ids[resourceAddr] = id
			}
		}
	}
	return ids
}
