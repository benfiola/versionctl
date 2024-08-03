import logging
import sys
from typing import Literal

import versionctl

LogLevel = Literal["debug"] | Literal["info"] | Literal["warning"] | Literal["error"]
log_level_map: dict[LogLevel, int] = {
    "debug": logging.DEBUG,
    "info": logging.INFO,
    "warning": logging.WARNING,
    "error": logging.ERROR,
}


def configure_log_level(log_level: LogLevel):
    """
    Configures logging for the application
    """
    logger = logging.getLogger(versionctl.__name__)
    handler = logging.StreamHandler(sys.stderr)
    formatter = logging.Formatter("%(msg)s")
    handler.setFormatter(formatter)
    logger.addHandler(handler)
    logger.setLevel(log_level_map[log_level])
