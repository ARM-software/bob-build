#!/usr/bin/env python3
#
# -----------------------------------------------------------------------------
# The proprietary software and information contained in this file is
# confidential and may only be used by an authorized person under a valid
# licensing agreement from Arm Limited or its affiliates.
#
# Copyright (C) 2026. Arm Limited or its affiliates. All rights reserved.
#
# This entire notice must be reproduced on all copies of this file and
# copies of this file may only be made by an authorized person under a valid
# licensing agreement from Arm Limited or its affiliates.
# -----------------------------------------------------------------------------

import argparse
import logging
import os
import re
import sys
from dataclasses import dataclass
from typing import List

LOGGER = logging.getLogger(__name__)


@dataclass(frozen=True)
class CodeMatcher:
    filename: str
    text: str

    def match(self, android_top: str) -> bool:
        path = os.path.join(android_top, self.filename)
        try:
            with open(path, encoding="utf-8") as f:
                matched = re.search(self.text, f.read()) is not None
        except OSError as err:
            LOGGER.debug(
                "Could not read %s while checking %r: %s", path, self.text, err
            )
            return False

        if not matched:
            LOGGER.debug("%r did not match %s", self.text, path)

        return matched


@dataclass(frozen=True)
class CompatVersion:
    matches: List[CodeMatcher]
    android_versions: List[int]
    src: str


def get_soong_compat_file(android_top: str, android_platform_version: int) -> str:
    list_of_android_mk_entries_matcher = CodeMatcher(
        filename="build/soong/android/androidmk.go",
        text=r"\n\tAndroidMkEntries\(\) \[\]AndroidMkEntries\n",
    )

    android_mk_extra_entries_context_matcher = CodeMatcher(
        filename="build/soong/android/androidmk.go",
        text=r"\ntype AndroidMkExtraEntriesContext interface {\n",
    )

    android_mk_soong_install_targets_matcher = CodeMatcher(
        filename="build/soong/android/androidmk.go",
        text=r'a.SetPath\("LOCAL_SOONG_INSTALLED_MODULE", (base|info).[Kk]atiInstalls\[len\((base|info).[Kk]atiInstalls\)-1\].to\)\n',
    )

    android_host_tool_provider_info_provider_matcher = CodeMatcher(
        filename="build/soong/android/module.go",
        text=r"\nvar HostToolProviderInfoProvider = blueprint.NewProvider\[HostToolProviderInfo\]\(\)\n",
    )

    android_common_module_info_provider_matcher = CodeMatcher(
        filename="build/soong/android/module.go",
        text=r"\nvar CommonModuleInfoProvider = blueprint.NewProvider\[\*CommonModuleInfo\]\(\)\n",
    )

    android_module_or_proxy_matcher = CodeMatcher(
        filename="build/soong/android/module_proxy.go",
        text=r"\ntype ModuleOrProxy interface {\n",
    )

    android_visit_module_proxy_matcher = CodeMatcher(
        filename="build/soong/android/base_module_context.go",
        text=r"\n\tVisitDirectDepsProxyWithTag\(tag blueprint.DependencyTag, visit func\(proxy ModuleProxy\)\)\n",
    )

    android_host_tool_info_matcher = CodeMatcher(
        filename="build/soong/android/module.go",
        text=r"\n\tHostToolInfo                   \*HostToolInfo\n",
    )

    android_custom_enc_provider_matcher = CodeMatcher(
        filename="build/blueprint/provider.go",
        text=r"\nfunc NewProvider\[K gobtools.CustomEnc\]\(\) ProviderKey\[K\] {\n",
    )

    # List of compatibility layers, ordered from oldest Soong version support to
    # newest.
    all_soong_compats = [
        # AndroidMkEntries() was made to return an array in 0b0e1b9
        CompatVersion(
            matches=[list_of_android_mk_entries_matcher],
            android_versions=[9, 10, 11, 12],
            src="soong_compat_00_pqr.go",
        ),
        # AndroidMkExtraEntriesContext was added in aa25553
        CompatVersion(
            matches=[
                list_of_android_mk_entries_matcher,
                android_mk_extra_entries_context_matcher,
            ],
            android_versions=[12],
            src="soong_compat_01_AndroidMkExtraEntries_ctx.go",
        ),
        # Soong install Mk targets have been added in 6301c3c
        CompatVersion(
            matches=[
                list_of_android_mk_entries_matcher,
                android_mk_extra_entries_context_matcher,
                android_mk_soong_install_targets_matcher,
            ],
            android_versions=[13, 14, 15],
            src="soong_compat_02_AndroidMkSoongInstallTargets.go",
        ),
        # Soong HostToolProviderInfoProvider
        CompatVersion(
            matches=[
                list_of_android_mk_entries_matcher,
                android_mk_extra_entries_context_matcher,
                android_mk_soong_install_targets_matcher,
                android_host_tool_provider_info_provider_matcher,
            ],
            android_versions=[16],
            src="soong_compat_03_HostBinProvider.go",
        ),
        # Soong HostToolProviderInfoProvider
        CompatVersion(
            matches=[
                list_of_android_mk_entries_matcher,
                android_mk_extra_entries_context_matcher,
                android_mk_soong_install_targets_matcher,
                android_host_tool_provider_info_provider_matcher,
                android_module_or_proxy_matcher,
                android_visit_module_proxy_matcher,
            ],
            android_versions=[16],
            src="soong_compat_04_ModuleProxy.go",
        ),
        CompatVersion(
            matches=[
                list_of_android_mk_entries_matcher,
                android_mk_extra_entries_context_matcher,
                android_mk_soong_install_targets_matcher,
                android_host_tool_provider_info_provider_matcher,
                android_module_or_proxy_matcher,
                android_visit_module_proxy_matcher,
                android_custom_enc_provider_matcher,
            ],
            android_versions=[16, 17],
            src="soong_compat_05_Provider_Enc_Dec.go",
        ),
        CompatVersion(
            matches=[
                list_of_android_mk_entries_matcher,
                android_mk_extra_entries_context_matcher,
                android_mk_soong_install_targets_matcher,
                android_common_module_info_provider_matcher,
                android_module_or_proxy_matcher,
                android_visit_module_proxy_matcher,
                android_host_tool_info_matcher,
                android_custom_enc_provider_matcher,
            ],
            android_versions=[16, 17],
            src="soong_compat_06_CommonModuleInfoProvider.go",
        ),
    ]

    soong_compats = [
        compat
        for compat in all_soong_compats
        if android_platform_version in compat.android_versions
    ]

    if len(soong_compats) == 1:
        return soong_compats[0].src

    if len(soong_compats) == 0:
        LOGGER.warning(
            "No available Soong compatibility layers found for "
            f"ANDROID_PLATFORM_VERSION = {android_platform_version}.\n"
            "Attempting text-based detection."
        )
        soong_compats = all_soong_compats

    # If there are multiple potential options for this Android version, try to
    # differentiate by matching specific lines in Soong's source. Search from
    # newest to oldest - newer Soong versions may contain older code fragments
    # too, so going the other way could mean incorrectly choosing an earlier
    # version.
    for compat in reversed(soong_compats):
        if all(matcher.match(android_top) for matcher in compat.matches):
            return compat.src

    LOGGER.warning(
        "Could not find an appropriate Soong compatibility layer "
        "based on code in build/soong.\n"
        "Falling back to default for this Android version. "
        "Compilation of Bob plugins may fail!"
    )
    return soong_compats[-1].src


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Detect the Bob Soong compatibility file for an Android tree."
    )
    parser.add_argument(
        "--android-top",
        required=True,
        help="Path to the root of the Android source tree.",
    )
    parser.add_argument(
        "--android-platform-version",
        required=True,
        type=int,
        help="Android PLATFORM_VERSION used by the DDK configuration.",
    )
    parser.add_argument(
        "--log-level",
        choices=["debug", "info", "warning", "error"],
        default="warning",
        help="Logging verbosity. Defaults to warning.",
    )
    args = parser.parse_args()

    logging.basicConfig(
        format="%(message)s",
        level=getattr(logging, args.log_level.upper()),
    )

    print(
        get_soong_compat_file(
            os.path.abspath(args.android_top),
            args.android_platform_version,
        )
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
