import json
import pathlib
import sys
from typing import Any

import click
import pydantic

from versionctl.core import (
    Analyzer,
    Configuration,
    Version,
    VersionFormat,
    get_tool_version,
    set_version,
)
from versionctl.data import data_folder
from versionctl.logs import LogLevel, configure_log_level


def main():
    try:
        grp_main()
    except Exception as e:
        click.echo(str(e), err=True)
        sys.exit(1)


def log_level_type(value: Any) -> LogLevel:
    return pydantic.TypeAdapter(LogLevel).validate_python(value)


def version_format_type(value: Any) -> VersionFormat:
    return pydantic.TypeAdapter(VersionFormat).validate_python(value)


pass_configuration = click.make_pass_decorator(Configuration)


@click.group()
@click.option("--config-file", type=pathlib.Path, default=None)
@click.option("--log-level", type=log_level_type, default="error")
@click.pass_context
def grp_main(ctx: click.Context, config_file: pathlib.Path | None, log_level: LogLevel):
    configure_log_level(log_level)
    config_file = config_file or data_folder.joinpath("default.json")
    config = Configuration.model_validate(json.loads(config_file.read_text()))
    ctx.obj = config


@grp_main.command("convert", help="convert a version into another format")
@click.argument("version")
@click.argument("format", type=version_format_type)
def cmd_convert(version: str, format: VersionFormat):
    version_ = Version.from_semver(version)
    version_ = version_.to_str(format)
    click.echo(version_)


@grp_main.command("current", help="calculate the current version for the local repo")
@pass_configuration
def cmd_current(config: Configuration):
    analyzer = Analyzer.create(config)
    version = analyzer.get_current_version()
    click.echo(version.to_str())


@grp_main.command("next", help="calculate the next version for the local repo")
@pass_configuration
def cmd_next(config: Configuration):
    analyzer = Analyzer.create(config)
    version = analyzer.get_next_version()
    click.echo(version.to_str())


@grp_main.command("set", help="writes version to file")
@click.argument("version", type=str)
@click.argument("file", type=pathlib.Path)
def cmd_set(version: str, file: pathlib.Path):
    set_version(version, file)


@grp_main.command("version", help="print tool version")
def cmd_version():
    tool_version = get_tool_version()
    click.echo(tool_version)


if __name__ == "__main__":
    grp_main()
