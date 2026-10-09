/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// writeSyntheticHelpBundles supplies small command trees for engine and help tests without reading released product metadata.
func writeSyntheticHelpBundles(t *testing.T) string {
	t.Helper()
	disableDevelopmentBundledPluginsForTest(t)
	root := t.TempDir()
	for _, name := range []string{"ecs", "region"} {
		dir := filepath.Join(root, name)
		writeVersionedBundle(t, dir, name, "0.1.0")
		mustWrite(t, filepath.Join(dir, "plugin.json"), `{"name":"`+name+`","version":"0.1.0","channel":"stable","quality":"reviewed","requires":{"ctyun":"`+testCompatibleCoreConstraint()+`"},"api":{"product":"`+name+`","ctyun_product_id":25,"endpoint_url":"https://example.test"}}`)
		for _, subdir := range []string{"i18n", "fixtures"} {
			if err := os.MkdirAll(filepath.Join(dir, subdir), 0o755); err != nil {
				t.Fatal(err)
			}
		}
	}
	ecs := filepath.Join(root, "ecs")
	mustWrite(t, filepath.Join(ecs, "commands.json"), `{"commands":[
{"id":"ecs.instance.list","path":["ecs","instance","list"],"operation":"list","table":"instances","fixture_response":"fixtures/list.json","parameters":[{"name":"name","flag":"name","target":"displayName","description":"Filter by exact instance name"}],"examples":["ctyun ecs instance list --name api-test01"],"docs_url":"https://example.test/instances"},
{"id":"ecs.instance.show","path":["ecs","instance","show","{instance_id}"],"operation":"show","table":"instances","fixture_response":"fixtures/list.json","examples":["ctyun ecs instance show ins-demo-1"]},
{"id":"ecs.instance.start","path":["ecs","instance","start","{instance_id}"],"operation":"start","table":"instances","fixture_response":"fixtures/list.json","dangerous":{"confirm":"yes"}},
{"id":"ecs.instance.describe-instances","path":["ecs","instance","describe-instances"],"operation":"list","table":"instances","fixture_response":"fixtures/list.json"}
]}`)
	mustWrite(t, filepath.Join(ecs, "apis.json"), `{"operations":{
"list":{"method":"POST","path":"/test/instances","body":{"regionID":"$profile.region","displayName":"$param.name"}},
"show":{"method":"POST","path":"/test/instance","body":{"instanceID":"$arg.instance_id"}},
"start":{"method":"POST","path":"/test/start","body":{"instanceID":"$arg.instance_id"}}
}}`)
	mustWrite(t, filepath.Join(ecs, "tables.json"), `{"tables":{"instances":{"row_path":"returnObj.results","columns":[
{"key":"instance_id","path":"instanceID","labels":{"en-US":"Instance ID","en-GB":"Instance ID","zh-CN":"实例 ID"}},
{"key":"name","path":"displayName","labels":{"en-US":"Name","en-GB":"Name","zh-CN":"云主机名称"}},
{"key":"status","path":"status","labels":{"en-US":"Status","en-GB":"Status","zh-CN":"状态"}}
]}}}`)
	mustWrite(t, filepath.Join(ecs, "fixtures", "list.json"), `{"statusCode":800,"returnObj":{"results":[{"instanceID":"ins-demo-1","displayName":"demo-instance","status":"running"}]}}`)
	for _, language := range []string{"en-US", "en-GB"} {
		mustWrite(t, filepath.Join(ecs, "i18n", language+".json"), `{"name":"Elastic Cloud Server","command.ecs.instance.list.description":"List ECS instances","command.ecs.instance.show.description":"Show an ECS instance","argument.ecs.instance.show.instance_id.description":"Instance ID"}`)
	}
	mustWrite(t, filepath.Join(ecs, "i18n", "zh-CN.json"), `{"name":"弹性云主机","command.ecs.instance.list.description":"查询云主机列表","parameter.ecs.instance.list.name.description":"按云主机名称过滤"}`)
	region := filepath.Join(root, "region")
	mustWrite(t, filepath.Join(region, "commands.json"), `{"commands":[
{"id":"region.list","path":["region","list"],"table":"regions","fixture_response":"fixtures/list.json"},
{"id":"region.demand.check","path":["region","demand","check","{region_id}"],"table":"regions","parameters":[{"name":"product_type","target":"product_type","flag":"product-type","required":true,"allowed_values":["ecs","eip","ebs"],"description":"Product type to check"},{"name":"zone","target":"zone","flag":"zone","description":"Availability zone name"},{"name":"ebs_type","target":"ebs_type","flag":"ebs-type","allowed_values":["SATA","SAS","SSD","SATA-KUNPENG","SATA-HAIGUANG","SAS-KUNPENG","SAS-HAIGUANG"],"description":"EBS disk type"}]},
{"id":"region.product.show","path":["region","product","show","{region_id}"],"table":"products"},
{"id":"region.resource.show","path":["region","resource","show","{region_id}"],"table":"resources"}
]}`)
	mustWrite(t, filepath.Join(region, "tables.json"), `{"tables":{
"regions":{"row_path":"returnObj","default_columns":["region_id","region_name"],"columns":[{"key":"region_id","path":"regionID","labels":{"en-US":"Region ID","en-GB":"Region ID","zh-CN":"资源池 ID"}},{"key":"region_name","path":"regionName","labels":{"en-US":"Region Name","en-GB":"Region Name","zh-CN":"资源池名称"}}]},
"products":{"row_path":"returnObj","layout":"vertical","default_columns":["flavors"],"columns":[{"key":"flavors","path":"flavors","labels":{"en-US":"ECS Flavor Types","en-GB":"ECS Flavor Types","zh-CN":"规格类型"}},{"key":"ecs","path":"ecs","labels":{"en-US":"ECS","en-GB":"ECS","zh-CN":"云主机"}}]},
"resources":{"row_path":"returnObj","layout":"vertical","default_columns":["ecs_total"],"columns":[{"key":"acl","path":"acl","labels":{"en-US":"ACL Summary","en-GB":"ACL Summary","zh-CN":"访问控制概要"}},{"key":"ecs_total","path":"ecsTotal","labels":{"en-US":"ECS Total","en-GB":"ECS Total","zh-CN":"云主机总数"}}]}
}}`)
	mustWrite(t, filepath.Join(region, "fixtures", "list.json"), `{"statusCode":800,"returnObj":[{"regionID":"region-demo-1","regionName":"Demo Region"}]}`)
	for _, language := range []string{"en-US", "en-GB"} {
		mustWrite(t, filepath.Join(region, "i18n", language+".json"), `{"name":"Region","command.region.list.description":"List regions","argument.region.demand.check.region_id.description":"Region ID"}`)
	}
	mustWrite(t, filepath.Join(region, "i18n", "zh-CN.json"), `{"name":"资源池","command.region.list.description":"查询资源池"}`)
	return root
}
