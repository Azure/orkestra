FROM golang:1.15 as builder

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY main.go main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -o executor main.go

FROM alpine:3.13
COPY --from=builder /workspace/executor .

ENTRYPOINT [ "./executor" ]