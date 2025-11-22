FROM --platform=$BUILDPLATFORM golang:1.25-bookworm AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

RUN mkdir -p /opt/harific

WORKDIR /opt/harific

COPY . ./

RUN go mod download && go mod verify
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-w -s -X 'main.version=$(git describe --tags --abbrev=0 2>/dev/null || echo dev)' -X 'main.date=$(date +%Y-%m-%dT%TZ)'" \
    -v -o harific harific.go

FROM debian:bookworm-slim

WORKDIR /work

COPY --from=builder /opt/harific/harific /usr/local/bin/harific

ENV PATH=$PATH:/usr/local/bin

ENTRYPOINT ["harific"]
