package state

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// PulumiCheckpoint is the top-level Pulumi checkpoint JSON structure.
type PulumiCheckpoint struct {
	Version  int           `json:"version"`
	Deployment *PulumiDeployment `json:"deployment"`
}

// PulumiDeployment contains the resource list in a Pulumi checkpoint.
type PulumiDeployment struct {
	Manifest  PulumiManifest  `json:"manifest"`
	Resources []PulumiResource `json:"resources"`
}

// PulumiManifest is metadata in a Pulumi checkpoint.
type PulumiManifest struct {
	Time    string `json:"time"`
	Version string `json:"version"`
}

// PulumiResource represents a single Pulumi resource in a checkpoint.
type PulumiResource struct {
	URN        string         `json:"urn"`
	Custom     bool           `json:"custom"`
	Type       string         `json:"type"` // e.g. "aws:rds/instance:Instance"
	Inputs     map[string]any `json:"inputs"`
	Outputs    map[string]any `json:"outputs"`
	Parent     string         `json:"parent,omitempty"`
	ID         string         `json:"id,omitempty"`
	Dependencies []string     `json:"dependencies,omitempty"`
}

// ImportPulumiCheckpoint parses a Pulumi checkpoint JSON byte slice and returns ResourceState entries.
// Only custom resources (not component resources) are included.
func ImportPulumiCheckpoint(data []byte) ([]ResourceState, error) {
	var checkpoint PulumiCheckpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, fmt.Errorf("parse pulumi checkpoint: %w", err)
	}

	if checkpoint.Deployment == nil {
		return nil, fmt.Errorf("pulumi checkpoint has no deployment section")
	}

	var states []ResourceState
	for _, res := range checkpoint.Deployment.Resources {
		if !res.Custom {
			continue
		}
		// Skip the root stack resource.
		if strings.Contains(res.Type, "pulumi:pulumi:Stack") {
			continue
		}

		name := urnToName(res.URN)
		provider := pulumiProviderFromType(res.Type)
		abstractType := pulumiTypeToAbstract(res.Type)

		rs := ResourceState{
			ID:            res.ID,
			Name:          name,
			Type:          abstractType,
			Provider:      provider,
			ProviderID:    res.ID,
			AppliedConfig: res.Inputs,
			Outputs:       res.Outputs,
			Dependencies:  res.Dependencies,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		states = append(states, rs)
	}
	return states, nil
}

// urnToName extracts the resource name from a Pulumi URN.
// Format: urn:pulumi:stack::project::type::name
func urnToName(urn string) string {
	parts := strings.Split(urn, "::")
	if len(parts) >= 4 {
		return parts[len(parts)-1]
	}
	return urn
}

// pulumiProviderFromType extracts the provider name from a Pulumi type string.
// e.g. "aws:rds/instance:Instance" → "aws"
// e.g. "gcp:compute/network:Network" → "gcp"
func pulumiProviderFromType(pulumiType string) string {
	if idx := strings.Index(pulumiType, ":"); idx > 0 {
		pkg := pulumiType[:idx]
		// Normalize common Pulumi package names.
		switch pkg {
		case "aws":
			return "aws"
		case "gcp", "google-native":
			return "gcp"
		case "azure-native", "azure":
			return "azure"
		case "digitalocean":
			return "digitalocean"
		default:
			return pkg
		}
	}
	return "unknown"
}

// pulumiTypeToAbstract maps Pulumi type strings to abstract infra resource types.
// Returns the original Pulumi type if no mapping exists.
func pulumiTypeToAbstract(pulumiType string) string {
	// Pulumi types are in the format "provider:module/resource:Resource".
	// We match on the lowercase type.
	lower := strings.ToLower(pulumiType)

	switch {
	// AWS
	case strings.Contains(lower, "aws:rds"):
		return "infra.database"
	case strings.Contains(lower, "aws:ec2/vpc:vpc"):
		return "infra.vpc"
	case strings.Contains(lower, "aws:ecs/service:service"):
		return "infra.container_service"
	case strings.Contains(lower, "aws:eks/cluster:cluster"):
		return "infra.k8s_cluster"
	case strings.Contains(lower, "aws:elasticache"):
		return "infra.cache"
	case strings.Contains(lower, "aws:lb"):
		return "infra.load_balancer"
	case strings.Contains(lower, "aws:route53"):
		return "infra.dns"
	case strings.Contains(lower, "aws:ecr"):
		return "infra.registry"
	case strings.Contains(lower, "aws:apigatewayv2"):
		return "infra.api_gateway"
	case strings.Contains(lower, "aws:ec2/securitygroup"):
		return "infra.firewall"
	case strings.Contains(lower, "aws:iam/role"):
		return "infra.iam_role"
	case strings.Contains(lower, "aws:s3/bucket"):
		return "infra.storage"
	case strings.Contains(lower, "aws:acm"):
		return "infra.certificate"
	// GCP
	case strings.Contains(lower, "gcp:sql") || strings.Contains(lower, "google-native:sqladmin"):
		return "infra.database"
	case strings.Contains(lower, "gcp:compute/network") || strings.Contains(lower, "google-native:compute/network"):
		return "infra.vpc"
	case strings.Contains(lower, "gcp:cloudrun") || strings.Contains(lower, "google-native:run"):
		return "infra.container_service"
	case strings.Contains(lower, "gcp:container/cluster") || strings.Contains(lower, "google-native:container/cluster"):
		return "infra.k8s_cluster"
	case strings.Contains(lower, "gcp:redis"):
		return "infra.cache"
	case strings.Contains(lower, "gcp:storage/bucket"):
		return "infra.storage"
	case strings.Contains(lower, "gcp:dns/managedzone"):
		return "infra.dns"
	case strings.Contains(lower, "gcp:artifactregistry"):
		return "infra.registry"
	// Azure
	case strings.Contains(lower, "azure-native:dbforpostgresql") || strings.Contains(lower, "azure:postgresql"):
		return "infra.database"
	case strings.Contains(lower, "azure-native:network/virtualnetwork") || strings.Contains(lower, "azure:network/virtualnetwork"):
		return "infra.vpc"
	case strings.Contains(lower, "azure-native:containerinstance") || strings.Contains(lower, "azure:containerservice/group"):
		return "infra.container_service"
	case strings.Contains(lower, "azure-native:containerservice/managedcluster") || strings.Contains(lower, "azure:containerservice/kubernetescluster"):
		return "infra.k8s_cluster"
	case strings.Contains(lower, "azure-native:cache") || strings.Contains(lower, "azure:redis"):
		return "infra.cache"
	case strings.Contains(lower, "azure-native:storage/storageaccount") || strings.Contains(lower, "azure:storage/account"):
		return "infra.storage"
	case strings.Contains(lower, "azure-native:network/loadbalancer") || strings.Contains(lower, "azure:lb"):
		return "infra.load_balancer"
	case strings.Contains(lower, "azure-native:network/dnszone") || strings.Contains(lower, "azure:dns/zone"):
		return "infra.dns"
	case strings.Contains(lower, "azure-native:containerregistry") || strings.Contains(lower, "azure:containerservice/registry"):
		return "infra.registry"
	// DigitalOcean
	case strings.Contains(lower, "digitalocean:index/databasecluster"):
		return "infra.database"
	case strings.Contains(lower, "digitalocean:index/vpc"):
		return "infra.vpc"
	case strings.Contains(lower, "digitalocean:index/app"):
		return "infra.container_service"
	case strings.Contains(lower, "digitalocean:index/kubernetescluster"):
		return "infra.k8s_cluster"
	case strings.Contains(lower, "digitalocean:index/loadbalancer"):
		return "infra.load_balancer"
	case strings.Contains(lower, "digitalocean:index/domain"):
		return "infra.dns"
	case strings.Contains(lower, "digitalocean:index/containerregistry"):
		return "infra.registry"
	case strings.Contains(lower, "digitalocean:index/spacesbucket"):
		return "infra.storage"
	case strings.Contains(lower, "digitalocean:index/firewall"):
		return "infra.firewall"
	case strings.Contains(lower, "digitalocean:index/certificate"):
		return "infra.certificate"
	default:
		return pulumiType
	}
}
