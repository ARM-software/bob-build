import logging
import os

from config_system import get_config_string, set_config

logger = logging.getLogger(__name__)


def plugin_exec():
    try:
        android_build_top = os.environ["ANDROID_BUILD_TOP"]
    except KeyError as e:
        logger.error("ANDROID_BUILD_TOP is not set - did you run 'lunch'?")
        raise e

    clang_prefix = android_build_top + "/" + get_config_string("TARGET_CLANG_PREFIX")

    set_config("TARGET_CLANG_PREFIX", clang_prefix)
    set_config("ANDROID_BUILD_TOP", android_build_top)  # should not be set
