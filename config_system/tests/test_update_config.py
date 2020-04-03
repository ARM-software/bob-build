# Copyright 2019-2020 Arm Limited.
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

import update_config

ignored_option_testdata = [
    # Attempt to set a non-user-settable option
    (
        """
config NON_USER_SETTABLE
	bool
	default n
""",
        ["NON_USER_SETTABLE=y"],
        "NON_USER_SETTABLE=y was ignored; it has no title, so is not user-settable " + \
        "(NON_USER_SETTABLE has no unmet dependencies)",
    ),

    # Specify the same option multiple times with different values
    (
        """
config USER_SETTABLE_INT_VALUE
    int "This integer can be set by the user"

config BLA
    bool
""",
        ["USER_SETTABLE_INT_VALUE=3", "BLA=n", "USER_SETTABLE_INT_VALUE=4"],
        "USER_SETTABLE_INT_VALUE=3 was overridden by later argument USER_SETTABLE_INT_VALUE=4",
    ),

    # Test the formatting of a simple unmet dependency
    (
        """
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
    (
        """
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
    (
        """
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

    mocker.patch("update_config.parse_args", new=lambda: argparse.Namespace(
        config=str(config_fname),
        database=str(mconfig_fname),
        json=None,
        new=True,
        plugin=[],
        ignore_missing=False,
        args=args,
    ))

    update_config.counter.reset()
    returncode = update_config.main()

    errors = []
    for record in caplog.records:
        if record.levelno == logging.ERROR:
            errors.append(record.message)

    assert returncode != 0
    assert len(errors) == 1
    assert errors[0] == error


# Test handling of dependent boolean values
select_depend_testdata = [
    # Depend on with no issue
    (
        """
config GATE
    bool "gating"

config OPTION
    bool "option"
    depends on GATE
""",
        None,
        ["GATE=y", "OPTION=y"],
        [],
        [],
        [],
    ),
    # Set the option when the dependency is not met.
    # Check of command line options will fail
    (
        """
config GATE
    bool "gating"

config OPTION
    bool "option"
    depends on GATE
""",
        None,
        ["OPTION=y"],
        [],
        [],
        ["OPTION=y was ignored; its dependencies were not met: GATE[=n]"],
    ),
    # Force the option. Error raised to fix Mconfig.
    (
        """
config GATE
    bool "gating"

config OPTION
    bool "option"
    depends on GATE

config FORCE
    bool "force"
    select OPTION
""",
        None,
        ["FORCE=y"],
        [],
        [],
        ["Inconsistent values: unmet direct dependencies: OPTION depends on GATE, but is selected by [FORCE]. Update the Mconfig so that this can't happen"],
    ),
    # Input contains an inconsistency on read, fix up on read,
    # Error still produced
    (
        """
config GATE
    bool "gating"

config OPTION
    bool "option"
    depends on GATE

config FORCE
    bool "force"
    select OPTION
""",
        """
#CONFIG_GATE=n
CONFIG_OPTION=y
CONFIG_FORCE=y
        """,
        [],
        ["Inconsistency prior to plugins: unmet direct dependencies: OPTION depends on GATE, but is selected by [FORCE]."],
        ["Inconsistent input, correcting: unmet direct dependencies: OPTION depends on GATE, but is selected by [FORCE]."],
        ["Inconsistent values: unmet direct dependencies: OPTION depends on GATE, but is selected by [FORCE]. Update the Mconfig so that this can't happen"],
    ),
    # Input contains an inconsistency on read, fix up on read,
    # No error.
    (
        """
config GATE
    bool "gating"

config OPTION
    bool "option"
    depends on GATE
""",
        """
#CONFIG_GATE=n
CONFIG_OPTION=y
        """,
        [],
        [],
        [],
        [],
    ),
]

@pytest.mark.parametrize("mconfig,config,args,expected_infos,expected_warnings,expected_errors", select_depend_testdata)
def test_select_depend(caplog, mocker, tmpdir,
                       mconfig, config, args, expected_infos,
                       expected_warnings, expected_errors):
    """
    For each test case, run update_config's `main()` function with the
    provided Mconfig and command-line arguments. No errors
    should be reported.
    """
    caplog.set_level(logging.INFO)

    config_fname = tmpdir.join("bob.config")
    mconfig_fname = tmpdir.join("Mconfig")
    mconfig_fname.write(mconfig, "wt")
    if config != None:
        config_fname.write(config)

    mocker.patch("update_config.parse_args", new=lambda: argparse.Namespace(
        config=str(config_fname),
        database=str(mconfig_fname),
        json=None,
        new=(config==None),
        plugin=[],
        ignore_missing=False,
        args=args,
    ))

    update_config.counter.reset()
    returncode = update_config.main()

    infos = []
    warnings = []
    errors = []
    for record in caplog.records:
        if record.levelno == logging.INFO:
            infos.append(record.message)
        if record.levelno == logging.WARNING:
            warnings.append(record.message)
        if record.levelno == logging.ERROR:
            errors.append(record.message)

    for single_info in expected_infos:
        assert single_info in infos

    assert warnings == expected_warnings
    assert errors == expected_errors
    if len(expected_errors) < 1:
        assert returncode == 0
    else:
        assert returncode != 0

option_depends_on_plugin_testdata = [
    # Attempt to set an option that depends on something set by a plugin
    (
        """
from config_system import set_config

def plugin_exec():
    set_config("PLUGIN_SET_OPTION", True)
""",
        """
config PLUGIN_SET_OPTION
    bool "plugin set"
    default n

config USER_OPTION
    bool "user"
    depends on PLUGIN_SET_OPTION
    default n
""",
        ["USER_OPTION=y"],
    ),
]


@pytest.mark.parametrize("plugin,mconfig,args", option_depends_on_plugin_testdata)
def test_option_depends_on_plugin(caplog, mocker, tmpdir, plugin, mconfig, args):
    """
    For each test case, run update_config's `main()` function with the
    provided plugin, Mconfig and command-line arguments. No errors
    should be reported.
    """

    config_fname = tmpdir.join("bob.config")
    mconfig_fname = tmpdir.join("Mconfig")
    plugin_fname = tmpdir.join("plugin.py")
    mconfig_fname.write(mconfig, "wt")
    plugin_fname.write(plugin, "wt")

    mocker.patch("update_config.parse_args", new=lambda: argparse.Namespace(
        config=str(config_fname),
        database=str(mconfig_fname),
        json=None,
        new=True,
        plugin=[os.path.splitext(str(plugin_fname))[0]],
        ignore_missing=False,
        args=args,
    ))

    update_config.counter.reset()
    returncode = update_config.main()

    errors = []
    for record in caplog.records:
        if record.levelno == logging.ERROR:
            errors.append(record.message)

    assert returncode == 0
    assert len(errors) == 0
