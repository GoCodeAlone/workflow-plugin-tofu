package generator

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

var doSizingDB = map[string]string{
	"xs": "db-s-1vcpu-1gb",
	"s":  "db-s-1vcpu-2gb",
	"m":  "db-s-2vcpu-4gb",
	"l":  "db-s-4vcpu-8gb",
	"xl": "db-s-8vcpu-16gb",
}

func doDatabase(body *hclwrite.Body, spec ResourceSpec) error {
	size := "m"
	if s, ok := spec.Config["size"].(string); ok {
		size = s
	}
	nodeSize := doSizingDB[size]
	if nodeSize == "" {
		nodeSize = doSizingDB["m"]
	}

	blk := body.AppendNewBlock("resource", []string{"digitalocean_database_cluster", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("engine", strVal(spec.Config, "engine", "pg"))
	b.SetAttributeValue("version", strVal(spec.Config, "version", "16"))
	b.SetAttributeValue("size", cty.StringVal(nodeSize))
	b.SetAttributeValue("region", cty.StringVal("nyc3"))
	b.SetAttributeValue("node_count", cty.NumberIntVal(1))
	return nil
}

func doVPC(body *hclwrite.Body, spec ResourceSpec) error {
	cidr := "10.10.10.0/24"
	if c, ok := spec.Config["cidr"].(string); ok {
		cidr = c
	}

	blk := body.AppendNewBlock("resource", []string{"digitalocean_vpc", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("region", cty.StringVal("nyc3"))
	b.SetAttributeValue("ip_range", cty.StringVal(cidr))
	return nil
}

func doContainerService(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"digitalocean_app", spec.Name})
	b := blk.Body()

	spec_ := b.AppendNewBlock("spec", nil)
	spec_.Body().SetAttributeValue("name", cty.StringVal(spec.Name))
	spec_.Body().SetAttributeValue("region", cty.StringVal("nyc"))

	service := spec_.Body().AppendNewBlock("service", nil)
	sb := service.Body()
	sb.SetAttributeValue("name", cty.StringVal(spec.Name))
	image := "nginx:latest"
	if img, ok := spec.Config["image"].(string); ok {
		image = img
	}
	imgBlk := sb.AppendNewBlock("image", nil)
	imgBlk.Body().SetAttributeValue("registry_type", cty.StringVal("DOCKER_HUB"))
	imgBlk.Body().SetAttributeValue("repository", cty.StringVal(image))
	sb.SetAttributeValue("instance_count", cty.NumberIntVal(replicaCount(spec.Config)))
	sb.SetAttributeValue("instance_size_slug", cty.StringVal("professional-xs"))
	return nil
}

func doK8sCluster(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"digitalocean_kubernetes_cluster", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("region", cty.StringVal("nyc3"))
	b.SetAttributeValue("version", strVal(spec.Config, "version", "1.31"))
	pool := b.AppendNewBlock("node_pool", nil)
	pool.Body().SetAttributeValue("name", cty.StringVal(spec.Name+"-pool"))
	pool.Body().SetAttributeValue("size", cty.StringVal("s-2vcpu-2gb"))
	pool.Body().SetAttributeValue("node_count", cty.NumberIntVal(1))
	return nil
}

func doCache(body *hclwrite.Body, spec ResourceSpec) error {
	nodeSize := "db-s-1vcpu-1gb"
	if s, ok := spec.Config["size"].(string); ok {
		sizes := map[string]string{"xs": "db-s-1vcpu-1gb", "s": "db-s-1vcpu-2gb", "m": "db-s-2vcpu-4gb", "l": "db-s-4vcpu-8gb", "xl": "db-s-8vcpu-16gb"}
		if n, ok := sizes[s]; ok {
			nodeSize = n
		}
	}
	blk := body.AppendNewBlock("resource", []string{"digitalocean_database_cluster", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("engine", cty.StringVal("redis"))
	b.SetAttributeValue("version", strVal(spec.Config, "version", "7"))
	b.SetAttributeValue("size", cty.StringVal(nodeSize))
	b.SetAttributeValue("region", cty.StringVal("nyc3"))
	b.SetAttributeValue("node_count", cty.NumberIntVal(1))
	return nil
}

func doLoadBalancer(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"digitalocean_loadbalancer", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("region", cty.StringVal("nyc3"))
	rule := b.AppendNewBlock("forwarding_rule", nil)
	rule.Body().SetAttributeValue("entry_port", cty.NumberIntVal(80))
	rule.Body().SetAttributeValue("entry_protocol", cty.StringVal("http"))
	rule.Body().SetAttributeValue("target_port", cty.NumberIntVal(80))
	rule.Body().SetAttributeValue("target_protocol", cty.StringVal("http"))
	return nil
}

func doDNS(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"digitalocean_domain", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", strVal(spec.Config, "zone", spec.Name+".example.com"))
	return nil
}

func doRegistry(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"digitalocean_container_registry", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("subscription_tier_slug", cty.StringVal("basic"))
	return nil
}

func doAPIGateway(body *hclwrite.Body, spec ResourceSpec) error {
	// DO uses App Platform routing for API gateway functionality.
	blk := body.AppendNewBlock("resource", []string{"digitalocean_app", spec.Name + "_gw"})
	b := blk.Body()
	spec_ := b.AppendNewBlock("spec", nil)
	spec_.Body().SetAttributeValue("name", cty.StringVal(spec.Name+"-gateway"))
	spec_.Body().SetAttributeValue("region", cty.StringVal("nyc"))
	return nil
}

func doFirewall(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"digitalocean_firewall", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	in := b.AppendNewBlock("inbound_rule", nil)
	in.Body().SetAttributeValue("protocol", cty.StringVal("tcp"))
	in.Body().SetAttributeValue("port_range", cty.StringVal("80"))
	in.Body().SetAttributeValue("source_addresses", cty.ListVal([]cty.Value{cty.StringVal("0.0.0.0/0")}))
	return nil
}

func doIAMRole(body *hclwrite.Body, spec ResourceSpec) error {
	// DO uses API tokens. Use a placeholder token resource via a null_resource.
	blk := body.AppendNewBlock("resource", []string{"digitalocean_token", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("expiry_seconds", cty.NumberIntVal(0))
	return nil
}

func doStorage(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"digitalocean_spaces_bucket", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("region", cty.StringVal("nyc3"))
	return nil
}

func doCertificate(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"digitalocean_certificate", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("type", cty.StringVal("lets_encrypt"))
	b.SetAttributeValue("domains", cty.ListVal([]cty.Value{
		strVal(spec.Config, "domain", "example.com"),
	}))
	return nil
}
