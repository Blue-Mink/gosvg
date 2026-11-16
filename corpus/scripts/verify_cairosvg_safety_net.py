#!/usr/bin/env python3
"""验证「cairosvg 对账安全网」在本机是否完整可用（只读，不改任何资产）。

背景：光栅器核心优化（Rasterizer 复用 / even-odd 根治 / 离位图池化）必须靠
golden 回归 + cairosvg 逐像素对账双兜底。历史上曾误判「cairosvg 参照图丢失」，
实际资产一直散落在 /tmp 各处。本脚本把它们串起来，一键自检 + 秒级复算，
避免再靠记忆判断。

它检查四要素：
  1. 850 SVG 语料目录
  2. cairosvg 参照图目录（ref/ 与 ref_nofont_backup/）
  3. cairosvg Python venv（含 cairosvg + PIL）
  4. gosvg 渲染结果目录（got/）

若 --recheck，则用现成的 got/ 与 ref/ 复算 IoU（不重渲染 gosvg，秒级），
并可与历史 JSON 的 average_iou 比对，验证端到端口径一致。

用法：
  python3 corpus/scripts/verify_cairosvg_safety_net.py \
      --corpus /tmp/corpus_recheck_850 \
      [--venv /tmp/iou_venv] \
      [--cairo-lib /tmp/cairo_root/usr/lib/x86_64-linux-gnu] \
      [--recheck] [--expect-json <历史JSON>]
"""
import argparse
import glob
import os
import subprocess
import sys
from pathlib import Path


def ok(msg):
    print("  [OK]   " + msg)


def miss(msg):
    print("  [MISS] " + msg)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--corpus", required=True, help="语料根目录（含 svg/ ref/ got/）")
    ap.add_argument("--venv", default="/tmp/iou_venv", help="含 cairosvg 的 Python venv")
    ap.add_argument("--cairo-lib", default="/tmp/cairo_root/usr/lib/x86_64-linux-gnu",
                    help="libcairo.so.2 所在目录（设 LD_LIBRARY_PATH）")
    ap.add_argument("--recheck", action="store_true", help="用 got/+ref/ 复算 IoU")
    ap.add_argument("--expect-json", default=None, help="历史 IoU JSON，比对 average_iou")
    args = ap.parse_args()

    root = Path(args.corpus)
    all_ok = True

    print("== cairosvg 对账安全网自检 ==")
    # 1) 语料
    svg_dir = root / "svg"
    svgs = sorted(glob.glob(str(svg_dir / "*.svg")))
    if svgs:
        ok(f"SVG 语料: {len(svgs)} 个 ({svg_dir})")
    else:
        miss(f"SVG 语料为空: {svg_dir}")
        all_ok = False

    # 2) 参照图
    for name in ("ref", "ref_nofont_backup"):
        refs = sorted(glob.glob(str(root / name / "*.png")))
        if refs:
            ok(f"参照图 {name}/: {len(refs)} 张")
        else:
            miss(f"参照图缺失: {root / name}")

    # 3) cairosvg venv + cairo 库
    py = Path(args.venv) / "bin" / "python"
    env = dict(os.environ)
    env["LD_LIBRARY_PATH"] = args.cairo_lib
    if py.exists():
        try:
            out = subprocess.run(
                [str(py), "-c",
                 "import cairosvg,PIL;print(cairosvg.__version__,PIL.__version__)"],
                env=env, capture_output=True, text=True, timeout=30,
            )
            if out.returncode == 0:
                ok(f"cairosvg venv: {out.stdout.strip()} ({py})")
            else:
                miss(f"cairosvg 导入失败: {out.stderr.strip()[:200]}")
                all_ok = False
        except Exception as exc:  # noqa: BLE001
            miss(f"cairosvg 探测异常: {exc}")
            all_ok = False
    else:
        miss(f"venv 不存在: {py}")
        all_ok = False

    cairo_so = Path(args.cairo_lib) / "libcairo.so.2"
    if cairo_so.exists():
        ok(f"libcairo.so.2: {cairo_so}")
    else:
        miss(f"libcairo.so.2 缺失: {args.cairo_lib}")
        all_ok = False

    # 4) gosvg 渲染结果
    got_dir = root / "got"
    gots = sorted(glob.glob(str(got_dir / "*.png")))
    if gots:
        ok(f"gosvg 渲染结果 got/: {len(gots)} 张")
    else:
        miss(f"gosvg 渲染结果为空: {got_dir}")

    # 5) 可选复算
    if args.recheck and gots:
        print("== 复算 IoU（got/ vs ref/，纯 PIL，秒级）==")
        script = r'''
import sys
from PIL import Image, ImageChops
from pathlib import Path
got=Path(sys.argv[1]); ref=Path(sys.argv[2])
def m(p,thr): return Image.open(p).convert("RGBA").getchannel("A").point(lambda v:255 if v>thr else 0)
def c(img): return img.histogram()[255]
for thr in (0,8):
    vals=[]
    for rp in sorted(ref.glob("*.png")):
        gp=got/rp.name
        if not gp.exists(): continue
        a,b=m(gp,thr),m(rp,thr)
        inter=c(ImageChops.darker(a,b)); union=c(ImageChops.lighter(a,b))
        vals.append(inter/union if union else 1.0)
    avg=sum(vals)/len(vals) if vals else 0
    below=sum(1 for v in vals if v<0.5)
    print(f"{thr}\t{len(vals)}\t{avg:.6f}\t{below}")
'''
        try:
            out = subprocess.run(
                [str(py), "-c", script, str(got_dir), str(root / "ref")],
                env=env, capture_output=True, text=True, timeout=180,
            )
            if out.returncode == 0:
                for line in out.stdout.strip().splitlines():
                    thr, n, avg, below = line.split("\t")
                    print(f"  alpha>{thr}: compared={n} avg_iou={avg} below0.5={below}")
                if args.expect_json:
                    import json
                    exp = json.load(open(args.expect_json))
                    print(f"  历史 JSON: avg_iou={exp.get('average_iou'):.6f} "
                          f"(供人工比对上面 alpha>0 行)")
            else:
                miss(f"复算失败: {out.stderr.strip()[:200]}")
                all_ok = False
        except Exception as exc:  # noqa: BLE001
            miss(f"复算异常: {exc}")
            all_ok = False

    print("== 结论: " + ("安全网完整可用" if all_ok else "安全网不完整，见 [MISS]") + " ==")
    return 0 if all_ok else 1


if __name__ == "__main__":
    sys.exit(main())
