package createhosttemplate

import (
	"strings"
	"testing"

	"github.com/syself/hrobot-go/models"

	sshclient "github.com/syself/cluster-api-provider-hetzner/pkg/services/baremetal/client/ssh"
)

func TestDisksFromStorageOutput(t *testing.T) {
	t.Parallel()

	out := sshclient.Output{
		StdOut: `NAME="loop0" TYPE="loop" SIZE="3068773888" WWN=""
NAME="sda" TYPE="disk" SIZE="1000000000" WWN="0x-too-small"
NAME="sdb" TYPE="disk" SIZE="2000000000" WWN="0x0002"
NAME="sdc" TYPE="disk" SIZE="2000000000" WWN="0x0001"
NAME="sdd" TYPE="disk" SIZE="4000000000" WWN=""
NAME="sde" TYPE="disk" SIZE="8000000000" WWN="0x0003"`,
	}

	disks, err := disksFromStorageOutput(out)
	if err != nil {
		t.Fatalf("disksFromStorageOutput() error = %v", err)
	}

	if len(disks) != 4 {
		t.Fatalf("disksFromStorageOutput() len = %d, want 4", len(disks))
	}

	if disks[0].WWN != "0x-too-small" || disks[0].SizeBytes != 1000000000 {
		t.Fatalf("first disk = %+v, want WWN 0x-too-small and size 1000000000", disks[0])
	}
	if disks[1].WWN != "0x0001" || disks[1].SizeBytes != 2000000000 {
		t.Fatalf("second disk = %+v, want WWN 0x0001 and size 2000000000", disks[1])
	}
	if disks[2].WWN != "0x0002" || disks[2].SizeBytes != 2000000000 {
		t.Fatalf("third disk = %+v, want WWN 0x0002 and size 2000000000", disks[2])
	}
	if disks[3].WWN != "0x0003" || disks[3].SizeBytes != 8000000000 {
		t.Fatalf("fourth disk = %+v, want WWN 0x0003 and size 8000000000", disks[3])
	}

	selected, selectedIndex, err := selectDisk(disks)
	if err != nil {
		t.Fatalf("selectDisk() error = %v", err)
	}
	if selectedIndex != 1 || selected.WWN != "0x0001" {
		t.Fatalf("selectDisk() = (%+v, %d), want WWN 0x0001 at index 1", selected, selectedIndex)
	}
}

func TestRenderTemplate(t *testing.T) {
	t.Parallel()

	server := &models.Server{
		ServerNumber: 1751550,
		ServerIP:     "144.76.74.13",
		Name:         "ci-box-1751550",
	}
	disks := []disk{
		{Name: "nvme1n1", WWN: "0x0001", SizeBytes: 2000000000},
		{Name: "nvme2n1", WWN: "0x0002", SizeBytes: 4000000000},
	}

	got := renderTemplate(server, effectiveName("", server.ServerNumber), disks)

	wantContains := []string{
		`name: "bm-1751550"`,
		`serverID: 1751550 # Robot name: ci-box-1751550, IP: 144.76.74.13`,
		`wwn: "0x0001"`,
		`# wwn: "0x0002"`,
		`maintenanceMode: false`,
		`description: "ci-box-1751550"`,
	}

	for _, want := range wantContains {
		if !strings.Contains(got, want) {
			t.Fatalf("renderTemplate() missing %q in output:\n%s", want, got)
		}
	}
}
