#!/usr/bin/env python3


"""JSON Config generator

This script allows the user to generate JSON configuration based on Mconfig file
to be used by Gazelle plugin.

Script reads the input from the `stdin` with the JSON format:

`{"root_path": ".","rel_package_path": "subdir","file_name": "Mconfig","ignore_source":false}`

where:

- `root_path` - the root directory of the repository
- `rel_package_path` - the relative path to `root_path` where to start parse
  (mostly where the root Mconfig file resides)
- `file_name` (optional) - the name of the Mconfig file to read (`Mconfig` by default)
- `ignore_source` (optional) - ignore all `source` and `source_local` directives
  from the parsed Mconfig file.

In case `root_path` and `rel_package_path` are the pointing the same directory,
it means we want to start reading configs form the `root_path` Mconfig.

Multiline JSON input can be provided to parse every input line separately.

"""

import json
import sys

from pathlib import Path

from config_system import lex_wrapper, syntax


class SetEncoder(json.JSONEncoder):
    def default(self, obj):
        if isinstance(obj, set):
            return list(obj)
        if isinstance(obj, Path):
            return str(obj)
        return json.JSONEncoder.default(self, obj)


def main(stdin, stdout) -> int:
    if not sys.stdin.isatty():
        for request in stdin:
            r = json.loads(request.strip())
            ignore_source = r.get("ignore_source", False)
            root_path = Path(r["root_path"])
            rel_path = Path(r["rel_package_path"])
            file_name = r.get("file_name", "Mconfig")

            if rel_path.is_absolute():
                raise ValueError(f"Absolute path is not allowed! '{rel_path}'")

            # root Mconfig file to parse
            root_file = root_path / rel_path / file_name

            # ignore_missing = True
            lexer = lex_wrapper.LexWrapper(True)
            lexer.source(root_file)

            syntax.parser.ignore_source = ignore_source
            cfg = syntax.parser.parse(None, debug=False, lexer=lexer)["config"]

            for k, v in cfg.items():
                cfg[k]["relPath"] = rel_path / v["relPath"]

            print(
                f"{json.dumps(cfg, indent=4, cls=SetEncoder)}",
                end="",
                file=stdout,
                flush=True,
            )

            # write delimiter
            stdout.buffer.write(bytes([0]))
            stdout.flush()

    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.stdin, sys.stdout))
