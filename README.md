# caphcli

`caphcli` is a standalone CLI for [Cluster API Provider
Hetzner](https://github.com/syself/cluster-api-provider-hetzner/)

## Required Environment Variables

Depending on the command, these environment variables are needed.

- `HETZNER_ROBOT_USER` and `HETZNER_ROBOT_PASSWORD` for Hetzner Robot API access.
- `SSH_KEY_NAME` for the Robot SSH key name to use or create.
- One of `HETZNER_SSH_PUB_PATH` or `HETZNER_SSH_PUB` for the SSH public key.
- One of `HETZNER_SSH_PRIV_PATH` or `HETZNER_SSH_PRIV` for the SSH private key.

## Common Usage

If you have Go installed, the easiest way is to run the code like this:

```console
go run github.com/syself/caphcli@latest -h
```

If you have new Hetzner Baremetal (Robot) Server, then create a HetznerBareMetalHost YAML file:

```console
go run github.com/syself/caphcli@latest create-host-yaml 1234567 1234567.yaml
```

This will create a HetznerBareMetalHost YAML file: `1234567.yaml`

The generated host starts with `spec.maintenanceMode: true`, and the command prints a hint to run `check-bm-server` next.

After that you can check if the rescue system is reachable reliably:

```console
go run github.com/syself/caphcli@latest check-bm-server 1234567.yaml
```

`check-bm-server` refuses to run unless `spec.maintenanceMode` is `true`. After a successful check it prints a hint to disable maintenance mode again.

<!-- readmegen:cli-help:start -->

## CLI Help

### `caphcli --help`

```text
CAPH developer and operations CLI

Usage:
  caphcli [command]

Available Commands:
  check-bm-server  Validate rescue and provisioning reliability for one bare-metal server
  completion       Generate the autocompletion script for the specified shell
  create-host-yaml Generate a HetznerBareMetalHost YAML file for one Robot server
  help             Help about any command

Flags:
  -h, --help   help for caphcli

Use "caphcli [command] --help" for more information about a command.
```

### `caphcli check-bm-server --help`

```text
Validate rescue and provisioning reliability for one HetznerBareMetalHost from a local YAML file.

The command does not talk to Kubernetes. It reads one local YAML file containing
HetznerBareMetalHost objects and then talks directly to Hetzner Robot plus the
target server.

Usage:
  caphcli check-bm-server FILE [flags]

Examples:
  caphcli check-bm-server \
    test/e2e/data/infrastructure-hetzner/v1beta1/bases/hetznerbaremetalhosts.yaml \
    --name bm-e2e-1731561

Flags:
      --force                                Skip the destructive-action confirmation prompt
  -h, --help                                 help for check-bm-server
      --image-path string                    Installimage IMAGE path for operating system inside the Hetzner rescue system (default "/root/.oldroot/nfs/images/Ubuntu-2404-noble-amd64-base.tar.gz")
      --name string                          HetznerBareMetalHost metadata.name. Optional if YAML contains exactly one host
      --poll-interval duration               Polling interval for wait steps (default 10s)
      --timeout-activate-rescue duration     Timeout for activating rescue boot (default 45s)
      --timeout-check-disk-rescue duration   Timeout for checking target disks in rescue (default 1m0s)
      --timeout-ensure-ssh-key duration      Timeout for ensuring SSH key in Robot (default 1m0s)
      --timeout-fetch-server duration        Timeout for fetching server details from Robot (default 30s)
      --timeout-install duration             Timeout for one Ubuntu install step (default 9m0s)
      --timeout-load-input duration          Timeout for input parsing + env loading (default 30s)
      --timeout-reboot-os duration           Timeout for rebooting into installed OS (default 45s)
      --timeout-reboot-rescue duration       Timeout for requesting reboot to rescue (default 45s)
      --timeout-wait-os duration             Timeout for waiting until installed OS is reachable (default 6m0s)
      --timeout-wait-rescue duration         Timeout for waiting until rescue SSH is reachable (default 8m0s)
```

### `caphcli create-host-yaml --help`

```text
Generate a HetznerBareMetalHost YAML file for one Hetzner Robot server.

The command talks directly to Hetzner Robot, ensures rescue SSH access, reboots
the target server into rescue once, inspects the available disks, and writes a
YAML file to the requested output path. Progress and confirmation prompts go to stderr.

Usage:
  caphcli create-host-yaml SERVER_ID OUTPUT_FILE [flags]

Examples:
  caphcli create-host-yaml 1751550 host.yaml
  caphcli create-host-yaml --force --name bm-e2e-1751550 1751550 host.yaml

Flags:
      --force                              Skip the reboot confirmation prompt
  -h, --help                               help for create-host-yaml
      --name string                        metadata.name for the generated HetznerBareMetalHost (default: bm-SERVER_ID)
      --poll-interval duration             Polling interval while waiting for rescue SSH (default 10s)
      --timeout-activate-rescue duration   Timeout for activating rescue boot (default 45s)
      --timeout-ensure-ssh-key duration    Timeout for ensuring SSH key in Robot (default 1m0s)
      --timeout-fetch-server duration      Timeout for fetching server details from Robot (default 30s)
      --timeout-load-input duration        Timeout for env loading + initial validation (default 30s)
      --timeout-reboot-rescue duration     Timeout for requesting reboot to rescue (default 45s)
      --timeout-wait-rescue duration       Timeout for waiting until rescue SSH is reachable (default 8m0s)
```

<!-- readmegen:cli-help:end -->
