package generator_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoCodeAlone/workflow-plugin-tofu/internal/generator"
)

func TestGenerateHCL_AWS_Database(t *testing.T) {
	dir := t.TempDir()
	specs := []generator.ResourceSpec{
		{Name: "mydb", Type: "infra.database", Provider: "aws", Config: map[string]any{
			"engine": "postgres", "version": "16", "size": "m", "storage_gb": 100,
		}},
	}
	if err := generator.GenerateHCL(specs, "aws", dir); err != nil {
		t.Fatalf("GenerateHCL: %v", err)
	}

	content := readFile(t, filepath.Join(dir, "mydb.tf"))
	assertContains(t, content, `aws_db_instance`)
	assertContains(t, content, `"mydb"`)
	assertContains(t, content, `db.r6g.large`)
}

func TestGenerateHCL_AWS_VPC(t *testing.T) {
	dir := t.TempDir()
	specs := []generator.ResourceSpec{
		{Name: "network", Type: "infra.vpc", Provider: "aws", Config: map[string]any{
			"cidr": "10.0.0.0/16",
		}},
	}
	if err := generator.GenerateHCL(specs, "aws", dir); err != nil {
		t.Fatalf("GenerateHCL: %v", err)
	}

	content := readFile(t, filepath.Join(dir, "network.tf"))
	assertContains(t, content, `aws_vpc`)
	assertContains(t, content, `10.0.0.0/16`)
	assertContains(t, content, `aws_internet_gateway`)
}

func TestGenerateHCL_GCP_Database(t *testing.T) {
	dir := t.TempDir()
	specs := []generator.ResourceSpec{
		{Name: "proddb", Type: "infra.database", Provider: "gcp", Config: map[string]any{
			"engine": "postgres", "size": "l",
		}},
	}
	if err := generator.GenerateHCL(specs, "gcp", dir); err != nil {
		t.Fatalf("GenerateHCL: %v", err)
	}

	content := readFile(t, filepath.Join(dir, "proddb.tf"))
	assertContains(t, content, `google_sql_database_instance`)
	assertContains(t, content, `db-custom-4-16384`)
}

func TestGenerateHCL_GCP_ContainerService(t *testing.T) {
	dir := t.TempDir()
	specs := []generator.ResourceSpec{
		{Name: "api", Type: "infra.container_service", Provider: "gcp", Config: map[string]any{
			"image": "myorg/api:latest",
		}},
	}
	if err := generator.GenerateHCL(specs, "gcp", dir); err != nil {
		t.Fatalf("GenerateHCL: %v", err)
	}

	content := readFile(t, filepath.Join(dir, "api.tf"))
	assertContains(t, content, `google_cloud_run_service`)
	assertContains(t, content, `myorg/api:latest`)
}

func TestGenerateHCL_Azure_Database(t *testing.T) {
	dir := t.TempDir()
	specs := []generator.ResourceSpec{
		{Name: "azdb", Type: "infra.database", Provider: "azure", Config: map[string]any{
			"size": "s",
		}},
	}
	if err := generator.GenerateHCL(specs, "azure", dir); err != nil {
		t.Fatalf("GenerateHCL: %v", err)
	}

	content := readFile(t, filepath.Join(dir, "azdb.tf"))
	assertContains(t, content, `azurerm_postgresql_flexible_server`)
	assertContains(t, content, `GP_Gen5_2`)
}

func TestGenerateHCL_DO_Database(t *testing.T) {
	dir := t.TempDir()
	specs := []generator.ResourceSpec{
		{Name: "dodb", Type: "infra.database", Provider: "digitalocean", Config: map[string]any{
			"engine": "pg", "size": "xs",
		}},
	}
	if err := generator.GenerateHCL(specs, "digitalocean", dir); err != nil {
		t.Fatalf("GenerateHCL: %v", err)
	}

	content := readFile(t, filepath.Join(dir, "dodb.tf"))
	assertContains(t, content, `digitalocean_database_cluster`)
	assertContains(t, content, `db-s-1vcpu-1gb`)
}

func TestGenerateHCL_ProviderBlock_AWS(t *testing.T) {
	dir := t.TempDir()
	if err := generator.GenerateHCL([]generator.ResourceSpec{}, "aws", dir); err != nil {
		t.Fatalf("GenerateHCL: %v", err)
	}

	content := readFile(t, filepath.Join(dir, "provider.tf"))
	assertContains(t, content, `hashicorp/aws`)
	assertContains(t, content, `provider`)
}

func TestGenerateHCL_ProviderBlock_GCP(t *testing.T) {
	dir := t.TempDir()
	if err := generator.GenerateHCL([]generator.ResourceSpec{}, "gcp", dir); err != nil {
		t.Fatalf("GenerateHCL: %v", err)
	}

	content := readFile(t, filepath.Join(dir, "provider.tf"))
	assertContains(t, content, `hashicorp/google`)
}

func TestGenerateHCL_ProviderBlock_Azure(t *testing.T) {
	dir := t.TempDir()
	if err := generator.GenerateHCL([]generator.ResourceSpec{}, "azure", dir); err != nil {
		t.Fatalf("GenerateHCL: %v", err)
	}

	content := readFile(t, filepath.Join(dir, "provider.tf"))
	assertContains(t, content, `hashicorp/azurerm`)
}

func TestGenerateHCL_ProviderBlock_DO(t *testing.T) {
	dir := t.TempDir()
	if err := generator.GenerateHCL([]generator.ResourceSpec{}, "digitalocean", dir); err != nil {
		t.Fatalf("GenerateHCL: %v", err)
	}

	content := readFile(t, filepath.Join(dir, "provider.tf"))
	assertContains(t, content, `digitalocean/digitalocean`)
}

func TestGenerateHCL_AllResourceTypes_AWS(t *testing.T) {
	resourceTypes := []string{
		"infra.database",
		"infra.vpc",
		"infra.container_service",
		"infra.k8s_cluster",
		"infra.cache",
		"infra.load_balancer",
		"infra.dns",
		"infra.registry",
		"infra.api_gateway",
		"infra.firewall",
		"infra.iam_role",
		"infra.storage",
		"infra.certificate",
	}
	for _, rt := range resourceTypes {
		t.Run(rt, func(t *testing.T) {
			dir := t.TempDir()
			specs := []generator.ResourceSpec{
				{Name: "test_resource", Type: rt, Provider: "aws", Config: map[string]any{}},
			}
			if err := generator.GenerateHCL(specs, "aws", dir); err != nil {
				t.Fatalf("GenerateHCL(%s, aws): %v", rt, err)
			}
			if _, err := os.Stat(filepath.Join(dir, "test_resource.tf")); err != nil {
				t.Fatalf("expected test_resource.tf: %v", err)
			}
		})
	}
}

func TestGenerateHCL_AllProviders_AllResourceTypes(t *testing.T) {
	providers := []string{"aws", "gcp", "azure", "digitalocean"}
	resourceTypes := []string{
		"infra.database", "infra.vpc", "infra.container_service", "infra.k8s_cluster",
		"infra.cache", "infra.load_balancer", "infra.dns", "infra.registry",
		"infra.api_gateway", "infra.firewall", "infra.iam_role", "infra.storage",
		"infra.certificate",
	}
	for _, provider := range providers {
		for _, rt := range resourceTypes {
			t.Run(provider+"/"+rt, func(t *testing.T) {
				dir := t.TempDir()
				specs := []generator.ResourceSpec{
					{Name: "res", Type: rt, Provider: provider, Config: map[string]any{}},
				}
				if err := generator.GenerateHCL(specs, provider, dir); err != nil {
					t.Fatalf("GenerateHCL(%s, %s): %v", rt, provider, err)
				}
			})
		}
	}
}

func TestGenerateHCL_UnsupportedProvider(t *testing.T) {
	dir := t.TempDir()
	specs := []generator.ResourceSpec{
		{Name: "db", Type: "infra.database", Provider: "unknown", Config: map[string]any{}},
	}
	if err := generator.GenerateHCL(specs, "unknown", dir); err == nil {
		t.Fatal("expected error for unsupported provider")
	}
}

func TestGenerateHCL_SanitizedFilename(t *testing.T) {
	dir := t.TempDir()
	specs := []generator.ResourceSpec{
		{Name: "my.resource/name", Type: "infra.storage", Provider: "aws", Config: map[string]any{}},
	}
	if err := generator.GenerateHCL(specs, "aws", dir); err != nil {
		t.Fatalf("GenerateHCL: %v", err)
	}
	// File should exist with sanitized name
	entries, _ := os.ReadDir(dir)
	found := false
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tf") && e.Name() != "provider.tf" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected .tf file to be created")
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func assertContains(t *testing.T, content, substr string) {
	t.Helper()
	if !strings.Contains(content, substr) {
		t.Errorf("expected content to contain %q\nContent:\n%s", substr, content)
	}
}
