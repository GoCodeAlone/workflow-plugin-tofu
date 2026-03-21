package state

import (
	"encoding/json"
	"fmt"
	"time"
)

// ExportTFState converts a slice of ResourceState into a .tfstate-compatible JSON byte slice.
// The resulting format is Terraform state version 4.
func ExportTFState(states []ResourceState) ([]byte, error) {
	tfstate := TFState{
		Version:          4,
		TerraformVersion: "1.6.0",
		Serial:           1,
		Lineage:          generateLineage(),
		Resources:        make([]TFResource, 0, len(states)),
	}

	for _, rs := range states {
		tfRes := TFResource{
			Mode:     "managed",
			Type:     abstractToTFType(rs.Type, rs.Provider),
			Name:     rs.Name,
			Provider: tfProviderString(rs.Provider),
			Instances: []TFInstance{
				{
					SchemaVersion: 0,
					Attributes:    mergeAttributes(rs),
					Dependencies:  rs.Dependencies,
				},
			},
		}
		tfstate.Resources = append(tfstate.Resources, tfRes)
	}

	data, err := json.MarshalIndent(tfstate, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal tfstate: %w", err)
	}
	return data, nil
}

// mergeAttributes combines ProviderID, AppliedConfig, and Outputs into a single attribute map.
func mergeAttributes(rs ResourceState) map[string]any {
	attrs := make(map[string]any)
	attrs["id"] = rs.ProviderID
	for k, v := range rs.AppliedConfig {
		attrs[k] = v
	}
	for k, v := range rs.Outputs {
		attrs[k] = v
	}
	return attrs
}

// abstractToTFType maps abstract infra types back to a representative Terraform resource type.
// If no mapping exists, the abstract type is returned as-is.
func abstractToTFType(abstractType, provider string) string {
	type key struct{ t, p string }
	mappings := map[key]string{
		{"infra.database", "aws"}:           "aws_db_instance",
		{"infra.vpc", "aws"}:                "aws_vpc",
		{"infra.container_service", "aws"}:  "aws_ecs_service",
		{"infra.k8s_cluster", "aws"}:        "aws_eks_cluster",
		{"infra.cache", "aws"}:              "aws_elasticache_replication_group",
		{"infra.load_balancer", "aws"}:      "aws_lb",
		{"infra.dns", "aws"}:                "aws_route53_zone",
		{"infra.registry", "aws"}:           "aws_ecr_repository",
		{"infra.api_gateway", "aws"}:        "aws_apigatewayv2_api",
		{"infra.firewall", "aws"}:           "aws_security_group",
		{"infra.iam_role", "aws"}:           "aws_iam_role",
		{"infra.storage", "aws"}:            "aws_s3_bucket",
		{"infra.certificate", "aws"}:        "aws_acm_certificate",
		{"infra.database", "gcp"}:           "google_sql_database_instance",
		{"infra.vpc", "gcp"}:                "google_compute_network",
		{"infra.container_service", "gcp"}:  "google_cloud_run_service",
		{"infra.k8s_cluster", "gcp"}:        "google_container_cluster",
		{"infra.cache", "gcp"}:              "google_redis_instance",
		{"infra.load_balancer", "gcp"}:      "google_compute_global_forwarding_rule",
		{"infra.dns", "gcp"}:                "google_dns_managed_zone",
		{"infra.registry", "gcp"}:           "google_artifact_registry_repository",
		{"infra.api_gateway", "gcp"}:        "google_api_gateway_api",
		{"infra.firewall", "gcp"}:           "google_compute_firewall",
		{"infra.iam_role", "gcp"}:           "google_service_account",
		{"infra.storage", "gcp"}:            "google_storage_bucket",
		{"infra.certificate", "gcp"}:        "google_compute_managed_ssl_certificate",
		{"infra.database", "azure"}:         "azurerm_postgresql_flexible_server",
		{"infra.vpc", "azure"}:              "azurerm_virtual_network",
		{"infra.container_service", "azure"}: "azurerm_container_group",
		{"infra.k8s_cluster", "azure"}:      "azurerm_kubernetes_cluster",
		{"infra.cache", "azure"}:            "azurerm_redis_cache",
		{"infra.load_balancer", "azure"}:    "azurerm_lb",
		{"infra.dns", "azure"}:              "azurerm_dns_zone",
		{"infra.registry", "azure"}:         "azurerm_container_registry",
		{"infra.api_gateway", "azure"}:      "azurerm_api_management",
		{"infra.firewall", "azure"}:         "azurerm_network_security_group",
		{"infra.iam_role", "azure"}:         "azurerm_user_assigned_identity",
		{"infra.storage", "azure"}:          "azurerm_storage_account",
		{"infra.certificate", "azure"}:      "azurerm_app_service_certificate",
		{"infra.database", "digitalocean"}:  "digitalocean_database_cluster",
		{"infra.vpc", "digitalocean"}:       "digitalocean_vpc",
		{"infra.container_service", "digitalocean"}: "digitalocean_app",
		{"infra.k8s_cluster", "digitalocean"}:    "digitalocean_kubernetes_cluster",
		{"infra.load_balancer", "digitalocean"}:  "digitalocean_loadbalancer",
		{"infra.dns", "digitalocean"}:            "digitalocean_domain",
		{"infra.registry", "digitalocean"}:       "digitalocean_container_registry",
		{"infra.firewall", "digitalocean"}:       "digitalocean_firewall",
		{"infra.iam_role", "digitalocean"}:       "digitalocean_token",
		{"infra.storage", "digitalocean"}:        "digitalocean_spaces_bucket",
		{"infra.certificate", "digitalocean"}:    "digitalocean_certificate",
	}
	if tfType, ok := mappings[key{abstractType, provider}]; ok {
		return tfType
	}
	return abstractType
}

// tfProviderString returns the TF provider registry string for a provider name.
func tfProviderString(provider string) string {
	switch provider {
	case "aws":
		return `provider["registry.terraform.io/hashicorp/aws"]`
	case "gcp":
		return `provider["registry.terraform.io/hashicorp/google"]`
	case "azure":
		return `provider["registry.terraform.io/hashicorp/azurerm"]`
	case "digitalocean":
		return `provider["registry.terraform.io/digitalocean/digitalocean"]`
	default:
		return `provider["registry.terraform.io/hashicorp/` + provider + `"]`
	}
}

// generateLineage creates a deterministic placeholder lineage string.
func generateLineage() string {
	return fmt.Sprintf("workflow-plugin-tofu-%d", time.Now().Unix())
}
