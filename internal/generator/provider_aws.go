package generator

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// awsSizingDB maps abstract sizes to AWS RDS instance classes.
var awsSizingDB = map[string]string{
	"xs": "db.t3.micro",
	"s":  "db.t3.small",
	"m":  "db.r6g.large",
	"l":  "db.r6g.xlarge",
	"xl": "db.r6g.2xlarge",
}

// awsSizingApp maps abstract sizes to AWS Fargate CPU/memory pairs.
var awsSizingApp = map[string][2]string{
	"xs": {"256", "512"},
	"s":  {"512", "1024"},
	"m":  {"1024", "2048"},
	"l":  {"2048", "4096"},
	"xl": {"4096", "8192"},
}

func awsDatabase(body *hclwrite.Body, spec ResourceSpec) error {
	size := "m"
	if s, ok := spec.Config["size"].(string); ok {
		size = s
	}
	instanceClass := awsSizingDB[size]
	if instanceClass == "" {
		instanceClass = awsSizingDB["m"]
	}

	blk := body.AppendNewBlock("resource", []string{"aws_db_instance", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("identifier", cty.StringVal(spec.Name))
	b.SetAttributeValue("instance_class", cty.StringVal(instanceClass))
	b.SetAttributeValue("engine", strVal(spec.Config, "engine", "postgres"))
	b.SetAttributeValue("engine_version", strVal(spec.Config, "version", "16"))
	b.SetAttributeValue("allocated_storage", cty.NumberIntVal(storageGB(spec.Config, 20)))
	b.SetAttributeValue("db_name", cty.StringVal(spec.Name))
	b.SetAttributeValue("username", cty.StringVal("admin"))
	b.SetAttributeValue("password", cty.StringVal("CHANGE_ME"))
	b.SetAttributeValue("skip_final_snapshot", cty.True)
	b.SetAttributeValue("multi_az", boolVal(spec.Config, "ha", false))
	return nil
}

func awsVPC(body *hclwrite.Body, spec ResourceSpec) error {
	cidr := "10.0.0.0/16"
	if c, ok := spec.Config["cidr"].(string); ok {
		cidr = c
	}

	vpcBlk := body.AppendNewBlock("resource", []string{"aws_vpc", spec.Name})
	vpcBlk.Body().SetAttributeValue("cidr_block", cty.StringVal(cidr))
	vpcBlk.Body().SetAttributeValue("enable_dns_hostnames", cty.True)
	vpcBlk.Body().SetAttributeValue("enable_dns_support", cty.True)
	vpcBlk.Body().AppendNewBlock("tags", nil).Body().SetAttributeValue("Name", cty.StringVal(spec.Name))

	igwBlk := body.AppendNewBlock("resource", []string{"aws_internet_gateway", spec.Name + "_igw"})
	igwBlk.Body().SetAttributeRaw("vpc_id", hclwrite.TokensForIdentifier("aws_vpc."+spec.Name+".id"))

	return nil
}

func awsContainerService(body *hclwrite.Body, spec ResourceSpec) error {
	size := "m"
	if s, ok := spec.Config["size"].(string); ok {
		size = s
	}
	cpu, mem := "1024", "2048"
	if pair, ok := awsSizingApp[size]; ok {
		cpu, mem = pair[0], pair[1]
	}

	// ECS Cluster
	clusterBlk := body.AppendNewBlock("resource", []string{"aws_ecs_cluster", spec.Name})
	clusterBlk.Body().SetAttributeValue("name", cty.StringVal(spec.Name))

	// ECS Task Definition
	taskBlk := body.AppendNewBlock("resource", []string{"aws_ecs_task_definition", spec.Name})
	tb := taskBlk.Body()
	tb.SetAttributeValue("family", cty.StringVal(spec.Name))
	tb.SetAttributeValue("requires_compatibilities", cty.ListVal([]cty.Value{cty.StringVal("FARGATE")}))
	tb.SetAttributeValue("network_mode", cty.StringVal("awsvpc"))
	tb.SetAttributeValue("cpu", cty.StringVal(cpu))
	tb.SetAttributeValue("memory", cty.StringVal(mem))
	image := "nginx:latest"
	if img, ok := spec.Config["image"].(string); ok {
		image = img
	}
	tb.SetAttributeValue("container_definitions", cty.StringVal(
		`[{"name":"`+spec.Name+`","image":"`+image+`","essential":true}]`,
	))

	// ECS Service
	svcBlk := body.AppendNewBlock("resource", []string{"aws_ecs_service", spec.Name})
	sb := svcBlk.Body()
	sb.SetAttributeValue("name", cty.StringVal(spec.Name))
	sb.SetAttributeRaw("cluster", hclwrite.TokensForIdentifier("aws_ecs_cluster."+spec.Name+".id"))
	sb.SetAttributeRaw("task_definition", hclwrite.TokensForIdentifier("aws_ecs_task_definition."+spec.Name+".arn"))
	sb.SetAttributeValue("desired_count", cty.NumberIntVal(replicaCount(spec.Config)))
	sb.SetAttributeValue("launch_type", cty.StringVal("FARGATE"))

	return nil
}

func awsK8sCluster(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"aws_eks_cluster", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("version", strVal(spec.Config, "version", "1.31"))
	roleBlk := b.AppendNewBlock("role_arn", nil)
	_ = roleBlk
	b.SetAttributeValue("role_arn", cty.StringVal("arn:aws:iam::ACCOUNT_ID:role/eks-cluster-role"))
	vpcConfig := b.AppendNewBlock("vpc_config", nil)
	vpcConfig.Body().SetAttributeValue("subnet_ids", cty.ListValEmpty(cty.String))
	return nil
}

func awsCache(body *hclwrite.Body, spec ResourceSpec) error {
	nodeType := "cache.t3.micro"
	if s, ok := spec.Config["size"].(string); ok {
		types := map[string]string{"xs": "cache.t3.micro", "s": "cache.t3.small", "m": "cache.r6g.large", "l": "cache.r6g.xlarge", "xl": "cache.r6g.2xlarge"}
		if t, ok := types[s]; ok {
			nodeType = t
		}
	}
	blk := body.AppendNewBlock("resource", []string{"aws_elasticache_replication_group", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("replication_group_id", cty.StringVal(spec.Name))
	b.SetAttributeValue("description", cty.StringVal(spec.Name+" cache"))
	b.SetAttributeValue("node_type", cty.StringVal(nodeType))
	b.SetAttributeValue("num_cache_clusters", cty.NumberIntVal(1))
	b.SetAttributeValue("engine_version", strVal(spec.Config, "version", "7.0"))
	return nil
}

func awsLoadBalancer(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"aws_lb", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("internal", cty.False)
	b.SetAttributeValue("load_balancer_type", cty.StringVal("application"))
	b.SetAttributeValue("subnets", cty.ListValEmpty(cty.String))
	return nil
}

func awsDNS(body *hclwrite.Body, spec ResourceSpec) error {
	zone := strVal(spec.Config, "zone", spec.Name+".example.com")
	blk := body.AppendNewBlock("resource", []string{"aws_route53_zone", spec.Name})
	blk.Body().SetAttributeValue("name", zone)
	return nil
}

func awsRegistry(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"aws_ecr_repository", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("image_tag_mutability", cty.StringVal("MUTABLE"))
	return nil
}

func awsAPIGateway(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"aws_apigatewayv2_api", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("protocol_type", cty.StringVal("HTTP"))
	return nil
}

func awsFirewall(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"aws_security_group", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("description", cty.StringVal("Managed by workflow-plugin-tofu"))
	return nil
}

func awsIAMRole(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"aws_iam_role", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("name", cty.StringVal(spec.Name))
	b.SetAttributeValue("assume_role_policy", cty.StringVal(`{"Version":"2012-10-17","Statement":[]}`))
	return nil
}

func awsStorage(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"aws_s3_bucket", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("bucket", cty.StringVal(spec.Name))
	return nil
}

func awsCertificate(body *hclwrite.Body, spec ResourceSpec) error {
	blk := body.AppendNewBlock("resource", []string{"aws_acm_certificate", spec.Name})
	b := blk.Body()
	b.SetAttributeValue("domain_name", strVal(spec.Config, "domain", "*.example.com"))
	b.SetAttributeValue("validation_method", cty.StringVal("DNS"))
	return nil
}

// helpers

func storageGB(cfg map[string]any, def int64) int64 {
	if v, ok := cfg["storage_gb"]; ok {
		switch n := v.(type) {
		case int:
			return int64(n)
		case int64:
			return n
		case float64:
			return int64(n)
		}
	}
	return def
}

func boolVal(cfg map[string]any, key string, def bool) cty.Value {
	if v, ok := cfg[key]; ok {
		if b, ok := v.(bool); ok {
			if b {
				return cty.True
			}
			return cty.False
		}
	}
	if def {
		return cty.True
	}
	return cty.False
}

func replicaCount(cfg map[string]any) int64 {
	if v, ok := cfg["replicas"]; ok {
		switch n := v.(type) {
		case int:
			return int64(n)
		case int64:
			return n
		case float64:
			return int64(n)
		}
	}
	return 1
}

