# Build the manager binary
FROM golang:1.16 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY pkg/ pkg/
COPY controllers/ controllers/
# COPY config.yaml config.yaml

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -o manager main.go

FROM alpine:3.7
RUN apk add --no-cache bash
RUN mkdir -p /etc/orkestra/charts/pull/

WORKDIR /
COPY --from=builder /workspace/manager .
# COPY --from=builder /workspace/config.yaml /etc/controller/config.yaml

ENTRYPOINT ["/manager"]
