# Deploying nadi-agent

The agent is a single static binary that only makes outbound HTTPS requests.
It is packaged for Linux (systemd) and macOS (launchd).

## Install a release (recommended)

Download the prebuilt tarball for your platform — no Go toolchain is required
on the host. Each tarball also contains the service unit and an example config.

| Platform    | Tarball                          |
| ----------- | -------------------------------- |
| Linux amd64 | `nadi-agent-linux-amd64.tar.gz`  |
| Linux arm64 | `nadi-agent-linux-arm64.tar.gz`  |
| macOS amd64 | `nadi-agent-darwin-amd64.tar.gz` |
| macOS arm64 | `nadi-agent-darwin-arm64.tar.gz` |

```sh
# Example: Linux amd64
curl -L -o nadi-agent.tar.gz \
  https://github.com/izzudin96/nadi-agent/releases/latest/download/nadi-agent-linux-amd64.tar.gz
tar -xzf nadi-agent.tar.gz
```

Then continue with the systemd or launchd steps below.

## Linux (systemd) — the primary target

```sh
# 1. Install the binary and create the service user
sudo install -m 0755 nadi-agent /usr/local/bin/nadi-agent
sudo useradd --system --no-create-home --home /var/lib/nadi-agent nadi-agent

# 2. Config (chmod 600 — it holds the API key). Start from the example.
sudo mkdir -p /etc/nadi-agent
sudo install -m 0600 agent.yaml.example /etc/nadi-agent/agent.yaml
sudoedit /etc/nadi-agent/agent.yaml
#   set device_id, server_url, api_key, and an absolute buffer_path:
#   buffer_path: "/var/lib/nadi-agent/agent-buffer.db"

# 3. Writable buffer directory
sudo mkdir -p /var/lib/nadi-agent
sudo chown nadi-agent:nadi-agent /var/lib/nadi-agent

# 4. Install and enable the unit
sudo install -m 0644 deploy/nadi-agent.service /etc/systemd/system/nadi-agent.service
sudo systemctl daemon-reload
sudo systemctl enable --now nadi-agent

# 5. Watch the logs
journalctl -u nadi-agent -f
```

The unit runs as an unprivileged `nadi-agent` user with a hardened sandbox
(`ProtectSystem=strict`, no capabilities). Config is read-only; only the buffer
directory is writable.

### disk.smart privileges

Reading SMART data needs raw disk access. The agent runs unprivileged, so grant
just the `smartctl` binary the needed capability (matches the current code,
which calls `smartctl` directly):

```sh
sudo setcap cap_sys_rawio+ep /usr/sbin/smartctl
```

Alternative: a passwordless sudoers rule for `smartctl` only
(`deploy/nadi-agent.sudoers`) — but that requires the agent to shell out via
`sudo smartctl`, which is not yet implemented.

## macOS (launchd) — dev machine

```sh
sudo install -m 0755 nadi-agent /usr/local/bin/nadi-agent
mkdir -p "$HOME/Library/Application Support/nadi-agent"
cp agent.yaml.example "$HOME/Library/Application Support/nadi-agent/agent.yaml"
chmod 600 "$HOME/Library/Application Support/nadi-agent/agent.yaml"
# edit the config (use an absolute buffer_path) and the plist (set USERNAME)
cp deploy/com.izzudin96.nadi-agent.plist ~/Library/LaunchAgents/
launchctl load ~/Library/LaunchAgents/com.izzudin96.nadi-agent.plist
```

## Windows

Not packaged in v1; a Windows Service wrapper can be added later if needed.

## Build from source

For contributors, or platforms without a prebuilt binary. Requires Go (see
`go.mod`) and GNU make.

```sh
git clone https://github.com/izzudin96/nadi-agent
cd nadi-agent
make build-linux   # cross-compiles bin/nadi-agent-linux-amd64
make build         # builds bin/nadi-agent for the current OS/arch
```

Then use the binary from `bin/` in place of `nadi-agent` in the steps above.
