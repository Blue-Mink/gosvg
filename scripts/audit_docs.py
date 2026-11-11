#!/usr/bin/env python3
"""gosvg 文档深度审计（可选，不阻塞门禁）。

用法：python3 scripts/audit_docs.py [--strict]

输出四类「人眼容易漏」的清单，供发版前人工复核：
  A 相对时间词（当前/最新/尚未/仍是/…）——判断是否过期
  B 数字声明（N 个样例 / 覆盖率 / IoU）——对照实测
  C 版本号分布——检查是否有旧版本仍被当作当前
  D 仓库内路径引用悬空（含同目录简写判定）

与 check_docs.sh 的分工：
  - check_docs.sh 只查可机械判定的硬不变量，失败即挂；
  - 本脚本查的是「需要人判断」的软信号，默认只报告，永不因内容失败。
  - --strict 在有「疑似过期模式」时以退出码 2 提示（仍不阻断 verify.sh）。
"""
from __future__ import annotations

import argparse
import collections
import pathlib
import re

ROOT = pathlib.Path(__file__).resolve().parent.parent

TIME = re.compile(r"当前|最新|尚未|仍是|仍为|现为|现在|至今|暂不|暂未|未发版|待查项")
NUM = re.compile(
    r"\b(\d+)\s*个(?:固定|基线|可比|确定性)?样例|\b(\d+)\s*个(?:基线)?样本"
    r"|\b(\d+)\s*个测试函数|\b(\d+)\s*个\s*[Ff]uzz|\b(\d+)\s*个\s*低分"
    r"|覆盖率\s*[\d.]+\s*%|\b8[0-9]\.\d%|IoU<0\.5|平均\s*alpha\s*IoU"
)
VER = re.compile(r"v0\.\d+\.\d+")
REF = re.compile(r"`([A-Za-z0-9_./-]+\.(?:md|go|sh|yml|yaml|json|sha256))`")

# 已知的「过期但已就地加注」的模式；出现这些时提示人工确认注记仍在。
STALE_PATTERNS = {
    "「当前 v0.2.3 已发」": "当前 v0.2.3 已发",
    "「当前 86.7%」": "当前 86.7%",
    "「6 个固定样例」": "6 个固定样例",
    "「215 个测试函数」": "215 个测试函数",
    "「仍是待查项」": "仍是待查项",
    "「（= main）」": "（= main）",
}


def md_files() -> list[pathlib.Path]:
    seen = {p.resolve() for p in ROOT.glob("*.md")}
    for p in ROOT.rglob("*.md"):
        if ".git" not in p.parts:
            seen.add(p.resolve())
    return sorted(seen)


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--strict", action="store_true", help="发现疑似过期模式时以退出码 2 提示")
    args = ap.parse_args()

    files = md_files()
    tracked = {
        p.relative_to(ROOT).as_posix()
        for p in ROOT.rglob("*")
        if p.is_file() and ".git" not in p.parts
    }

    print(f"# gosvg 文档审计（{len(files)} 个 .md）\n")

    print("## A. 相对时间词（逐条判定是否过期）")
    n = 0
    for p in files:
        for i, line in enumerate(p.read_text(errors="ignore").splitlines(), 1):
            if TIME.search(line):
                n += 1
                print(f"{p.relative_to(ROOT).as_posix()}:{i}| {line.strip()[:160]}")
    print(f"-- 共 {n} 行\n")

    print("## B. 数字声明（逐条对照实测）")
    n = 0
    for p in files:
        for i, line in enumerate(p.read_text(errors="ignore").splitlines(), 1):
            if NUM.search(line):
                n += 1
                print(f"{p.relative_to(ROOT).as_posix()}:{i}| {line.strip()[:160]}")
    print(f"-- 共 {n} 行\n")

    print("## C. 版本号出现次数")
    counts: collections.Counter[str] = collections.Counter()
    for p in files:
        counts.update(m.group() for m in VER.finditer(p.read_text(errors="ignore")))
    print("  " + ", ".join(f"{k}={v}" for k, v in sorted(counts.items())) + "\n")

    print("## D. 仓库内路径引用悬空")
    miss: list[str] = []
    for p in files:
        for i, line in enumerate(p.read_text(errors="ignore").splitlines(), 1):
            for m in REF.finditer(line):
                r = m.group(1)
                base = r.rsplit("/", 1)[-1]
                if (r.startswith("github.com/") or "*" in r or r.startswith("/")
                        or base.startswith("_") or base.startswith(".")):
                    continue
                if r in tracked:
                    continue
                if "/" in r:
                    # 带目录前缀的相对路径必须相对仓库根存在（不设前缀白名单）。
                    miss.append(f"{p.relative_to(ROOT).as_posix()}:{i} -> {r}")
                elif "/" not in r and not any(t.endswith("/" + r) for t in tracked):
                    # 纯 basename 允许「同目录简写」（如两份 UPSTREAM.md）。
                    miss.append(f"{p.relative_to(ROOT).as_posix()}:{i} -> {r}")
    for x in miss:
        print("  " + x)
    print(f"-- 共 {len(miss)} 处\n")

    print("## E. 疑似过期模式（须确认已就地加注，而非裸断言）")
    hits = 0
    for label, needle in STALE_PATTERNS.items():
        locs = []
        for p in files:
            for i, line in enumerate(p.read_text(errors="ignore").splitlines(), 1):
                if needle in line:
                    locs.append(f"{p.relative_to(ROOT).as_posix()}:{i}")
        if locs:
            hits += len(locs)
            print(f"  {label}: {', '.join(locs)}")
    print(f"-- 共 {hits} 处命中（命中≠错误，逐条确认有时间限定/⚠️ 注记即可）\n")

    print("提示：历史快照按仓库纪律保留原文、就地加注，不要删改当时结论。")
    if args.strict and hits:
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
