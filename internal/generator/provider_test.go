package generator_test

import (
	"path/filepath"
	"testing"

	"github.com/GoCodeAlone/workflow-plugin-tofu/internal/generator"
)

func TestAWSContainerService_Fargate(t *testing.T) {
	dir := t.TempDir()
	specs := []generator.ResourceSpec{
		{Name: "app", Type: "infra.container_service", Provider: "aws", Config: map[string]any{
			"image": "myapp:v1", "size": "l", "replicas": 3,
		}},
	}
	if err := generator.GenerateHCL(specs, "aws", dir); err != nil {
		t.Fatalf("GenerateHCL: %v", err)
	}

	content := readFile(t, filepath.Join(dir, "app.tf"))
	assertContains(t, content, `aws_ecs_cluster`)
	assertContains(t, content, `aws_ecs_task_definition`)
	assertContains(t, content, `aws_ecs_service`)
	assertContains(t, content, `FARGATE`)
	assertContains(t, content, `myapp:v1`)
}

func TestAWSK8sCluster(t *testing.T) {
	dir := t.TempDir()
	specs := []generator.ResourceSpec{
		{Name: "mycluster", Type: "infra.k8s_cluster", Provider: "aws", Config: map[string]any{}},
	}
	if err := generator.GenerateHCL(specs, "aws", dir); err != nil {
		t.Fatalf("GenerateHCL: %v", err)
	}

	content := readFile(t, filepath.Join(dir, "mycluster.tf"))
	assertContains(t, content, `aws_eks_cluster`)
}

func TestGCPK8sCluster(t *testing.T) {
	dir := t.TempDir()
	specs := []generator.ResourceSpec{
		{Name: "gke", Type: "infra.k8s_cluster", Provider: "gcp", Config: map[string]any{}},
	}
	if err := generator.GenerateHCL(specs, "gcp", dir); err != nil {
		t.Fatalf("GenerateHCL: %v", err)
	}

	content := readFile(t, filepath.Join(dir, "gke.tf"))
	assertContains(t, content, `google_container_cluster`)
}

func TestAzureK8sCluster(t *testing.T) {
	dir := t.TempDir()
	specs := []generator.ResourceSpec{
		{Name: "aks", Type: "infra.k8s_cluster", Provider: "azure", Config: map[string]any{}},
	}
	if err := generator.GenerateHCL(specs, "azure", dir); err != nil {
		t.Fatalf("GenerateHCL: %v", err)
	}

	content := readFile(t, filepath.Join(dir, "aks.tf"))
	assertContains(t, content, `azurerm_kubernetes_cluster`)
}

func TestDOK8sCluster(t *testing.T) {
	dir := t.TempDir()
	specs := []generator.ResourceSpec{
		{Name: "doks", Type: "infra.k8s_cluster", Provider: "digitalocean", Config: map[string]any{}},
	}
	if err := generator.GenerateHCL(specs, "digitalocean", dir); err != nil {
		t.Fatalf("GenerateHCL: %v", err)
	}

	content := readFile(t, filepath.Join(dir, "doks.tf"))
	assertContains(t, content, `digitalocean_kubernetes_cluster`)
}

func TestAWSSizingDB_AllTiers(t *testing.T) {
	tiers := map[string]string{
		"xs": "db.t3.micro",
		"s":  "db.t3.small",
		"m":  "db.r6g.large",
		"l":  "db.r6g.xlarge",
		"xl": "db.r6g.2xlarge",
	}
	for size, expectedClass := range tiers {
		t.Run("size="+size, func(t *testing.T) {
			dir := t.TempDir()
			specs := []generator.ResourceSpec{
				{Name: "db", Type: "infra.database", Provider: "aws", Config: map[string]any{"size": size}},
			}
			if err := generator.GenerateHCL(specs, "aws", dir); err != nil {
				t.Fatalf("GenerateHCL: %v", err)
			}
			content := readFile(t, filepath.Join(dir, "db.tf"))
			assertContains(t, content, expectedClass)
		})
	}
}

func TestGCPSizingDB_AllTiers(t *testing.T) {
	tiers := map[string]string{
		"xs": "db-f1-micro",
		"s":  "db-g1-small",
		"m":  "db-custom-2-8192",
		"l":  "db-custom-4-16384",
		"xl": "db-custom-8-32768",
	}
	for size, expectedTier := range tiers {
		t.Run("size="+size, func(t *testing.T) {
			dir := t.TempDir()
			specs := []generator.ResourceSpec{
				{Name: "db", Type: "infra.database", Provider: "gcp", Config: map[string]any{"size": size}},
			}
			if err := generator.GenerateHCL(specs, "gcp", dir); err != nil {
				t.Fatalf("GenerateHCL: %v", err)
			}
			content := readFile(t, filepath.Join(dir, "db.tf"))
			assertContains(t, content, expectedTier)
		})
	}
}

func TestDOContainerService(t *testing.T) {
	dir := t.TempDir()
	specs := []generator.ResourceSpec{
		{Name: "doapp", Type: "infra.container_service", Provider: "digitalocean", Config: map[string]any{
			"image": "nginx:1.25", "replicas": 2,
		}},
	}
	if err := generator.GenerateHCL(specs, "digitalocean", dir); err != nil {
		t.Fatalf("GenerateHCL: %v", err)
	}

	content := readFile(t, filepath.Join(dir, "doapp.tf"))
	assertContains(t, content, `digitalocean_app`)
	assertContains(t, content, `nginx:1.25`)
}

func TestAzureContainerService(t *testing.T) {
	dir := t.TempDir()
	specs := []generator.ResourceSpec{
		{Name: "aciapp", Type: "infra.container_service", Provider: "azure", Config: map[string]any{
			"image": "myapp:latest",
		}},
	}
	if err := generator.GenerateHCL(specs, "azure", dir); err != nil {
		t.Fatalf("GenerateHCL: %v", err)
	}

	content := readFile(t, filepath.Join(dir, "aciapp.tf"))
	assertContains(t, content, `azurerm_container_group`)
	assertContains(t, content, `myapp:latest`)
}

func TestMultipleResources(t *testing.T) {
	dir := t.TempDir()
	specs := []generator.ResourceSpec{
		{Name: "vpc1", Type: "infra.vpc", Provider: "aws", Config: map[string]any{"cidr": "10.0.0.0/16"}},
		{Name: "db1", Type: "infra.database", Provider: "aws", Config: map[string]any{"size": "s"}},
		{Name: "cache1", Type: "infra.cache", Provider: "aws", Config: map[string]any{"size": "s"}},
	}
	if err := generator.GenerateHCL(specs, "aws", dir); err != nil {
		t.Fatalf("GenerateHCL: %v", err)
	}

	// Each resource gets its own file.
	for _, name := range []string{"vpc1", "db1", "cache1"} {
		content := readFile(t, filepath.Join(dir, name+".tf"))
		if content == "" {
			t.Errorf("expected non-empty %s.tf", name)
		}
	}
}
