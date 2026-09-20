/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
	"github.com/ArvinZJC/ctyun-cli/internal/client"
	"github.com/ArvinZJC/ctyun-cli/internal/openapipipeline"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
	"github.com/ArvinZJC/ctyun-cli/internal/waiter"
)

// TestPromotedWaiterProvenanceAndFixtures prevents lost or invented waiter
// definitions and verifies each bound command's actual captured response shape.
func TestPromotedWaiterProvenanceAndFixtures(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join(repoPath(t, "plugins"), "*", "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		name := filepath.Base(filepath.Dir(path))
		t.Run(name, func(t *testing.T) {
			bundle, err := plugin.LoadBundle(filepath.Dir(path), version.Version)
			if err != nil {
				t.Fatal(err)
			}
			baseline, err := (openapipipeline.Workspace{Root: filepath.Dir(repoPath(t, "plugins"))}).ReadBaseline(name)
			if err != nil {
				t.Fatal(err)
			}
			if len(bundle.Waiters.Waiters) != len(baseline.Waiters) {
				t.Fatal("waiter inventory differs from baseline")
			}
			if len(bundle.Waiters.Waiters) > 0 {
				if _, err := plugin.LoadBundle(filepath.Dir(path), "0.4.999"); err == nil {
					t.Fatal("bound waiters accept a pre-0.5.0 core")
				}
			}
			for id, spec := range bundle.Waiters.Waiters {
				if !reflect.DeepEqual(spec, baseline.Waiters[id].Waiter) {
					t.Fatalf("%s differs from baseline", id)
				}
				if len(spec.Commands) == 0 {
					t.Fatalf("%s lacks explicit bindings", id)
				}
				for _, command := range bundle.Commands.Commands {
					if !plugin.WaiterApplies(bundle, command, spec) {
						continue
					}
					data, err := os.ReadFile(filepath.Join(filepath.Dir(path), command.FixtureResponse))
					if err != nil {
						t.Fatal(err)
					}
					payload, err := client.DecodeResponse(data)
					if operation := bundle.APIs.Operations[command.Operation]; operation.Response != nil {
						var response *client.HTTPResponse
						response, err = client.DecodeFixture(data)
						if err == nil {
							var result *client.Result
							result, err = client.DecodeHTTPResponse(response, client.RequestSpec{Method: operation.Method, Response: operation.Response})
							if err == nil {
								payload = result.Payload
							}
						}
					}
					if err != nil {
						t.Fatal(err)
					}
					if err := waiter.ValidateExample(waiter.Spec{Selector: spec.Selector, Path: spec.Path, Success: spec.Success, Failure: spec.Failure, SuccessValues: spec.SuccessValues, FailureValues: spec.FailureValues}, payload); err != nil {
						t.Fatalf("%s fixture for %s: %v", id, command.ID, err)
					}
				}
			}
		})
	}
}

// TestBundledWaiterOfflineFlows checks real metadata through the CLI and keeps
// JSON stdout parseable while terminal states go to stderr.
func TestBundledWaiterOfflineFlows(t *testing.T) {
	for _, tc := range []struct {
		args      []string
		id, state string
	}{
		{[]string{"vbs", "backup", "show", "--backup-id", "test"}, "vbs.backup.available", "success"},
		{[]string{"vbs", "policy", "show-task", "--policy-id", "test", "--task-id", "test"}, "vbs.policy.task.completed", "failure"},
		{[]string{"hpfs", "dataflow-task", "show", "--task-id", "test"}, "hpfs.dataflow-task.completed", "success"},
		{[]string{"zos", "service", "show-status"}, "zos.service.activated", "success"},
		{[]string{"ims", "image", "show", "8d8e8888-8ed8-88b8-88cb-888f8b8cf8fa"}, "ims.image.active", "success"},
		{[]string{"ecs", "snapshot", "show", "c7a7f06d-fb0f-8d5a-e710-9262995b6b6d"}, "ecs.snapshot.detail-available", "success"},
		{[]string{"evs", "snapshot", "list", "--snapshot-id", "cafa8c74-42e7-4e32-921c-d7b2c75dac27"}, "evs.snapshot.available", "success"},
		{[]string{"cdr", "task", "show", "--task-id", "d009e332-c8c4-5568-1a1a-363ed3874f93"}, "cdr.task.completed", "failure"},
		{[]string{"cbr", "restore", "list", "--restore-task-id", "d81b06d4-c0ea-41d8-b397-3275cf5e2478"}, "cbr.restore.completed", "success"},
		{[]string{"ecpc", "desktop", "describe-desktops", "--desktop-oid", "XXXXXXXXXXXXX6925"}, "ecpc.desktop.running", "success"},
		{[]string{"as", "group", "list", "--group-id", "540"}, "as.group.disabled", "success"},
	} {
		t.Run(tc.id, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			args := append(append([]string{}, tc.args...), "--offline", "--wait", tc.id, "--output", "json", "--lang", "en-US")
			if err := cli.Run(cli.Config{Args: args, Stdout: &stdout, Stderr: &stderr, PluginRoot: t.TempDir()}); err != nil {
				t.Fatal(err)
			}
			if !json.Valid(stdout.Bytes()) || !strings.Contains(stderr.String(), tc.id+": "+tc.state) {
				t.Fatalf("stdout=%s stderr=%s", stdout.String(), stderr.String())
			}
		})
	}
}

// TestBundledWaiterStateSemantics guards upstream distinctions that must not
// be generalized across products or inferred from a success-shaped value.
func TestBundledWaiterStateSemantics(t *testing.T) {
	for _, tc := range []struct {
		name, id string
		value    any
		want     waiter.State
	}{
		{"ims", "ims.image.integrity-checked", "compute_success", waiter.Pending},
		{"ims", "ims.image.integrity-checked", "check_success", waiter.Success},
		{"ims", "ims.image.integrity-checked", "abnormal", waiter.Failure},
		{"ecs", "ecs.instance.task-succeeded", float64(1), waiter.Success},
		{"ecs", "ecs.instance.task-succeeded", "running", waiter.Pending},
		{"job", "job.succeeded", "success", waiter.Success},
		{"job", "job.succeeded", float64(1), waiter.Pending},
		{"ecs", "ecs.backup.available", "ACTIVE", waiter.Success},
		{"ecs", "ecs.backup.available", "active", waiter.Pending},
		{"ecs", "ecs.snapshot.available", "available", waiter.Success},
		{"order", "order.completed", "22", waiter.Failure},
		{"order", "order.completed", "7", waiter.Pending},
		{"as", "as.activity.rule-succeeded", nil, waiter.Pending},
		{"as", "as.activity.succeeded", float64(0), waiter.Pending},
		{"zos", "zos.migration.completed", "finished_error", waiter.Failure},
		{"evs", "evs.volume.ready", "in-use", waiter.Success},
		{"evs", "evs.volume.available", "in-use", waiter.Pending},
	} {
		t.Run(tc.id+"/"+fmt.Sprint(tc.value), func(t *testing.T) {
			bundle, err := plugin.LoadBundle(filepath.Join(repoPath(t, "plugins"), tc.name), version.Version)
			if err != nil {
				t.Fatal(err)
			}
			spec, ok := bundle.Waiters.Waiters[tc.id]
			if !ok {
				t.Fatal("missing waiter")
			}
			// Identity selection is exercised against captured fixtures separately.
			state, err := waiter.Evaluate(waiter.Spec{Path: "state", Success: spec.Success, Failure: spec.Failure, SuccessValues: spec.SuccessValues, FailureValues: spec.FailureValues}, map[string]any{"state": tc.value})
			if err != nil || state != tc.want {
				t.Fatalf("got %s, %v; want %s", state, err, tc.want)
			}
		})
	}
}
