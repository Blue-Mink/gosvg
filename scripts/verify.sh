#!/bin/sh
# gosvg 全套验证：CI 与本地手动同源。
# 用法：sh ./scripts/verify.sh
# 内网若 proxy.golang.org 不通，可设 GOPROXY=https://goproxy.cn,direct
set -eu
cd "$(dirname "$0")/.."
if [ -z "${GOPROXY:-}" ]; then
  export GOPROXY=https://goproxy.cn,direct
fi

echo "== gofmt =="
FMT=$(gofmt -l .)
if [ -n "$FMT" ]; then echo "$FMT"; exit 1; fi

echo "== go vet ==";        go vet ./...
echo "== go test ==";       go test ./...
echo "== go test -race =="; go test -race ./...
echo "== arm64 build ==";   GOARCH=arm64 GOOS=linux CGO_ENABLED=0 go build ./...
echo "== fuzz 30s ==";      go test -run '^$' -fuzz '^FuzzParse$' -fuzztime 30s .
echo "ALL GREEN"
