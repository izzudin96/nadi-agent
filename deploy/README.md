# Deploying nadi-agent

The agent is a single static binary that only makes outbound HTTPS requests.
It is packaged for Linux (systemd) and macOS (launchd).

## Linux (systemd) — the primary target

```sh
# 1. Build the static Linux binary
make build-linux

# 2. Install the binary and create the service user
sudo install -m 0755 bin/nadi-agent-linux-amd64 /usr/local/bin/nadi-agent
sudo useradd --system --no-create-home --home /var/lib/nadi-agent nadi-agent

# 3. Config (chmod 600 — it holds the API key)
sudo mkdir -p /etc/nadi-agent
sudo install -m 0600 agent.yaml /etc/nadi-agent/agent.yaml

# 4. Writable buffer directory
sudo mkdir -p /var/lib/nadi-agent
sudo chown nadi-agent:nadi-agent /var/lib/nadi-agent

# 5. Install and enable the unit
sudo install -m 0644 deploy/nadi-agent.service /etc/systemd/system/nadi-agent.service
sudo systemctl daemon-reload
sudo systemctl enable --now nadi-agent

# 6. Watch the logs
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
make build
sudo cp bin/nadi-agent /usr/local/bin/nadi-agent
mkdir -p "$HOME/Library/Application Support/nadi-agent"
# create agent.yaml there (chmod 600), then edit the path in the plist
cp deploy/com.izzudin96.nadi-agent.plist ~/Library/LaunchAgents/
launchctl load ~/Library/LaunchAgents/com.izzudin96.nadi-agent.plist
```

## Windows

Not packaged in v1; a Windows Service wrapper can be added later if needed.
