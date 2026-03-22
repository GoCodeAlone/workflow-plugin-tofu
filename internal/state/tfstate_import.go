// Package state provides adapters between Terraform/Pulumi state formats and the
// workflow engine's ResourceState representation.
package state

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/GoCodeAlone/workflow/interfaces"
)

// TFState is the top-level structure of a terraform.tfstate file.
type TFState struct {
	Version          int             `json:"version"`
	TerraformVersion string          `json:"terraform_version"`
	Serial           int             `json:"serial"`
	Lineage          string          `json:"lineage"`
	Resources        []TFResource    `json:"resources"`
}

// TFResource is a single resource block in a tfstate file.
type TFResource struct {
	Module    string       `json:"module,omitempty"`
	Mode      string       `json:"mode"` // managed, data
	Type      string       `json:"type"`
	Name      string       `json:"name"`
	Provider  string       `json:"provider"`
	Instances []TFInstance `json:"instances"`
}

// TFInstance is a single instance of a resource in a tfstate file.
type TFInstance struct {
	SchemaVersion int            `json:"schema_version"`
	Attributes    map[string]any `json:"attributes"`
	Dependencies  []string       `json:"dependencies,omitempty"`
}

// ImportTFState parses a .tfstate JSON byte slice and returns a slice of ResourceState.
// Only "managed" mode resources with at least one instance are included.
func ImportTFState(data []byte) ([]interfaces.ResourceState, error) {
	var tfstate TFState
	if err := json.Unmarshal(data, &tfstate); err != nil {
		return nil, fmt.Errorf("parse tfstate: %w", err)
	}

		var states []interfaces.ResourceState
	for _, res := range tfstate.Resources {
		if res.Mode != "managed" {
			continue
		}
		for _, inst := range res.Instances {
			rs := interfaces.ResourceState{
				ID:            resourceID(inst.Attributes),
				Name:          res.Name,
				Type:          tfTypeToAbstract(res.Type),
				Provider:      providerFromTF(res.Provider),
				ProviderID:    resourceID(inst.Attributes),
				AppliedConfig: inst.Attributes,
				Outputs:       extractOutputs(res.Type, inst.Attributes),
				Dependencies:  inst.Dependencies,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}
			states = append(states, rs)
		}
	}
	return states, nil
}

// resourceID extracts the "id" attribute from a TF instance attribute map.
func resourceID(attrs map[string]any) string {
	if v, ok := attrs["id"]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// tfTypeToAbstract maps Terraform resource types to abstract infra types.
// Returns the original tf type if no mapping exists.
func tfTypeToAbstract(tfType string) string {
	mappings := map[string]string{
		// AWS
		"aws_db_instance":                     "infra.database",
		"aws_rds_cluster":                     "infra.database",
		"aws_vpc":                             "infra.vpc",
		"aws_ecs_service":                     "infra.container_service",
		"aws_eks_cluster":                     "infra.k8s_cluster",
		"aws_elasticache_cluster":             "infra.cache",
		"aws_elasticache_replication_group":   "infra.cache",
		"aws_lb":                              "infra.load_balancer",
		"aws_route53_zone":                    "infra.dns",
		"aws_route53_record":                  "infra.dns",
		"aws_ecr_repository":                  "infra.registry",
		"aws_apigatewayv2_api":                "infra.api_gateway",
		"aws_security_group":                  "infra.firewall",
		"aws_iam_role":                        "infra.iam_role",
		"aws_s3_bucket":                       "infra.storage",
		"aws_acm_certificate":                 "infra.certificate",
		// GCP
		"google_sql_database_instance":        "infra.database",
		"google_compute_network":              "infra.vpc",
		"google_cloud_run_service":            "infra.container_service",
		"google_container_cluster":            "infra.k8s_cluster",
		"google_redis_instance":               "infra.cache",
		"google_compute_forwarding_rule":      "infra.load_balancer",
		"google_compute_global_forwarding_rule": "infra.load_balancer",
		"google_dns_managed_zone":             "infra.dns",
		"google_artifact_registry_repository": "infra.registry",
		"google_api_gateway_api":              "infra.api_gateway",
		"google_compute_firewall":             "infra.firewall",
		"google_service_account":              "infra.iam_role",
		"google_storage_bucket":               "infra.storage",
		"google_compute_ssl_certificate":      "infra.certificate",
		"google_compute_managed_ssl_certificate": "infra.certificate",
		// Azure
		"azurerm_mssql_server":               "infra.database",
		"azurerm_postgresql_flexible_server": "infra.database",
		"azurerm_virtual_network":            "infra.vpc",
		"azurerm_container_group":            "infra.container_service",
		"azurerm_kubernetes_cluster":         "infra.k8s_cluster",
		"azurerm_redis_cache":                "infra.cache",
		"azurerm_lb":                         "infra.load_balancer",
		"azurerm_dns_zone":                   "infra.dns",
		"azurerm_container_registry":         "infra.registry",
		"azurerm_api_management":             "infra.api_gateway",
		"azurerm_network_security_group":     "infra.firewall",
		"azurerm_user_assigned_identity":     "infra.iam_role",
		"azurerm_storage_account":            "infra.storage",
		"azurerm_app_service_certificate":    "infra.certificate",
		// DigitalOcean
		"digitalocean_database_cluster":      "infra.database",
		"digitalocean_vpc":                   "infra.vpc",
		"digitalocean_app":                   "infra.container_service",
		"digitalocean_kubernetes_cluster":    "infra.k8s_cluster",
		"digitalocean_loadbalancer":          "infra.load_balancer",
		"digitalocean_domain":                "infra.dns",
		"digitalocean_container_registry":    "infra.registry",
		"digitalocean_firewall":              "infra.firewall",
		"digitalocean_token":                 "infra.iam_role",
		"digitalocean_spaces_bucket":         "infra.storage",
		"digitalocean_certificate":           "infra.certificate",
	}
	if abstract, ok := mappings[tfType]; ok {
		return abstract
	}
	return tfType
}

// providerFromTF extracts a short provider name from the TF provider string.
// e.g. "provider[\"registry.terraform.io/hashicorp/aws\"]" → "aws"
// e.g. "provider[\"registry.terraform.io/hashicorp/google\"]" → "gcp"
func providerFromTF(provider string) string {
	// Extract last path component after /
	for i := len(provider) - 1; i >= 0; i-- {
		if provider[i] == '/' {
			name := provider[i+1:]
			// Strip trailing "]
			for j := range name {
				if name[j] == '"' || name[j] == ']' {
					name = name[:j]
					break
				}
			}
			// Normalize provider names to canonical short forms.
			switch name {
			case "google":
				return "gcp"
			case "azurerm":
				return "azure"
			default:
				return name
			}
		}
	}
	return provider
}

// extractOutputs pulls common endpoint/connection outputs from TF attributes.
func extractOutputs(tfType string, attrs map[string]any) map[string]any {
	outputs := make(map[string]any)
	// Common output keys by resource type.
	outputKeys := map[string][]string{
		"aws_db_instance":                    {"endpoint", "address", "port", "arn"},
		"aws_vpc":                            {"id", "cidr_block", "default_route_table_id"},
		"aws_ecs_service":                    {"id", "cluster"},
		"aws_lb":                             {"dns_name", "arn", "zone_id"},
		"aws_s3_bucket":                      {"id", "bucket", "bucket_regional_domain_name"},
		"google_sql_database_instance":       {"connection_name", "first_ip_address", "service_account_email_address"},
		"google_cloud_run_service":           {"status.0.url"},
		"google_storage_bucket":              {"url", "self_link"},
		"azurerm_postgresql_flexible_server": {"fqdn", "id"},
		"azurerm_storage_account":            {"primary_blob_endpoint", "primary_access_key"},
		"digitalocean_database_cluster":      {"host", "port", "uri", "private_uri"},
		"digitalocean_spaces_bucket":         {"bucket_domain_name", "endpoint"},
	}
	keys, ok := outputKeys[tfType]
	if !ok {
		// For unknown types, copy id and common fields.
		keys = []string{"id", "name", "arn"}
	}
	for _, k := range keys {
		if v, ok := attrs[k]; ok {
			outputs[k] = v
		}
	}
	return outputs
}
