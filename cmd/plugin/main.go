// Command workflow-plugin-tofu is a workflow engine external plugin that provides
// OpenTofu/Terraform adapter capabilities: HCL generation, plan/apply execution,
// and state import/export.
// It runs as a subprocess and communicates with the host workflow engine via the
// go-plugin protocol.
package main

import (
	"github.com/GoCodeAlone/workflow-plugin-tofu/internal"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func main() {
	sdk.Serve(internal.NewTofuPlugin())
}
