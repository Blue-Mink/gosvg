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
# -race 强制要求 cgo + 一个可用的 C 编译器（$(go env CC)，默认 gcc）。
# 纯运行时的「零 cgo」不等于「-race 不需要 cgo」：在没有 gcc 的最小容器 /
# 精简 NAS 上，旧脚本会因 set -eu 在 race 这行直接退出，后面 arm64/wasm/
# fuzz/docs 全部不跑，让人误以为「脚本挂了=代码有问题」。
# 这里只在「确实没有 C 编译器」时跳过 race 并继续跑完其余门禁；有编译器时
# 行为完全不变（照跑、失败照 fail），不放宽任何检查，且结尾如实标注 SKIPPED。
RACE_SKIPPED=0
if ! command -v "$(go env CC)" >/dev/null 2>&1; then
  echo "== go test -race ==       SKIPPED: $(go env CC) 不可用（本机无 C 编译器）；补跑需在带 gcc 的机器执行本脚本"
  RACE_SKIPPED=1
else
  echo "== go test -race ==";     go test -count=1 -race ./...
fi
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
echo "== docs consistency =="; sh ./scripts/check_docs.sh

if [ "$RACE_SKIPPED" = "1" ]; then
  echo "ALL GREEN (race SKIPPED: no C compiler; 其余门禁全部实跑通过)"
else
  echo "ALL GREEN"
fi
