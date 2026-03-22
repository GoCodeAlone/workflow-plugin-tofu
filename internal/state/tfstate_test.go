package state_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/GoCodeAlone/workflow-plugin-tofu/internal/state"
)

// sampleTFState is a minimal but representative .tfstate fixture.
var sampleTFState = []byte(`{
  "version": 4,
  "terraform_version": "1.6.0",
  "serial": 5,
  "lineage": "abc123",
  "resources": [
    {
      "mode": "managed",
      "type": "aws_db_instance",
      "name": "mydb",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [
        {
          "schema_version": 2,
          "attributes": {
            "id": "mydb-abc123",
            "identifier": "mydb",
            "instance_class": "db.r6g.large",
            "engine": "postgres",
            "engine_version": "16.1",
            "endpoint": "mydb.abc123.us-east-1.rds.amazonaws.com:5432",
            "address": "mydb.abc123.us-east-1.rds.amazonaws.com",
            "port": 5432,
            "arn": "arn:aws:rds:us-east-1:123456789:db:mydb"
          },
          "dependencies": []
        }
      ]
    },
    {
      "mode": "managed",
      "type": "aws_vpc",
      "name": "network",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [
        {
          "schema_version": 1,
          "attributes": {
            "id": "vpc-12345",
            "cidr_block": "10.0.0.0/16",
            "enable_dns_hostnames": true,
            "enable_dns_support": true
          },
          "dependencies": []
        }
      ]
    },
    {
      "mode": "data",
      "type": "aws_ami",
      "name": "latest",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {"id": "ami-xxx"}
        }
      ]
    }
  ]
}`)

func TestImportTFState_BasicParse(t *testing.T) {
	states, err := state.ImportTFState(sampleTFState)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}

	// data resources are excluded, so we expect 2 managed resources.
	if len(states) != 2 {
		t.Errorf("expected 2 states (managed only), got %d", len(states))
	}
}

func TestImportTFState_DatabaseResource(t *testing.T) {
	states, err := state.ImportTFState(sampleTFState)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}

	var found bool
	for _, s := range states {
		if s.Name == "mydb" {
			found = true
			if s.Type != "infra.database" {
				t.Errorf("expected type infra.database, got %q", s.Type)
			}
			if s.Provider != "aws" {
				t.Errorf("expected provider aws, got %q", s.Provider)
			}
			if s.ProviderID != "mydb-abc123" {
				t.Errorf("expected provider ID mydb-abc123, got %q", s.ProviderID)
			}
		}
	}
	if !found {
		t.Fatal("expected resource named 'mydb'")
	}
}

func TestImportTFState_VPCResource(t *testing.T) {
	states, err := state.ImportTFState(sampleTFState)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}

	var found bool
	for _, s := range states {
		if s.Name == "network" {
			found = true
			if s.Type != "infra.vpc" {
				t.Errorf("expected type infra.vpc, got %q", s.Type)
			}
			if s.ProviderID != "vpc-12345" {
				t.Errorf("expected provider ID vpc-12345, got %q", s.ProviderID)
			}
		}
	}
	if !found {
		t.Fatal("expected resource named 'network'")
	}
}

func TestImportTFState_InvalidJSON(t *testing.T) {
	_, err := state.ImportTFState([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestImportTFState_EmptyResources(t *testing.T) {
	data := []byte(`{"version":4,"terraform_version":"1.6.0","serial":1,"lineage":"x","resources":[]}`)
	states, err := state.ImportTFState(data)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}
	if len(states) != 0 {
		t.Errorf("expected 0 states, got %d", len(states))
	}
}

func TestExportTFState_RoundTrip(t *testing.T) {
	imported, err := state.ImportTFState(sampleTFState)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}

	exported, err := state.ExportTFState(imported)
	if err != nil {
		t.Fatalf("ExportTFState: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(exported, &result); err != nil {
		t.Fatalf("exported tfstate is invalid JSON: %v", err)
	}

	if result["version"].(float64) != 4 {
		t.Errorf("expected version 4, got %v", result["version"])
	}
	if _, ok := result["resources"]; !ok {
		t.Error("expected 'resources' key in exported tfstate")
	}
}

func TestExportTFState_ContainsProviderID(t *testing.T) {
	imported, err := state.ImportTFState(sampleTFState)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}

	exported, err := state.ExportTFState(imported)
	if err != nil {
		t.Fatalf("ExportTFState: %v", err)
	}

	content := string(exported)
	if !strings.Contains(content, "mydb-abc123") {
		t.Error("expected provider ID 'mydb-abc123' in exported tfstate")
	}
	if !strings.Contains(content, "vpc-12345") {
		t.Error("expected provider ID 'vpc-12345' in exported tfstate")
	}
}

func TestExportTFState_ResourceTypeMapping(t *testing.T) {
	imported, err := state.ImportTFState(sampleTFState)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}

	exported, err := state.ExportTFState(imported)
	if err != nil {
		t.Fatalf("ExportTFState: %v", err)
	}

	if !strings.Contains(string(exported), "aws_db_instance") {
		t.Error("expected 'aws_db_instance' in exported tfstate (round-trip mapping)")
	}
	if !strings.Contains(string(exported), "aws_vpc") {
		t.Error("expected 'aws_vpc' in exported tfstate (round-trip mapping)")
	}
}

func TestExportTFState_Empty(t *testing.T) {
	exported, err := state.ExportTFState(nil)
	if err != nil {
		t.Fatalf("ExportTFState with nil: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(exported, &result); err != nil {
		t.Fatalf("exported tfstate is invalid JSON: %v", err)
	}
	resources := result["resources"].([]any)
	if len(resources) != 0 {
		t.Errorf("expected empty resources, got %d", len(resources))
	}
}

// samplePulumiCheckpoint is a minimal Pulumi checkpoint fixture.
var samplePulumiCheckpoint = []byte(`{
  "version": 3,
  "deployment": {
    "manifest": {
      "time": "2024-01-01T00:00:00.000000000Z",
      "version": "v3.100.0"
    },
    "resources": [
      {
        "urn": "urn:pulumi:prod::myapp::pulumi:pulumi:Stack::myapp-prod",
        "custom": false,
        "type": "pulumi:pulumi:Stack",
        "inputs": {},
        "outputs": {}
      },
      {
        "urn": "urn:pulumi:prod::myapp::aws:rds/instance:Instance::mydb",
        "custom": true,
        "id": "mydb-pulumi-123",
        "type": "aws:rds/instance:Instance",
        "inputs": {
          "instanceClass": "db.r6g.large",
          "engine": "postgres",
          "engineVersion": "16.1",
          "allocatedStorage": 100
        },
        "outputs": {
          "endpoint": "mydb-pulumi.abc.us-east-1.rds.amazonaws.com:5432",
          "address": "mydb-pulumi.abc.us-east-1.rds.amazonaws.com"
        },
        "dependencies": []
      },
      {
        "urn": "urn:pulumi:prod::myapp::gcp:storage/bucket:Bucket::assets",
        "custom": true,
        "id": "assets-bucket-xyz",
        "type": "gcp:storage/bucket:Bucket",
        "inputs": {
          "location": "US",
          "forceDestroy": false
        },
        "outputs": {
          "url": "gs://assets-bucket-xyz",
          "selfLink": "https://www.googleapis.com/storage/v1/b/assets-bucket-xyz"
        },
        "dependencies": []
      }
    ]
  }
}`)

func TestImportPulumiCheckpoint_BasicParse(t *testing.T) {
	states, err := state.ImportPulumiCheckpoint(samplePulumiCheckpoint)
	if err != nil {
		t.Fatalf("ImportPulumiCheckpoint: %v", err)
	}

	// Stack resource is excluded, so 2 custom resources.
	if len(states) != 2 {
		t.Errorf("expected 2 states, got %d", len(states))
	}
}

func TestImportPulumiCheckpoint_DatabaseResource(t *testing.T) {
	states, err := state.ImportPulumiCheckpoint(samplePulumiCheckpoint)
	if err != nil {
		t.Fatalf("ImportPulumiCheckpoint: %v", err)
	}

	var found bool
	for _, s := range states {
		if s.Name == "mydb" {
			found = true
			if s.Type != "infra.database" {
				t.Errorf("expected type infra.database, got %q", s.Type)
			}
			if s.Provider != "aws" {
				t.Errorf("expected provider aws, got %q", s.Provider)
			}
			if s.ProviderID != "mydb-pulumi-123" {
				t.Errorf("expected provider ID mydb-pulumi-123, got %q", s.ProviderID)
			}
		}
	}
	if !found {
		t.Fatal("expected resource named 'mydb'")
	}
}

func TestImportPulumiCheckpoint_GCPStorage(t *testing.T) {
	states, err := state.ImportPulumiCheckpoint(samplePulumiCheckpoint)
	if err != nil {
		t.Fatalf("ImportPulumiCheckpoint: %v", err)
	}

	var found bool
	for _, s := range states {
		if s.Name == "assets" {
			found = true
			if s.Type != "infra.storage" {
				t.Errorf("expected type infra.storage, got %q", s.Type)
			}
			if s.Provider != "gcp" {
				t.Errorf("expected provider gcp, got %q", s.Provider)
			}
		}
	}
	if !found {
		t.Fatal("expected resource named 'assets'")
	}
}

func TestImportPulumiCheckpoint_InvalidJSON(t *testing.T) {
	_, err := state.ImportPulumiCheckpoint([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestImportPulumiCheckpoint_MissingDeployment(t *testing.T) {
	data := []byte(`{"version": 3}`)
	_, err := state.ImportPulumiCheckpoint(data)
	if err == nil {
		t.Fatal("expected error for missing deployment section")
	}
}

func TestImportPulumiCheckpoint_URNNameExtraction(t *testing.T) {
	checkpoint := []byte(`{
  "version": 3,
  "deployment": {
    "manifest": {"time": "2024-01-01T00:00:00Z", "version": "v3.0.0"},
    "resources": [
      {
        "urn": "urn:pulumi:staging::webapp::aws:ec2/vpc:Vpc::main-network",
        "custom": true,
        "id": "vpc-stage-001",
        "type": "aws:ec2/vpc:Vpc",
        "inputs": {},
        "outputs": {},
        "dependencies": []
      }
    ]
  }
}`)
	states, err := state.ImportPulumiCheckpoint(checkpoint)
	if err != nil {
		t.Fatalf("ImportPulumiCheckpoint: %v", err)
	}
	if len(states) != 1 {
		t.Fatalf("expected 1 state, got %d", len(states))
	}
	if states[0].Name != "main-network" {
		t.Errorf("expected name 'main-network', got %q", states[0].Name)
	}
	if states[0].Type != "infra.vpc" {
		t.Errorf("expected type infra.vpc, got %q", states[0].Type)
	}
}

// ---------------------------------------------------------------------------
// Multi-provider TF state fixtures
// ---------------------------------------------------------------------------

// awsAllTypesTFState covers all 13 AWS resource types mapped to abstract types.
var awsAllTypesTFState = []byte(`{
  "version": 4,
  "terraform_version": "1.6.0",
  "serial": 10,
  "lineage": "aws-all",
  "resources": [
    {
      "mode": "managed", "type": "aws_ecs_service", "name": "api_svc",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "arn:aws:ecs:us-east-1:123:service/api-svc", "cluster": "arn:aws:ecs:us-east-1:123:cluster/prod"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "aws_eks_cluster", "name": "k8s",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "prod-cluster", "arn": "arn:aws:eks:us-east-1:123:cluster/prod-cluster", "endpoint": "https://eks.example.com", "version": "1.29"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "aws_db_instance", "name": "main_db",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [{"schema_version": 2, "attributes": {"id": "main-db-id", "endpoint": "main-db.abc.us-east-1.rds.amazonaws.com:5432", "address": "main-db.abc.us-east-1.rds.amazonaws.com", "port": 5432, "arn": "arn:aws:rds:us-east-1:123:db:main-db"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "aws_elasticache_cluster", "name": "redis",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "prod-redis", "cluster_address": "prod-redis.cache.amazonaws.com", "engine": "redis", "engine_version": "7.0"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "aws_vpc", "name": "main_vpc",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [{"schema_version": 1, "attributes": {"id": "vpc-main-001", "cidr_block": "10.0.0.0/16"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "aws_lb", "name": "alb",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/app/prod-alb/abc", "dns_name": "prod-alb.us-east-1.elb.amazonaws.com", "arn": "arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/app/prod-alb/abc", "zone_id": "Z35SXDOTRQ7X7K"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "aws_route53_zone", "name": "public_zone",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "Z1234567890", "name": "example.com"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "aws_route53_record", "name": "api_record",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [{"schema_version": 2, "attributes": {"id": "Z1234567890_api.example.com_A", "name": "api.example.com", "type": "A"}, "dependencies": ["aws_route53_zone.public_zone"]}]
    },
    {
      "mode": "managed", "type": "aws_ecr_repository", "name": "app_repo",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "app-repo", "repository_url": "123.dkr.ecr.us-east-1.amazonaws.com/app-repo", "arn": "arn:aws:ecr:us-east-1:123:repository/app-repo"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "aws_apigatewayv2_api", "name": "http_api",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "abc123api", "name": "prod-api", "api_endpoint": "https://abc123api.execute-api.us-east-1.amazonaws.com"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "aws_security_group", "name": "web_sg",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [{"schema_version": 1, "attributes": {"id": "sg-web-001", "name": "web-sg", "vpc_id": "vpc-main-001"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "aws_iam_role", "name": "app_role",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "app-role", "arn": "arn:aws:iam::123:role/app-role", "name": "app-role"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "aws_s3_bucket", "name": "assets",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "prod-assets-bucket", "bucket": "prod-assets-bucket", "bucket_regional_domain_name": "prod-assets-bucket.s3.us-east-1.amazonaws.com"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "aws_acm_certificate", "name": "tls_cert",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "arn:aws:acm:us-east-1:123:certificate/abc-def", "domain_name": "example.com", "arn": "arn:aws:acm:us-east-1:123:certificate/abc-def"}, "dependencies": []}]
    }
  ]
}`)

// gcpAllTypesTFState covers all 13 GCP resource types.
var gcpAllTypesTFState = []byte(`{
  "version": 4,
  "terraform_version": "1.6.0",
  "serial": 10,
  "lineage": "gcp-all",
  "resources": [
    {
      "mode": "managed", "type": "google_cloud_run_service", "name": "api",
      "provider": "provider[\"registry.terraform.io/hashicorp/google\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "locations/us-central1/namespaces/my-project/services/api", "name": "api", "project": "my-project"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "google_container_cluster", "name": "gke",
      "provider": "provider[\"registry.terraform.io/hashicorp/google\"]",
      "instances": [{"schema_version": 2, "attributes": {"id": "projects/my-project/locations/us-central1/clusters/prod-gke", "name": "prod-gke", "endpoint": "34.123.45.67"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "google_sql_database_instance", "name": "db",
      "provider": "provider[\"registry.terraform.io/hashicorp/google\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "my-project:us-central1:prod-db", "connection_name": "my-project:us-central1:prod-db", "first_ip_address": "10.1.2.3", "service_account_email_address": "p123@developer.gserviceaccount.com"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "google_redis_instance", "name": "cache",
      "provider": "provider[\"registry.terraform.io/hashicorp/google\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "projects/my-project/locations/us-central1/instances/prod-redis", "host": "10.1.2.4", "port": 6379}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "google_compute_network", "name": "vpc",
      "provider": "provider[\"registry.terraform.io/hashicorp/google\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "projects/my-project/global/networks/prod-vpc", "name": "prod-vpc", "self_link": "https://www.googleapis.com/compute/v1/projects/my-project/global/networks/prod-vpc"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "google_compute_forwarding_rule", "name": "lb",
      "provider": "provider[\"registry.terraform.io/hashicorp/google\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "projects/my-project/regions/us-central1/forwardingRules/prod-lb", "name": "prod-lb", "ip_address": "34.100.200.1"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "google_dns_managed_zone", "name": "dns",
      "provider": "provider[\"registry.terraform.io/hashicorp/google\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "projects/my-project/managedZones/example-com", "name": "example-com", "dns_name": "example.com."}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "google_artifact_registry_repository", "name": "repo",
      "provider": "provider[\"registry.terraform.io/hashicorp/google\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "projects/my-project/locations/us-central1/repositories/app-repo", "name": "app-repo", "format": "DOCKER"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "google_api_gateway_api", "name": "apigw",
      "provider": "provider[\"registry.terraform.io/hashicorp/google\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "projects/my-project/locations/global/apis/prod-api", "api_id": "prod-api"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "google_compute_firewall", "name": "fw",
      "provider": "provider[\"registry.terraform.io/hashicorp/google\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "projects/my-project/global/firewalls/allow-web", "name": "allow-web"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "google_service_account", "name": "svc_acct",
      "provider": "provider[\"registry.terraform.io/hashicorp/google\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "projects/my-project/serviceAccounts/app@my-project.iam.gserviceaccount.com", "email": "app@my-project.iam.gserviceaccount.com"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "google_storage_bucket", "name": "assets",
      "provider": "provider[\"registry.terraform.io/hashicorp/google\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "prod-assets", "url": "gs://prod-assets", "self_link": "https://www.googleapis.com/storage/v1/b/prod-assets"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "google_compute_ssl_certificate", "name": "cert",
      "provider": "provider[\"registry.terraform.io/hashicorp/google\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "projects/my-project/global/sslCertificates/prod-cert", "name": "prod-cert"}, "dependencies": []}]
    }
  ]
}`)

// azureAllTypesTFState covers all 13 Azure resource types.
var azureAllTypesTFState = []byte(`{
  "version": 4,
  "terraform_version": "1.6.0",
  "serial": 10,
  "lineage": "azure-all",
  "resources": [
    {
      "mode": "managed", "type": "azurerm_container_group", "name": "api",
      "provider": "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "/subscriptions/sub-id/resourceGroups/prod-rg/providers/Microsoft.ContainerInstance/containerGroups/api", "name": "api", "ip_address": "10.0.1.5"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "azurerm_kubernetes_cluster", "name": "aks",
      "provider": "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
      "instances": [{"schema_version": 2, "attributes": {"id": "/subscriptions/sub-id/resourceGroups/prod-rg/providers/Microsoft.ContainerService/managedClusters/prod-aks", "name": "prod-aks", "fqdn": "prod-aks.azmk8s.io"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "azurerm_mssql_server", "name": "sqldb",
      "provider": "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "/subscriptions/sub-id/resourceGroups/prod-rg/providers/Microsoft.Sql/servers/prod-sql", "name": "prod-sql", "fully_qualified_domain_name": "prod-sql.database.windows.net"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "azurerm_redis_cache", "name": "cache",
      "provider": "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
      "instances": [{"schema_version": 1, "attributes": {"id": "/subscriptions/sub-id/resourceGroups/prod-rg/providers/Microsoft.Cache/Redis/prod-redis", "hostname": "prod-redis.redis.cache.windows.net", "ssl_port": 6380}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "azurerm_virtual_network", "name": "vnet",
      "provider": "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "/subscriptions/sub-id/resourceGroups/prod-rg/providers/Microsoft.Network/virtualNetworks/prod-vnet", "name": "prod-vnet", "address_space": ["10.0.0.0/8"]}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "azurerm_lb", "name": "lb",
      "provider": "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "/subscriptions/sub-id/resourceGroups/prod-rg/providers/Microsoft.Network/loadBalancers/prod-lb", "name": "prod-lb"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "azurerm_dns_zone", "name": "zone",
      "provider": "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "/subscriptions/sub-id/resourceGroups/prod-rg/providers/Microsoft.Network/dnsZones/example.com", "name": "example.com"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "azurerm_container_registry", "name": "acr",
      "provider": "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
      "instances": [{"schema_version": 2, "attributes": {"id": "/subscriptions/sub-id/resourceGroups/prod-rg/providers/Microsoft.ContainerRegistry/registries/prodacr", "login_server": "prodacr.azurecr.io"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "azurerm_api_management", "name": "apim",
      "provider": "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "/subscriptions/sub-id/resourceGroups/prod-rg/providers/Microsoft.ApiManagement/service/prod-apim", "gateway_url": "https://prod-apim.azure-api.net"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "azurerm_network_security_group", "name": "nsg",
      "provider": "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "/subscriptions/sub-id/resourceGroups/prod-rg/providers/Microsoft.Network/networkSecurityGroups/prod-nsg", "name": "prod-nsg"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "azurerm_user_assigned_identity", "name": "identity",
      "provider": "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "/subscriptions/sub-id/resourceGroups/prod-rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/prod-identity", "client_id": "client-uuid", "principal_id": "principal-uuid"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "azurerm_storage_account", "name": "storage",
      "provider": "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
      "instances": [{"schema_version": 4, "attributes": {"id": "/subscriptions/sub-id/resourceGroups/prod-rg/providers/Microsoft.Storage/storageAccounts/prodstore", "primary_blob_endpoint": "https://prodstore.blob.core.windows.net/", "primary_access_key": "base64key=="}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "azurerm_app_service_certificate", "name": "cert",
      "provider": "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "/subscriptions/sub-id/resourceGroups/prod-rg/providers/Microsoft.Web/certificates/prod-cert", "thumbprint": "AABBCCDDEEFF"}, "dependencies": []}]
    }
  ]
}`)

// doAllTypesTFState covers all DigitalOcean resource types.
var doAllTypesTFState = []byte(`{
  "version": 4,
  "terraform_version": "1.6.0",
  "serial": 10,
  "lineage": "do-all",
  "resources": [
    {
      "mode": "managed", "type": "digitalocean_app", "name": "api",
      "provider": "provider[\"registry.terraform.io/digitalocean/digitalocean\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "do-app-uuid-001", "default_ingress": "https://api-xyz.ondigitalocean.app"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "digitalocean_kubernetes_cluster", "name": "k8s",
      "provider": "provider[\"registry.terraform.io/digitalocean/digitalocean\"]",
      "instances": [{"schema_version": 3, "attributes": {"id": "do-k8s-uuid-001", "name": "prod-k8s", "endpoint": "https://do-k8s-uuid.k8s.ondigitalocean.com"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "digitalocean_database_cluster", "name": "db",
      "provider": "provider[\"registry.terraform.io/digitalocean/digitalocean\"]",
      "instances": [{"schema_version": 1, "attributes": {"id": "do-db-uuid-001", "engine": "pg", "host": "db.example.db.ondigitalocean.com", "port": 25060, "uri": "postgresql://user:pass@db.example.db.ondigitalocean.com:25060/defaultdb", "private_uri": "postgresql://user:pass@private-db.example.db.ondigitalocean.com:25060/defaultdb"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "digitalocean_database_cluster", "name": "redis",
      "provider": "provider[\"registry.terraform.io/digitalocean/digitalocean\"]",
      "instances": [{"schema_version": 1, "attributes": {"id": "do-redis-uuid-001", "engine": "redis", "host": "redis.example.db.ondigitalocean.com", "port": 25061, "uri": "rediss://user:pass@redis.example.db.ondigitalocean.com:25061", "private_uri": "rediss://user:pass@private-redis.example.db.ondigitalocean.com:25061"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "digitalocean_vpc", "name": "vpc",
      "provider": "provider[\"registry.terraform.io/digitalocean/digitalocean\"]",
      "instances": [{"schema_version": 1, "attributes": {"id": "do-vpc-uuid-001", "name": "prod-vpc", "ip_range": "10.10.0.0/16", "region": "nyc3"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "digitalocean_loadbalancer", "name": "lb",
      "provider": "provider[\"registry.terraform.io/digitalocean/digitalocean\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "do-lb-uuid-001", "name": "prod-lb", "ip": "159.89.200.1"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "digitalocean_domain", "name": "dns",
      "provider": "provider[\"registry.terraform.io/digitalocean/digitalocean\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "example.com", "name": "example.com", "ttl": 1800}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "digitalocean_container_registry", "name": "registry",
      "provider": "provider[\"registry.terraform.io/digitalocean/digitalocean\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "prod-registry", "name": "prod-registry", "endpoint": "registry.digitalocean.com/prod-registry"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "digitalocean_firewall", "name": "fw",
      "provider": "provider[\"registry.terraform.io/digitalocean/digitalocean\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "do-fw-uuid-001", "name": "prod-fw", "status": "succeeded"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "digitalocean_spaces_bucket", "name": "assets",
      "provider": "provider[\"registry.terraform.io/digitalocean/digitalocean\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "prod-assets", "bucket_domain_name": "prod-assets.nyc3.digitaloceanspaces.com", "endpoint": "nyc3.digitaloceanspaces.com"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "digitalocean_certificate", "name": "cert",
      "provider": "provider[\"registry.terraform.io/digitalocean/digitalocean\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "do-cert-uuid-001", "name": "prod-cert", "domains": ["example.com"]}, "dependencies": []}]
    }
  ]
}`)

// ---------------------------------------------------------------------------
// AWS all-types tests
// ---------------------------------------------------------------------------

func TestImportTFState_AWS_AllTypes(t *testing.T) {
	states, err := state.ImportTFState(awsAllTypesTFState)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}

	want := map[string]string{
		"api_svc":    "infra.container_service",
		"k8s":        "infra.k8s_cluster",
		"main_db":    "infra.database",
		"redis":      "infra.cache",
		"main_vpc":   "infra.vpc",
		"alb":        "infra.load_balancer",
		"public_zone": "infra.dns",
		"api_record": "infra.dns",
		"app_repo":   "infra.registry",
		"http_api":   "infra.api_gateway",
		"web_sg":     "infra.firewall",
		"app_role":   "infra.iam_role",
		"assets":     "infra.storage",
		"tls_cert":   "infra.certificate",
	}

	if len(states) != len(want) {
		t.Errorf("expected %d resources, got %d", len(want), len(states))
	}

	found := make(map[string]string)
	for _, s := range states {
		found[s.Name] = s.Type
		if s.Provider != "aws" {
			t.Errorf("resource %q: expected provider 'aws', got %q", s.Name, s.Provider)
		}
	}
	for name, wantType := range want {
		if gotType, ok := found[name]; !ok {
			t.Errorf("resource %q not found in import result", name)
		} else if gotType != wantType {
			t.Errorf("resource %q: expected type %q, got %q", name, wantType, gotType)
		}
	}
}

func TestImportTFState_AWS_ProviderIDExtracted(t *testing.T) {
	states, err := state.ImportTFState(awsAllTypesTFState)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}

	wantIDs := map[string]string{
		"api_svc":    "arn:aws:ecs:us-east-1:123:service/api-svc",
		"k8s":        "prod-cluster",
		"main_db":    "main-db-id",
		"redis":      "prod-redis",
		"main_vpc":   "vpc-main-001",
		"alb":        "arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/app/prod-alb/abc",
		"public_zone": "Z1234567890",
		"api_record": "Z1234567890_api.example.com_A",
		"app_repo":   "app-repo",
		"http_api":   "abc123api",
		"web_sg":     "sg-web-001",
		"app_role":   "app-role",
		"assets":     "prod-assets-bucket",
		"tls_cert":   "arn:aws:acm:us-east-1:123:certificate/abc-def",
	}

	for _, s := range states {
		if want, ok := wantIDs[s.Name]; ok {
			if s.ProviderID != want {
				t.Errorf("resource %q: expected ProviderID %q, got %q", s.Name, want, s.ProviderID)
			}
		}
	}
}

func TestImportTFState_AWS_AttributesPreserved(t *testing.T) {
	states, err := state.ImportTFState(awsAllTypesTFState)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}

	for _, s := range states {
		switch s.Name {
		case "main_db":
			if s.Outputs["endpoint"] == nil {
				t.Error("aws_db_instance: expected 'endpoint' in Outputs")
			}
			if s.Outputs["arn"] == nil {
				t.Error("aws_db_instance: expected 'arn' in Outputs")
			}
		case "main_vpc":
			if s.Outputs["cidr_block"] == nil {
				t.Error("aws_vpc: expected 'cidr_block' in Outputs")
			}
		case "alb":
			if s.Outputs["dns_name"] == nil {
				t.Error("aws_lb: expected 'dns_name' in Outputs")
			}
		case "assets":
			if s.Outputs["bucket"] == nil {
				t.Error("aws_s3_bucket: expected 'bucket' in Outputs")
			}
		}
	}
}

// ---------------------------------------------------------------------------
// GCP all-types tests
// ---------------------------------------------------------------------------

func TestImportTFState_GCP_AllTypes(t *testing.T) {
	states, err := state.ImportTFState(gcpAllTypesTFState)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}

	want := map[string]string{
		"api":      "infra.container_service",
		"gke":      "infra.k8s_cluster",
		"db":       "infra.database",
		"cache":    "infra.cache",
		"vpc":      "infra.vpc",
		"lb":       "infra.load_balancer",
		"dns":      "infra.dns",
		"repo":     "infra.registry",
		"apigw":    "infra.api_gateway",
		"fw":       "infra.firewall",
		"svc_acct": "infra.iam_role",
		"assets":   "infra.storage",
		"cert":     "infra.certificate",
	}

	if len(states) != len(want) {
		t.Errorf("expected %d resources, got %d", len(want), len(states))
	}

	found := make(map[string]string)
	for _, s := range states {
		found[s.Name] = s.Type
		if s.Provider != "gcp" {
			t.Errorf("resource %q: expected provider 'gcp', got %q", s.Name, s.Provider)
		}
	}
	for name, wantType := range want {
		if gotType, ok := found[name]; !ok {
			t.Errorf("resource %q not found in import result", name)
		} else if gotType != wantType {
			t.Errorf("resource %q: expected type %q, got %q", name, wantType, gotType)
		}
	}
}

func TestImportTFState_GCP_ProviderIDExtracted(t *testing.T) {
	states, err := state.ImportTFState(gcpAllTypesTFState)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}

	wantIDs := map[string]string{
		"api":    "locations/us-central1/namespaces/my-project/services/api",
		"gke":    "projects/my-project/locations/us-central1/clusters/prod-gke",
		"db":     "my-project:us-central1:prod-db",
		"cache":  "projects/my-project/locations/us-central1/instances/prod-redis",
		"vpc":    "projects/my-project/global/networks/prod-vpc",
		"lb":     "projects/my-project/regions/us-central1/forwardingRules/prod-lb",
		"assets": "prod-assets",
		"cert":   "projects/my-project/global/sslCertificates/prod-cert",
	}

	for _, s := range states {
		if want, ok := wantIDs[s.Name]; ok {
			if s.ProviderID != want {
				t.Errorf("resource %q: expected ProviderID %q, got %q", s.Name, want, s.ProviderID)
			}
		}
	}
}

func TestImportTFState_GCP_AttributesPreserved(t *testing.T) {
	states, err := state.ImportTFState(gcpAllTypesTFState)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}

	for _, s := range states {
		switch s.Name {
		case "db":
			if s.Outputs["connection_name"] == nil {
				t.Error("google_sql_database_instance: expected 'connection_name' in Outputs")
			}
			if s.Outputs["first_ip_address"] == nil {
				t.Error("google_sql_database_instance: expected 'first_ip_address' in Outputs")
			}
		case "assets":
			if s.Outputs["url"] == nil {
				t.Error("google_storage_bucket: expected 'url' in Outputs")
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Azure all-types tests
// ---------------------------------------------------------------------------

func TestImportTFState_Azure_AllTypes(t *testing.T) {
	states, err := state.ImportTFState(azureAllTypesTFState)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}

	want := map[string]string{
		"api":      "infra.container_service",
		"aks":      "infra.k8s_cluster",
		"sqldb":    "infra.database",
		"cache":    "infra.cache",
		"vnet":     "infra.vpc",
		"lb":       "infra.load_balancer",
		"zone":     "infra.dns",
		"acr":      "infra.registry",
		"apim":     "infra.api_gateway",
		"nsg":      "infra.firewall",
		"identity": "infra.iam_role",
		"storage":  "infra.storage",
		"cert":     "infra.certificate",
	}

	if len(states) != len(want) {
		t.Errorf("expected %d resources, got %d", len(want), len(states))
	}

	found := make(map[string]string)
	for _, s := range states {
		found[s.Name] = s.Type
		if s.Provider != "azurerm" {
			t.Errorf("resource %q: expected provider 'azurerm', got %q", s.Name, s.Provider)
		}
	}
	for name, wantType := range want {
		if gotType, ok := found[name]; !ok {
			t.Errorf("resource %q not found in import result", name)
		} else if gotType != wantType {
			t.Errorf("resource %q: expected type %q, got %q", name, wantType, gotType)
		}
	}
}

func TestImportTFState_Azure_ProviderIDExtracted(t *testing.T) {
	states, err := state.ImportTFState(azureAllTypesTFState)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}

	wantIDs := map[string]string{
		"sqldb":   "/subscriptions/sub-id/resourceGroups/prod-rg/providers/Microsoft.Sql/servers/prod-sql",
		"vnet":    "/subscriptions/sub-id/resourceGroups/prod-rg/providers/Microsoft.Network/virtualNetworks/prod-vnet",
		"storage": "/subscriptions/sub-id/resourceGroups/prod-rg/providers/Microsoft.Storage/storageAccounts/prodstore",
	}

	for _, s := range states {
		if want, ok := wantIDs[s.Name]; ok {
			if s.ProviderID != want {
				t.Errorf("resource %q: expected ProviderID %q, got %q", s.Name, want, s.ProviderID)
			}
		}
	}
}

func TestImportTFState_Azure_AttributesPreserved(t *testing.T) {
	states, err := state.ImportTFState(azureAllTypesTFState)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}

	for _, s := range states {
		switch s.Name {
		case "storage":
			if s.Outputs["primary_blob_endpoint"] == nil {
				t.Error("azurerm_storage_account: expected 'primary_blob_endpoint' in Outputs")
			}
		}
	}
}

// ---------------------------------------------------------------------------
// DigitalOcean all-types tests
// ---------------------------------------------------------------------------

func TestImportTFState_DO_AllTypes(t *testing.T) {
	states, err := state.ImportTFState(doAllTypesTFState)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}

	// Both db and redis use digitalocean_database_cluster → infra.database (no cache alias for DO).
	want := map[string]string{
		"api":      "infra.container_service",
		"k8s":      "infra.k8s_cluster",
		"db":       "infra.database",
		"redis":    "infra.database",
		"vpc":      "infra.vpc",
		"lb":       "infra.load_balancer",
		"dns":      "infra.dns",
		"registry": "infra.registry",
		"fw":       "infra.firewall",
		"assets":   "infra.storage",
		"cert":     "infra.certificate",
	}

	if len(states) != len(want) {
		t.Errorf("expected %d resources, got %d", len(want), len(states))
	}

	found := make(map[string]string)
	for _, s := range states {
		found[s.Name] = s.Type
		if s.Provider != "digitalocean" {
			t.Errorf("resource %q: expected provider 'digitalocean', got %q", s.Name, s.Provider)
		}
	}
	for name, wantType := range want {
		if gotType, ok := found[name]; !ok {
			t.Errorf("resource %q not found in import result", name)
		} else if gotType != wantType {
			t.Errorf("resource %q: expected type %q, got %q", name, wantType, gotType)
		}
	}
}

func TestImportTFState_DO_ProviderIDExtracted(t *testing.T) {
	states, err := state.ImportTFState(doAllTypesTFState)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}

	wantIDs := map[string]string{
		"api":      "do-app-uuid-001",
		"k8s":      "do-k8s-uuid-001",
		"db":       "do-db-uuid-001",
		"redis":    "do-redis-uuid-001",
		"vpc":      "do-vpc-uuid-001",
		"lb":       "do-lb-uuid-001",
		"dns":      "example.com",
		"registry": "prod-registry",
		"fw":       "do-fw-uuid-001",
		"assets":   "prod-assets",
		"cert":     "do-cert-uuid-001",
	}

	for _, s := range states {
		if want, ok := wantIDs[s.Name]; ok {
			if s.ProviderID != want {
				t.Errorf("resource %q: expected ProviderID %q, got %q", s.Name, want, s.ProviderID)
			}
		}
	}
}

func TestImportTFState_DO_AttributesPreserved(t *testing.T) {
	states, err := state.ImportTFState(doAllTypesTFState)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}

	for _, s := range states {
		switch s.Name {
		case "db":
			if s.Outputs["host"] == nil {
				t.Error("digitalocean_database_cluster: expected 'host' in Outputs")
			}
			if s.Outputs["uri"] == nil {
				t.Error("digitalocean_database_cluster: expected 'uri' in Outputs")
			}
		case "assets":
			if s.Outputs["bucket_domain_name"] == nil {
				t.Error("digitalocean_spaces_bucket: expected 'bucket_domain_name' in Outputs")
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Complex TF state scenarios
// ---------------------------------------------------------------------------

func TestImportTFState_WithModules(t *testing.T) {
	data := []byte(`{
  "version": 4,
  "terraform_version": "1.6.0",
  "serial": 3,
  "lineage": "modules-test",
  "resources": [
    {
      "module": "module.networking",
      "mode": "managed", "type": "aws_vpc", "name": "main",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [{"schema_version": 1, "attributes": {"id": "vpc-module-001", "cidr_block": "10.1.0.0/16"}, "dependencies": []}]
    },
    {
      "module": "module.app",
      "mode": "managed", "type": "aws_ecs_service", "name": "web",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "arn:aws:ecs:us-east-1:123:service/web"}, "dependencies": ["module.networking.aws_vpc.main"]}]
    }
  ]
}`)
	states, err := state.ImportTFState(data)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}
	if len(states) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(states))
	}

	for _, s := range states {
		switch s.Name {
		case "main":
			if s.Type != "infra.vpc" {
				t.Errorf("module vpc: expected infra.vpc, got %q", s.Type)
			}
			if s.ProviderID != "vpc-module-001" {
				t.Errorf("module vpc: expected ProviderID 'vpc-module-001', got %q", s.ProviderID)
			}
		case "web":
			if s.Type != "infra.container_service" {
				t.Errorf("module ecs: expected infra.container_service, got %q", s.Type)
			}
		default:
			t.Errorf("unexpected resource name %q", s.Name)
		}
	}
}

func TestImportTFState_WithCount(t *testing.T) {
	data := []byte(`{
  "version": 4,
  "terraform_version": "1.6.0",
  "serial": 2,
  "lineage": "count-test",
  "resources": [
    {
      "mode": "managed", "type": "aws_s3_bucket", "name": "logs",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [
        {"schema_version": 0, "attributes": {"id": "logs-bucket-0", "bucket": "logs-bucket-0"}, "dependencies": []},
        {"schema_version": 0, "attributes": {"id": "logs-bucket-1", "bucket": "logs-bucket-1"}, "dependencies": []},
        {"schema_version": 0, "attributes": {"id": "logs-bucket-2", "bucket": "logs-bucket-2"}, "dependencies": []}
      ]
    }
  ]
}`)
	states, err := state.ImportTFState(data)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}
	if len(states) != 3 {
		t.Fatalf("expected 3 instances from count resource, got %d", len(states))
	}

	ids := make(map[string]bool)
	for _, s := range states {
		if s.Type != "infra.storage" {
			t.Errorf("count instance: expected infra.storage, got %q", s.Type)
		}
		if s.Provider != "aws" {
			t.Errorf("count instance: expected provider 'aws', got %q", s.Provider)
		}
		ids[s.ProviderID] = true
	}
	for _, want := range []string{"logs-bucket-0", "logs-bucket-1", "logs-bucket-2"} {
		if !ids[want] {
			t.Errorf("expected instance with ProviderID %q", want)
		}
	}
}

func TestImportTFState_WithForEach(t *testing.T) {
	data := []byte(`{
  "version": 4,
  "terraform_version": "1.6.0",
  "serial": 2,
  "lineage": "foreach-test",
  "resources": [
    {
      "mode": "managed", "type": "aws_security_group", "name": "zones",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [
        {"schema_version": 1, "attributes": {"id": "sg-us-east", "name": "us-east"}, "dependencies": []},
        {"schema_version": 1, "attributes": {"id": "sg-eu-west", "name": "eu-west"}, "dependencies": []},
        {"schema_version": 1, "attributes": {"id": "sg-ap-south", "name": "ap-south"}, "dependencies": []}
      ]
    }
  ]
}`)
	states, err := state.ImportTFState(data)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}
	if len(states) != 3 {
		t.Fatalf("expected 3 for_each instances, got %d", len(states))
	}
	for _, s := range states {
		if s.Type != "infra.firewall" {
			t.Errorf("for_each instance: expected infra.firewall, got %q", s.Type)
		}
	}
}

func TestImportTFState_WithDependencies(t *testing.T) {
	data := []byte(`{
  "version": 4,
  "terraform_version": "1.6.0",
  "serial": 3,
  "lineage": "deps-test",
  "resources": [
    {
      "mode": "managed", "type": "aws_vpc", "name": "base",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [{"schema_version": 1, "attributes": {"id": "vpc-base"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "aws_security_group", "name": "web",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [{"schema_version": 1, "attributes": {"id": "sg-web"}, "dependencies": ["aws_vpc.base"]}]
    },
    {
      "mode": "managed", "type": "aws_db_instance", "name": "db",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [{"schema_version": 2, "attributes": {"id": "db-prod"}, "dependencies": ["aws_vpc.base", "aws_security_group.web"]}]
    }
  ]
}`)
	states, err := state.ImportTFState(data)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}
	if len(states) != 3 {
		t.Fatalf("expected 3 resources, got %d", len(states))
	}

	for _, s := range states {
		switch s.Name {
		case "base":
			if len(s.Dependencies) != 0 {
				t.Errorf("vpc has no deps, expected empty, got %v", s.Dependencies)
			}
		case "web":
			if len(s.Dependencies) != 1 || s.Dependencies[0] != "aws_vpc.base" {
				t.Errorf("web sg: expected dep [aws_vpc.base], got %v", s.Dependencies)
			}
		case "db":
			if len(s.Dependencies) != 2 {
				t.Errorf("db: expected 2 deps, got %v", s.Dependencies)
			}
		}
	}
}

func TestImportTFState_MixedProviders(t *testing.T) {
	data := []byte(`{
  "version": 4,
  "terraform_version": "1.6.0",
  "serial": 5,
  "lineage": "mixed-providers",
  "resources": [
    {
      "mode": "managed", "type": "aws_vpc", "name": "aws_net",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [{"schema_version": 1, "attributes": {"id": "vpc-aws-001"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "aws_s3_bucket", "name": "aws_bucket",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "my-bucket", "bucket": "my-bucket"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "google_compute_network", "name": "gcp_net",
      "provider": "provider[\"registry.terraform.io/hashicorp/google\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "projects/p/global/networks/gcp-vpc"}, "dependencies": []}]
    },
    {
      "mode": "managed", "type": "google_storage_bucket", "name": "gcp_bucket",
      "provider": "provider[\"registry.terraform.io/hashicorp/google\"]",
      "instances": [{"schema_version": 0, "attributes": {"id": "gcp-assets", "url": "gs://gcp-assets", "self_link": "https://..."}, "dependencies": []}]
    }
  ]
}`)
	states, err := state.ImportTFState(data)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}
	if len(states) != 4 {
		t.Fatalf("expected 4 resources, got %d", len(states))
	}

	providers := make(map[string]int)
	for _, s := range states {
		providers[s.Provider]++
	}
	if providers["aws"] != 2 {
		t.Errorf("expected 2 aws resources, got %d", providers["aws"])
	}
	if providers["gcp"] != 2 {
		t.Errorf("expected 2 gcp resources, got %d", providers["gcp"])
	}
}

// ---------------------------------------------------------------------------
// Export round-trip tests for all providers
// ---------------------------------------------------------------------------

func testExportRoundTrip(t *testing.T, fixture []byte, providerName string) {
	t.Helper()

	imported, err := state.ImportTFState(fixture)
	if err != nil {
		t.Fatalf("ImportTFState: %v", err)
	}
	if len(imported) == 0 {
		t.Fatal("import produced 0 resources")
	}

	exported, err := state.ExportTFState(imported)
	if err != nil {
		t.Fatalf("ExportTFState: %v", err)
	}

	// Re-import the exported state.
	reimported, err := state.ImportTFState(exported)
	if err != nil {
		t.Fatalf("re-ImportTFState: %v", err)
	}

	if len(reimported) != len(imported) {
		t.Errorf("round-trip: expected %d resources, got %d", len(imported), len(reimported))
	}

	// Build maps by name+type for comparison.
	orig := make(map[string]string)
	for _, s := range imported {
		orig[s.Name] = s.Type
	}
	for _, s := range reimported {
		if s.Provider != providerName {
			t.Errorf("round-trip resource %q: expected provider %q, got %q", s.Name, providerName, s.Provider)
		}
		if orig[s.Name] != s.Type {
			t.Errorf("round-trip resource %q: type changed from %q to %q", s.Name, orig[s.Name], s.Type)
		}
	}

	// Verify exported JSON is valid.
	var raw map[string]any
	if err := json.Unmarshal(exported, &raw); err != nil {
		t.Fatalf("exported JSON invalid: %v", err)
	}
	if raw["version"].(float64) != 4 {
		t.Errorf("expected version 4, got %v", raw["version"])
	}
}

func TestExportTFState_RoundTrip_AWS(t *testing.T) {
	testExportRoundTrip(t, awsAllTypesTFState, "aws")
}

func TestExportTFState_RoundTrip_GCP(t *testing.T) {
	testExportRoundTrip(t, gcpAllTypesTFState, "gcp")
}

func TestExportTFState_RoundTrip_Azure(t *testing.T) {
	testExportRoundTrip(t, azureAllTypesTFState, "azurerm")
}

func TestExportTFState_RoundTrip_DO(t *testing.T) {
	testExportRoundTrip(t, doAllTypesTFState, "digitalocean")
}

// ---------------------------------------------------------------------------
// Pulumi multi-provider fixtures (Azure + DigitalOcean)
// ---------------------------------------------------------------------------

var azurePulumiCheckpoint = []byte(`{
  "version": 3,
  "deployment": {
    "manifest": {"time": "2024-06-01T00:00:00Z", "version": "v3.100.0"},
    "resources": [
      {
        "urn": "urn:pulumi:prod::infra::pulumi:pulumi:Stack::infra-prod",
        "custom": false,
        "type": "pulumi:pulumi:Stack",
        "inputs": {}, "outputs": {}
      },
      {
        "urn": "urn:pulumi:prod::infra::azure:containerservice/group:Group::api-containers",
        "custom": true,
        "id": "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.ContainerInstance/containerGroups/api-containers",
        "type": "azure:containerservice/group:Group",
        "inputs": {"resourceGroupName": "rg", "osType": "Linux"},
        "outputs": {"ipAddress": "10.0.0.5"},
        "dependencies": []
      },
      {
        "urn": "urn:pulumi:prod::infra::azure:sql/server:Server::prod-sql",
        "custom": true,
        "id": "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Sql/servers/prod-sql",
        "type": "azure:sql/server:Server",
        "inputs": {"resourceGroupName": "rg", "administratorLogin": "sqladmin"},
        "outputs": {"fullyQualifiedDomainName": "prod-sql.database.windows.net"},
        "dependencies": []
      },
      {
        "urn": "urn:pulumi:prod::infra::azure:network/virtualNetwork:VirtualNetwork::prod-vnet",
        "custom": true,
        "id": "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/prod-vnet",
        "type": "azure:network/virtualNetwork:VirtualNetwork",
        "inputs": {"resourceGroupName": "rg", "addressSpaces": ["10.0.0.0/16"]},
        "outputs": {"id": "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/prod-vnet"},
        "dependencies": []
      }
    ]
  }
}`)

var doPulumiCheckpoint = []byte(`{
  "version": 3,
  "deployment": {
    "manifest": {"time": "2024-06-01T00:00:00Z", "version": "v3.100.0"},
    "resources": [
      {
        "urn": "urn:pulumi:prod::infra::pulumi:pulumi:Stack::infra-prod",
        "custom": false,
        "type": "pulumi:pulumi:Stack",
        "inputs": {}, "outputs": {}
      },
      {
        "urn": "urn:pulumi:prod::infra::digitalocean:index/app:App::api-app",
        "custom": true,
        "id": "do-app-uuid-pulumi",
        "type": "digitalocean:index/app:App",
        "inputs": {"spec": {"name": "api-app", "region": "nyc"}},
        "outputs": {"defaultIngress": "https://api-app.ondigitalocean.app"},
        "dependencies": []
      },
      {
        "urn": "urn:pulumi:prod::infra::digitalocean:index/databaseCluster:DatabaseCluster::prod-db",
        "custom": true,
        "id": "do-db-uuid-pulumi",
        "type": "digitalocean:index/databaseCluster:DatabaseCluster",
        "inputs": {"engine": "pg", "version": "15", "size": "db-s-1vcpu-1gb", "region": "nyc3", "nodeCount": 1},
        "outputs": {"host": "prod-db.db.ondigitalocean.com", "port": 25060, "uri": "postgresql://..."},
        "dependencies": []
      }
    ]
  }
}`)

func TestImportPulumiCheckpoint_Azure_AllTypes(t *testing.T) {
	states, err := state.ImportPulumiCheckpoint(azurePulumiCheckpoint)
	if err != nil {
		t.Fatalf("ImportPulumiCheckpoint: %v", err)
	}
	if len(states) != 3 {
		t.Fatalf("expected 3 custom resources (stack excluded), got %d", len(states))
	}

	want := map[string]string{
		"api-containers": "infra.container_service",
		"prod-sql":       "infra.database",
		"prod-vnet":      "infra.vpc",
	}
	for _, s := range states {
		if s.Provider != "azure" {
			t.Errorf("resource %q: expected provider 'azure', got %q", s.Name, s.Provider)
		}
		if wantType, ok := want[s.Name]; ok {
			if s.Type != wantType {
				t.Errorf("resource %q: expected type %q, got %q", s.Name, wantType, s.Type)
			}
		} else {
			t.Errorf("unexpected resource name %q", s.Name)
		}
	}
}

func TestImportPulumiCheckpoint_Azure_ProviderIDExtracted(t *testing.T) {
	states, err := state.ImportPulumiCheckpoint(azurePulumiCheckpoint)
	if err != nil {
		t.Fatalf("ImportPulumiCheckpoint: %v", err)
	}

	for _, s := range states {
		if s.ProviderID == "" {
			t.Errorf("resource %q: ProviderID should not be empty", s.Name)
		}
		if s.Name == "prod-sql" && !strings.Contains(s.ProviderID, "prod-sql") {
			t.Errorf("prod-sql: expected ProviderID to contain 'prod-sql', got %q", s.ProviderID)
		}
	}
}

func TestImportPulumiCheckpoint_DO_AllTypes(t *testing.T) {
	states, err := state.ImportPulumiCheckpoint(doPulumiCheckpoint)
	if err != nil {
		t.Fatalf("ImportPulumiCheckpoint: %v", err)
	}
	if len(states) != 2 {
		t.Fatalf("expected 2 custom resources (stack excluded), got %d", len(states))
	}

	want := map[string]string{
		"api-app": "infra.container_service",
		"prod-db": "infra.database",
	}
	for _, s := range states {
		if s.Provider != "digitalocean" {
			t.Errorf("resource %q: expected provider 'digitalocean', got %q", s.Name, s.Provider)
		}
		if wantType, ok := want[s.Name]; ok {
			if s.Type != wantType {
				t.Errorf("resource %q: expected type %q, got %q", s.Name, wantType, s.Type)
			}
		} else {
			t.Errorf("unexpected resource name %q", s.Name)
		}
	}
}

func TestImportPulumiCheckpoint_DO_ProviderIDExtracted(t *testing.T) {
	states, err := state.ImportPulumiCheckpoint(doPulumiCheckpoint)
	if err != nil {
		t.Fatalf("ImportPulumiCheckpoint: %v", err)
	}

	wantIDs := map[string]string{
		"api-app": "do-app-uuid-pulumi",
		"prod-db": "do-db-uuid-pulumi",
	}
	for _, s := range states {
		if want, ok := wantIDs[s.Name]; ok {
			if s.ProviderID != want {
				t.Errorf("resource %q: expected ProviderID %q, got %q", s.Name, want, s.ProviderID)
			}
		}
	}
}
