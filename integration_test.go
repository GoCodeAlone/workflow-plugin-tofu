package tofu_test

import (
	"testing"

	"github.com/GoCodeAlone/workflow/wftest"
)

func TestIntegration_TofuPlan(t *testing.T) {
	h := wftest.New(t,
		wftest.WithYAML(`
pipelines:
  iac-plan:
    steps:
      - name: plan
        type: step.tofu_plan
        config:
          working_dir: /tmp/test
`),
		wftest.MockStep("step.tofu_plan", wftest.Returns(map[string]any{
			"has_changes": true,
			"plan_file":   "/tmp/plan.tfplan",
		})),
	)
	result := h.ExecutePipeline("iac-plan", nil)
	if result.Error != nil {
		t.Fatalf("pipeline failed: %v", result.Error)
	}
	if !result.StepExecuted("plan") {
		t.Error("expected plan step to execute")
	}
	if result.StepOutput("plan")["has_changes"] != true {
		t.Error("expected has_changes to be true")
	}
	if result.StepOutput("plan")["plan_file"] != "/tmp/plan.tfplan" {
		t.Errorf("expected plan_file to be /tmp/plan.tfplan, got %v", result.StepOutput("plan")["plan_file"])
	}
}

func TestIntegration_TofuApply(t *testing.T) {
	h := wftest.New(t,
		wftest.WithYAML(`
pipelines:
  iac-apply:
    steps:
      - name: init
        type: step.tofu_init
        config:
          working_dir: /tmp/test
      - name: plan
        type: step.tofu_plan
        config:
          working_dir: /tmp/test
      - name: apply
        type: step.tofu_apply
        config:
          working_dir: /tmp/test
`),
		wftest.MockStep("step.tofu_init", wftest.Returns(map[string]any{
			"initialized": true,
		})),
		wftest.MockStep("step.tofu_plan", wftest.Returns(map[string]any{
			"has_changes": true,
			"plan_file":   "/tmp/plan.tfplan",
		})),
		wftest.MockStep("step.tofu_apply", wftest.Returns(map[string]any{
			"applied":    true,
			"changed":    3,
			"destroyed":  0,
		})),
	)
	result := h.ExecutePipeline("iac-apply", nil)
	if result.Error != nil {
		t.Fatalf("pipeline failed: %v", result.Error)
	}
	if !result.StepExecuted("init") {
		t.Error("expected init step to execute")
	}
	if !result.StepExecuted("plan") {
		t.Error("expected plan step to execute")
	}
	if !result.StepExecuted("apply") {
		t.Error("expected apply step to execute")
	}
	if result.StepOutput("apply")["applied"] != true {
		t.Error("expected apply to succeed")
	}
	if result.StepOutput("apply")["changed"] != 3 {
		t.Errorf("expected 3 changes, got %v", result.StepOutput("apply")["changed"])
	}
	if result.StepCount() != 3 {
		t.Errorf("expected 3 steps executed, got %d", result.StepCount())
	}
}

func TestIntegration_TofuInit(t *testing.T) {
	h := wftest.New(t,
		wftest.WithYAML(`
pipelines:
  iac-init:
    steps:
      - name: init
        type: step.tofu_init
        config:
          working_dir: /tmp/tofu-workspace
          backend_config: {}
`),
		wftest.MockStep("step.tofu_init", wftest.Returns(map[string]any{
			"initialized":     true,
			"backend_type":    "local",
			"working_dir":     "/tmp/tofu-workspace",
		})),
	)
	result := h.ExecutePipeline("iac-init", nil)
	if result.Error != nil {
		t.Fatalf("pipeline failed: %v", result.Error)
	}
	if !result.StepExecuted("init") {
		t.Error("expected init step to execute")
	}
	if result.StepOutput("init")["initialized"] != true {
		t.Error("expected initialized to be true")
	}
	if result.StepOutput("init")["backend_type"] != "local" {
		t.Errorf("expected backend_type to be local, got %v", result.StepOutput("init")["backend_type"])
	}
}

func TestIntegration_HCLGenerate(t *testing.T) {
	h := wftest.New(t,
		wftest.WithYAML(`
pipelines:
  hcl-generate:
    steps:
      - name: generate
        type: step.iac_generate_hcl
        config:
          output_dir: /tmp/hcl-output
          resource_type: aws_instance
`),
		wftest.MockStep("step.iac_generate_hcl", wftest.Returns(map[string]any{
			"generated":   true,
			"output_file": "/tmp/hcl-output/main.tf",
			"resource_count": 1,
		})),
	)
	result := h.ExecutePipeline("hcl-generate", nil)
	if result.Error != nil {
		t.Fatalf("pipeline failed: %v", result.Error)
	}
	if !result.StepExecuted("generate") {
		t.Error("expected generate step to execute")
	}
	if result.StepOutput("generate")["generated"] != true {
		t.Error("expected generated to be true")
	}
	if result.StepOutput("generate")["output_file"] != "/tmp/hcl-output/main.tf" {
		t.Errorf("expected output_file to be /tmp/hcl-output/main.tf, got %v", result.StepOutput("generate")["output_file"])
	}
}
