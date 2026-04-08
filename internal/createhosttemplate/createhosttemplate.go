package createhosttemplate

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/syself/hrobot-go/models"

	infrav1 "github.com/syself/cluster-api-provider-hetzner/api/v1beta1"
	robotclient "github.com/syself/cluster-api-provider-hetzner/pkg/services/baremetal/client/robot"
	sshclient "github.com/syself/cluster-api-provider-hetzner/pkg/services/baremetal/client/ssh"

	"github.com/syself/caphcli/internal/provisioncheck"
)

const (
	minDiskSizeBytes int64 = 1_000_000_000
	rescueHostName         = "rescue"
	sshPort                = 22
)

type Timeouts struct {
	LoadInput          time.Duration
	EnsureSSHKey       time.Duration
	FetchServerDetails time.Duration
	ActivateRescue     time.Duration
	RebootToRescue     time.Duration
	WaitForRescue      time.Duration
}

type Config struct {
	ServerID     int
	Name         string
	Force        bool
	PollInterval time.Duration
	Timeouts     Timeouts
	Input        io.Reader
	Output       io.Writer
	LogOutput    io.Writer
}

type runner struct {
	cfg         Config
	sshFactory  sshclient.Factory
	robotClient robotclient.Client
	creds       envCredentials
	fingerprint string
	server      *models.Server
}

type envCredentials struct {
	robotUser  string
	robotPass  string
	sshKeyName string
	sshPub     string
	sshPriv    string
}

type disk struct {
	Name      string
	WWN       string
	SizeBytes int64
}

type storageDetails struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
	Size string `json:"size,omitempty"`
	WWN  string `json:"wwn,omitempty"`
}

func DefaultConfig() Config {
	return Config{
		PollInterval: provisioncheck.DefaultPollInterval,
		Timeouts: Timeouts{
			LoadInput:          provisioncheck.DefaultLoadInputTimeout,
			EnsureSSHKey:       provisioncheck.DefaultEnsureSSHKeyTimeout,
			FetchServerDetails: provisioncheck.DefaultFetchServerDetailsTimeout,
			ActivateRescue:     provisioncheck.DefaultActivateRescueTimeout,
			RebootToRescue:     provisioncheck.DefaultRebootToRescueTimeout,
			WaitForRescue:      provisioncheck.DefaultWaitForRescueTimeout,
		},
		Input:     os.Stdin,
		Output:    os.Stdout,
		LogOutput: os.Stderr,
	}
}

func (cfg Config) withDefaults() Config {
	defaults := DefaultConfig()

	if cfg.PollInterval == 0 {
		cfg.PollInterval = defaults.PollInterval
	}
	if cfg.Input == nil {
		cfg.Input = defaults.Input
	}
	if cfg.Output == nil {
		cfg.Output = defaults.Output
	}
	if cfg.LogOutput == nil {
		cfg.LogOutput = defaults.LogOutput
	}
	if cfg.Timeouts.LoadInput == 0 {
		cfg.Timeouts.LoadInput = defaults.Timeouts.LoadInput
	}
	if cfg.Timeouts.EnsureSSHKey == 0 {
		cfg.Timeouts.EnsureSSHKey = defaults.Timeouts.EnsureSSHKey
	}
	if cfg.Timeouts.FetchServerDetails == 0 {
		cfg.Timeouts.FetchServerDetails = defaults.Timeouts.FetchServerDetails
	}
	if cfg.Timeouts.ActivateRescue == 0 {
		cfg.Timeouts.ActivateRescue = defaults.Timeouts.ActivateRescue
	}
	if cfg.Timeouts.RebootToRescue == 0 {
		cfg.Timeouts.RebootToRescue = defaults.Timeouts.RebootToRescue
	}
	if cfg.Timeouts.WaitForRescue == 0 {
		cfg.Timeouts.WaitForRescue = defaults.Timeouts.WaitForRescue
	}

	return cfg
}

func (cfg Config) Validate() error {
	if cfg.ServerID <= 0 {
		return fmt.Errorf("server id must be > 0, got %d", cfg.ServerID)
	}
	if cfg.Input == nil {
		return errors.New("config Input must not be nil")
	}
	if cfg.Output == nil {
		return errors.New("config Output must not be nil")
	}
	if cfg.LogOutput == nil {
		return errors.New("config LogOutput must not be nil")
	}
	if cfg.PollInterval <= 0 {
		return fmt.Errorf("--poll-interval must be > 0, got %s", cfg.PollInterval)
	}
	if err := validateTimeout("--timeout-load-input", cfg.Timeouts.LoadInput); err != nil {
		return err
	}
	if err := validateTimeout("--timeout-ensure-ssh-key", cfg.Timeouts.EnsureSSHKey); err != nil {
		return err
	}
	if err := validateTimeout("--timeout-fetch-server", cfg.Timeouts.FetchServerDetails); err != nil {
		return err
	}
	if err := validateTimeout("--timeout-activate-rescue", cfg.Timeouts.ActivateRescue); err != nil {
		return err
	}
	if err := validateTimeout("--timeout-reboot-rescue", cfg.Timeouts.RebootToRescue); err != nil {
		return err
	}
	if err := validateTimeout("--timeout-wait-rescue", cfg.Timeouts.WaitForRescue); err != nil {
		return err
	}

	return nil
}

func Run(ctx context.Context, cfg Config) error {
	cfg = cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		return err
	}

	r := &runner{
		cfg:        cfg,
		sshFactory: sshclient.NewFactory(),
	}

	if err := runWithTimeout(ctx, cfg.Timeouts.LoadInput, func(context.Context) error {
		creds, err := loadEnvCredentials()
		if err != nil {
			return err
		}
		r.creds = creds
		return nil
	}); err != nil {
		return err
	}

	r.robotClient = robotclient.NewFactory().NewClient(robotclient.Credentials{
		Username: r.creds.robotUser,
		Password: r.creds.robotPass,
	})

	if err := r.ensureSSHKey(ctx); err != nil {
		return err
	}
	if err := r.fetchServerDetails(ctx); err != nil {
		return err
	}
	if err := r.confirmRescueReboot(); err != nil {
		return err
	}
	if err := r.activateRescue(ctx); err != nil {
		return err
	}
	if err := r.rebootToRescue(ctx); err != nil {
		return err
	}

	ssh, err := r.waitForRescue(ctx)
	if err != nil {
		return err
	}

	disks, err := disksFromStorageOutput(ssh.GetHardwareDetailsStorage())
	if err != nil {
		return err
	}

	template := renderTemplate(r.server, effectiveName(cfg.Name, cfg.ServerID), disks)
	if _, err := io.WriteString(cfg.Output, template); err != nil {
		return fmt.Errorf("write template: %w", err)
	}

	return nil
}

func (r *runner) ensureSSHKey(ctx context.Context) error {
	return runWithTimeout(ctx, r.cfg.Timeouts.EnsureSSHKey, func(context.Context) error {
		r.logf("ensuring Robot SSH key %q", r.creds.sshKeyName)

		fingerprint, err := ensureRobotSSHKey(r.robotClient, r.creds.sshKeyName, r.creds.sshPub)
		if err != nil {
			return err
		}

		r.fingerprint = fingerprint
		r.logf("using Robot SSH key fingerprint %q", r.fingerprint)
		return nil
	})
}

func (r *runner) fetchServerDetails(ctx context.Context) error {
	return runWithTimeout(ctx, r.cfg.Timeouts.FetchServerDetails, func(context.Context) error {
		r.logf("fetching Robot server %d", r.cfg.ServerID)

		server, err := r.robotClient.GetBMServer(r.cfg.ServerID)
		if err != nil {
			return fmt.Errorf("get robot server %d: %w", r.cfg.ServerID, err)
		}
		if server.ServerIP == "" {
			return fmt.Errorf("server %d has empty server_ip in Robot API", r.cfg.ServerID)
		}

		r.server = server
		r.logf("server %d name=%q ip=%s", r.cfg.ServerID, server.Name, server.ServerIP)
		return nil
	})
}

func (r *runner) confirmRescueReboot() error {
	if r.cfg.Force {
		r.logf("confirmation skipped because --force was provided")
		return nil
	}

	_, err := fmt.Fprintf(
		r.cfg.LogOutput,
		"WARNING: this will reboot server %d (%q, %s) into rescue to inspect its disks.\nType \"yes\" to continue: ",
		r.cfg.ServerID,
		r.server.Name,
		r.server.ServerIP,
	)
	if err != nil {
		return fmt.Errorf("write confirmation prompt: %w", err)
	}

	reader := bufio.NewReader(r.cfg.Input)
	confirmation, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read confirmation: %w", err)
	}

	confirmation = strings.TrimSpace(confirmation)
	if confirmation != "yes" {
		return fmt.Errorf("confirmation failed: expected %q, got %q", "yes", confirmation)
	}

	r.logf("reboot confirmed for server %d", r.cfg.ServerID)
	return nil
}

func (r *runner) activateRescue(ctx context.Context) error {
	return runWithTimeout(ctx, r.cfg.Timeouts.ActivateRescue, func(context.Context) error {
		r.logf("activating rescue boot")

		_, deleteErr := r.robotClient.DeleteBootRescue(r.cfg.ServerID)
		if deleteErr != nil && !models.IsError(deleteErr, models.ErrorCodeNotFound) {
			return fmt.Errorf("delete boot rescue: %w", deleteErr)
		}
		if _, err := r.robotClient.SetBootRescue(r.cfg.ServerID, r.fingerprint); err != nil {
			return fmt.Errorf("set boot rescue: %w", err)
		}

		r.logf("rescue boot activated")
		return nil
	})
}

func (r *runner) rebootToRescue(ctx context.Context) error {
	return runWithTimeout(ctx, r.cfg.Timeouts.RebootToRescue, func(context.Context) error {
		r.logf("requesting hardware reboot into rescue")

		if _, err := r.robotClient.RebootBMServer(r.cfg.ServerID, infrav1.RebootTypeHardware); err != nil {
			return fmt.Errorf("robot reboot hw: %w", err)
		}

		return nil
	})
}

func (r *runner) waitForRescue(ctx context.Context) (sshclient.Client, error) {
	var ssh sshclient.Client
	err := runWithTimeout(ctx, r.cfg.Timeouts.WaitForRescue, func(stepCtx context.Context) error {
		ssh = r.sshFactory.NewClient(sshclient.Input{
			IP:         r.server.ServerIP,
			Port:       sshPort,
			PrivateKey: r.creds.sshPriv,
		})

		r.logf("waiting for rescue SSH on %s", r.server.ServerIP)
		return waitUntil(stepCtx, r.cfg.PollInterval, func(format string, args ...any) {
			r.logf(format, args...)
		}, func() (bool, string, error) {
			out := ssh.GetHostName()
			if out.Err == nil {
				hostName := strings.TrimSpace(out.StdOut)
				if hostName == rescueHostName {
					return true, fmt.Sprintf("rescue reachable (hostname=%q)", hostName), nil
				}
				if hostName == "" {
					return false, "connected but empty hostname", nil
				}
				return false, fmt.Sprintf("host reachable but hostname=%q (want=%q)", hostName, rescueHostName), nil
			}
			return false, fmt.Sprintf("waiting for rescue ssh: %v", out.Err), nil
		})
	})
	if err != nil {
		return nil, err
	}

	return ssh, nil
}

func (r *runner) logf(format string, args ...any) {
	_, _ = fmt.Fprintf(r.cfg.LogOutput, "%s\n", fmt.Sprintf(format, args...))
}

func runWithTimeout(ctx context.Context, timeout time.Duration, fn func(context.Context) error) error {
	stepCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := fn(stepCtx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(stepCtx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("timed out after %s: %w", timeout, err)
		}
		return err
	}

	return nil
}

func validateTimeout(flagName string, timeout time.Duration) error {
	if timeout <= 0 {
		return fmt.Errorf("%s must be > 0, got %s", flagName, timeout)
	}
	return nil
}

func waitUntil(ctx context.Context, pollInterval time.Duration, progress func(format string, args ...any), check func() (done bool, message string, err error)) error {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		done, message, err := check()
		if err != nil {
			return err
		}
		if message != "" {
			progress("%s", message)
		}
		if done {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func ensureRobotSSHKey(cli robotclient.Client, keyName, publicKey string) (string, error) {
	keys, err := cli.ListSSHKeys()
	if err != nil {
		return "", fmt.Errorf("list ssh keys: %w", err)
	}
	for _, key := range keys {
		if key.Name == keyName {
			return key.Fingerprint, nil
		}
	}

	created, err := cli.SetSSHKey(keyName, publicKey)
	if err != nil {
		return "", fmt.Errorf("create ssh key %q: %w", keyName, err)
	}
	return created.Fingerprint, nil
}

func loadEnvCredentials() (envCredentials, error) {
	user := strings.TrimSpace(os.Getenv("HETZNER_ROBOT_USER"))
	pass := strings.TrimSpace(os.Getenv("HETZNER_ROBOT_PASSWORD"))
	if user == "" || pass == "" {
		return envCredentials{}, errors.New("HETZNER_ROBOT_USER and HETZNER_ROBOT_PASSWORD are required")
	}

	keyName := strings.TrimSpace(os.Getenv("SSH_KEY_NAME"))
	if keyName == "" {
		return envCredentials{}, errors.New("SSH_KEY_NAME is required")
	}

	sshPub, err := loadKeyMaterial("HETZNER_SSH_PUB_PATH", "HETZNER_SSH_PUB")
	if err != nil {
		return envCredentials{}, fmt.Errorf("load public key: %w", err)
	}
	sshPriv, err := loadKeyMaterial("HETZNER_SSH_PRIV_PATH", "HETZNER_SSH_PRIV")
	if err != nil {
		return envCredentials{}, fmt.Errorf("load private key: %w", err)
	}

	return envCredentials{
		robotUser:  user,
		robotPass:  pass,
		sshKeyName: keyName,
		sshPub:     strings.TrimSpace(sshPub),
		sshPriv:    strings.TrimSpace(sshPriv),
	}, nil
}

func loadKeyMaterial(pathVar, base64Var string) (string, error) {
	path := strings.TrimSpace(os.Getenv(pathVar))
	if path != "" {
		data, err := os.ReadFile(path) // #nosec G304,G703 -- file path is intentionally provided via environment variable.
		if err != nil {
			return "", fmt.Errorf("read %s (%s): %w", pathVar, path, err)
		}
		if len(data) == 0 {
			return "", fmt.Errorf("%s points to empty file: %s", pathVar, path)
		}
		return string(data), nil
	}

	raw := strings.TrimSpace(os.Getenv(base64Var))
	if raw == "" {
		return "", fmt.Errorf("set either %s or %s", pathVar, base64Var)
	}

	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err == nil {
		if len(decoded) == 0 {
			return "", fmt.Errorf("%s decoded to empty value", base64Var)
		}
		return string(decoded), nil
	}

	return raw, nil
}

func disksFromStorageOutput(out sshclient.Output) ([]disk, error) {
	if out.Err != nil {
		return nil, fmt.Errorf("get hardware details storage: %w", out.Err)
	}
	if strings.TrimSpace(out.StdOut) == "" {
		return nil, errors.New("storage output is empty")
	}

	lines := strings.Split(strings.TrimSpace(out.StdOut), "\n")
	disks := make([]disk, 0, len(lines))
	for _, line := range lines {
		var diskInfo storageDetails
		if err := json.Unmarshal([]byte(validJSONFromSSHOutput(line)), &diskInfo); err != nil {
			return nil, fmt.Errorf("parse lsblk line %q: %w", line, err)
		}
		if diskInfo.Type != "disk" {
			continue
		}

		sizeBytes, err := strconv.ParseInt(strings.TrimSpace(diskInfo.Size), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse disk size %q for %q: %w", diskInfo.Size, diskInfo.Name, err)
		}

		wwn := strings.TrimSpace(diskInfo.WWN)
		if wwn == "" {
			continue
		}

		disks = append(disks, disk{
			Name:      strings.TrimSpace(diskInfo.Name),
			WWN:       wwn,
			SizeBytes: sizeBytes,
		})
	}

	sort.Slice(disks, func(i, j int) bool {
		if disks[i].SizeBytes != disks[j].SizeBytes {
			return disks[i].SizeBytes < disks[j].SizeBytes
		}
		return normalizeWWN(disks[i].WWN) < normalizeWWN(disks[j].WWN)
	})

	if len(disks) == 0 {
		return nil, errors.New("no disk with WWN found")
	}
	if _, _, err := selectDisk(disks); err != nil {
		return nil, err
	}

	return disks, nil
}

func selectDisk(disks []disk) (disk, int, error) {
	for idx, disk := range disks {
		if disk.SizeBytes > minDiskSizeBytes {
			return disk, idx, nil
		}
	}

	return disk{}, -1, fmt.Errorf("no disk with WWN and size > %d bytes found", minDiskSizeBytes)
}

func effectiveName(name string, serverID int) string {
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	return fmt.Sprintf("bm-%d", serverID)
}

func renderTemplate(server *models.Server, name string, disks []disk) string {
	selected, selectedIndex, err := selectDisk(disks)
	if err != nil {
		panic(err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Generated from Hetzner Robot server %d.\n", server.ServerNumber)
	b.WriteString("# Candidate WWNs are sorted by disk size, then WWN.\n")
	fmt.Fprintf(&b, "# The selected WWN is the smallest disk above %d bytes.\n", minDiskSizeBytes)
	b.WriteString("# Review the selected disk before applying this object.\n")
	b.WriteString("apiVersion: infrastructure.cluster.x-k8s.io/v1beta1\n")
	b.WriteString("kind: HetznerBareMetalHost\n")
	b.WriteString("metadata:\n")
	fmt.Fprintf(&b, "  name: %q\n", name)
	b.WriteString("spec:\n")
	fmt.Fprintf(&b, "  serverID: %d", server.ServerNumber)
	if suffix := robotServerComment(server); suffix != "" {
		fmt.Fprintf(&b, " # %s", suffix)
	}
	b.WriteString("\n")
	b.WriteString("  rootDeviceHints:\n")
	for idx, disk := range disks {
		if idx == selectedIndex {
			fmt.Fprintf(&b, "    wwn: %q\n", selected.WWN)
			continue
		}
		fmt.Fprintf(&b, "    # wwn: %q\n", disk.WWN)
	}
	b.WriteString("  maintenanceMode: false\n")
	fmt.Fprintf(&b, "  description: %q\n", defaultDescription(server))
	return b.String()
}

func robotServerComment(server *models.Server) string {
	parts := make([]string, 0, 2)
	if name := sanitizeComment(server.Name); name != "" {
		parts = append(parts, fmt.Sprintf("Robot name: %s", name))
	}
	if ip := sanitizeComment(server.ServerIP); ip != "" {
		parts = append(parts, fmt.Sprintf("IP: %s", ip))
	}
	return strings.Join(parts, ", ")
}

func defaultDescription(server *models.Server) string {
	if name := strings.TrimSpace(server.Name); name != "" {
		return name
	}
	return fmt.Sprintf("Robot server %d", server.ServerNumber)
}

func sanitizeComment(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "#", "")
	return strings.TrimSpace(value)
}

func validJSONFromSSHOutput(str string) string {
	tempString1 := strings.ReplaceAll(str, `" `, `","`)
	tempString2 := strings.ReplaceAll(tempString1, `="`, `":"`)
	return fmt.Sprintf(`{"%s}`, strings.TrimSpace(tempString2))
}

func normalizeWWN(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
