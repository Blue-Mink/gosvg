#!/bin/sh
# 文档一致性门禁：把「可机械判定的不变量」固化，防止发版后文档状态倒挂。
# 用法：sh ./scripts/check_docs.sh   （由 verify.sh 自动调用，也可单跑）
# 设计原则：只查能机械判定的事实；历史快照、口径说明、结论演进一律不查。
# 深度审计（相对时间词 / 数字声明 / 版本号分布）见 scripts/audit_docs.py，不阻塞。
set -eu
cd "$(dirname "$0")/.."

FAIL=0
bad() { echo "  ✗ $*"; FAIL=1; }
ok()  { echo "  ✓ $*"; }

echo "== docs: golden 样例数 =="
N_SVG=$(find testdata -maxdepth 1 -name '*.svg' -type f | wc -l | tr -d ' ')
N_HASH=$(grep -c . testdata/golden.sha256 | tr -d ' ')
[ "$N_SVG" = "$N_HASH" ] \
  && ok "testdata/*.svg=$N_SVG 与 golden.sha256 行数=$N_HASH 一致" \
  || bad "testdata/*.svg=$N_SVG 但 golden.sha256=$N_HASH 行（增删样例须同步重生成基线）"

echo "== docs: 源码注释不得写死固定样例数 =="
# 数据真相是 testdata/*.svg 的实际数量；注释里写死「N 个固定样例」在增删样例后必然
# 过期。PR #24 的门禁只扫 .md，cmd/corpus/main.go 因此漏网。排除两个检查器自身：
# 它们含这些字面量是模式/文案，不是状态声明。
HARD=$(grep -rnE '[0-9]+ 个(固定|基线|确定性|golden)样例' \
       --include='*.go' --include='*.sh' --include='*.py' \
       --exclude='check_docs.sh' --exclude='audit_docs.py' \
       --exclude-dir=.git . 2>/dev/null || true)
if [ -n "$HARD" ]; then
  echo "$HARD" | sed 's/^/    /'
  bad "源码注释写死了固定样例数（改为引用 testdata/*.svg 自动枚举，勿写死）"
else
  ok "源码注释未写死固定样例数"
fi

echo "== docs: 质量基线锚点 =="
ANCHOR=$(grep -o 'test=[0-9]* example=[0-9]* fuzz=[0-9]* golden=[0-9]*' GOSVG_TODO.md | head -1)
if [ -z "$ANCHOR" ]; then
  bad "GOSVG_TODO.md 缺少机器校验锚点行（\`test=... example=... fuzz=... golden=...\`）"
else
  for kv in $ANCHOR; do
    key=${kv%%=*}; want=${kv#*=}
    case "$key" in
      test)    got=$(grep -rh '^func Test'     --include='*_test.go' . | wc -l | tr -d ' ') ;;
      example) got=$(grep -rh '^func Example'  --include='*_test.go' . | wc -l | tr -d ' ') ;;
      fuzz)    got=$(grep -rh '^func Fuzz'     --include='*_test.go' . | wc -l | tr -d ' ') ;;
      golden)  got="$N_SVG" ;;
      *)       got='' ;;
    esac
    if [ -z "$got" ]; then bad "锚点含未知键 $key"; elif [ "$got" = "$want" ]; then
      ok "$key=$want 与实测一致"
    else
      bad "锚点 $key=$want 但实测 $got（更新 GOSVG_TODO.md 锚点行，勿改口径凑数）"
    fi
  done
fi

echo "== docs: 版本状态一致性 =="
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || true)
if [ -z "$LATEST_TAG" ]; then
  echo "  · 无 git tag（浅克隆/CI archive），跳过版本一致性检查"
else
  CL_VER=$(sed -n 's/^## \[\(v[0-9.]*\)\].*/\1/p' CHANGELOG.md | head -1)
  IOU_VER=$(sed -n 's/^>.*最新 release 为 \*\*\(v[0-9.]*\)\*\*.*/\1/p' CAIROSVG_IOU.md | head -1)
  [ "$CL_VER" = "$LATEST_TAG" ] && ok "CHANGELOG 最新版本 $CL_VER = 最新 tag" \
    || bad "CHANGELOG 最新版本 ${CL_VER:-<缺>} ≠ 最新 tag $LATEST_TAG"
  [ "$IOU_VER" = "$LATEST_TAG" ] && ok "CAIROSVG_IOU 文首 $IOU_VER = 最新 tag" \
    || bad "CAIROSVG_IOU 文首 ${IOU_VER:-<缺>} ≠ 最新 tag $LATEST_TAG"

  echo "== docs: tag 之后有提交时 [Unreleased] 不得为空 =="
  N_AFTER=$(git rev-list --count "$LATEST_TAG"..HEAD 2>/dev/null || echo 0)
  UNREL=$(sed -n '/^## \[Unreleased\]/,/^## \[v/p' CHANGELOG.md | grep -v '^## \[' | grep -c . || true)
  if [ "$N_AFTER" -gt 0 ]; then
    grep -q '（暂无未发布改动）' CHANGELOG.md \
      && bad "tag $LATEST_TAG 之后有 $N_AFTER 个提交，但 CHANGELOG 仍写「（暂无未发布改动）」" \
      || { [ "$UNREL" -gt 0 ] && ok "tag 后 $N_AFTER 个提交，[Unreleased] 有 $UNREL 行记录" \
           || bad "tag 后 $N_AFTER 个提交但 [Unreleased] 段为空"; }
  else
    ok "HEAD 即 tag $LATEST_TAG，[Unreleased] 可为空"
  fi

  echo "== docs: 版本状态块不得写死 main commit =="
  # CAIROSVG_IOU.md / CHANGELOG.md 的文首版本状态是「当前状态」声明；main 会持续
  # 前移，写死 hash 在下次合并后必然过期。只查文件顶部，不误伤正文历史快照。
  HEAD_HARD=$( { head -12 CAIROSVG_IOU.md; head -25 CHANGELOG.md; } \
               | grep -nE 'main\s*=\s*`[0-9a-f]{7,40}`' || true)
  if [ -n "$HEAD_HARD" ]; then
    echo "$HEAD_HARD" | sed 's/^/    /'
    bad "版本状态块写死了 main commit（应改为「以 git rev-parse HEAD 为准」）"
  else
    ok "版本状态块未写死 main commit"
  fi

  echo "== docs: 禁止裸「= main」断言 =="
  # tag 与 HEAD 分离后，「tag → xxx（= main…）」必然过期；只允许带「发布当时」限定。
  # 用宽松前缀匹配，避免「（= main；」这类变体绕过检查。
  BARE=$(grep -n '（= main' CHANGELOG.md CAIROSVG_IOU.md README.md 2>/dev/null \
         | grep -v '发布当时' || true)
  if [ -n "$BARE" ]; then
    echo "$BARE" | sed 's/^/    /'
    bad "存在无「发布当时」限定的「（= main」断言"
  else
    ok "无裸「= main」断言"
  fi
fi

echo "== docs: 仓库内路径引用悬空 =="
# 精确检查：带目录前缀的路径必须相对仓库根存在；纯 basename 允许同目录简写。
# 纯 sh 实现，不依赖 python3 —— verify.sh 是 Go/sh 门禁，CI 不保证有 Python。
MISS=''
REFS=$(grep -rhoE '`[A-Za-z0-9_./-]+\.(md|go|sh|yml|yaml|json|sha256)`' \
       --include='*.md' --exclude-dir=.git . 2>/dev/null | tr -d '`' | sort -u)
for r in $REFS; do
  case "$r" in
    github.com/*|/*|*\**) continue ;;
  esac
  base=${r##*/}
  case "$base" in
    _*|.*) continue ;;
  esac
  case "$r" in
    */*)
      if [ ! -f "$r" ]; then
        LOC=$(grep -rn -F "`$r`" --include='*.md' --exclude-dir=.git . 2>/dev/null | head -1 | cut -d: -f1,2)
        MISS="${MISS}${LOC} -> ${r}
"
      fi
      ;;
    *)
      # 同目录简写（如两份 UPSTREAM.md）：任意路径以该 basename 结尾即算有效。
      if ! find . -type f -name "$base" -not -path './.git/*' 2>/dev/null | grep -q .; then
        LOC=$(grep -rn -F "`$r`" --include='*.md' --exclude-dir=.git . 2>/dev/null | head -1 | cut -d: -f1,2)
        MISS="${MISS}${LOC} -> ${r}
"
      fi
      ;;
  esac
done
if [ -n "$MISS" ]; then
  printf '%s' "$MISS" | sed 's/^/    /'
  bad "存在悬空路径引用（见上）"
else
  ok "仓库内路径引用全部可解析"
fi

echo "== docs: 关键文件存在 =="
for f in CHANGELOG.md CAIROSVG_IOU.md GOSVG_TODO.md GOSVG_GAP_TODO.md \
         CORPUS_BASELINE.md BENCHMARK.md CONTRIBUTING.md SECURITY.md \
         OPEN_SOURCE_CHECKLIST.md README.md NOTICE scripts/verify.sh \
         scripts/check_docs.sh scripts/audit_docs.py; do
  [ -f "$f" ] || bad "缺少 $f"
done

echo "== docs: 双 CI workflow 逐字一致 =="
if cmp -s .gitea/workflows/ci.yml .github/workflows/ci.yml; then
  ok "两份 workflow 内容一致"
else
  bad "两份 workflow 已漂移（改一处须同步改另一处）"
fi

echo "== docs: Go 版本单一来源 =="
GO_MOD_VER=$(awk '$1=="go" {print $2; exit}' go.mod)
VER_BAD=0
for wf in .gitea/workflows/ci.yml .github/workflows/ci.yml; do
  WF_VER=$(grep -oE "go-version: *'?[0-9]+\.[0-9]+(\.[0-9]+)?'?" "$wf" \
           | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1)
  if [ "$WF_VER" != "$GO_MOD_VER" ]; then
    echo "    $wf: go-version=$WF_VER ≠ go.mod go $GO_MOD_VER"
    VER_BAD=1
  fi
done
[ "$VER_BAD" -eq 0 ] \
  && ok "两份 workflow 的 go-version 与 go.mod 一致（$GO_MOD_VER）" \
  || bad "Go 版本声明漂移（go.mod 是单一来源，勿在多处硬编码不同值）"

[ "$FAIL" -eq 0 ] && ok "关键文件齐全"

if [ "$FAIL" -ne 0 ]; then
  echo "DOCS CHECK FAILED"; exit 1
fi
echo "DOCS CHECK PASSED"
