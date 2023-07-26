import os
import sys
import logging
import subprocess
import re

from config_system import (
    get_config_bool,
    get_config_string,
    set_config,
    get_mconfig_dir,
)

logger = logging.getLogger(__name__)


def check_output(command, dir=None):
    """
    Executes the command, while making sure the executable is found in the $PATH,
    and returns the output. If the executable wasn't found, returns an empty string.
    The 'command' needs to be an array of arguments.
    """

    output = ""
    try:
        output = subprocess.check_output(command, cwd=dir).strip()
        output = output.decode(sys.getdefaultencoding())
    except OSError as e:
        logger.error("%s executing '%s'" % (str(e), command[0]))
    except subprocess.CalledProcessError as e:
        logger.warning("Problem executing command: %s" % str(e))

    return output


def pkg_config():
    """
    If package configuration is enabled, then for each library in PKG_CONFIG_PACKAGES, the
    pkg-config utility will be invoked to populate configuration variables.
    The cflags, linker paths and libraries will be assigned to XXX_CFLAGS, XXX_LDFLAGS
    and XXX_LIBS respectively, where XXX is the uppercase package name with any non
    alphanumeric letters replaced by '_'.
    Where no package information exists the default configuration value will be used.
    """
    if get_config_bool("PKG_CONFIG"):
        cmd = [get_config_string("PKG_CONFIG_BINARY")]
        pkg_config_flags = get_config_string("PKG_CONFIG_FLAGS")
        pkg_config_flags = pkg_config_flags.replace("%MCONFIGDIR%", get_mconfig_dir())
        cmd.extend(pkg_config_flags.split(" "))

        # clean already existing env vars, as leaving them may be erroneous
        for k in ("PKG_CONFIG_PATH", "PKG_CONFIG_SYSROOT_DIR"):
            if k in os.environ:
                logger.warning(
                    "Environment variable %s is already defined. "
                    "It will be ignored." % k
                )
                del os.environ[k]

        pkg_config_path = get_config_string("PKG_CONFIG_PATH")
        if pkg_config_path != "":
            pkg_config_path = pkg_config_path.replace("%MCONFIGDIR%", get_mconfig_dir())
            os.putenv("PKG_CONFIG_PATH", pkg_config_path)

        pkg_config_sys_root = get_config_string("PKG_CONFIG_SYSROOT_DIR")
        if pkg_config_sys_root != "":
            pkg_config_sys_root = pkg_config_sys_root.replace(
                "%MCONFIGDIR%", get_mconfig_dir()
            )
            os.putenv("PKG_CONFIG_SYSROOT_DIR", pkg_config_sys_root)

        pkg_config_packages = get_config_string("PKG_CONFIG_PACKAGES")

        pkg_config_packages_list = pkg_config_packages.split(",")

        for pkg in pkg_config_packages_list:
            pkg = pkg.strip()
            if pkg == "":
                continue
            # convert library name to upper case alpha numeric
            pkg_uc_alnum = re.sub("[^a-zA-Z0-9_]", "_", pkg.upper())

            pkg_config_cflags = "%s%s" % (pkg_uc_alnum, "_CFLAGS")
            pkg_config_ldflags = "%s%s" % (pkg_uc_alnum, "_LDFLAGS")
            pkg_config_libs = "%s%s" % (pkg_uc_alnum, "_LDLIBS")

            cflags = check_output(cmd + [pkg, "--cflags"])
            if cflags != "":
                set_config(pkg_config_cflags, cflags)

            ldflags = check_output(cmd + [pkg, "--libs-only-L"])
            if ldflags != "":
                set_config(pkg_config_ldflags, ldflags)

            libs = check_output(cmd + [pkg, "--libs-only-l"])
            if libs != "":
                set_config(pkg_config_libs, libs)


def plugin_exec():
    if get_config_bool("ALLOW_HOST_EXPLORE"):
        pkg_config()
