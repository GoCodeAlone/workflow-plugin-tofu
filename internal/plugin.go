// Package internal implements the workflow-plugin-tofu plugin.
package internal

import (
	"fmt"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

// tofuPlugin implements sdk.PluginProvider and sdk.StepProvider.
type tofuPlugin struct{}

// NewTofuPlugin returns a new tofuPlugin instance.
func NewTofuPlugin() sdk.PluginProvider {
	return &tofuPlugin{}
}

// Manifest returns plugin metadata.
func (p *tofuPlugin) Manifest() sdk.PluginManifest {
	return sdk.PluginManifest{
		Name:        "workflow-plugin-tofu",
		Version:     "0.1.0",
		Author:      "GoCodeAlone",
		Description: "OpenTofu/Terraform adapter: HCL generation, plan/apply execution, and state import/export",
	}
}

// StepTypes returns the step type names this plugin provides.
func (p *tofuPlugin) StepTypes() []string {
	return []string{
		"step.iac_generate_hcl",
		"step.tofu_init",
		"step.tofu_plan",
		"step.tofu_apply",
		"step.tofu_state_import",
		"step.tofu_state_export",
	}
}

// CreateStep creates a step instance of the given type.
func (p *tofuPlugin) CreateStep(typeName, name string, config map[string]any) (sdk.StepInstance, error) {
	switch typeName {
	case "step.iac_generate_hcl":
		return newGenerateHCLStep(name, config), nil
	case "step.tofu_init":
		return newTofuInitStep(name, config), nil
	case "step.tofu_plan":
		return newTofuPlanStep(name, config), nil
	case "step.tofu_apply":
		return newTofuApplyStep(name, config), nil
	case "step.tofu_state_import":
		return newStateImportStep(name, config), nil
	case "step.tofu_state_export":
		return newStateExportStep(name, config), nil
	default:
		return nil, fmt.Errorf("tofu plugin: unknown step type %q", typeName)
	}
}
