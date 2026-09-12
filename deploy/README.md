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

# 2. Config (it holds the API key). The service runs as the unprivileged
#    nadi-agent user, so the file must be readable by that group. Install as
#    0600, edit, then hand it to root:nadi-agent with mode 0640 — readable by
#    the service, not world-readable.
sudo mkdir -p /etc/nadi-agent
sudo install -m 0600 agent.yaml.example /etc/nadi-agent/agent.yaml
sudoedit /etc/nadi-agent/agent.yaml
#   set device_id, server_url, api_key, and an absolute buffer_path:
#   buffer_path: "/var/lib/nadi-agent/agent-buffer.db"
sudo chown root:nadi-agent /etc/nadi-agent/agent.yaml
sudo chmod 0640 /etc/nadi-agent/agent.yaml

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

## Upgrade

Config (`/etc/nadi-agent/`) and the buffer (`/var/lib/nadi-agent/`) live outside
the binary, so upgrading is just replacing the binary and restarting. Stop the
service first — overwriting a running executable fails with `Text file busy`.

```sh
# 1. Download the new release. Check uname -m: x86_64 -> amd64, aarch64 -> arm64.
cd /tmp
curl -L -o nadi-agent.tar.gz \
  https://github.com/izzudin96/nadi-agent/releases/latest/download/nadi-agent-linux-amd64.tar.gz
tar -xzf nadi-agent.tar.gz

# 2. Stop, replace the binary, start
sudo systemctl stop nadi-agent
sudo install -m 0755 nadi-agent /usr/local/bin/nadi-agent
sudo systemctl start nadi-agent

# 3. Confirm the new version in the startup log
journalctl -u nadi-agent -n 5 --no-pager
```

The version is only reported in the startup log (there is no `--version` flag).
If the release ships an updated unit file, reinstall it and reload systemd:

```sh
sudo install -m 0644 deploy/nadi-agent.service /etc/systemd/system/nadi-agent.service
sudo systemctl daemon-reload
sudo systemctl restart nadi-agent
```

macOS: `launchctl unload` the plist, replace `/usr/local/bin/nadi-agent`, then
`launchctl load` it again.

To pin a specific version instead of tracking `latest`, use the tagged URL:
`.../releases/download/vX.Y.Z/nadi-agent-linux-amd64.tar.gz`.

### Rollback

There is no built-in rollback. Keep the previous tarball and install it the same
way — the binary is stateless, so a downgrade is just swapping it back:

```sh
curl -L -o nadi-agent.tar.gz \
  https://github.com/izzudin96/nadi-agent/releases/download/v0.1.0/nadi-agent-linux-amd64.tar.gz
tar -xzf nadi-agent.tar.gz
sudo systemctl stop nadi-agent
sudo install -m 0755 nadi-agent /usr/local/bin/nadi-agent
sudo systemctl start nadi-agent
```

Before downgrading, check the release notes: the older payload must still match
the server's heartbeat API and be able to read the existing buffer file.

## Uninstall

Removing the agent does **not** delete the device on the server — its stored
metrics stay in the dashboard. If you are decommissioning the host, also delete
the device from the dashboard (that removes its stored metrics).

### Linux (systemd)

```sh
# 1. Stop and disable the service
sudo systemctl disable --now nadi-agent

# 2. Remove the unit
sudo rm /etc/systemd/system/nadi-agent.service
sudo systemctl daemon-reload
sudo systemctl reset-failed nadi-agent

# 3. Remove the binary, config, and buffered data
sudo rm /usr/local/bin/nadi-agent
sudo rm -rf /etc/nadi-agent /var/lib/nadi-agent

# 4. Remove the service user
sudo userdel nadi-agent

# 5. If you granted SMART access, undo it
sudo setcap -r /usr/sbin/smartctl
```

`/etc/nadi-agent/agent.yaml` holds the API key, so step 3 removes the agent's
local copy — but the key stays valid server-side until you rotate or delete the
device.

### macOS (launchd)

```sh
launchctl unload ~/Library/LaunchAgents/com.izzudin96.nadi-agent.plist
rm ~/Library/LaunchAgents/com.izzudin96.nadi-agent.plist
sudo rm /usr/local/bin/nadi-agent
rm -rf "$HOME/Library/Application Support/nadi-agent"
```

On recent macOS, `launchctl bootout gui/$(id -u)/com.izzudin96.nadi-agent` is
the modern equivalent of `unload`.

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
