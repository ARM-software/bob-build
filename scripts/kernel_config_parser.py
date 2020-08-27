# Copyright 2018-2020 Arm Limited.
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

import logging
import os

logger = logging.getLogger(__name__)

g_kernel_configs = dict()


def parse_kernel_config(kdir):
    """Parse kernel configuration from provided directory"""
    config_file = os.path.join(kdir, '.config')
    config = dict()

    try:
        with open(config_file, "rt") as fp:
            for line in fp.readlines():
                try:
                    (key, val) = line.split("=")
                    config[key.strip()] = val.strip().strip('"')
                except ValueError:
                    pass
    except IOError as e:
        logger.error("Failed to open kernel config file in %s:", config_file)

    return config


def get_value(kdir, option):
    """Return value of the kernel config opption"""
    global g_kernel_configs

    if kdir not in g_kernel_configs:
        g_kernel_configs[kdir] = parse_kernel_config(kdir)

    return g_kernel_configs[kdir].get(option)


def option_enabled(kdir, option):
    """Return true if a given kernel config option is enabled"""
    return get_value(kdir, option) == 'y'


def get_arch(kdir):
    arch_dir = os.path.join(kdir, "arch")
    if not os.path.exists(arch_dir):
        logger.error("'arch' subdirectory in kernel %s does not exist", kdir)
        return None

    # Each directory in $KDIR/arch has a config option with the same name.
    for arch in os.listdir(arch_dir):
        if not os.path.isfile(os.path.join(arch_dir, arch, "Kconfig")):
            continue

        if option_enabled(kdir, "CONFIG_" + arch.upper()):
            return arch

    if option_enabled(kdir, "CONFIG_UML"):
        return "um"
    elif option_enabled(kdir, "CONFIG_X86_32"):
        return "i386"
    elif option_enabled(kdir, "CONFIG_X86_64"):
        return "x86_64"
    elif option_enabled(kdir, "CONFIG_PPC32") or option_enabled(kdir, "CONFIG_PPC64"):
        return "powerpc"
    elif (option_enabled(kdir, "CONFIG_SUPERH") or option_enabled(kdir, "CONFIG_SUPERH32") or
          option_enabled(kdir, "CONFIG_SUPERH64")):
        return "sh"

    logger.error("Couldn't get ARCH for kernel %s", kdir)
    return None
