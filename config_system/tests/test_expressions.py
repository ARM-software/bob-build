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

import pytest

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
        7, None
    ),
    (
        {  # addition of identifier and literal
            "type": "int",
            "expr": "NUMBER_A+4",
        },
        11, None
    ),
    (
        {  # addition of literal and identifier
            "type": "int",
            "expr": "5+NUMBER_A",
        },
        12, None
    ),
    (
        {  # addition of identifiers
            "type": "int",
            "expr": "NUMBER_A+NUMBER_A",
        },
        14, None
    ),
    (
        {  # literal subtraction
            "type": "int",
            "expr": "10-4",
        },
        6, None
    ),
    (
        {  # subtraction of identifier and literal
            "type": "int",
            "expr": "NUMBER_A-4",
        },
        3, None
    ),
    (
        {  # subtraction of literal and identifier
            "type": "int",
            "expr": "20-NUMBER_A",
        },
        13, None
    ),
    (
        {  # subtraction of identifiers
            "type": "int",
            "expr": "NUMBER_B-NUMBER_A",
        },
        120, None
    ),
    (
        {  # parenthesis in expression
            "type": "int",
            "expr": "NUMBER_B - (NUMBER_A + 10)",
        },
        110, None
    ),
    (
        {  # parenthesis in expression (on LHS)
            "type": "int",
            "expr": "(NUMBER_B + NUMBER_A) - 10",
        },
        124, None
    ),
    (
        {  # string literal concatenation
            "type": "string",
            "expr": '"ban"+"ana"',
        },
        "banana", None
    ),
    (
        {  # string and literal concatenation
            "type": "string",
            "expr": 'PREFIX+"ana"',
        },
        "abraana", None
    ),
    (
        {  # string and literal concatenation
            "type": "string",
            "expr": '"ban"+SUFFIX',
        },
        "bancadabra", None
    ),
    (
        {  # string concatenation
            "type": "string",
            "expr": "PREFIX+SUFFIX",
        },
        "abracadabra", None
    ),
    (
        {  # string concatenation with parens (for completeness)
            "type": "string",
            "expr": 'PREFIX+(SUFFIX+"boo")',
        },
        "abracadabraboo", None
    ),
    (
        {  # string subtraction
            "type": "string",
            "expr": "PREFIX-SUFFIX",
        },
        "", "'-' operator is not valid on strings"
    ),
    (
        {  # expression with bools (+)
            "type": "bool",
            "expr": "y+n",
        },
        "", "'+' operator is not valid on booleans"
    ),
    (
        {  # expression with bools (-)
            "type": "bool",
            "expr": "TRUE-n",
        },
        "", "'-' operator is not valid on booleans"
    ),
    (
        {  # mixed expression '+'
            "type": "string",
            "expr": "PREFIX+NUMBER_A",
        },
        "", "'+' operator is not valid with mixed types"
    ),
    (
        {  # mixed expression '-'
            "type": "string",
            "expr": "PREFIX-NUMBER_A",
        },
        "", "'-' operator is not valid with mixed types"
    ),
    (
        {  # mixed expression with boolean. Note we can't detect mixed,
           # so this complains about boolean
            "type": "string",
            "expr": "PREFIX-TRUE",
        },
        "", "'-' operator is not valid on booleans"
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

    val = expr.expr_value(c['default'])

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
        "y", None
    ),
    (
        {  # literal AND
            "expr": "n && y",
        },
        "n", None
    ),
    (
        {  # literal AND
            "expr": "y && n",
        },
        "n", None
    ),
    (
        {  # literal AND
            "expr": "n && n",
        },
        "n", None
    ),
    (
        {  # AND
            "expr": "TRUE && TRUE",
        },
        "y", None
    ),
    (
        {  # AND
            "expr": "FALSE && TRUE",
        },
        "n", None
    ),
    (
        {  # AND
            "expr": "TRUE && FALSE",
        },
        "n", None
    ),
    (
        {  # AND
            "expr": "FALSE && FALSE",
        },
        "n", None
    ),
    (
        {  # literal OR
            "expr": "y || y",
        },
        "y", None
    ),
    (
        {  # literal OR
            "expr": "n || y",
        },
        "y", None
    ),
    (
        {  # literal OR
            "expr": "y || n",
        },
        "y", None
    ),
    (
        {  # literal OR
            "expr": "n || n",
        },
        "n", None
    ),
    (
        {  # OR
            "expr": "TRUE || TRUE",
        },
        "y", None
    ),
    (
        {  # OR
            "expr": "FALSE || TRUE",
        },
        "y", None
    ),
    (
        {  # OR
            "expr": "TRUE || FALSE",
        },
        "y", None
    ),
    (
        {  # OR
            "expr": "FALSE || FALSE",
        },
        "n", None
    ),
    (
        {  # NOT
            "expr": "!TRUE",
        },
        "n", None
    ),
    (
        {  # NOT
            "expr": "!FALSE",
        },
        "y", None
    ),
    (
        {  # Greater than
            "expr": "NUMBER_B > 126",
        },
        "y", None
    ),
    (
        {  # Greater than
            "expr": "NUMBER_B > 127",
        },
        "n", None
    ),
    (
        {  # Greater than eq
            "expr": "NUMBER_B >= 127",
        },
        "y", None
    ),
    (
        {  # Greater than eq
            "expr": "NUMBER_B >= 128",
        },
        "n", None
    ),
    (
        {  # Less than
            "expr": "NUMBER_B < 128",
        },
        "y", None
    ),
    (
        {  # Less than
            "expr": "NUMBER_B < 127",
        },
        "n", None
    ),
    (
        {  # Less than eq
            "expr": "NUMBER_B <= 127",
        },
        "y", None
    ),
    (
        {  # Less than eq
            "expr": "NUMBER_B <= 126",
        },
        "n", None
    ),
    (
        {  # Equal
            "expr": "NUMBER_B = 127",
        },
        "y", None
    ),
    (
        {  # Equal
            "expr": "NUMBER_B = 126",
        },
        "n", None
    ),
    (
        {  # Equal
            "expr": "NUMBER_B = 128",
        },
        "n", None
    ),
    (
        {  # Not equal
            "expr": "NUMBER_B != 127",
        },
        "n", None
    ),
    (
        {  # Not equal
            "expr": "NUMBER_B != 126",
        },
        "y", None
    ),
    (
        {  # Not equal
            "expr": "NUMBER_B != 128",
        },
        "y", None
    ),
    (
        {  # Comparison with numeric expression
            "expr": "NUMBER_B < (121 + NUMBER_A)"
        },
        "y", None
    ),
    (
        {  # Comparison with string
            "expr": '"abracadabra" = PREFIX+SUFFIX'
        },
        "y", None
    ),
    (
        {  # Precedence of || vs <
            "expr": "FALSE || NUMBER_B < 128"
        },
        "y", None
    ),
    (
        {  # Precedence of && vs <
            "expr": "TRUE && NUMBER_B < 128"
        },
        "y", None
    ),
    (
        {  # Precedence of && vs < vs +
            "expr": "TRUE && NUMBER_B < 121 + NUMBER_A"
        },
        "y", None
    ),
    (
        {  # Precedence of && vs || - && has higher precedence
            "expr": "FALSE && TRUE || TRUE"
        },
        "y", None
    ),
    (
        {  # Precedence of && vs || - && has higher precedence
            "expr": "TRUE || FALSE && FALSE"
        },
        "y", None
    ),
    (
        {  # Parenthesis to give || precedence
            "expr": "FALSE && (TRUE || TRUE)"
        },
        "n", None
    ),
    (
        {  # Parenthesis to give || precedence
            "expr": "(TRUE || FALSE) && FALSE"
        },
        "n", None
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

    val = expr.condexpr_value(c['default_cond'][0]['cond'])

    if error is not None:
        assert error in caplog.text
    else:
        assert val == result
        assert caplog.text == ""
