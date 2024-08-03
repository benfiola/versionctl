import io
import logging
import pathlib
import select
import shlex
import subprocess
from typing import IO

logger = logging.getLogger(__name__)


def reader(popen: subprocess.Popen, outs: tuple[IO[str], IO[str]]):
    """
    Returns a read method that reads from the given popen into the provided 'outs' buffers.

    outs[0] is stdout
    outs[1] is stderr
    """

    def read():
        readables, _, _ = select.select([popen.stdout, popen.stderr], [], [], 0.01)
        while readables:
            readable = readables.pop()

            prefix = "out: "
            out = outs[0]
            if readable == popen.stderr:
                prefix = "err: "
                out = outs[1]
            for line in readable:
                out.writelines(line)
                if line.endswith("\n"):
                    line = line[:-1]   
                logger.debug(prefix + line)

    return read


def run_command(
    cmd: list[str], cwd: pathlib.Path | None = None, env: dict[str, str] | None = None
) -> str:
    """
    Runs a command and returns its stdout.

    If the commit fails, raises a subprocess.CalledProcessError
    """
    kwargs = {
        "cwd": cwd,
        "encoding": "utf-8",
        "env": env,
        "stderr": subprocess.PIPE,
        "stdout": subprocess.PIPE,
    }

    stdout = io.StringIO()
    stderr = io.StringIO()
    logger.debug(f"cmd: {shlex.join(cmd)}")
    popen = subprocess.Popen(cmd, **kwargs)
    read = reader(popen, (stdout, stderr))

    while popen.returncode is None:
        popen.poll()
        read()
    read()

    stdout.seek(0)
    stderr.seek(0)
    stdout = stdout.read()

    if popen.returncode != 0:
        stderr = stderr.read()
        raise subprocess.CalledProcessError(
            cmd=cmd,
            output=stdout,
            returncode=popen.returncode,
            stderr=stderr,
        )

    return stdout
