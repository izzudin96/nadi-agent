# nadi-agent

Device agent for Nadi: collects system metrics and pushes heartbeats to the nadi-server.

## Deployment

See [`deploy/README.md`](deploy/README.md) for installing the agent as a systemd
service (Linux) or launchd agent (macOS).

## Development

Prerequisites: Go 1.27+ (see `go.mod`), GNU make.

```sh
make build        # builds bin/nadi-agent for the current OS/arch
make build-linux  # cross-compiles a Linux amd64 binary (CGO disabled)
make test         # run tests
make lint         # gofmt check + go vet
make run          # run from source
```

## Project layout

```
cmd/nadi-agent/   entrypoint
```

See `PLAN.md` (parent repo) for the full roadmap.
