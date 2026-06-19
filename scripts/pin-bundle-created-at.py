#!/usr/bin/env python3
"""Pin CSV createdAt so bundle regeneration produces a reproducible git diff."""

from __future__ import annotations

import pathlib
import re
import sys

CSV = pathlib.Path("bundle/manifests/project-onboarding-operator.clusterserviceversion.yaml")
PIN = pathlib.Path("bundle/metadata/created-at")


def main() -> int:
    if not CSV.is_file():
        print(f"skip pin createdAt: {CSV} not found", file=sys.stderr)
        return 0

    text = CSV.read_text()
    match = re.search(r'^    createdAt: "([^"]+)"', text, flags=re.M)
    if not match:
        print("skip pin createdAt: field not found in CSV", file=sys.stderr)
        return 0

    if not PIN.is_file():
        print(f"error: missing pin file {PIN} (commit bundle/metadata/created-at)", file=sys.stderr)
        return 1

    created_at = PIN.read_text().strip()
    if not created_at:
        print(f"error: empty pin file {PIN}", file=sys.stderr)
        return 1

    pinned = f'    createdAt: "{created_at}"'
    text, count = re.subn(r'^    createdAt: "[^"]+"', pinned, text, count=1, flags=re.M)
    if count != 1:
        print("failed to pin createdAt in CSV", file=sys.stderr)
        return 1

    CSV.write_text(text)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
