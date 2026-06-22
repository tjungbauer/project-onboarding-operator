#!/usr/bin/env python3
"""Pin the operator container image in the published CSV to an immutable @sha256 digest."""

from __future__ import annotations

import argparse
import pathlib
import re
import sys


def normalize_digest(digest: str) -> str:
    digest = digest.strip()
    if digest.startswith("sha256:"):
        return digest
    return f"sha256:{digest}"


def pin_csv(csv_path: pathlib.Path, image_tag: str, digest: str) -> None:
    pinned = f"{image_tag.rsplit(':', 1)[0]}@{normalize_digest(digest)}"
    text = csv_path.read_text()

    text, count = re.subn(
        rf"^(\s+)image: {re.escape(image_tag)}\s*$",
        rf"\1image: {pinned}",
        text,
        flags=re.M,
    )
    if count < 1:
        sys.exit(f"error: no deployment image line found for {image_tag}")

    if re.search(r"^\s+containerImage:\s", text, flags=re.M):
        text, _ = re.subn(
            r"^(\s+containerImage:\s).*$",
            rf"\1{pinned}",
            text,
            count=1,
            flags=re.M,
        )
    else:
        text = text.replace(
            "  annotations:\n",
            f"  annotations:\n    containerImage: {pinned}\n",
            1,
        )

    related_block = (
        "  relatedImages:\n"
        f"  - image: {pinned}\n"
        "    name: manager\n"
    )
    if re.search(r"^  relatedImages:\s*$", text, flags=re.M):
        text, _ = re.subn(
            r"^  relatedImages:\n(?:  - .*\n)+",
            related_block,
            text,
            count=1,
            flags=re.M,
        )
    else:
        text, _ = re.subn(
            r"^  version: ",
            related_block + "  version: ",
            text,
            count=1,
            flags=re.M,
        )

    csv_path.write_text(text)
    print(f"==> Pinned CSV operator image to {pinned}")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--csv",
        default="bundle/manifests/project-onboarding-operator.clusterserviceversion.yaml",
    )
    parser.add_argument("--image", required=True, help="Tagged image ref (quay.io/...:vX.Y.Z)")
    parser.add_argument("--digest", required=True, help="sha256 digest from registry")
    args = parser.parse_args()

    csv_path = pathlib.Path(args.csv)
    if not csv_path.is_file():
        sys.exit(f"error: CSV not found: {csv_path}")

    pin_csv(csv_path, args.image, args.digest)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
