#!/usr/bin/env python3
"""Extract a versioned CHANGELOG.md section for GoReleaser release notes."""

import argparse
import re
from pathlib import Path


VERSION_HEADING_PATTERN = re.compile(
    r"^##\s+v?"
    r"(?P<version>[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?)"
    r"(?:\s+\([^)]*\))?\s*$"
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Extract the CHANGELOG.md section for a release version.",
    )
    parser.add_argument(
        "--changelog",
        default="CHANGELOG.md",
        type=Path,
        help="Path to CHANGELOG.md.",
    )
    parser.add_argument(
        "--version",
        required=True,
        help="Release version to extract, without a leading v.",
    )
    parser.add_argument(
        "--output",
        required=True,
        type=Path,
        help="Path where extracted release notes should be written.",
    )
    return parser.parse_args()


def extract_release_notes(changelog: Path, version: str) -> str:
    changelog_lines = changelog.read_text(encoding="utf-8").splitlines()

    start = None
    end = None
    for index, line in enumerate(changelog_lines):
        match = VERSION_HEADING_PATTERN.match(line)
        if start is None:
            if match and match.group("version") == version:
                start = index
            continue

        if line.startswith("## "):
            end = index
            break

    if start is None:
        raise SystemExit(f"{changelog} does not contain a release section for version {version}")

    if end is None:
        end = len(changelog_lines)

    release_notes = "\n".join(changelog_lines[start:end]).strip()
    if not release_notes:
        raise SystemExit(f"{changelog} release section for version {version} is empty")

    return release_notes


def main() -> None:
    args = parse_args()
    release_notes = extract_release_notes(args.changelog, args.version)
    args.output.write_text(release_notes + "\n", encoding="utf-8")
    print(f"Extracted {args.changelog} section for version {args.version} to {args.output}")


if __name__ == "__main__":
    main()
