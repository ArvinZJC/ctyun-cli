/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPluginCommandDispatchUsesMetadataWithoutProductBranch(t *testing.T) {
	pluginRoot := t.TempDir()
	writeVPCBundle(t, filepath.Join(pluginRoot, "vpc"))

	var stdout bytes.Buffer
	err := Run(Config{
		Args:       []string{"--offline", "--lang", "en-US", "vpc", "subnet", "list", "--cols", "subnet_id,name"},
		Stdout:     &stdout,
		PluginRoot: pluginRoot,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got := stdout.String()
	for _, want := range []string{"Subnet ID", "Name", "subnet-demo-1", "app-subnet"} {
		if !strings.Contains(got, want) {
			t.Fatalf("generic plugin output missing %q:\n%s", want, got)
		}
	}
}

func TestDangerousCommandPromptsForConfirmation(t *testing.T) {
	pluginRoot := t.TempDir()
	writeDangerBundle(t, filepath.Join(pluginRoot, "ecs"))

	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args:       []string{"ecs", "instance", "delete", "ins-demo-1"},
		Stdout:     &stdout,
		Stderr:     &stderr,
		Stdin:      strings.NewReader("n\n"),
		PluginRoot: pluginRoot,
	})
	if err == nil {
		t.Fatal("dangerous command declined confirmation returned nil error")
	}
	if !strings.Contains(stderr.String(), "Continue? [y/N]:") {
		t.Fatalf("confirmation prompt missing from stderr:\n%s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = Run(Config{
		Args:       []string{"--offline", "ecs", "instance", "delete", "ins-demo-1"},
		Stdout:     &stdout,
		Stderr:     &stderr,
		Stdin:      strings.NewReader("y\n"),
		PluginRoot: pluginRoot,
	})
	if err != nil {
		t.Fatalf("dangerous command accepted confirmation returned error: %v", err)
	}
	if !strings.Contains(stderr.String(), "Continue? [y/N]:") {
		t.Fatalf("confirmation prompt missing from stderr:\n%s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "job-demo-1") {
		t.Fatalf("confirmed dangerous output missing fixture response:\n%s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = Run(Config{
		Args:       []string{"--offline", "--yes", "ecs", "instance", "delete", "ins-demo-1"},
		Stdout:     &stdout,
		Stderr:     &stderr,
		PluginRoot: pluginRoot,
	})
	if err != nil {
		t.Fatalf("dangerous command with --yes returned error: %v", err)
	}
	if stderr.String() != "" {
		t.Fatalf("--yes should skip confirmation prompt, stderr:\n%s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "job-demo-1") {
		t.Fatalf("confirmed dangerous output missing fixture response:\n%s", stdout.String())
	}
}

func TestDangerousCommandLocalizesConfirmationPrompt(t *testing.T) {
	pluginRoot := t.TempDir()
	writeDangerBundle(t, filepath.Join(pluginRoot, "ecs"))

	var stderr bytes.Buffer
	err := Run(Config{
		Args:       []string{"--lang", "zh-CN", "ecs", "instance", "delete", "ins-demo-1"},
		Stdout:     io.Discard,
		Stderr:     &stderr,
		Stdin:      strings.NewReader("n\n"),
		PluginRoot: pluginRoot,
	})
	if err == nil {
		t.Fatal("dangerous command declined confirmation returned nil error")
	}
	if !strings.Contains(stderr.String(), "需要确认") || !strings.Contains(stderr.String(), "是否继续？[y/N]：") {
		t.Fatalf("prompt = %q, want localized confirmation prompt", stderr.String())
	}
}

func TestWaitFlagEvaluatesWaiterMetadata(t *testing.T) {
	pluginRoot := t.TempDir()
	writeWaitBundle(t, filepath.Join(pluginRoot, "ecs"))

	var stdout bytes.Buffer
	err := Run(Config{
		Args:       []string{"--offline", "--wait", "ecs.instance.running", "ecs", "instance", "show", "ins-demo-1"},
		Stdout:     &stdout,
		PluginRoot: pluginRoot,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "waiter ecs.instance.running: success") {
		t.Fatalf("waiter output missing success:\n%s", stdout.String())
	}
}

func TestWaitFlagPollsUntilWaiterSucceeds(t *testing.T) {
	pluginRoot := t.TempDir()
	writePollingWaitBundle(t, filepath.Join(pluginRoot, "ecs"))

	attempts := 0
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		status := "starting"
		if attempts == 2 {
			status = "running"
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
  "returnObj": {
    "status": "` + status + `",
    "instances": [{"instanceID": "ins-demo-1"}]
  }
}`)),
		}, nil
	})

	var stdout bytes.Buffer
	err := Run(Config{
		Args:          []string{"--wait", "ecs.instance.running", "ecs", "instance", "show", "ins-demo-1", "--cols", "instance_id"},
		Stdout:        &stdout,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env: func(key string) string {
			switch key {
			case "CTYUN_AK":
				return "ak-test"
			case "CTYUN_SK":
				return "sk-test"
			default:
				return ""
			}
		},
		Config: []byte(`{
  "active_profile": "default",
  "profiles": {
    "default": {
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test"
    }
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
	if !strings.Contains(stdout.String(), "waiter ecs.instance.running: success") {
		t.Fatalf("waiter output missing success:\n%s", stdout.String())
	}
}

func TestWaitFlagReportsTimeoutAfterMaxAttempts(t *testing.T) {
	pluginRoot := t.TempDir()
	writePollingWaitBundle(t, filepath.Join(pluginRoot, "ecs"))

	attempts := 0
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
  "returnObj": {
    "status": "starting",
    "instances": [{"instanceID": "ins-demo-1"}]
  }
}`)),
		}, nil
	})

	var stdout bytes.Buffer
	err := Run(Config{
		Args:          []string{"--wait", "ecs.instance.running", "ecs", "instance", "show", "ins-demo-1", "--cols", "instance_id"},
		Stdout:        &stdout,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env: func(key string) string {
			switch key {
			case "CTYUN_AK":
				return "ak-test"
			case "CTYUN_SK":
				return "sk-test"
			default:
				return ""
			}
		},
		Config: []byte(`{
  "active_profile": "default",
  "profiles": {
    "default": {
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test"
    }
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want timeout after max attempts", attempts)
	}
	if !strings.Contains(stdout.String(), "waiter ecs.instance.running: timeout") {
		t.Fatalf("waiter output missing timeout:\n%s", stdout.String())
	}
}

func TestTimeoutFlagOverridesProfileRequestTimeout(t *testing.T) {
	pluginRoot := t.TempDir()
	writePollingWaitBundle(t, filepath.Join(pluginRoot, "ecs"))

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		deadline, ok := req.Context().Deadline()
		if !ok {
			t.Fatal("request context has no timeout deadline")
		}
		remaining := time.Until(deadline)
		if remaining < 18*time.Second || remaining > 22*time.Second {
			t.Fatalf("request timeout = %v, want about 20s", remaining)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
  "returnObj": {
    "status": "running",
    "instances": [{"instanceID": "ins-demo-1"}]
  }
}`)),
		}, nil
	})

	var stdout bytes.Buffer
	err := Run(Config{
		Args:          []string{"--timeout", "20", "ecs", "instance", "show", "ins-demo-1", "--cols", "instance_id"},
		Stdout:        &stdout,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env: func(key string) string {
			switch key {
			case "CTYUN_AK":
				return "ak-test"
			case "CTYUN_SK":
				return "sk-test"
			default:
				return ""
			}
		},
		Config: []byte(`{
  "active_profile": "default",
  "profiles": {
    "default": {
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test",
      "timeout_seconds": 7
    }
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func TestPluginCommandWithoutFixtureUsesAPIMetadataAndEnvCredentials(t *testing.T) {
	pluginRoot := t.TempDir()
	writeIMSBundleWithoutFixture(t, filepath.Join(pluginRoot, "ims"))

	var seenBody string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Eop-Authorization") == "" {
			t.Fatal("request was not signed")
		}
		if _, ok := req.Context().Deadline(); !ok {
			t.Fatal("request context has no profile timeout deadline")
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		seenBody = string(body)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"returnObj":{"images":[{"imageID":"img-demo-1","name":"base"}]}}`)),
		}, nil
	})

	var stdout bytes.Buffer
	err := Run(Config{
		Args:          []string{"--lang", "en-US", "ims", "image", "list", "--cols", "image_id,name"},
		Stdout:        &stdout,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env: func(key string) string {
			switch key {
			case "CTYUN_AK":
				return "ak-test"
			case "CTYUN_SK":
				return "sk-test"
			default:
				return ""
			}
		},
		Config: []byte(`{
  "active_profile": "default",
  "profiles": {
    "default": {
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test",
      "timeout_seconds": 7
    }
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.Contains(seenBody, `"regionID":"cn-huadong1"`) {
		t.Fatalf("request body did not include profile region: %s", seenBody)
	}
	if !strings.Contains(stdout.String(), "img-demo-1") {
		t.Fatalf("output missing API response row:\n%s", stdout.String())
	}
}

func TestPluginCommandWarnsWhenLiveRequestUsesConfigCredentials(t *testing.T) {
	pluginRoot := t.TempDir()
	writeIMSBundleWithoutFixture(t, filepath.Join(pluginRoot, "ims"))
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"returnObj":{"images":[]}}`)),
		}, nil
	})

	var stderr bytes.Buffer
	err := Run(Config{
		Args:          []string{"--lang", "en-US", "ims", "image", "list"},
		Stdout:        io.Discard,
		Stderr:        &stderr,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env:           func(string) string { return "" },
		Config: []byte(`{
  "active_profile": "default",
  "profiles": {
    "default": {
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test",
      "ak": "profile-ak",
      "sk": "profile-sk"
    }
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stderr.String(), "Warning: using CTyun AK/SK from config.") ||
		!strings.Contains(stderr.String(), "environment variable") ||
		!strings.Contains(stderr.String(), "ctyun config set warn_config_credentials false") {
		t.Fatalf("stderr missing config credential warning: %q", stderr.String())
	}
}

func TestPluginCommandLocalizesConfigCredentialWarning(t *testing.T) {
	pluginRoot := t.TempDir()
	writeIMSBundleWithoutFixture(t, filepath.Join(pluginRoot, "ims"))
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"returnObj":{"images":[]}}`)),
		}, nil
	})

	var stderr bytes.Buffer
	err := Run(Config{
		Args:          []string{"--lang", "zh-CN", "ims", "image", "list"},
		Stdout:        io.Discard,
		Stderr:        &stderr,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env:           func(string) string { return "" },
		Config: []byte(`{
  "active_profile": "default",
  "profiles": {
    "default": {
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test",
      "ak": "profile-ak",
      "sk": "profile-sk"
    }
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stderr.String(), "警告：正在使用配置中的天翼云 AK/SK。") ||
		!strings.Contains(stderr.String(), "环境变量") ||
		!strings.Contains(stderr.String(), "ctyun config set warn_config_credentials false") {
		t.Fatalf("stderr missing localized config credential warning: %q", stderr.String())
	}
}

func TestPluginCommandDoesNotWarnForEnvCredentialsOrDisabledConfigWarning(t *testing.T) {
	cases := []struct {
		name   string
		env    func(string) string
		config []byte
	}{
		{
			name: "env credentials",
			env: func(key string) string {
				switch key {
				case "CTYUN_AK":
					return "env-ak"
				case "CTYUN_SK":
					return "env-sk"
				default:
					return ""
				}
			},
			config: []byte(`{
  "active_profile": "default",
  "profiles": {
    "default": {
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test",
      "ak": "profile-ak",
      "sk": "profile-sk"
    }
  }
}`),
		},
		{
			name: "config warning disabled",
			env:  func(string) string { return "" },
			config: []byte(`{
  "warn_config_credentials": false,
  "active_profile": "default",
  "profiles": {
    "default": {
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test",
      "ak": "profile-ak",
      "sk": "profile-sk"
    }
  }
}`),
		},
		{
			name: "config warning disabled by env",
			env: func(key string) string {
				if key == "CTYUN_WARN_CONFIG_CREDENTIALS" {
					return "false"
				}
				return ""
			},
			config: []byte(`{
  "active_profile": "default",
  "profiles": {
    "default": {
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test",
      "ak": "profile-ak",
      "sk": "profile-sk"
    }
  }
}`),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pluginRoot := t.TempDir()
			writeIMSBundleWithoutFixture(t, filepath.Join(pluginRoot, "ims"))
			transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"returnObj":{"images":[]}}`)),
				}, nil
			})

			var stderr bytes.Buffer
			err := Run(Config{
				Args:          []string{"--lang", "en-US", "ims", "image", "list"},
				Stdout:        io.Discard,
				Stderr:        &stderr,
				PluginRoot:    pluginRoot,
				HTTPTransport: transport,
				Env:           tc.env,
				Config:        tc.config,
			})
			if err != nil {
				t.Fatalf("Run returned error: %v", err)
			}
			if strings.Contains(stderr.String(), "using CTyun AK/SK from config") {
				t.Fatalf("stderr contained config credential warning: %q", stderr.String())
			}
		})
	}
}

func TestPluginCommandBindsPathArgumentsIntoAPIBody(t *testing.T) {
	pluginRoot := t.TempDir()
	writeArgumentBundle(t, filepath.Join(pluginRoot, "ims"))

	var seenBody string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		seenBody = string(body)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"returnObj":{"images":[{"imageID":"img-demo-1","name":"base"}]}}`)),
		}, nil
	})

	var stdout bytes.Buffer
	err := Run(Config{
		Args:          []string{"ims", "image", "show", "img-demo-1", "--cols", "image_id,name"},
		Stdout:        &stdout,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env: func(key string) string {
			switch key {
			case "CTYUN_AK":
				return "ak-test"
			case "CTYUN_SK":
				return "sk-test"
			default:
				return ""
			}
		},
		Config: []byte(`{
  "active_profile": "default",
  "profiles": {
    "default": {
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test"
    }
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.Contains(seenBody, `"imageID":"img-demo-1"`) {
		t.Fatalf("request body did not bind path argument: %s", seenBody)
	}
	if !strings.Contains(stdout.String(), "img-demo-1") {
		t.Fatalf("output missing response row:\n%s", stdout.String())
	}
}

func TestPluginCommandBindsMetadataFlagsIntoAPIBody(t *testing.T) {
	pluginRoot := t.TempDir()
	writeFlagBundle(t, filepath.Join(pluginRoot, "ecs"))

	var seenBody string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		seenBody = string(body)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"returnObj":{"instances":[{"instanceID":"ins-demo-1","displayName":"demo-web","status":"running"}]}}`)),
		}, nil
	})

	var stdout bytes.Buffer
	err := Run(Config{
		Args:          []string{"ecs", "instance", "list", "--name", "demo-web", "--cols", "instance_id,name"},
		Stdout:        &stdout,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env: func(key string) string {
			switch key {
			case "CTYUN_AK":
				return "ak-test"
			case "CTYUN_SK":
				return "sk-test"
			default:
				return ""
			}
		},
		Config: []byte(`{
  "active_profile": "default",
  "profiles": {
    "default": {
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test"
    }
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.Contains(seenBody, `"displayName":"demo-web"`) {
		t.Fatalf("request body did not bind metadata flag: %s", seenBody)
	}
	if !strings.Contains(stdout.String(), "demo-web") {
		t.Fatalf("output missing response row:\n%s", stdout.String())
	}
}

func TestPluginCommandBindsQueryAndHeaderMetadata(t *testing.T) {
	pluginRoot := t.TempDir()
	writeQueryHeaderBundle(t, filepath.Join(pluginRoot, "ecs"))

	var seenQuery, seenHeader, seenBody string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		seenQuery = req.URL.RawQuery
		seenHeader = req.Header.Get("x-ctyun-resource")
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		seenBody = string(body)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"returnObj":{"instances":[{"instanceID":"ins-demo-1","displayName":"demo-web"}]}}`)),
		}, nil
	})

	err := Run(Config{
		Args:          []string{"ecs", "instance", "list", "--page", "2", "--cols", "instance_id"},
		Stdout:        io.Discard,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env: func(key string) string {
			switch key {
			case "CTYUN_AK":
				return "ak-test"
			case "CTYUN_SK":
				return "sk-test"
			default:
				return ""
			}
		},
		Config: []byte(`{
  "active_profile": "default",
  "profiles": {
    "default": {
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test"
    }
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if seenQuery != "pageNo=2&regionID=cn-huadong1" {
		t.Fatalf("query = %q, want pageNo=2&regionID=cn-huadong1", seenQuery)
	}
	if seenHeader != "ecs" {
		t.Fatalf("header = %q, want ecs", seenHeader)
	}
	if seenBody != "" {
		t.Fatalf("body = %q, want empty body for query/header-only operation", seenBody)
	}
}

func TestPluginCommandRequiresMetadataFlags(t *testing.T) {
	pluginRoot := t.TempDir()
	writeFlagBundle(t, filepath.Join(pluginRoot, "ecs"))

	err := Run(Config{
		Args:       []string{"ecs", "instance", "list"},
		Stdout:     io.Discard,
		PluginRoot: pluginRoot,
		Config: []byte(`{
  "active_profile": "default",
  "profiles": {
    "default": {
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test"
    }
  }
}`),
	})
	if err == nil {
		t.Fatal("Run returned nil error for missing required flag")
	}
	if !strings.Contains(err.Error(), "--name") {
		t.Fatalf("error = %v, want missing --name", err)
	}
}

func TestPluginCommandValidatesAllowedParameterValues(t *testing.T) {
	pluginRoot := t.TempDir()
	writeValidationBundle(t, filepath.Join(pluginRoot, "ecs"))

	err := Run(Config{
		Args:       []string{"--lang", "en-US", "ecs", "instance", "list", "--status", "paused", "--name", "demo-web"},
		Stdout:     io.Discard,
		PluginRoot: pluginRoot,
	})
	if err == nil {
		t.Fatal("Run returned nil error for invalid status")
	}
	if !strings.Contains(err.Error(), "--status must be one of running,stopped") {
		t.Fatalf("error = %v, want allowed-values validation", err)
	}
}

func TestPluginCommandValidatesParameterPattern(t *testing.T) {
	pluginRoot := t.TempDir()
	writeValidationBundle(t, filepath.Join(pluginRoot, "ecs"))

	err := Run(Config{
		Args:       []string{"--lang", "en-US", "ecs", "instance", "list", "--status", "running", "--name", "bad name"},
		Stdout:     io.Discard,
		PluginRoot: pluginRoot,
	})
	if err == nil {
		t.Fatal("Run returned nil error for invalid name")
	}
	if !strings.Contains(err.Error(), "--name does not match") {
		t.Fatalf("error = %v, want pattern validation", err)
	}
}

func TestPluginCommandLocalizesValidationErrors(t *testing.T) {
	pluginRoot := t.TempDir()
	writeValidationBundle(t, filepath.Join(pluginRoot, "ecs"))

	err := Run(Config{
		Args:       []string{"--lang", "zh-CN", "ecs", "instance", "list", "--status", "paused", "--name", "demo-web"},
		Stdout:     io.Discard,
		PluginRoot: pluginRoot,
	})
	if err == nil {
		t.Fatal("Run returned nil error for invalid localized status")
	}
	if !strings.Contains(err.Error(), "--status 必须是以下值之一：running,stopped") {
		t.Fatalf("error = %v, want localized allowed-values validation", err)
	}
}

func TestDefaultExecutionBypassesFixtureForRetrievalCommand(t *testing.T) {
	var seen bool
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		seen = true
		if req.URL.Path != "/v4/ecs/list-instances" {
			t.Fatalf("path = %q, want /v4/ecs/list-instances", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"returnObj":{"results":[{"instanceID":"ins-live-1","displayName":"live","instanceStatus":"running"}]}}`)),
		}, nil
	})

	var stdout bytes.Buffer
	err := Run(Config{
		Args:          []string{"--lang", "en-US", "ecs", "instance", "list", "--cols", "instance_id,name"},
		Stdout:        &stdout,
		HTTPTransport: transport,
		Env: func(key string) string {
			switch key {
			case "CTYUN_AK":
				return "ak-test"
			case "CTYUN_SK":
				return "sk-test"
			default:
				return ""
			}
		},
		Config: []byte(`{
  "active_profile": "default",
  "profiles": {
    "default": {
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test"
    }
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !seen {
		t.Fatal("HTTP transport was not used")
	}
	if !strings.Contains(stdout.String(), "ins-live-1") {
		t.Fatalf("live output missing response row:\n%s", stdout.String())
	}
}

func TestOptionalMetadataFlagIsOmittedWhenUnset(t *testing.T) {
	var seenBody string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		seenBody = string(body)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"returnObj":{"results":[{"instanceID":"ins-live-1","displayName":"live","instanceStatus":"running"}]}}`)),
		}, nil
	})

	err := Run(Config{
		Args:          []string{"ecs", "instance", "list", "--cols", "instance_id,name"},
		Stdout:        io.Discard,
		HTTPTransport: transport,
		Env: func(key string) string {
			switch key {
			case "CTYUN_AK":
				return "ak-test"
			case "CTYUN_SK":
				return "sk-test"
			default:
				return ""
			}
		},
		Config: []byte(`{
  "active_profile": "default",
  "profiles": {
    "default": {
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test"
    }
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.Contains(seenBody, "displayName") {
		t.Fatalf("optional displayName was sent without --name: %s", seenBody)
	}
	if !strings.Contains(seenBody, `"regionID":"cn-huadong1"`) {
		t.Fatalf("request body missing region: %s", seenBody)
	}
}

func TestDebugFlagWritesRedactedHTTPDetailsToStderr(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`ak-test sk-test Signature=debug-signature-secret`)),
		}, nil
	})

	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args:          []string{"--debug", "ecs", "instance", "list"},
		Stdout:        &stdout,
		Stderr:        &stderr,
		HTTPTransport: transport,
		Env: func(key string) string {
			switch key {
			case "CTYUN_AK":
				return "ak-test"
			case "CTYUN_SK":
				return "sk-test"
			default:
				return ""
			}
		},
		Config: []byte(`{
  "active_profile": "default",
  "profiles": {
    "default": {
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test"
    }
  }
}`),
	})
	if err == nil {
		t.Fatal("Run returned nil error for HTTP 400")
	}
	got := stderr.String()
	for _, secret := range []string{"ak-test", "sk-test", "debug-signature-secret"} {
		if strings.Contains(got, secret) {
			t.Fatalf("debug stderr still contains %q: %s", secret, got)
		}
	}
	if !strings.Contains(got, "request POST https://ctapi.example.test/v4/ecs/list-instances") {
		t.Fatalf("debug stderr missing request line: %s", got)
	}
	if !strings.Contains(got, "response 400") {
		t.Fatalf("debug stderr missing response status: %s", got)
	}
}
