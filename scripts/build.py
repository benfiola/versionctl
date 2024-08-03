#!/usr/bin/env python
import argparse
import pathlib
import subprocess
import sys


def main(file: pathlib.Path):
    subprocess.run(
        [
            "python",
            # these python flags are passed through to the binary nuitka creates
            "-u",
            "-X",
            "utf8",
            "-m",
            "nuitka",
            # embed distribution metadata (i.e., version) in binary
            "--include-distribution-metadata=versionctl",
            # embed non-python files in binary
            "--include-package-data=versionctl",
            "--onefile",
            # disable compression (slower startup, but larger file)
            "--onefile-no-compression",
            f"--output-dir={file.parent}",
            f"--output-filename={file.name}",
            "--standalone",
            # python must be statically linked (otherwise, a standalone binary makes less sense)
            "--static-libpython=yes",
            "versionctl/cli.py",
        ]
    )


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", default="build/versionctl")
    data = vars(parser.parse_args())
    file = pathlib.Path(data["output"])
    main(file)
