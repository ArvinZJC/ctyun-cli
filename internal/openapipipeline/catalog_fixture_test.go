/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"encoding/json"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

func loadCatalogFixture(t *testing.T) Catalog {
	t.Helper()
	return Catalog{
		SchemaVersion: 1,
		Product: Product{
			PluginName:     "ecs",
			APIProduct:     "ecs",
			CtyunProductID: 25,
			SourceRevision: "81",
			DisplayName: map[string]string{
				"en-US": "Elastic Cloud Server",
				"en-GB": "Elastic Cloud Server",
				"zh-CN": "弹性云主机",
			},
			EndpointURL: "https://ctecs-global.ctapi.ctyun.cn",
			SourceURL:   "https://eop.ctyun.cn/ebp/ctapiDocument/search?sid=25",
			APIScope: plugin.APIScope{
				IncludeURIPrefixes: []string{"/v4/ecs/"},
				Notes:              "All official ECS APIs whose URI starts with /v4/ecs/.",
			},
		},
		Operations: []Operation{
			{
				ID:          "v4.ecs.instance.list",
				APIID:       "8309",
				Title:       "查询云主机列表",
				Description: map[string]string{"en-US": "List ECS instances", "en-GB": "List ECS instances", "zh-CN": "列出云主机"},
				Category:    "instance",
				Method:      "POST",
				Path:        "/v4/ecs/list-instances",
				ContentType: "application/json",
				DocsURL:     "https://eop.ctyun.cn/ebp/ctapiDocument/search?sid=25&api=8309&data=87",
				Retryable:   true,
				Examples:    []string{"ctyun ecs instance list", "ctyun ecs instance list --name demo"},
				Parameters: []Parameter{
					{Name: "regionID", Location: "body", Required: true, Type: "string", Profile: "region"},
					{
						Name:         "keyword",
						Location:     "body",
						Type:         "string",
						CLIName:      "name",
						CLIFlag:      "name",
						TableTarget:  "displayName",
						Description:  "Filter by instance name",
						Descriptions: map[string]string{"en-US": "Filter by instance name", "en-GB": "Filter by instance name", "zh-CN": "按云主机名称过滤"},
					},
					{
						Name:        "pageNo",
						Location:    "body",
						Type:        "integer",
						CLIName:     "page_no",
						CLIFlag:     "page-no",
						TableTarget: "pageNo",
						Description: "页码，取值范围：正整数（≥1），注：默认值为1",
					},
				},
				Response: Response{
					SuccessCode: "800",
					ResultPath:  "returnObj",
					RowPath:     "returnObj.results",
					Columns: []Column{
						{Key: "instance_id", Path: "instanceID", LabelEN: "Instance ID", LabelZH: "实例 ID"},
						{Key: "name", Path: "displayName", LabelEN: "Name", LabelZH: "名称"},
						{Key: "status", Path: "status", LabelEN: "Status", LabelZH: "状态"},
						{Key: "created_time", Path: "createdTime", LabelEN: "Created Time", LabelZH: "创建时间"},
					},
				},
				ExampleResponse: json.RawMessage(`{"statusCode":800,"returnObj":{"results":[{"instanceID":"ins-demo-1","displayName":"demo","status":"running"}]}}`),
			},
			{
				ID:          "v4.ecs.instance.show",
				APIID:       "8310",
				Title:       "查询云主机详情",
				Description: map[string]string{"en-US": "Show an ECS instance", "en-GB": "Show an ECS instance", "zh-CN": "查看云主机详情"},
				Category:    "instance",
				Method:      "GET",
				Path:        "/v4/ecs/instance-details",
				ContentType: "application/json",
				DocsURL:     "https://eop.ctyun.cn/ebp/ctapiDocument/search?sid=25",
				Retryable:   true,
				Examples:    []string{"ctyun ecs instance show {instance_id}"},
				Parameters: []Parameter{
					{Name: "regionID", Location: "query", Required: true, Type: "string", Profile: "region"},
					{Name: "instanceID", Location: "query", Required: true, Type: "string", Argument: "instance_id"},
				},
				Response: Response{
					SuccessCode: "800",
					ResultPath:  "returnObj",
					RowPath:     "returnObj",
					Columns: []Column{
						{Key: "instance_id", Path: "instanceID", LabelEN: "Instance ID", LabelZH: "实例 ID"},
						{Key: "name", Path: "displayName", LabelEN: "Name", LabelZH: "名称"},
						{Key: "status", Path: "instanceStatus", LabelEN: "Status", LabelZH: "状态"},
					},
				},
				ExampleResponse: json.RawMessage(`{"statusCode":800,"returnObj":{"instanceID":"ins-demo-1","displayName":"demo","instanceStatus":"running"}}`),
			},
			{
				ID:          "v4.ecs.instance.start",
				APIID:       "8311",
				Title:       "开启一台云主机",
				Description: map[string]string{"en-US": "Start an ECS instance", "en-GB": "Start an ECS instance", "zh-CN": "启动云主机"},
				Category:    "instance",
				Method:      "POST",
				Path:        "/v4/ecs/start-instance",
				ContentType: "application/json",
				DocsURL:     "https://eop.ctyun.cn/ebp/ctapiDocument/search?sid=25",
				Parameters: []Parameter{
					{Name: "regionID", Location: "body", Required: true, Type: "string", Profile: "region"},
					{Name: "instanceID", Location: "body", Required: true, Type: "string", Argument: "instance_id", ExampleUnavailable: true},
				},
				Response: Response{
					SuccessCode: "800",
					ResultPath:  "returnObj",
					JobIDPath:   "returnObj.jobID",
					RowPath:     "returnObj",
					Columns:     []Column{{Key: "job_id", Path: "jobID", LabelEN: "Job ID", LabelZH: "任务 ID"}},
				},
				Dangerous:       true,
				ExampleResponse: json.RawMessage(`{"statusCode":800,"returnObj":{"jobID":"job-demo-1"}}`),
			},
			{
				ID:          "v4.ecs.zone.list",
				APIID:       "8312",
				Title:       "查询云主机可用区",
				Description: map[string]string{"en-US": "List ECS availability zones", "en-GB": "List ECS availability zones", "zh-CN": "列出云主机可用区"},
				Category:    "zone",
				Method:      "GET",
				Path:        "/v4/ecs/get-zones",
				ContentType: "application/json",
				DocsURL:     "https://eop.ctyun.cn/ebp/ctapiDocument/search?sid=25",
				Retryable:   true,
				Parameters:  []Parameter{{Name: "regionID", Location: "query", Required: true, Type: "string", Argument: "region_id", ExampleUnavailable: true}},
				Response: Response{
					SuccessCode: "800",
					ResultPath:  "returnObj",
					RowPath:     "returnObj.zoneList",
					Columns:     []Column{{Key: "zone_name", Path: "name", LabelEN: "Zone Name", LabelZH: "可用区名称"}},
				},
				ExampleResponse: json.RawMessage(`{"statusCode":800,"returnObj":{"zoneList":[{"name":"az1"}]}}`),
			},
		},
	}
}

func catalogFixtureJSON(t *testing.T) []byte {
	t.Helper()
	data, err := json.Marshal(loadCatalogFixture(t))
	if err != nil {
		t.Fatalf("marshal catalog fixture: %v", err)
	}
	return data
}
