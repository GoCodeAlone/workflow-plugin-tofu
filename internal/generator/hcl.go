// Package generator converts abstract infra resource specs into Terraform HCL files.
package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// ResourceSpec is the abstract declaration of a single infrastructure resource.
type ResourceSpec struct {
	Name     string
	Type     string
	Provider string
	Config   map[string]any
}

// GenerateHCL writes .tf files to outputDir for each resource in specs.
// provider is the cloud provider name (aws, gcp, azure, digitalocean).
func GenerateHCL(specs []ResourceSpec, provider, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	// Write one file per resource spec.
	for _, spec := range specs {
		f := hclwrite.NewEmptyFile()
		body := f.Body()

		if err := appendResource(body, spec, provider); err != nil {
			return fmt.Errorf("generate resource %q: %w", spec.Name, err)
		}

		filename := filepath.Join(outputDir, sanitizeName(spec.Name)+".tf")
		if err := os.WriteFile(filename, f.Bytes(), 0644); err != nil {
			return fmt.Errorf("write %s: %w", filename, err)
		}
	}

	// Write provider block.
	pf := hclwrite.NewEmptyFile()
	if err := appendProviderBlock(pf.Body(), provider); err != nil {
		return fmt.Errorf("write provider block: %w", err)
	}
	providerFile := filepath.Join(outputDir, "provider.tf")
	if err := os.WriteFile(providerFile, pf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write provider.tf: %w", err)
	}

	return nil
}

// appendResource adds the Terraform resource block(s) for one abstract spec.
func appendResource(body *hclwrite.Body, spec ResourceSpec, provider string) error {
	switch spec.Type {
	case "infra.database":
		return appendDatabase(body, spec, provider)
	case "infra.vpc":
		return appendVPC(body, spec, provider)
	case "infra.container_service":
		return appendContainerService(body, spec, provider)
	case "infra.k8s_cluster":
		return appendK8sCluster(body, spec, provider)
	case "infra.cache":
		return appendCache(body, spec, provider)
	case "infra.load_balancer":
		return appendLoadBalancer(body, spec, provider)
	case "infra.dns":
		return appendDNS(body, spec, provider)
	case "infra.registry":
		return appendRegistry(body, spec, provider)
	case "infra.api_gateway":
		return appendAPIGateway(body, spec, provider)
	case "infra.firewall":
		return appendFirewall(body, spec, provider)
	case "infra.iam_role":
		return appendIAMRole(body, spec, provider)
	case "infra.storage":
		return appendStorage(body, spec, provider)
	case "infra.certificate":
		return appendCertificate(body, spec, provider)
	default:
		return fmt.Errorf("unsupported resource type %q for provider %q", spec.Type, provider)
	}
}

// appendProviderBlock writes the Terraform required_providers + provider block.
func appendProviderBlock(body *hclwrite.Body, provider string) error {
	terraform := body.AppendNewBlock("terraform", nil)
	required := terraform.Body().AppendNewBlock("required_providers", nil)

	switch provider {
	case "aws":
		setProviderSource(required.Body(), "aws", "hashicorp/aws", "~> 5.0")
		awsBlock := body.AppendNewBlock("provider", []string{"aws"})
		awsBlock.Body().SetAttributeValue("region", cty.StringVal("us-east-1"))
	case "gcp":
		setProviderSource(required.Body(), "google", "hashicorp/google", "~> 5.0")
		body.AppendNewBlock("provider", []string{"google"})
	case "azure":
		setProviderSource(required.Body(), "azurerm", "hashicorp/azurerm", "~> 3.0")
		az := body.AppendNewBlock("provider", []string{"azurerm"})
		az.Body().AppendNewBlock("features", nil)
	case "digitalocean":
		setProviderSource(required.Body(), "digitalocean", "digitalocean/digitalocean", "~> 2.0")
		body.AppendNewBlock("provider", []string{"digitalocean"})
	default:
		return fmt.Errorf("unsupported provider %q", provider)
	}
	return nil
}

func setProviderSource(body *hclwrite.Body, name, source, version string) {
	blk := body.AppendNewBlock(name, nil)
	blk.Body().SetAttributeValue("source", cty.StringVal(source))
	blk.Body().SetAttributeValue("version", cty.StringVal(version))
}

// sanitizeName replaces characters invalid in filenames with underscores.
func sanitizeName(name string) string {
	out := make([]byte, len(name))
	for i, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' {
			out[i] = byte(c)
		} else {
			out[i] = '_'
		}
	}
	return string(out)
}

// strVal returns a cty string value from a map key, or the fallback.
func strVal(cfg map[string]any, key, fallback string) cty.Value {
	if v, ok := cfg[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return cty.StringVal(s)
		}
	}
	return cty.StringVal(fallback)
}

// appendDatabase dispatches to the provider-specific database generator.
func appendDatabase(body *hclwrite.Body, spec ResourceSpec, provider string) error {
	switch provider {
	case "aws":
		return awsDatabase(body, spec)
	case "gcp":
		return gcpDatabase(body, spec)
	case "azure":
		return azureDatabase(body, spec)
	case "digitalocean":
		return doDatabase(body, spec)
	default:
		return fmt.Errorf("unsupported provider %q for infra.database", provider)
	}
}

func appendVPC(body *hclwrite.Body, spec ResourceSpec, provider string) error {
	switch provider {
	case "aws":
		return awsVPC(body, spec)
	case "gcp":
		return gcpVPC(body, spec)
	case "azure":
		return azureVPC(body, spec)
	case "digitalocean":
		return doVPC(body, spec)
	default:
		return fmt.Errorf("unsupported provider %q for infra.vpc", provider)
	}
}

func appendContainerService(body *hclwrite.Body, spec ResourceSpec, provider string) error {
	switch provider {
	case "aws":
		return awsContainerService(body, spec)
	case "gcp":
		return gcpContainerService(body, spec)
	case "azure":
		return azureContainerService(body, spec)
	case "digitalocean":
		return doContainerService(body, spec)
	default:
		return fmt.Errorf("unsupported provider %q for infra.container_service", provider)
	}
}

func appendK8sCluster(body *hclwrite.Body, spec ResourceSpec, provider string) error {
	switch provider {
	case "aws":
		return awsK8sCluster(body, spec)
	case "gcp":
		return gcpK8sCluster(body, spec)
	case "azure":
		return azureK8sCluster(body, spec)
	case "digitalocean":
		return doK8sCluster(body, spec)
	default:
		return fmt.Errorf("unsupported provider %q for infra.k8s_cluster", provider)
	}
}

func appendCache(body *hclwrite.Body, spec ResourceSpec, provider string) error {
	switch provider {
	case "aws":
		return awsCache(body, spec)
	case "gcp":
		return gcpCache(body, spec)
	case "azure":
		return azureCache(body, spec)
	case "digitalocean":
		return doCache(body, spec)
	default:
		return fmt.Errorf("unsupported provider %q for infra.cache", provider)
	}
}

func appendLoadBalancer(body *hclwrite.Body, spec ResourceSpec, provider string) error {
	switch provider {
	case "aws":
		return awsLoadBalancer(body, spec)
	case "gcp":
		return gcpLoadBalancer(body, spec)
	case "azure":
		return azureLoadBalancer(body, spec)
	case "digitalocean":
		return doLoadBalancer(body, spec)
	default:
		return fmt.Errorf("unsupported provider %q for infra.load_balancer", provider)
	}
}

func appendDNS(body *hclwrite.Body, spec ResourceSpec, provider string) error {
	switch provider {
	case "aws":
		return awsDNS(body, spec)
	case "gcp":
		return gcpDNS(body, spec)
	case "azure":
		return azureDNS(body, spec)
	case "digitalocean":
		return doDNS(body, spec)
	default:
		return fmt.Errorf("unsupported provider %q for infra.dns", provider)
	}
}

func appendRegistry(body *hclwrite.Body, spec ResourceSpec, provider string) error {
	switch provider {
	case "aws":
		return awsRegistry(body, spec)
	case "gcp":
		return gcpRegistry(body, spec)
	case "azure":
		return azureRegistry(body, spec)
	case "digitalocean":
		return doRegistry(body, spec)
	default:
		return fmt.Errorf("unsupported provider %q for infra.registry", provider)
	}
}

func appendAPIGateway(body *hclwrite.Body, spec ResourceSpec, provider string) error {
	switch provider {
	case "aws":
		return awsAPIGateway(body, spec)
	case "gcp":
		return gcpAPIGateway(body, spec)
	case "azure":
		return azureAPIGateway(body, spec)
	case "digitalocean":
		return doAPIGateway(body, spec)
	default:
		return fmt.Errorf("unsupported provider %q for infra.api_gateway", provider)
	}
}

func appendFirewall(body *hclwrite.Body, spec ResourceSpec, provider string) error {
	switch provider {
	case "aws":
		return awsFirewall(body, spec)
	case "gcp":
		return gcpFirewall(body, spec)
	case "azure":
		return azureFirewall(body, spec)
	case "digitalocean":
		return doFirewall(body, spec)
	default:
		return fmt.Errorf("unsupported provider %q for infra.firewall", provider)
	}
}

func appendIAMRole(body *hclwrite.Body, spec ResourceSpec, provider string) error {
	switch provider {
	case "aws":
		return awsIAMRole(body, spec)
	case "gcp":
		return gcpIAMRole(body, spec)
	case "azure":
		return azureIAMRole(body, spec)
	case "digitalocean":
		return doIAMRole(body, spec)
	default:
		return fmt.Errorf("unsupported provider %q for infra.iam_role", provider)
	}
}

func appendStorage(body *hclwrite.Body, spec ResourceSpec, provider string) error {
	switch provider {
	case "aws":
		return awsStorage(body, spec)
	case "gcp":
		return gcpStorage(body, spec)
	case "azure":
		return azureStorage(body, spec)
	case "digitalocean":
		return doStorage(body, spec)
	default:
		return fmt.Errorf("unsupported provider %q for infra.storage", provider)
	}
}

func appendCertificate(body *hclwrite.Body, spec ResourceSpec, provider string) error {
	switch provider {
	case "aws":
		return awsCertificate(body, spec)
	case "gcp":
		return gcpCertificate(body, spec)
	case "azure":
		return azureCertificate(body, spec)
	case "digitalocean":
		return doCertificate(body, spec)
	default:
		return fmt.Errorf("unsupported provider %q for infra.certificate", provider)
	}
}
