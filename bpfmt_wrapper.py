#!/usr/bin/env python3

from __future__ import annotations

import os
from pathlib import Path
from shlex import join
from sys import argv

from argparse import (
    SUPPRESS,
    REMAINDER,
    ArgumentParser,
    RawTextHelpFormatter,
)
from subprocess import CalledProcessError, run
from python.runfiles import Runfiles  # type: ignore


class RunfileNotFoundError(FileNotFoundError):
    pass


def parser() -> ArgumentParser:
    p = ArgumentParser(
        formatter_class=RawTextHelpFormatter,
        argument_default=SUPPRESS,
        description="bpftm wrapper",
        fromfile_prefix_chars="@",
    )

    p.add_argument(
        "--bpfmt",
        required=True,
        metavar="BPFMT",
        help="bpfmt binary path",
    )

    return p


def runfile(path: str) -> Path:
    runfiles = Runfiles.Create()
    resolved = Path(runfiles.Rlocation(path))
    if not resolved.exists():
        raise RunfileNotFoundError(path)
    return resolved


def exec(exec: Path, *args: str) -> int:
    cmd = (
        exec,
        *args,
    )
    try:
        result = run(
            cmd,
            capture_output=True,
            encoding="utf8",
            cwd=os.environ.get("BUILD_WORKSPACE_DIRECTORY", "."),
        )

        if result.returncode != 0:
            print(result.stderr.strip())
        else:
            print(result.stdout.strip())

        return result.returncode

    except CalledProcessError as e:
        print(
            "Subprocess command failed:\n%s\n\nstdout:\n%s\n\nstderr:\n%s",
            join(e.cmd),
            e.stdout,
            e.stderr,
        )
        return e.returncode


def main(*args: str) -> None:
    p = parser()

    print("args: ", args)

    parsed, rest = p.parse_known_args()

    bpfmt = runfile(parsed.bpfmt)

    print("parsed: ", parsed)
    print("rest: ", rest)
    print("bpfmt: ", bpfmt)

    return exec(bpfmt, *rest)


if __name__ == "__main__":
    exit(main(*argv[1:]))
