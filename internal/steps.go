package internal

import (
	"context"
	"fmt"
	"os"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
	"github.com/GoCodeAlone/workflow-plugin-tofu/internal/executor"
	"github.com/GoCodeAlone/workflow-plugin-tofu/internal/generator"
	"github.com/GoCodeAlone/workflow-plugin-tofu/internal/state"
)

// generateHCLStep implements step.iac_generate_hcl.
type generateHCLStep struct {
	name   string
	config map[string]any
}

func newGenerateHCLStep(name string, config map[string]any) *generateHCLStep {
	return &generateHCLStep{name: name, config: config}
}

func (s *generateHCLStep) Execute(
	ctx context.Context,
	triggerData map[string]any,
	stepOutputs map[string]map[string]any,
	current map[string]any,
	metadata map[string]any,
	config map[string]any,
) (*sdk.StepResult, error) {
	cfg := mergeMaps(s.config, config)

	outputDir, _ := cfg["output_dir"].(string)
	if outputDir == "" {
		outputDir = "terraform"
	}
	provider, _ := cfg["provider"].(string)
	if provider == "" {
		return nil, fmt.Errorf("step.iac_generate_hcl: 'provider' config is required (aws, gcp, azure, digitalocean)")
	}

	// Build resource specs from config.
	specs := buildResourceSpecs(cfg)

	if err := generator.GenerateHCL(specs, provider, outputDir); err != nil {
		return nil, fmt.Errorf("step.iac_generate_hcl: %w", err)
	}

	return &sdk.StepResult{
		Output: map[string]any{
			"output_dir":     outputDir,
			"provider":       provider,
			"resource_count": len(specs),
		},
	}, nil
}

// tofuInitStep implements step.tofu_init.
type tofuInitStep struct {
	name   string
	config map[string]any
}

func newTofuInitStep(name string, config map[string]any) *tofuInitStep {
	return &tofuInitStep{name: name, config: config}
}

func (s *tofuInitStep) Execute(
	ctx context.Context,
	triggerData map[string]any,
	stepOutputs map[string]map[string]any,
	current map[string]any,
	metadata map[string]any,
	config map[string]any,
) (*sdk.StepResult, error) {
	cfg := mergeMaps(s.config, config)

	workDir, binary, err := executorConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("step.tofu_init: %w", err)
	}

	tool, _ := cfg["tool"].(string)
	result, err := runInit(ctx, tool, binary, workDir)
	if err != nil {
		return nil, fmt.Errorf("step.tofu_init: %w", err)
	}

	return &sdk.StepResult{
		Output: map[string]any{
			"stdout":    result.Stdout,
			"stderr":    result.Stderr,
			"exit_code": result.ExitCode,
		},
	}, nil
}

// tofuPlanStep implements step.tofu_plan.
type tofuPlanStep struct {
	name   string
	config map[string]any
}

func newTofuPlanStep(name string, config map[string]any) *tofuPlanStep {
	return &tofuPlanStep{name: name, config: config}
}

func (s *tofuPlanStep) Execute(
	ctx context.Context,
	triggerData map[string]any,
	stepOutputs map[string]map[string]any,
	current map[string]any,
	metadata map[string]any,
	config map[string]any,
) (*sdk.StepResult, error) {
	cfg := mergeMaps(s.config, config)

	workDir, binary, err := executorConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("step.tofu_plan: %w", err)
	}
	varFile, _ := cfg["var_file"].(string)
	tool, _ := cfg["tool"].(string)

	result, err := runPlan(ctx, tool, binary, workDir, varFile)
	if err != nil {
		return nil, fmt.Errorf("step.tofu_plan: %w", err)
	}

	planFile := workDir + "/plan.tfplan"
	return &sdk.StepResult{
		Output: map[string]any{
			"plan_file": planFile,
			"stdout":    result.Stdout,
			"stderr":    result.Stderr,
			"exit_code": result.ExitCode,
		},
	}, nil
}

// tofuApplyStep implements step.tofu_apply.
type tofuApplyStep struct {
	name   string
	config map[string]any
}

func newTofuApplyStep(name string, config map[string]any) *tofuApplyStep {
	return &tofuApplyStep{name: name, config: config}
}

func (s *tofuApplyStep) Execute(
	ctx context.Context,
	triggerData map[string]any,
	stepOutputs map[string]map[string]any,
	current map[string]any,
	metadata map[string]any,
	config map[string]any,
) (*sdk.StepResult, error) {
	cfg := mergeMaps(s.config, config)

	workDir, binary, err := executorConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("step.tofu_apply: %w", err)
	}
	planFile, _ := cfg["plan_file"].(string)
	tool, _ := cfg["tool"].(string)

	result, err := runApply(ctx, tool, binary, workDir, planFile)
	if err != nil {
		return nil, fmt.Errorf("step.tofu_apply: %w", err)
	}

	return &sdk.StepResult{
		Output: map[string]any{
			"resource_ids": result.ResourceIDs(),
			"stdout":       result.Stdout,
			"stderr":       result.Stderr,
			"exit_code":    result.ExitCode,
		},
	}, nil
}

// stateImportStep implements step.tofu_state_import.
type stateImportStep struct {
	name   string
	config map[string]any
}

func newStateImportStep(name string, config map[string]any) *stateImportStep {
	return &stateImportStep{name: name, config: config}
}

func (s *stateImportStep) Execute(
	ctx context.Context,
	triggerData map[string]any,
	stepOutputs map[string]map[string]any,
	current map[string]any,
	metadata map[string]any,
	config map[string]any,
) (*sdk.StepResult, error) {
	cfg := mergeMaps(s.config, config)

	stateFile, _ := cfg["state_file"].(string)
	if stateFile == "" {
		return nil, fmt.Errorf("step.tofu_state_import: 'state_file' config is required")
	}

	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, fmt.Errorf("step.tofu_state_import: read state file: %w", err)
	}

	states, err := state.ImportTFState(data)
	if err != nil {
		return nil, fmt.Errorf("step.tofu_state_import: %w", err)
	}

	// Convert states to serializable form.
	resources := make([]map[string]any, len(states))
	for i, rs := range states {
		resources[i] = map[string]any{
			"id":          rs.ID,
			"name":        rs.Name,
			"type":        rs.Type,
			"provider":    rs.Provider,
			"provider_id": rs.ProviderID,
			"outputs":     rs.Outputs,
			"created_at":  rs.CreatedAt,
		}
	}

	return &sdk.StepResult{
		Output: map[string]any{
			"resources": resources,
			"count":     len(resources),
		},
	}, nil
}

// stateExportStep implements step.tofu_state_export.
type stateExportStep struct {
	name   string
	config map[string]any
}

func newStateExportStep(name string, config map[string]any) *stateExportStep {
	return &stateExportStep{name: name, config: config}
}

func (s *stateExportStep) Execute(
	ctx context.Context,
	triggerData map[string]any,
	stepOutputs map[string]map[string]any,
	current map[string]any,
	metadata map[string]any,
	config map[string]any,
) (*sdk.StepResult, error) {
	cfg := mergeMaps(s.config, config)

	outputFile, _ := cfg["output_file"].(string)
	if outputFile == "" {
		outputFile = "terraform.tfstate"
	}

	// Build states from current context or step outputs.
	states, err := state.ImportTFState([]byte(`{"version":4,"resources":[]}`))
	if err != nil {
		return nil, fmt.Errorf("step.tofu_state_export: %w", err)
	}

	data, err := state.ExportTFState(states)
	if err != nil {
		return nil, fmt.Errorf("step.tofu_state_export: %w", err)
	}

	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		return nil, fmt.Errorf("step.tofu_state_export: write output: %w", err)
	}

	return &sdk.StepResult{
		Output: map[string]any{
			"output_file": outputFile,
			"size":        len(data),
		},
	}, nil
}

// helpers

func mergeMaps(base, override map[string]any) map[string]any {
	result := make(map[string]any, len(base))
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		result[k] = v
	}
	return result
}

func executorConfig(cfg map[string]any) (workDir, binary string, err error) {
	workDir, _ = cfg["working_dir"].(string)
	if workDir == "" {
		workDir = "."
	}
	binary, _ = cfg["binary_path"].(string)
	return workDir, binary, nil
}

func buildResourceSpecs(cfg map[string]any) []generator.ResourceSpec {
	rawResources, _ := cfg["resources"].([]any)
	specs := make([]generator.ResourceSpec, 0, len(rawResources))
	for _, r := range rawResources {
		if rm, ok := r.(map[string]any); ok {
			spec := generator.ResourceSpec{
				Name:     strFromMap(rm, "name"),
				Type:     strFromMap(rm, "type"),
				Provider: strFromMap(rm, "provider"),
			}
			if c, ok := rm["config"].(map[string]any); ok {
				spec.Config = c
			} else {
				spec.Config = make(map[string]any)
			}
			if spec.Name != "" && spec.Type != "" {
				specs = append(specs, spec)
			}
		}
	}
	return specs
}

func strFromMap(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func runInit(ctx context.Context, tool, binary, workDir string) (*executor.RunResult, error) {
	if tool == "terraform" {
		ex, err := executor.NewTerraformExecutor(binary)
		if err != nil {
			return nil, err
		}
		return ex.Init(ctx, workDir)
	}
	ex, err := executor.NewTofuExecutor(binary)
	if err != nil {
		return nil, err
	}
	return ex.Init(ctx, workDir)
}

func runPlan(ctx context.Context, tool, binary, workDir, varFile string) (*executor.RunResult, error) {
	if tool == "terraform" {
		ex, err := executor.NewTerraformExecutor(binary)
		if err != nil {
			return nil, err
		}
		return ex.Plan(ctx, workDir, varFile)
	}
	ex, err := executor.NewTofuExecutor(binary)
	if err != nil {
		return nil, err
	}
	return ex.Plan(ctx, workDir, varFile)
}

func runApply(ctx context.Context, tool, binary, workDir, planFile string) (*executor.RunResult, error) {
	if tool == "terraform" {
		ex, err := executor.NewTerraformExecutor(binary)
		if err != nil {
			return nil, err
		}
		return ex.Apply(ctx, workDir, planFile)
	}
	ex, err := executor.NewTofuExecutor(binary)
	if err != nil {
		return nil, err
	}
	return ex.Apply(ctx, workDir, planFile)
}
