package generator

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

var gcpSizingDB = map[string]string{
	"xs": "db-f1-micro",
	"s":  "db-g1-small",
	"m":  "db-custom-2-8192",
	"l":  "db-custom-4-16384",
	"xl": "db-custom-8-32768",
}

func gcpDatabase(body *hclwrite.Body, spec ResourceSpec) error {
	size := "m"
	if s, ok := spec.Config["size"].(string); ok {
		size = s
	}
	tier := gcpSizingDB[size]
	if tier == "" {
		tier = gcpSizingDB["m"]
	}

	blk := body.AppendNewBlock("resource", []string{"google_sql_database_instance", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("database_version", cty.StringVal("POSTGRES_16"))
	settings := b.AppendNewBlock("settings", nil)
	settings.Body().SetAttributeValue("tier", cty.StringVal(tier))
	return nil
}

func gcpVPC(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"google_compute_network", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("auto_create_subnetworks", cty.False)
	return nil
}

func gcpContainerService(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"google_cloud_run_service", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("location", cty.StringVal("us-central1"))

	template := b.AppendNewBlock("template", nil)
	containers := template.Body().AppendNewBlock("spec", nil).Body().AppendNewBlock("containers", nil)
	image := "nginx:latest"
	if img, ok := spec.Config["image"].(string); ok {
		image = img
	}
	containers.Body().SetAttributeValue("image", cty.StringVal(image))

	traffic := b.AppendNewBlock("traffic", nil)
	traffic.Body().SetAttributeValue("percent", cty.NumberIntVal(100))
	traffic.Body().SetAttributeValue("latest_revision", cty.True)
	return nil
}

func gcpK8sCluster(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"google_container_cluster", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("location", cty.StringVal("us-central1"))
	b.SetAttributeValue("initial_node_count", cty.NumberIntVal(1))
	b.SetAttributeValue("remove_default_node_pool", cty.True)
	return nil
}

func gcpCache(body *hclwrite.Body, spec ResourceSpec) error {
	tier := "STANDARD_HA"
	memorySizeMb := int64(1024)
	if s, ok := spec.Config["size"].(string); ok {
		sizes := map[string]int64{"xs": 256, "s": 1024, "m": 4096, "l": 16384, "xl": 65536}
		if mb, ok := sizes[s]; ok {
			memorySizeMb = mb
		}
	}
	blk := body.AppendNewBlock("resource", []string{"google_redis_instance", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("tier", cty.StringVal(tier))
	b.SetAttributeValue("memory_size_gb", cty.NumberIntVal(memorySizeMb/1024))
	b.SetAttributeValue("redis_version", strVal(spec.Config, "version", "REDIS_7_0"))
	return nil
}

func gcpLoadBalancer(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"google_compute_global_forwarding_rule", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("port_range", cty.StringVal("80"))
	b.SetAttributeValue("target", cty.StringVal(""))
	return nil
}

func gcpDNS(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"google_dns_managed_zone", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("dns_name", strVal(spec.Config, "zone", spec.Name+".example.com."))
	return nil
}

func gcpRegistry(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"google_artifact_registry_repository", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("repository_id", cty.StringVal(spec.Name))
	b.SetAttributeValue("location", cty.StringVal("us-central1"))
	b.SetAttributeValue("format", cty.StringVal("DOCKER"))
	return nil
}

func gcpAPIGateway(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"google_api_gateway_api", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("api_id", cty.StringVal(spec.Name))
	return nil
}

func gcpFirewall(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"google_compute_firewall", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("network", cty.StringVal("default"))
	allow := b.AppendNewBlock("allow", nil)
	allow.Body().SetAttributeValue("protocol", cty.StringVal("tcp"))
	return nil
}

func gcpIAMRole(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"google_service_account", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("account_id", cty.StringVal(spec.Name))
	b.SetAttributeValue("display_name", cty.StringVal(spec.Name))
	return nil
}

func gcpStorage(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"google_storage_bucket", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("location", cty.StringVal("US"))
	b.SetAttributeValue("force_destroy", cty.False)
	return nil
}

func gcpCertificate(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"google_compute_managed_ssl_certificate", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	managed := b.AppendNewBlock("managed", nil)
	managed.Body().SetAttributeValue("domains", cty.ListVal([]cty.Value{
		strVal(spec.Config, "domain", "example.com"),
	}))
	return nil
}
