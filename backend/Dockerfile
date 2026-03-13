FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN mkdir -p /out \
  && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/server ./cmd/server \
  && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/indexer ./cmd/indexer \
  && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/hedger ./cmd/hedger \
  && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/liquidator ./cmd/liquidator \
  && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/matcher ./cmd/matcher \
  && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/fundingd ./cmd/fundingd

FROM alpine:3.21

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata wget python3 py3-pip \
  && python3 -m pip install --break-system-packages --no-cache-dir hyperliquid-python-sdk eth-account

COPY --from=builder /out /app/bin
COPY scripts /app/scripts

ENV TZ=Asia/Shanghai
