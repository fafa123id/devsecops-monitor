# -------------------------------------------------
# Build stage
# -------------------------------------------------
FROM golang:1.27.3-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS:-linux} \
    GOARCH=${TARGETARCH:-amd64} \
    go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/devsecops-agent .

# -------------------------------------------------
# Runtime stage
# -------------------------------------------------
# Docker CLI is required by the agent's DockerExecutor.
FROM docker:cli

RUN apk add --no-cache ca-certificates tzdata

ENV TZ=Asia/Jakarta

WORKDIR /app

COPY --from=builder /out/devsecops-agent /usr/local/bin/devsecops-agent

ENTRYPOINT ["/usr/local/bin/devsecops-agent"]

CMD []