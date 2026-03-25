# caphcli

`caphcli` is a standalone CLI for [Cluster API Provider
Hetzner](https://github.com/syself/cluster-api-provider-hetzner/)

## Required Environment Variables

Depending on the command, these environment variables are needed.

- `HETZNER_ROBOT_USER` and `HETZNER_ROBOT_PASSWORD` for Hetzner Robot API access.
- `SSH_KEY_NAME` for the Robot SSH key name to use or create.
- One of `HETZNER_SSH_PUB_PATH` or `HETZNER_SSH_PUB` for the SSH public key.
- One of `HETZNER_SSH_PRIV_PATH` or `HETZNER_SSH_PRIV` for the SSH private key.

<!-- readmegen:cli-help:start -->

## CLI Help

### `caphcli --help`

```text
CAPH developer and operations CLI

Usage:
  caphcli [command]

Available Commands:
  check-bm-servers Validate rescue and provisioning reliability for one bare-metal server
  completion       Generate the autocompletion script for the specified shell
  help             Help about any command

Flags:
  -h, --help   help for caphcli

Use "caphcli [command] --help" for more information about a command.
```

### `caphcli check-bm-servers --help`

```text
Validate rescue and provisioning reliability for one HetznerBareMetalHost from a local YAML file.

The command does not talk to Kubernetes. It reads one local YAML file containing
HetznerBareMetalHost objects and then talks directly to Hetzner Robot plus the
target server.

Usage:
  caphcli check-bm-servers [flags]

Examples:
  caphcli check-bm-servers \
    --file test/e2e/data/infrastructure-hetzner/v1beta1/bases/hetznerbaremetalhosts.yaml \
    --name bm-e2e-1731561

Flags:
      --file string                          Path to a local YAML file containing HetznerBareMetalHost objects (required)
      --force                                Skip the destructive-action confirmation prompt
  -h, --help                                 help for check-bm-servers
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
      --timeout-wait-rescue duration         Timeout for waiting until rescue SSH is reachable (default 6m0s)
```

<!-- readmegen:cli-help:end -->
