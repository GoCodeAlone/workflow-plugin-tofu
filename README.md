# workflow-plugin-tofu

> ⚠️ **Experimental** — This plugin compiles and passes its unit tests but has not been validated in any active GoCodeAlone-internal production deployment. Use with caution. Please [open an issue](https://github.com/GoCodeAlone/workflow-plugin-tofu/issues/new) if you adopt it so we can promote it to **verified** status.

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/GoCodeAlone/workflow-plugin-tofu.svg)](https://pkg.go.dev/github.com/GoCodeAlone/workflow-plugin-tofu)

OpenTofu/Terraform adapter for workflow IaC — generates HCL from abstract infra specs, executes plan/apply, and handles state import/export.

## What it provides

**Pipeline step types:**
- `step.iac_generate_hcl` — Generate HCL configuration from abstract workflow infra specs
- `step.tofu_init` — Run `tofu init` (downloads providers + modules)
- `step.tofu_plan` — Run `tofu plan` and capture the execution plan
- `step.tofu_apply` — Run `tofu apply` to provision infrastructure
- `step.tofu_state_import` — Import existing resources into Tofu state
- `step.tofu_state_export` — Export Tofu state for use by other workflow steps

## Prerequisites

- [OpenTofu](https://opentofu.org/docs/intro/install/) or [Terraform](https://developer.hashicorp.com/terraform/install) installed and on `$PATH`
- Provider credentials configured via env vars (e.g. `AWS_ACCESS_KEY_ID`, `GOOGLE_CREDENTIALS`, `ARM_CLIENT_ID`)

## Install

```yaml
# In your wfctl.yaml
version: 1
plugins:
  - name: workflow-plugin-tofu
    version: 0.1.3
    source: github.com/GoCodeAlone/workflow-plugin-tofu
```

Then:

```sh
wfctl plugin install
```

## Minimal example

See [`examples/minimal/config.yaml`](examples/minimal/config.yaml).

## Documentation

- [Plugin authoring guide (upstream)](https://github.com/GoCodeAlone/workflow/blob/main/docs/PLUGIN_AUTHORING.md)
- [Workflow engine docs](https://github.com/GoCodeAlone/workflow)
- [IaC guide](https://github.com/GoCodeAlone/workflow/blob/main/docs/iac/)

## License

MIT. See [LICENSE](LICENSE).
