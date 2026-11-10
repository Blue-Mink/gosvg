#!/bin/sh
# gosvg 全套验证：CI 与本地手动同源。
# 用法：sh ./scripts/verify.sh
# 内网若 proxy.golang.org 不通，可设 GOPROXY=https://goproxy.cn,direct
set -eu
cd "$(dirname "$0")/.."
if [ -z "${GOPROXY:-}" ]; then
  export GOPROXY=https://goproxy.cn,direct
fi

echo "== go mod verify ==";    go mod verify
echo "== go mod tidy -diff =="; go mod tidy -diff
echo "== gofmt =="
FMT=$(gofmt -l .)
if [ -n "$FMT" ]; then echo "$FMT"; exit 1; fi

echo "== go vet ==";            go vet ./...
echo "== go test (cover) ==";   go test -count=1 -cover ./...
echo "== go test -race ==";     go test -count=1 -race ./...
echo "== arm64 build ==";       GOARCH=arm64 GOOS=linux CGO_ENABLED=0 go build ./...
echo "== wasm build ==";        GOOS=js GOARCH=wasm CGO_ENABLED=0 go build ./...
echo "== bench smoke ==";       go test -run '^$' -bench . -benchtime=1x ./...
echo "== fuzz FuzzParse 30s =="; go test -run '^$' -fuzz '^FuzzParse$'  -fuzztime 30s .
echo "== fuzz FuzzRender 20s ==";go test -run '^$' -fuzz '^FuzzRender$' -fuzztime 20s .
echo "== fuzz 其余目标各 5s =="
for f in $(go test -list '^Fuzz' . | grep '^Fuzz' | grep -vE '^FuzzParse$|^FuzzRender$'); do
  echo "  -> $f"
  go test -run '^$' -fuzz "^${f}$" -fuzztime 5s .
done
echo "ALL GREEN"
