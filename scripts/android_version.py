#!/usr/bin/env python3


import argparse
import logging
import os
import subprocess
import sys


logger = logging.getLogger(__name__)


def get_platform_version():
    android_build_top = os.getenv("ANDROID_BUILD_TOP")
    if android_build_top is None:
        logger.error("ANDROID_BUILD_TOP not set")
        return None

    soong_ui = os.path.join(android_build_top, "build", "soong", "soong_ui.bash")
    try:
        cmd = [soong_ui, "--dumpvar-mode", "PLATFORM_VERSION"]
        # Ignore soong_ui's stderr output by redirecting it. This does not end
        # up in the captured output.
        platform_version = (
            subprocess.check_output(cmd, stderr=subprocess.PIPE).decode().strip()
        )
    except (OSError, subprocess.CalledProcessError) as e:
        logger.error("%s", str(e))
        return None

    if platform_version.isalpha():
        # aosp master may have a single letter for PLATFORM_VERSION eg. 'Q' for Android 10
        platform_version = ord(platform_version[0]) - 71
    return int(platform_version)


if __name__ == "__main__":
    logging.basicConfig()

    version = get_platform_version()
    if version is not None:
        sys.stdout.write(str(version) + "\n")
    else:
        sys.stderr.write("Could not get Android version\n")
        sys.exit(1)
