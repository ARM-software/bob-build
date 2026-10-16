#!/usr/bin/env python3
"""Publish the current GitLab CI commit as a SHA-versioned Bazel release."""

import base64
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess


def prepare_release(sha, output_dir, project_url):
    version = f"sha-{sha}"
    prefix = f"bob-{sha}"
    manifest = subprocess.check_output(
        ["git", "show", f"{sha}:MODULE.bazel"], text=True
    )
    module = re.search(r"(?ms)^module\(\n(.*?)^\)", manifest)
    if module is None or re.search(r"\bversion\s*=", module[1]):
        raise ValueError("MODULE.bazel must declare module() without a version")
    offset = module.start(1)
    manifest = manifest[:offset] + f'    version = "{version}",\n' + manifest[offset:]

    output_dir.mkdir(parents=True)
    archive = output_dir / "src.tar.gz"
    subprocess.run(
        [
            "git",
            "archive",
            "--format=tar.gz",
            f"--prefix={prefix}/",
            f"--output={archive}",
            sha,
        ],
        check=True,
    )
    checksum = hashlib.sha256(archive.read_bytes()).digest()
    module_dir = output_dir / "modules" / "bob-build"
    version_dir = module_dir / version
    version_dir.mkdir(parents=True)
    (version_dir / "MODULE.bazel").write_text(manifest, encoding="utf-8")
    metadata = {
        version_dir
        / "source.json": {
            "url": f"{project_url}/-/releases/{version}/downloads/src.tar.gz",
            "integrity": "sha256-" + base64.b64encode(checksum).decode("ascii"),
            "strip_prefix": prefix,
        },
        module_dir / "metadata.json": {"versions": [version], "yanked_versions": {}},
    }
    for path, data in metadata.items():
        path.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")


def api(path, *args):
    return json.loads(
        subprocess.check_output(
            [
                "glab",
                "api",
                f"projects/{os.environ['CI_PROJECT_ID']}/{path}",
                "--hostname",
                os.environ["CI_SERVER_HOST"],
                *args,
            ],
            text=True,
        )
    )


def main():
    sha = os.environ["CI_COMMIT_SHA"]
    version = f"sha-{sha}"
    project_url = os.environ["CI_PROJECT_URL"].rstrip("/")
    output_dir = Path("release")

    # Stop before uploading anything. Incomplete publications need manual cleanup.
    if api("repository/tags", "--method", "GET", "--raw-field", f"search=^{version}$"):
        raise SystemExit(f"Tag {version} already exists")
    if api(
        "packages",
        "--method",
        "GET",
        "--raw-field",
        "package_type=generic",
        "--raw-field",
        "package_name=release",
        "--raw-field",
        f"package_version={version}",
    ):
        raise SystemExit(
            f"Package {version} already exists; clean up incomplete publication manually"
        )

    prepare_release(sha, output_dir, project_url)
    links = []
    for path in sorted(output_dir.rglob("*")):
        if not path.is_file():
            continue
        name = path.relative_to(output_dir).as_posix()
        package_path = f"packages/generic/release/{version}/{name}"
        api(
            package_path,
            "--method",
            "PUT",
            "--input",
            str(path),
            "--header",
            "Content-Type: application/octet-stream",
        )
        links.append(
            {
                "name": name,
                "url": f"{os.environ['CI_API_V4_URL']}/projects/{os.environ['CI_PROJECT_ID']}/{package_path}",
                "direct_asset_path": f"/{name}",
                "link_type": "package",
            }
        )
    subprocess.run(
        [
            "glab",
            "release",
            "create",
            version,
            "--repo",
            project_url,
            "--ref",
            sha,
            "--name",
            f"Bob {version}",
            "--notes",
            f"Bob Bazel module from commit `{sha}`.",
            "--assets-links",
            json.dumps(links),
            "--no-update",
        ],
        check=True,
    )


if __name__ == "__main__":
    main()
