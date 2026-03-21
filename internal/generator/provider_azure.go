package generator

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

var azureSizingDB = map[string]string{
	"xs": "GP_Gen5_1",
	"s":  "GP_Gen5_2",
	"m":  "GP_Gen5_4",
	"l":  "MO_Gen5_8",
	"xl": "MO_Gen5_16",
}

func azureDatabase(body *hclwrite.Body, spec ResourceSpec) error {
	size := "m"
	if s, ok := spec.Config["size"].(string); ok {
		size = s
	}
	sku := azureSizingDB[size]
	if sku == "" {
		sku = azureSizingDB["m"]
	}

	blk := body.AppendNewBlock("resource", []string{"azurerm_postgresql_flexible_server", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("resource_group_name", cty.StringVal("rg-"+spec.Name))
	b.SetAttributeValue("location", cty.StringVal("East US"))
	b.SetAttributeValue("administrator_login", cty.StringVal("admin"))
	b.SetAttributeValue("administrator_password", cty.StringVal("CHANGE_ME"))
	b.SetAttributeValue("sku_name", cty.StringVal(sku))
	b.SetAttributeValue("version", strVal(spec.Config, "version", "16"))
	b.SetAttributeValue("storage_mb", cty.NumberIntVal(storageGB(spec.Config, 32)*1024))
	return nil
}

func azureVPC(body *hclwrite.Body, spec ResourceSpec) error {
	cidr := "10.0.0.0/16"
	if c, ok := spec.Config["cidr"].(string); ok {
		cidr = c
	}

	blk := body.AppendNewBlock("resource", []string{"azurerm_virtual_network", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("resource_group_name", cty.StringVal("rg-"+spec.Name))
	b.SetAttributeValue("location", cty.StringVal("East US"))
	b.SetAttributeValue("address_space", cty.ListVal([]cty.Value{cty.StringVal(cidr)}))
	return nil
}

func azureContainerService(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"azurerm_container_group", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("resource_group_name", cty.StringVal("rg-"+spec.Name))
	b.SetAttributeValue("location", cty.StringVal("East US"))
	b.SetAttributeValue("ip_address_type", cty.StringVal("Public"))
	b.SetAttributeValue("os_type", cty.StringVal("Linux"))

	image := "nginx:latest"
	if img, ok := spec.Config["image"].(string); ok {
		image = img
	}
	container := b.AppendNewBlock("container", nil)
	cb := container.Body()
	cb.SetAttributeValue("name", cty.StringVal(spec.Name))
	cb.SetAttributeValue("image", cty.StringVal(image))
	cb.SetAttributeValue("cpu", cty.NumberFloatVal(1.0))
	cb.SetAttributeValue("memory", cty.NumberFloatVal(2.0))
	return nil
}

func azureK8sCluster(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"azurerm_kubernetes_cluster", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("location", cty.StringVal("East US"))
	b.SetAttributeValue("resource_group_name", cty.StringVal("rg-"+spec.Name))
	b.SetAttributeValue("dns_prefix", cty.StringVal(spec.Name))
	pool := b.AppendNewBlock("default_node_pool", nil)
	pool.Body().SetAttributeValue("name", cty.StringVal("default"))
	pool.Body().SetAttributeValue("node_count", cty.NumberIntVal(1))
	pool.Body().SetAttributeValue("vm_size", cty.StringVal("Standard_D2_v2"))
	identity := b.AppendNewBlock("identity", nil)
	identity.Body().SetAttributeValue("type", cty.StringVal("SystemAssigned"))
	return nil
}

func azureCache(body *hclwrite.Body, spec ResourceSpec) error {
	skuName := "Standard"
	family := "C"
	capacity := int64(1)
	if s, ok := spec.Config["size"].(string); ok {
		caps := map[string]int64{"xs": 0, "s": 1, "m": 2, "l": 3, "xl": 4}
		if c, ok := caps[s]; ok {
			capacity = c
		}
	}
	blk := body.AppendNewBlock("resource", []string{"azurerm_redis_cache", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("location", cty.StringVal("East US"))
	b.SetAttributeValue("resource_group_name", cty.StringVal("rg-"+spec.Name))
	b.SetAttributeValue("capacity", cty.NumberIntVal(capacity))
	b.SetAttributeValue("family", cty.StringVal(family))
	b.SetAttributeValue("sku_name", cty.StringVal(skuName))
	return nil
}

func azureLoadBalancer(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"azurerm_lb", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("location", cty.StringVal("East US"))
	b.SetAttributeValue("resource_group_name", cty.StringVal("rg-"+spec.Name))
	b.SetAttributeValue("sku", cty.StringVal("Standard"))
	return nil
}

func azureDNS(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"azurerm_dns_zone", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", strVal(spec.Config, "zone", spec.Name+".example.com"))
	b.SetAttributeValue("resource_group_name", cty.StringVal("rg-"+spec.Name))
	return nil
}

func azureRegistry(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"azurerm_container_registry", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("resource_group_name", cty.StringVal("rg-"+spec.Name))
	b.SetAttributeValue("location", cty.StringVal("East US"))
	b.SetAttributeValue("sku", cty.StringVal("Standard"))
	return nil
}

func azureAPIGateway(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"azurerm_api_management", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("location", cty.StringVal("East US"))
	b.SetAttributeValue("resource_group_name", cty.StringVal("rg-"+spec.Name))
	b.SetAttributeValue("publisher_name", cty.StringVal("GoCodeAlone"))
	b.SetAttributeValue("publisher_email", cty.StringVal("admin@example.com"))
	b.SetAttributeValue("sku_name", cty.StringVal("Developer_1"))
	return nil
}

func azureFirewall(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"azurerm_network_security_group", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("location", cty.StringVal("East US"))
	b.SetAttributeValue("resource_group_name", cty.StringVal("rg-"+spec.Name))
	return nil
}

func azureIAMRole(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"azurerm_user_assigned_identity", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("resource_group_name", cty.StringVal("rg-"+spec.Name))
	b.SetAttributeValue("location", cty.StringVal("East US"))
	return nil
}

func azureStorage(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"azurerm_storage_account", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("resource_group_name", cty.StringVal("rg-"+spec.Name))
	b.SetAttributeValue("location", cty.StringVal("East US"))
	b.SetAttributeValue("account_tier", cty.StringVal("Standard"))
	b.SetAttributeValue("account_replication_type", cty.StringVal("LRS"))
	return nil
}

func azureCertificate(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"azurerm_app_service_certificate", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("resource_group_name", cty.StringVal("rg-"+spec.Name))
	b.SetAttributeValue("location", cty.StringVal("East US"))
	b.SetAttributeValue("pfx_blob", cty.StringVal(""))
	return nil
}
