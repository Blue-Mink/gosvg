#!/usr/bin/env python3
"""用 cairosvg 把 SVG 语料渲染成参照 PNG（IoU 对账的参照侧）。

口径与 CAIROSVG_IOU.md 一致：256x256、透明背景、bytestring 渲染、
stem 同名配对 foo.svg -> foo.png。

原始语料不在仓库内（见 corpus/README.md）；本脚本只负责在持有语料的机器上
重新生成参照图，供 cmd/iou 对账。

用法：
    python3 corpus/scripts/render_ref.py --svg <SVG目录> --ref <输出目录> \
        [--size 256] [--fontconfig <fonts.conf>] [--cairo-lib <dir>]
"""
import argparse
import glob
import os
import sys
import time


def main():
    ap = argparse.ArgumentParser(description="render SVG corpus to reference PNG via cairosvg")
    ap.add_argument("--svg", required=True, help="SVG 语料目录")
    ap.add_argument("--ref", required=True, help="参照 PNG 输出目录")
    ap.add_argument("--size", type=int, default=256, help="渲染边长（默认 256）")
    ap.add_argument("--fontconfig", default=None,
                    help="可选 fonts.conf；设 FONTCONFIG_FILE 以复现带字体口径")
    ap.add_argument("--cairo-lib", default=None,
                    help="可选 cairo 动态库目录；设 LD_LIBRARY_PATH")
    args = ap.parse_args()

    # 环境变量须在 import cairosvg 前设置
    if args.fontconfig:
        os.environ["FONTCONFIG_FILE"] = args.fontconfig
    if args.cairo_lib:
        os.environ["LD_LIBRARY_PATH"] = args.cairo_lib

    import cairosvg  # noqa: E402

    os.makedirs(args.ref, exist_ok=True)
    files = sorted(glob.glob(os.path.join(args.svg, "*.svg")))
    t0 = time.time()
    ok, fail = 0, []
    for path in files:
        name = os.path.basename(path)[:-4]
        try:
            with open(path, "rb") as fh:
                data = fh.read()
            cairosvg.svg2png(
                bytestring=data,
                write_to=os.path.join(args.ref, name + ".png"),
                output_width=args.size,
                output_height=args.size,
            )
            ok += 1
        except Exception as exc:  # noqa: BLE001 - 逐个记录失败，不中断整批
            fail.append((name, type(exc).__name__ + ":" + str(exc)[:100]))
        if (ok + len(fail)) % 200 == 0:
            print(f"  rendered {ok + len(fail)}/{len(files)} ok={ok} fail={len(fail)}",
                  flush=True)
    print(f"DONE total={len(files)} ok={ok} fail={len(fail)} "
          f"size={args.size} time={time.time() - t0:.1f}s")
    for name, err in fail[:20]:
        print("  FAIL", name, err)
    return 0 if not fail else 1


if __name__ == "__main__":
    sys.exit(main())
