import os
import sys
import pytest

TEST_DIR = os.path.dirname(os.path.abspath(__file__))
CFG_DIR = os.path.dirname(TEST_DIR)
sys.path.append(CFG_DIR)

from config_system import general

template_type_strictness_default_mconfig = """
config TRUE
    bool
    default y

config TEXT
    string
    default "text"

config OPTION
    bool
    {expr}
"""


type_strictness_default_testdata = [
    (
        {  # default with the wrong type (string)
            "expr": 'default "string_1"',
        },
        "Type mismatch in config OPTION: expected bool but got string",
    ),
    (
        {  # default with the wrong type (int)
            "expr": "default 1",
        },
        "Type mismatch in config OPTION: expected bool but got int",
    ),
    (
        {  # default with the right type (boolean true)
            "expr": "default y",
        },
        None,
    ),
    (
        {  # default with the right type (boolean false)
            "expr": "default n",
        },
        None,
    ),
    (
        {  # default with the wrong type (string)
            "expr": "default TEXT",
        },
        "Type mismatch in config OPTION: expected bool but got string",
    ),
    (
        {  # default with the right type (boolean true)
            "expr": "default TRUE",
        },
        None,
    ),
    (
        {  # default_cond with with the wrong type (string)
            "expr": 'default "y" if y',
        },
        "Type mismatch in config OPTION: expected bool but got string",
    ),
    (
        {  # default_cond with the right type
            "expr": "default y if y",
        },
        None,
    ),
    (
        {  # two default conds: one with the right type and one with the wrong type
            "expr": "\n".join(["default n if y", 'default "y" if n']),
        },
        "Type mismatch in config OPTION: expected bool but got string",
    ),
    (
        {  # two default conds: both with the right types
            "expr": "\n".join(["default n if y", "default y if n"]),
        },
        None,
    ),
]


@pytest.mark.parametrize("inputdata,error", type_strictness_default_testdata)
def test_type_strictness_default(caplog, tmpdir, inputdata, error):
    mconfig_file = tmpdir.join("Mconfig")

    mconfig = template_type_strictness_default_mconfig.format(**inputdata)
    mconfig_file.write(mconfig, "wt")

    general.init_config(str(mconfig_file), False)

    if error is not None:
        assert error in caplog.text
    else:
        assert caplog.text == ""


template_type_strictness_select_mconfig = """
config OPTION_1
  bool
  default n
config OPTION_2
  string
  default "text"
config OPTION_3
  int
  default 1
config OPTION_4
  bool
  {expr}
"""


type_strictness_select_testdata = [
    (
        {  # select with the wrong type (string)
            "expr": "select OPTION_2",
        },
        "Select option must have type bool but got type string instead",
    ),
    (
        {  # select with the right type (bool)
            "expr": "select OPTION_1",
        },
        None,
    ),
    (
        {  # select with the wrong type (string)
            "expr": "select OPTION_2 if OPTION_1",
        },
        "Select option must have type bool but got type string instead",
    ),
    (
        {  # select with the wrong type (int)
            "expr": "select OPTION_3 if !OPTION_1",
        },
        "Select option must have type bool but got type int instead",
    ),
    (
        {  # select with the right type (bool)
            "expr": "select OPTION_1 if OPTION_3 = 4",
        },
        None,
    ),
    (
        {  # select with the wrong type (int)
            "expr": "\n".join(
                ["select OPTION_1 if OPTION_3 = 4", "select OPTION_3 if OPTION_1"]
            ),
        },
        "Select option must have type bool but got type int instead",
    ),
    (
        {  # select with the wrong type (int)
            "expr": "\n".join(["select OPTION_1 if OPTION_3 = 4", "select OPTION_3"]),
        },
        "Select option must have type bool but got type int instead",
    ),
    (
        {  # select with the right type (bool)
            "expr": "\n".join(
                [
                    "select OPTION_1 if OPTION_3 = 4",
                    'select OPTION_1 if OPTION_2 = "str"',
                ]
            ),
        },
        None,
    ),
]


@pytest.mark.parametrize("inputdata,error", type_strictness_select_testdata)
def test_type_strictness_select(caplog, tmpdir, inputdata, error):
    mconfig_file = tmpdir.join("Mconfig")

    mconfig = template_type_strictness_select_mconfig.format(**inputdata)
    mconfig_file.write(mconfig, "wt")

    general.init_config(str(mconfig_file), False)

    if error is not None:
        assert error in caplog.text
    else:
        assert caplog.text == ""


if __name__ == "__main__":
    raise SystemExit(pytest.main(sys.argv))
