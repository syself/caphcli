/*
Copyright 2026 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package provisioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	infrav1 "github.com/syself/cluster-api-provider-hetzner/api/v1beta1"
)

func TestDefaultConfigWaitForRescueTimeout(t *testing.T) {
	t.Parallel()

	if got, want := DefaultConfig().Timeouts.WaitForRescue, 8*time.Minute; got != want {
		t.Fatalf("DefaultConfig().Timeouts.WaitForRescue = %s, want %s", got, want)
	}
}

func TestStepWarningPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		used float64
		want string
	}{
		{
			name: "below threshold",
			used: 79.9,
			want: "",
		},
		{
			name: "at threshold",
			used: 80.0,
			want: "⚠️ ",
		},
		{
			name: "above threshold",
			used: 95.5,
			want: "⚠️ ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := stepWarningPrefix(tt.used); got != tt.want {
				t.Fatalf("stepWarningPrefix(%v) = %q, want %q", tt.used, got, tt.want)
			}
		})
	}
}

func TestLoadHostsFromHBMHYAMLFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		content   string
		wantNames []string
	}{
		{
			name: "multi document",
			content: `apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: HetznerBareMetalHost
metadata:
  name: alpha
spec:
  rootDeviceHints:
    wwn: "0x1"
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: HetznerBareMetalHost
metadata:
  name: beta
spec:
  rootDeviceHints:
    wwn: "0x2"
`,
			wantNames: []string{"alpha", "beta"},
		},
		{
			name: "top level items list",
			content: `apiVersion: v1
items:
- apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
  kind: HetznerBareMetalHost
  metadata:
    name: alpha
  spec:
    rootDeviceHints:
      wwn: "0x1"
- apiVersion: v1
  kind: Secret
  metadata:
    name: ignored
- apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
  kind: HetznerBareMetalHost
  metadata:
    name: beta
  spec:
    rootDeviceHints:
      wwn: "0x2"
`,
			wantNames: []string{"alpha", "beta"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := filepath.Join(dir, "hosts.yaml")
			if err := os.WriteFile(path, []byte(tt.content), 0o600); err != nil {
				t.Fatalf("write test yaml: %v", err)
			}

			hosts, err := loadHostsFromHBMHYAMLFile(path)
			if err != nil {
				t.Fatalf("loadHostsFromHBMHYAMLFile() error = %v", err)
			}
			if len(hosts) != len(tt.wantNames) {
				t.Fatalf("len(hosts) = %d, want %d", len(hosts), len(tt.wantNames))
			}
			for i, wantName := range tt.wantNames {
				if hosts[i].Name != wantName {
					t.Fatalf("hosts[%d].Name = %q, want %q", i, hosts[i].Name, wantName)
				}
			}
		})
	}
}

func TestSelectHostRequiresMaintenanceMode(t *testing.T) {
	t.Parallel()

	trueValue := true
	falseValue := false

	tests := []struct {
		name    string
		host    infrav1.HetznerBareMetalHost
		wantErr string
	}{
		{
			name: "maintenance mode unset",
			host: infrav1.HetznerBareMetalHost{
				Spec: infrav1.HetznerBareMetalHostSpec{
					RootDeviceHints: &infrav1.RootDeviceHints{WWN: "0x1"},
				},
			},
			wantErr: `must set spec.maintenanceMode: true`,
		},
		{
			name: "maintenance mode false",
			host: infrav1.HetznerBareMetalHost{
				Spec: infrav1.HetznerBareMetalHostSpec{
					RootDeviceHints: &infrav1.RootDeviceHints{WWN: "0x1"},
					MaintenanceMode: &falseValue,
				},
			},
			wantErr: `must set spec.maintenanceMode: true`,
		},
		{
			name: "maintenance mode true",
			host: infrav1.HetznerBareMetalHost{
				Spec: infrav1.HetznerBareMetalHostSpec{
					RootDeviceHints: &infrav1.RootDeviceHints{WWN: "0x1"},
					MaintenanceMode: &trueValue,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.host.Name = tt.name
			_, err := selectHost([]infrav1.HetznerBareMetalHost{tt.host}, "")
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("selectHost() error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("selectHost() error = nil, want substring %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("selectHost() error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}
