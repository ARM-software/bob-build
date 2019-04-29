# Copyright 2019 Arm Limited.
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

import argparse
import logging
import os
import pytest
import pytest_catchlog
import pytest_mock
import sys
import tempfile

from config_system import general
from config_system import update_config


ignored_option_testdata = [
    # Attempt to set a non-user-settable option
    ("""
config NON_USER_SETTABLE
	bool
	default n
""",
        ["NON_USER_SETTABLE=y"],
        "NON_USER_SETTABLE=y was ignored; it has no title, so is not user-settable " + \
        "(NON_USER_SETTABLE has no unmet dependencies)",
    ),

    # Specify the same option multiple times with different values
    ("""
config USER_SETTABLE_INT_VALUE
    int "This integer can be set by the user"

config BLA
    bool
""",
        ["USER_SETTABLE_INT_VALUE=3", "BLA=n", "USER_SETTABLE_INT_VALUE=4"],
        "USER_SETTABLE_INT_VALUE=3 was overridden by later argument USER_SETTABLE_INT_VALUE=4",
    ),

    # Test the formatting of a simple unmet dependency
    ("""
config FALSE
	bool

config SIMPLE_DEPENDENCIES_NOT_MET
	bool "Is this simple dependency met?"
	depends on FALSE
""",
        ["SIMPLE_DEPENDENCIES_NOT_MET=y"],
        "SIMPLE_DEPENDENCIES_NOT_MET=y was ignored; its dependencies were not met: FALSE[=n]",
    ),

    # Test the formatting of a more complex dependency
    ("""
config STRING_VALUE
	string
	default "string"

config FALSE
	bool

config COMPLEX_DEPENDENCIES_NOT_MET
	bool "Something with a non-trivial dependency"
	depends on STRING_VALUE = "not_string" && !FALSE
""",
        ["COMPLEX_DEPENDENCIES_NOT_MET=y"],
        "COMPLEX_DEPENDENCIES_NOT_MET=y was ignored; its dependencies were not met: " + \
        "(STRING_VALUE[=string] = \"not_string\") && !FALSE[=n]",
    ),


    # Test the formatting of an integer expression, including all the
    # comparison operators
    ("""
config INT_VALUE
	int
	default 60221409

config ANOTHER_INT_VALUE
	int
	default 31415926

config INT_DEPENDENCIES_NOT_MET
	bool "Check some integer ranges"
	depends on (INT_VALUE >= 3 && INT_VALUE <= 25) || (INT_VALUE > 100 && INT_VALUE < 200) || INT_VALUE = ANOTHER_INT_VALUE || INT_VALUE != 60221409
""",
        ["INT_DEPENDENCIES_NOT_MET=y"],
        "INT_DEPENDENCIES_NOT_MET=y was ignored; its dependencies were not met: " + \
        "((((INT_VALUE[=60221409] >= 3) && (INT_VALUE[=60221409] <= 25)) || " + \
        "((INT_VALUE[=60221409] > 100) && (INT_VALUE[=60221409] < 200))) || " + \
        "(INT_VALUE[=60221409] = ANOTHER_INT_VALUE[=31415926])) || " + \
        "(INT_VALUE[=60221409] != 60221409)",
    ),

    # Check we get the right error message when trying to set an unknown option
    (
        "",
        ["UNKNOWN_CONFIGURATION_OPTION=n"],
        "unknown configuration option \"UNKNOWN_CONFIGURATION_OPTION\"",
    ),
]


@pytest.mark.parametrize("mconfig,args,error", ignored_option_testdata)
def test_ignored_config_option(caplog, mocker, tmpdir, mconfig, args, error):
    """For each test case, run update_config's `main()` function with the
    provided Mconfig and command-line arguments. Each case expects exactly one
    error message, relating to the command-line option provided - check that
    this is logged.
    """

    config_fname = tmpdir.join("bob.config")
    mconfig_fname = tmpdir.join("Mconfig")
    mconfig_fname.write(mconfig, "wt")

    mocker.patch("config_system.update_config.parse_args", new=lambda: argparse.Namespace(
        output=str(config_fname),
        database=str(mconfig_fname),
        plugin=[],
        ignore_missing=False,
        args=args,
    ))

    returncode = update_config.main()

    errors = []
    for record in caplog.records:
        if record.levelno == logging.ERROR:
            errors.append(record.message)

    assert returncode == 2
    assert len(errors) == 1
    assert errors[0] == error
