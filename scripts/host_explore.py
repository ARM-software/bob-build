# Copyright 2018 Arm Limited.
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

import os
import sys
import logging
import subprocess
import re
import tempfile

from config_system import get_config_bool, get_config_string, set_config

logger = logging.getLogger(__name__)

def which_binary(executable, path=None):
    '''
    Emulates 'which', trying to locate a binary 'executable' in the 'path',
    or environment '$PATH', if none is given.
    '''

    if not executable.startswith('/'):
        if path is None:
            path = os.environ['PATH']
        paths = path.split(os.pathsep)
        for p in paths:
            f = os.path.join(p, executable)
            if os.path.isfile(f) and os.access(f, os.X_OK):
                return f
        logger.error("Executable (%s) not found in path (%s)", executable, path)
    else:
        if os.path.isfile(executable) and os.access(executable, os.X_OK):
            return executable
        logger.error("Executable (%s) not found", executable)

    return None

def check_output(command, dir=None):
    '''
    Executes the command, while making sure the executable is found in the $PATH,
    and returns the output. If the executable wasn't found, returns an empty string.
    The 'command' needs to be an array of arguments.
    '''
    executable = which_binary(command[0])
    output = ''
    if executable:
        try:
            output = subprocess.check_output(command, cwd=dir).strip()
            output = output.decode(sys.getdefaultencoding())
        except (OSError, subprocess.CalledProcessError) as e:
            logger.warning("Problem executing command: %s" % str(e))

    return output

def compiler_config():
    '''
    Adds runtime-dependant entries by analyzing user-configured flags.
    This is mainly required to allow some configuration-specific flags that are
    impossible or hard to express in Blueprint, because of the need to, say, run
    a binary and get the output from it.
    '''
    extra_target_ldflags = ''
    extra_host_ldflags = ''
    host_cxx = get_config_string('HOST_CXX_BINARY')
    host_libstdcxx_path = check_output([host_cxx, '-print-file-name=libstdc++.so'])
    target_libstdcxx_path = ""

    # No toolchain prefix indicates a native build
    native_build = get_config_string('TARGET_GNU_TOOLCHAIN_PREFIX') == ''
    if get_config_bool('TARGET_TOOLCHAIN_CLANG'):
        if native_build:
            target_libstdcxx_path = host_libstdcxx_path
        else:
            cross_gcc = which_binary(get_config_string('TARGET_GNU_TOOLCHAIN_PREFIX') +
                get_config_string('GNU_CC_BINARY'))
            flags = get_config_string('TARGET_GNU_FLAGS').split(" ")
            flags = list(filter(None, flags))

            cross_sysroot = check_output([cross_gcc] + flags + ['-print-sysroot'])
            set_config('TARGET_SYSROOT', cross_sysroot)
            cross_version = check_output([cross_gcc] + flags + ['-dumpversion'])
            set_config('TARGET_GNU_TOOLCHAIN_VERSION', cross_version)
            target_libstdcxx_path = check_output([cross_gcc] + flags + ['-print-file-name=libstdc++.so'])
            crt_path = os.path.split(check_output([cross_gcc] + flags + ['-print-file-name=crt1.o']))[0]
            if crt_path != '':
                extra_target_ldflags += '-B{0} '.format(crt_path)

            libgcc_static_path = os.path.split(check_output([cross_gcc] + flags + ['-print-file-name=libgcc.a']))[0]
            if libgcc_static_path != '':
                extra_target_ldflags += '-B{0} -L{0} '.format(libgcc_static_path)

            crosslib_path = os.path.split(check_output([cross_gcc] + flags + ['-print-file-name=libgcc_s.so']))[0]
            if crosslib_path != '':
                extra_target_ldflags += '-L{0} '.format(crosslib_path)

    elif get_config_bool('TARGET_TOOLCHAIN_GNU'):
        flags = get_config_string('TARGET_GNU_FLAGS').split(" ")
        flags = list(filter(None, flags))
        target_libstdcxx_path = check_output([host_cxx] + flags + ['-print-file-name=libstdc++.so'])

    host_libstdcxx_dir = os.path.split(host_libstdcxx_path)[0]
    ld_library_path = os.getenv('LD_LIBRARY_PATH','')
    ld_paths = ld_library_path.split(":")
    if len(host_libstdcxx_dir) > 0:
        if host_libstdcxx_dir not in ld_paths:
            extra_host_ldflags += '-Wl,--enable-new-dtags -Wl,-rpath,{0} '.format(host_libstdcxx_dir)
            set_config('EXTRA_LD_LIBRARY_PATH', host_libstdcxx_dir)

    # In native builds we require the addition of rpath to the executable if not in load library path
    if native_build:
        target_libstdcxx_dir = os.path.split(target_libstdcxx_path)[0]
        if len(target_libstdcxx_dir) > 0:
            if target_libstdcxx_dir not in ld_paths:
                extra_target_ldflags += '-Wl,--enable-new-dtags -Wl,-rpath,{0} '.format(target_libstdcxx_dir)

    if not get_config_bool('BUILDER_ANDROID'):
        cross_ld = which_binary(get_config_string('TARGET_GNU_TOOLCHAIN_PREFIX') + 'ld')
        if cross_ld:
            cross_ld_version = check_output([cross_ld, '-version'])
            if cross_ld_version.count('gold') == 1:
                extra_target_ldflags += '-fuse-ld=bfd '

        host_ld = which_binary('ld')
        if host_ld:
            ld_version = check_output([host_ld, '-version'])
            if ld_version.count('gold') == 1:
                extra_host_ldflags += '-fuse-ld=bfd '

        set_config('EXTRA_TARGET_LDFLAGS', extra_target_ldflags)
    set_config('EXTRA_HOST_LDFLAGS', extra_host_ldflags)


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
