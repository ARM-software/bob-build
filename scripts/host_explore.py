# Copyright 2018-2019 Arm Limited.
# SPDX-License-Identifier: Apache-2.0
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import distutils.spawn
import os
import sys
import logging
import shlex
import subprocess
import re
import tempfile

from config_system import get_config_bool, get_config_string, set_config

logger = logging.getLogger(__name__)

def which_binary(executable):
    full_path = distutils.spawn.find_executable(executable)
    if not full_path:
        logger.error("Executable (%s) not found", executable)
    return full_path

def check_output(command, dir=None):
    '''
    Executes the command, while making sure the executable is found in the $PATH,
    and returns the output. If the executable wasn't found, returns an empty string.
    The 'command' needs to be an array of arguments.
    '''

    output = ''
    try:
        output = subprocess.check_output(command, cwd=dir).strip()
        output = output.decode(sys.getdefaultencoding())
    except OSError as e:
        logger.error("%s executing '%s'" % (str(e), command[0]))
    except subprocess.CalledProcessError as e:
        logger.warning("Problem executing command: %s" % str(e))

    return output


def gnu_toolchain_is_cross_compiler(tgtType):
    prefix = get_config_string(tgtType + "_GNU_TOOLCHAIN_PREFIX")
    # If the prefix ends with the path separator, it is being used as
    # alternative to adding the compiler to $PATH, and isn't specifying a
    # cross-compilation prefix.
    if prefix and not prefix.endswith(os.sep):
        return True
    return False


def get_cxx_compiler(tgtType):
    """Parse the config options prefixed by `tgtType` and determine the name of
    the compiler for that target type, as well as whether it is a cross
    compiler.
    """
    args = []
    cross_compiler = False

    if get_config_bool(tgtType + "_TOOLCHAIN_GNU"):
        args = shlex.split(get_config_string(tgtType + "_GNU_FLAGS"))
        cxx = get_config_string(tgtType + "_GNU_TOOLCHAIN_PREFIX") + get_config_string("GNU_CXX_BINARY")
        cross_compiler = gnu_toolchain_is_cross_compiler(tgtType)
    elif get_config_bool(tgtType + "_TOOLCHAIN_CLANG"):
        # To use the *_CLANG_TRIPLE option to decide whether this is a
        # cross-compiler would require discovering what target triples are
        # runnable on the host. Instead, just ask the GNU toolchain.
        cross_compiler = gnu_toolchain_is_cross_compiler(tgtType)
        cxx = get_config_string("CLANG_CXX_BINARY")
    elif get_config_bool(tgtType + "_TOOLCHAIN_ARMCLANG"):
        if tgtType != "HOST":
            cross_compiler = True
        cxx = get_config_string("ARMCLANG_CXX_BINARY")

    return cxx, args, cross_compiler


def stl_rpath_ldflags(tgtType):
    """If the toolchain is generating code that can run on the host, add the
    C++ STL library to the linker rpath so it can be found when the binary is
    executed.
    """

    stl_library = get_config_string(tgtType + "_STL_LIBRARY")
    if not stl_library:
        return []

    cxx, args, cross_compiler = get_cxx_compiler(tgtType)

    if cross_compiler:
        return []

    stl_lib = "lib{}.so".format(stl_library)
    stl_path = check_output([cxx] + args + ["-print-file-name=" + stl_lib])
    stl_dir = os.path.dirname(stl_path)
    ld_library_path = os.getenv("LD_LIBRARY_PATH", default="")
    ld_paths = ld_library_path.split(os.pathsep)
    if stl_dir and stl_dir not in ld_paths:
        return ["-Wl,--enable-new-dtags", "-Wl,-rpath,{}".format(stl_dir)]
    return []


def force_bfd_ldflags(tgtType):
    if get_config_bool("BUILDER_ANDROID"):
        return []

    ld = which_binary(get_config_string(tgtType + "_GNU_TOOLCHAIN_PREFIX") + "ld")
    if ld:
        ld_version = check_output([ld, "-version"])
        if ld_version.count("gold") == 1:
            return ["-fuse-ld=bfd"]
    return []


def compiler_config():
    host_ldflags = stl_rpath_ldflags("HOST") + force_bfd_ldflags("HOST")
    target_ldflags = stl_rpath_ldflags("TARGET") + force_bfd_ldflags("TARGET")

    set_config("EXTRA_HOST_LDFLAGS", " ".join(host_ldflags))
    set_config("EXTRA_TARGET_LDFLAGS", " ".join(target_ldflags))


def pkg_config():
    '''
    If package configuration is enabled, then for each library in PKG_CONFIG_PACKAGES, the
    pkg-config utility will be invoked to populate configuration variables.
    The cflags, linker paths and libraries will be assigned to XXX_CFLAGS, XXX_LDFLAGS
    and XXX_LIBS respectively, where XXX is the uppercase package name with any non
    alphanumeric letters replaced by '_'.
    Where no package information exists the default configuration value will be used.
    '''
    if get_config_bool('PKG_CONFIG'):
        pkg_config_path = get_config_string('PKG_CONFIG_PATH')
        if pkg_config_path != '':
            os.putenv('PKG_CONFIG_PATH', pkg_config_path)

        pkg_config_sys_root = get_config_string('PKG_CONFIG_SYSROOT_DIR')
        if pkg_config_sys_root != '':
            os.putenv('PKG_CONFIG_SYSROOT_DIR', pkg_config_sys_root)

        pkg_config_packages = get_config_string('PKG_CONFIG_PACKAGES')

        pkg_config_packages_list = pkg_config_packages.split(',')

        for pkg in pkg_config_packages_list:
            pkg = pkg.strip()
            if pkg == '': continue
            # convert library name to upper case alpha numeric
            pkg_uc_alnum = re.sub('[^a-zA-Z0-9_]', '_', pkg.upper())

            pkg_config_cflags = "%s%s" % (pkg_uc_alnum, '_CFLAGS' )
            pkg_config_ldflags = "%s%s" % (pkg_uc_alnum, '_LDFLAGS' )
            pkg_config_libs = "%s%s" % (pkg_uc_alnum, '_LDLIBS' )

            cflags = check_output(['pkg-config', pkg, '--cflags'])
            if cflags != '':
                set_config(pkg_config_cflags, cflags)

            ldflags = check_output(['pkg-config', pkg, '--libs-only-L'])
            if ldflags != '':
                set_config(pkg_config_ldflags, ldflags)

            libs = check_output(['pkg-config', pkg, '--libs-only-l'])
            if libs != '':
                set_config(pkg_config_libs, libs)


def plugin_exec():
    if get_config_bool('ALLOW_HOST_EXPLORE'):
        compiler_config()
        pkg_config()
