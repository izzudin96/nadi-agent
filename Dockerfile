# Build stage: static Linux binary.
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o /out/nadi-agent ./cmd/nadi-agent

# Runtime stage. This image is a test harness for Linux behaviour (PLAN §6.4):
# the agent needs host namespaces for real metrics, so run with e.g.
#   docker run --pid=host --net=host -v /:/host:ro nadi-agent ...
# Production deployment uses the systemd unit (deploy/nadi-agent.service), not
# this container.
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/nadi-agent /usr/local/bin/nadi-agent
ENTRYPOINT ["nadi-agent"]
