# Copyright 2023 Arm Limited.
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

import json
import io
import pytest

from pathlib import Path

import get_configs_gazelle


option_config = [
    (
        "subdir",
        "internal/libA",
    ),
    (
        "other_relative_path",
        "external/crypto/tool",
    ),
]


@pytest.mark.parametrize("relativePath,submodule", option_config)
def test_option_config_relPath(capfd, monkeypatch, tmp_path, relativePath, submodule):
    """
    For each test case, run get_configs_gazelle's `main()` function for two
    Mconfig files located as:

    tmp_path
    └── relativePath
        ├── Mconfig
        └── submodule
            └── Mconfig

    Check weather configurations `relPath` property is properly set.
    Starting point for parse is `tmp_path/relativePath` directory thus
    `relPath` property for all configs of the root Mconfig file
    (`tmp_path/relativePath/Mconfig`) should be "relativePath" where for the
    configs of internal one (`tmp_path/relativePath/submodule/Mconfig`)
    should be "relativePath/submodule".
    """

    # fmt: off
    input_json = (
        f'{{"root_path": "{tmp_path}","rel_package_path": "{relativePath}","file_name": "Mconfig"}}'
    )
    # fmt: on

    monkeypatch.setattr("sys.stdin", io.StringIO(input_json))

    mconfig_data = f"""
config SUB_FEATURE_X
    bool "Enable Feature X"
    default y
source "{submodule}/Mconfig"
config SUB_FEATURE_Y
    bool "Enable Feature Y"
    default n
"""

    submconfig_data = """
config FEATURE_A
    bool "Enable Feature A"
    default y
config OPTION_B
    string "Set Option XY"
    default "--secret"
"""

    mconfig_dir = tmp_path / relativePath
    mconfig_dir.mkdir(parents=True, exist_ok=True)
    mconfig_fname = tmp_path / relativePath / "Mconfig"
    mconfig_fname.write_text(mconfig_data)

    submconfig_dir = tmp_path / relativePath / submodule
    submconfig_dir.mkdir(parents=True, exist_ok=True)
    submconfig_fname = submconfig_dir / "Mconfig"
    submconfig_fname.write_text(submconfig_data)

    returncode = get_configs_gazelle.main()

    out = capfd.readouterr()

    configuration = json.loads(out.out.strip())

    for cfg in ["SUB_FEATURE_X", "SUB_FEATURE_Y"]:
        assert configuration[cfg]["relPath"] == relativePath

    for cfg in ["FEATURE_A", "OPTION_B"]:
        assert configuration[cfg]["relPath"] == str(Path(relativePath, submodule))

    assert returncode == 0
