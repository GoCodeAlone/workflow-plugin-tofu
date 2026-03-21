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
