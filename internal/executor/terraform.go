package executor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// TerraformExecutor runs Terraform (BSL) commands in a working directory.
// It requires the terraform binary to be installed externally.
type TerraformExecutor struct {
	BinaryPath string
}

// NewTerraformExecutor creates an executor for the terraform binary.
// It resolves the binary path from PATH if not explicitly set.
// If an explicit path is given, it verifies the binary exists.
func NewTerraformExecutor(binaryPath string) (*TerraformExecutor, error) {
	if binaryPath == "" {
		path, err := exec.LookPath("terraform")
		if err != nil {
			return nil, fmt.Errorf("terraform binary not found in PATH: %w (install Terraform from https://developer.hashicorp.com/terraform/install)", err)
		}
		binaryPath = path
	} else {
		if _, err := exec.LookPath(binaryPath); err != nil {
			return nil, fmt.Errorf("terraform binary not found at %q: %w", binaryPath, err)
		}
	}
	return &TerraformExecutor{BinaryPath: binaryPath}, nil
}

// Init runs `terraform init` in workDir.
func (e *TerraformExecutor) Init(ctx context.Context, workDir string) (*RunResult, error) {
	return e.run(ctx, workDir, "init", "-input=false", "-no-color")
}

// Plan runs `terraform plan -out=plan.tfplan`.
// varFile is optional; pass empty string to skip.
func (e *TerraformExecutor) Plan(ctx context.Context, workDir, varFile string) (*RunResult, error) {
	args := []string{"plan", "-out=plan.tfplan", "-input=false", "-no-color"}
	if varFile != "" {
		args = append(args, "-var-file="+varFile)
	}
	return e.run(ctx, workDir, args...)
}

// Apply runs `terraform apply plan.tfplan`.
func (e *TerraformExecutor) Apply(ctx context.Context, workDir, planFile string) (*RunResult, error) {
	if planFile == "" {
		planFile = "plan.tfplan"
	}
	return e.run(ctx, workDir, "apply", "-input=false", "-no-color", planFile)
}

// Destroy runs `terraform destroy -auto-approve`.
func (e *TerraformExecutor) Destroy(ctx context.Context, workDir string) (*RunResult, error) {
	return e.run(ctx, workDir, "destroy", "-auto-approve", "-input=false", "-no-color")
}

// run executes the terraform binary with given args in workDir.
func (e *TerraformExecutor) run(ctx context.Context, workDir string, args ...string) (*RunResult, error) {
	cmd := exec.CommandContext(ctx, e.BinaryPath, args...)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := &RunResult{
		Command:  e.BinaryPath + " " + strings.Join(args, " "),
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		result.Error = err.Error()
		return result, fmt.Errorf("terraform %s: %w\n%s", args[0], err, stderr.String())
	}
	return result, nil
}
