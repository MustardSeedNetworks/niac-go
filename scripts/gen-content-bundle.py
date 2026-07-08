#!/usr/bin/env python3
"""Generate a reproducible NIAC content bundle from the sanitized walk corpus.

The bundle format is locked by internal/content/extract.go: a gzip-compressed
tar mirroring the library layout (top-level walks/, networks/, pcaps/ dirs)
extracted straight into the library root. This script only ever populates
walks/ — there is no separate manifest, the tar layout IS the manifest.

Source corpus layout: <corpus>/<vendor>/<device>[-rNN].walk. Devices are
re-captured at multiple counter-advanced snapshots (the "-rNN" suffix); this
script dedupes down to one file per device, keeping the smallest snapshot
(smallest = least noise from counter/uptime drift) and dropping the "-rNN"
suffix from the tar entry name.

Two modes:
  full        Every deduped device, staged as walks/<device>.walk.
  essentials  The smallest walk per top-level vendor directory, sized for
              embedding in the daemon binary (L1 subset).

Usage:
    python3 scripts/gen-content-bundle.py --corpus <dir> --out <file.tar.gz> --mode full
    python3 scripts/gen-content-bundle.py --corpus <dir> --out <file.tar.gz> --mode essentials
"""

from __future__ import annotations

import argparse
import gzip
import re
import sys
import tarfile
from dataclasses import dataclass
from pathlib import Path

# Fixed mtime for every tar entry so byte-identical inputs always produce a
# byte-identical bundle (no timestamp drift across runs/machines).
FIXED_MTIME = 0

REVISION_SUFFIX = re.compile(r"-r\d+$")


@dataclass(frozen=True)
class WalkFile:
    path: Path
    vendor: str
    device: str
    size: int


def device_name(stem: str) -> str:
    """Strip a trailing "-rNN" revision suffix off a walk filename stem."""
    return REVISION_SUFFIX.sub("", stem)


def discover(corpus: Path) -> list[WalkFile]:
    walks = []
    for path in corpus.rglob("*.walk"):
        vendor = path.relative_to(corpus).parts[0]
        walks.append(
            WalkFile(
                path=path,
                vendor=vendor,
                device=device_name(path.stem),
                size=path.stat().st_size,
            )
        )
    return walks


def dedupe_by_device(walks: list[WalkFile]) -> list[WalkFile]:
    """Keep the smallest-size file per (vendor, device) group."""
    best: dict[tuple[str, str], WalkFile] = {}
    for walk in walks:
        key = (walk.vendor, walk.device)
        current = best.get(key)
        if current is None or walk.size < current.size:
            best[key] = walk
    return sorted(best.values(), key=lambda w: (w.vendor, w.device))


def smallest_per_vendor(walks: list[WalkFile]) -> list[WalkFile]:
    """Keep the single smallest deduped walk per top-level vendor dir."""
    best: dict[str, WalkFile] = {}
    for walk in walks:
        current = best.get(walk.vendor)
        if current is None or walk.size < current.size:
            best[walk.vendor] = walk
    return sorted(best.values(), key=lambda w: (w.vendor, w.device))


def write_bundle(walks: list[WalkFile], out: Path) -> None:
    out.parent.mkdir(parents=True, exist_ok=True)
    # gzip.GzipFile defaults its header mtime field to the current time,
    # which would make the compressed bytes differ run-to-run even though
    # the uncompressed tar is deterministic. Pin it to FIXED_MTIME too, and
    # drive the outer file handle ourselves so both layers are fixed.
    with (
        out.open("wb") as raw,
        gzip.GzipFile(filename="", mode="wb", fileobj=raw, mtime=FIXED_MTIME) as gz,
        tarfile.open(fileobj=gz, mode="w:") as tar,
    ):
        for walk in walks:
            info = tar.gettarinfo(str(walk.path), arcname=f"walks/{walk.device}.walk")
            info.mtime = FIXED_MTIME
            info.uid = 0
            info.gid = 0
            info.uname = ""
            info.gname = ""
            info.mode = 0o644
            with walk.path.open("rb") as fh:
                tar.addfile(info, fh)


def format_size(num_bytes: int) -> str:
    size = float(num_bytes)
    for unit in ("B", "KB", "MB", "GB"):
        if size < 1024:
            return f"{size:.1f}{unit}"
        size /= 1024
    return f"{size:.1f}TB"


def main() -> int:
    parser = argparse.ArgumentParser(
        description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter
    )
    parser.add_argument(
        "--corpus",
        required=True,
        type=Path,
        help="Sanitized walk corpus dir (vendor/device[-rNN].walk)",
    )
    parser.add_argument(
        "--out", required=True, type=Path, help="Output bundle path (.tar.gz)"
    )
    parser.add_argument(
        "--mode",
        required=True,
        choices=["full", "essentials"],
        help="full = all deduped devices; essentials = smallest walk per vendor",
    )
    args = parser.parse_args()

    if not args.corpus.is_dir():
        parser.error(f"corpus dir not found: {args.corpus}")

    all_walks = discover(args.corpus)
    if not all_walks:
        parser.error(f"no *.walk files found under {args.corpus}")

    deduped = dedupe_by_device(all_walks)
    selected = deduped if args.mode == "full" else smallest_per_vendor(deduped)

    write_bundle(selected, args.out)

    uncompressed = sum(w.size for w in selected)
    compressed = args.out.stat().st_size
    vendors = sorted({w.vendor for w in selected})

    print(f"mode:              {args.mode}")
    print(f"source walk files:  {len(all_walks)}")
    print(f"deduped devices:    {len(deduped)}")
    print(f"bundled devices:    {len(selected)}")
    print(f"vendors:            {len(vendors)}")
    print(f"uncompressed size:  {format_size(uncompressed)} ({uncompressed} bytes)")
    print(f"compressed size:    {format_size(compressed)} ({compressed} bytes)")
    print(f"output:             {args.out}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
