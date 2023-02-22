# Copyright 2019, 2023 Arm Limited.
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

import pytest
import sys
from config_system import data, expr, general


template_expr_mconfig = """
config TRUE
    bool
    default y

config NUMBER_A
    int "num"
    default 7

config NUMBER_B
    int "num"
    default 127

config PREFIX
    string "pre"
    default "abra"

config SUFFIX
    string "suffix"
    default "cadabra"

config OPTION
    {type} "option"
    default {expr}
"""


expr_testdata = [
    (
        {  # literal addition
            "type": "int",
            "expr": "3+4",
        },
        7,
        None,
    ),
    (
        {  # addition of identifier and literal
            "type": "int",
            "expr": "NUMBER_A+4",
        },
        11,
        None,
    ),
    (
        {  # addition of literal and identifier
            "type": "int",
            "expr": "5+NUMBER_A",
        },
        12,
        None,
    ),
    (
        {  # addition of identifiers
            "type": "int",
            "expr": "NUMBER_A+NUMBER_A",
        },
        14,
        None,
    ),
    (
        {  # literal subtraction
            "type": "int",
            "expr": "10-4",
        },
        6,
        None,
    ),
    (
        {  # subtraction of identifier and literal
            "type": "int",
            "expr": "NUMBER_A-4",
        },
        3,
        None,
    ),
    (
        {  # subtraction of literal and identifier
            "type": "int",
            "expr": "20-NUMBER_A",
        },
        13,
        None,
    ),
    (
        {  # subtraction of identifiers
            "type": "int",
            "expr": "NUMBER_B-NUMBER_A",
        },
        120,
        None,
    ),
    (
        {  # parenthesis in expression
            "type": "int",
            "expr": "NUMBER_B - (NUMBER_A + 10)",
        },
        110,
        None,
    ),
    (
        {  # parenthesis in expression (on LHS)
            "type": "int",
            "expr": "(NUMBER_B + NUMBER_A) - 10",
        },
        124,
        None,
    ),
    (
        {  # string literal concatenation
            "type": "string",
            "expr": '"ban"+"ana"',
        },
        "banana",
        None,
    ),
    (
        {  # string and literal concatenation
            "type": "string",
            "expr": 'PREFIX+"ana"',
        },
        "abraana",
        None,
    ),
    (
        {  # string and literal concatenation
            "type": "string",
            "expr": '"ban"+SUFFIX',
        },
        "bancadabra",
        None,
    ),
    (
        {  # string concatenation
            "type": "string",
            "expr": "PREFIX+SUFFIX",
        },
        "abracadabra",
        None,
    ),
    (
        {  # string concatenation with 'y' (not a bool)
            "type": "string",
            "expr": 'PREFIX+"y"',
        },
        "abray",
        None,
    ),
    (
        {  # string concatenation with 'n' (not a bool)
            "type": "string",
            "expr": '"n"+SUFFIX',
        },
        "ncadabra",
        None,
    ),
    (
        {  # string concatenation with parens (for completeness)
            "type": "string",
            "expr": 'PREFIX+(SUFFIX+"boo")',
        },
        "abracadabraboo",
        None,
    ),
    (
        {  # string subtraction
            "type": "string",
            "expr": "PREFIX-SUFFIX",
        },
        "",
        "'-' operator is not valid on strings",
    ),
    (
        {  # expression with bools (+)
            "type": "bool",
            "expr": "y+n",
        },
        "",
        "'+' operator is not valid on booleans",
    ),
    (
        {  # expression with bools (-)
            "type": "bool",
            "expr": "TRUE-n",
        },
        "",
        "'-' operator is not valid on booleans",
    ),
    (
        {  # mixed expression '+'
            "type": "string",
            "expr": "PREFIX+NUMBER_A",
        },
        "",
        "'+' operator is not valid with mixed types",
    ),
    (
        {  # mixed expression '-'
            "type": "string",
            "expr": "PREFIX-NUMBER_A",
        },
        "",
        "'-' operator is not valid with mixed types",
    ),
    (
        {  # mixed expression with boolean.
            "type": "string",
            "expr": "PREFIX-TRUE",
        },
        "",
        "'-' operator is not valid with mixed types",
    ),
]


@pytest.mark.parametrize("inputdata,result,error", expr_testdata)
def test_expr_evaluation(caplog, tmpdir, inputdata, result, error):
    mconfig_file = tmpdir.join("Mconfig")

    mconfig = template_expr_mconfig.format(**inputdata)
    mconfig_file.write(mconfig, "wt")

    data.init(str(mconfig_file), False)
    c = data.get_config("OPTION")
    general.set_initial_values()

    val = expr.expr_value(c["default"])

    if error is not None:
        assert error in caplog.text
    else:
        assert val == result
        assert caplog.text == ""


template_condexpr_mconfig = """
config TRUE
    bool
    default y

config FALSE
    bool
    default n

config NUMBER_A
    int "num"
    default 7

config NUMBER_B
    int "num"
    default 127

config PREFIX
    string "pre"
    default "abra"

config SUFFIX
    string "suffix"
    default "cadabra"

config OPTION
    bool "option"
    default y if {expr}
"""

condexpr_testdata = [
    (
        {  # literal AND
            "expr": "y && y",
        },
        True,
        None,
    ),
    (
        {  # literal AND
            "expr": "n && y",
        },
        False,
        None,
    ),
    (
        {  # literal AND
            "expr": "y && n",
        },
        False,
        None,
    ),
    (
        {  # literal AND
            "expr": "n && n",
        },
        False,
        None,
    ),
    (
        {  # AND
            "expr": "TRUE && TRUE",
        },
        True,
        None,
    ),
    (
        {  # AND
            "expr": "FALSE && TRUE",
        },
        False,
        None,
    ),
    (
        {  # AND
            "expr": "TRUE && FALSE",
        },
        False,
        None,
    ),
    (
        {  # AND
            "expr": "FALSE && FALSE",
        },
        False,
        None,
    ),
    (
        {  # literal OR
            "expr": "y || y",
        },
        True,
        None,
    ),
    (
        {  # literal OR
            "expr": "n || y",
        },
        True,
        None,
    ),
    (
        {  # literal OR
            "expr": "y || n",
        },
        True,
        None,
    ),
    (
        {  # literal OR
            "expr": "n || n",
        },
        False,
        None,
    ),
    (
        {  # OR
            "expr": "TRUE || TRUE",
        },
        True,
        None,
    ),
    (
        {  # OR
            "expr": "FALSE || TRUE",
        },
        True,
        None,
    ),
    (
        {  # OR
            "expr": "TRUE || FALSE",
        },
        True,
        None,
    ),
    (
        {  # OR
            "expr": "FALSE || FALSE",
        },
        False,
        None,
    ),
    (
        {  # NOT
            "expr": "!TRUE",
        },
        False,
        None,
    ),
    (
        {  # NOT
            "expr": "!FALSE",
        },
        True,
        None,
    ),
    (
        {  # Greater than
            "expr": "NUMBER_B > 126",
        },
        True,
        None,
    ),
    (
        {  # Greater than
            "expr": "NUMBER_B > 127",
        },
        False,
        None,
    ),
    (
        {  # Greater than eq
            "expr": "NUMBER_B >= 127",
        },
        True,
        None,
    ),
    (
        {  # Greater than eq
            "expr": "NUMBER_B >= 128",
        },
        False,
        None,
    ),
    (
        {  # Less than
            "expr": "NUMBER_B < 128",
        },
        True,
        None,
    ),
    (
        {  # Less than
            "expr": "NUMBER_B < 127",
        },
        False,
        None,
    ),
    (
        {  # Less than eq
            "expr": "NUMBER_B <= 127",
        },
        True,
        None,
    ),
    (
        {  # Less than eq
            "expr": "NUMBER_B <= 126",
        },
        False,
        None,
    ),
    (
        {  # Equal
            "expr": "NUMBER_B = 127",
        },
        True,
        None,
    ),
    (
        {  # Equal
            "expr": "NUMBER_B = 126",
        },
        False,
        None,
    ),
    (
        {  # Equal
            "expr": "NUMBER_B = 128",
        },
        False,
        None,
    ),
    (
        {  # Not equal
            "expr": "NUMBER_B != 127",
        },
        False,
        None,
    ),
    (
        {  # Not equal
            "expr": "NUMBER_B != 126",
        },
        True,
        None,
    ),
    (
        {  # Not equal
            "expr": "NUMBER_B != 128",
        },
        True,
        None,
    ),
    (
        {"expr": "NUMBER_B < (121 + NUMBER_A)"},  # Comparison with numeric expression
        True,
        None,
    ),
    ({"expr": '"abracadabra" = PREFIX+SUFFIX'}, True, None),  # Comparison with string
    ({"expr": "FALSE || NUMBER_B < 128"}, True, None),  # Precedence of || vs <
    ({"expr": "TRUE && NUMBER_B < 128"}, True, None),  # Precedence of && vs <
    (
        {"expr": "TRUE && NUMBER_B < 121 + NUMBER_A"},  # Precedence of && vs < vs +
        True,
        None,
    ),
    (
        {  # Precedence of && vs || - && has higher precedence
            "expr": "FALSE && TRUE || TRUE"
        },
        True,
        None,
    ),
    (
        {  # Precedence of && vs || - && has higher precedence
            "expr": "TRUE || FALSE && FALSE"
        },
        True,
        None,
    ),
    (
        {"expr": "FALSE && (TRUE || TRUE)"},  # Parenthesis to give || precedence
        False,
        None,
    ),
    (
        {"expr": "(TRUE || FALSE) && FALSE"},  # Parenthesis to give || precedence
        False,
        None,
    ),
]


@pytest.mark.parametrize("inputdata,result,error", condexpr_testdata)
def test_condexpr_evaluation(caplog, tmpdir, inputdata, result, error):
    mconfig_file = tmpdir.join("Mconfig")

    mconfig = template_condexpr_mconfig.format(**inputdata)
    mconfig_file.write(mconfig, "wt")

    data.init(str(mconfig_file), False)
    c = data.get_config("OPTION")
    general.set_initial_values()

    val = expr.condexpr_value(c["default_cond"][0]["cond"])

    if error is not None:
        assert error in caplog.text
    else:
        assert val == result
        assert caplog.text == ""


if __name__ == "__main__":
    raise SystemExit(pytest.main(sys.argv))
