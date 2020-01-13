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

import logging

from config_system.data import get_config


logger = logging.getLogger(__name__)
logger.addHandler(logging.NullHandler())


# Expression tuple for 'y' and 'n'
YES = ('boolean', True)
NO = ('boolean', False)


def check_depends(depends, value):
    """Check if the config identified in value is a simple dependency
    listed in the depends expression.
    A simple expression consists of just && and || boolean operators.
    If the expression uses any other operator, return False.

    This is used by the menu_parse below to indent dependent configs.
    """
    if depends is None:
        return False
    assert type(depends) == tuple
    assert len(depends) in [2, 3]

    if depends[0] == 'and':
        return (check_depends(depends[1], value) or
                check_depends(depends[2], value))
    elif depends[0] == 'or':
        return (check_depends(depends[1], value) and
                check_depends(depends[2], value))
    elif depends[0] == 'identifier':
        return depends[1] == value
    return False


# Operators where the result is not boolean
ARITH_SET = {"+", "-"}


def _expr_value(e):
    """Evaluate the value of the input expression.
    This isn't expected to use conditional operators.
    """
    assert type(e) == tuple
    assert len(e) in [2, 3]
    if len(e) == 3:
        left = _expr_value(e[1])
        right = _expr_value(e[2])
        if type(left) != type(right):
            raise TypeError("'{}' operator is not valid with mixed types".format(e[0]))
        elif type(left) == bool:
            raise TypeError("'{}' operator is not valid on booleans".format(e[0]))
        elif e[0] == '+':
            return left + right
        elif e[0] == '-':
            if type(left) is str:
                raise TypeError("'-' operator is not valid on strings")
                return left
            return left - right
    elif e[0] in ['string', 'number', 'boolean']:
        return e[1]
    elif e[0] == 'identifier':
        return get_config(e[1])['value']

    raise Exception("Unexpected depend list: " + str(e))


def expr_value(e):
    try:
        result = _expr_value(e)
    except TypeError as err:
        logger.error("{} in expression '{}'".format(str(err), format_dependency_list(e)))
        result = ""

    return result


def _condexpr_value(e):
    """Evaluate the value of the input expression.
    """
    assert type(e) == tuple
    assert len(e) in [2, 3]

    if len(e) == 3:
        if e[0] in ARITH_SET:
            return _expr_value(e)

        left = _condexpr_value(e[1])
        right = _condexpr_value(e[2])
        if type(left) != type(right):
            # Boolean result expected
            return False
        elif e[0] == 'and':
            return left and right
        elif e[0] == 'or':
            return left or right
        elif e[0] == '=':
            return left == right
        elif e[0] == '!=':
            return left != right
        elif e[0] == '>':
            return left > right
        elif e[0] == '>=':
            return left >= right
        elif e[0] == '<':
            return left < right
        elif e[0] == '<=':
            return left <= right
    elif e[0] == 'not':
        return not _condexpr_value(e[1])
    elif e[0] in ['string', 'number', 'boolean']:
        return e[1]
    elif e[0] == 'identifier':
        return get_config(e[1])['value']

    raise Exception("Unexpected depend list: " + str(e))


def condexpr_value(e):
    assert e is not None
    try:
        result = _condexpr_value(e)
        if type(result) is not bool:
            logger.error("Conditional expression '{}' does not return a boolean '{}'".format(
                format_dependency_list(e), str(result)))
            result = False
    except TypeError as err:
        logger.error("{} in expression '{}'".format(str(err), format_dependency_list(e)))
        result = False

    return result


def expr_type(e):
    """Return the Mconfig type string for the input expression.
    This isn't expected to use conditional operators.
    """
    assert type(e) == tuple
    assert len(e) in [2, 3]
    if len(e) == 3:
        left = expr_type(e[1])
        right = expr_type(e[2])
        if left != right:
            raise TypeError("'{}' operator is not valid with mixed types".format(e[0]))
        elif left == 'bool':
            raise TypeError("'{}' operator is not valid on booleans".format(e[0]))
        elif e[0] == '+':
            return left
        elif e[0] == '-':
            if left == 'string':
                raise TypeError("'-' operator is not valid on strings")
            return left
    elif e[0] == 'string':
        return 'string'
    elif e[0] == 'number':
        return 'int'
    elif e[0] == 'boolean':
        return 'bool'
    elif e[0] == 'identifier':
        return get_config(e[1])['datatype']

    raise Exception("Unexpected depend list: " + str(e))


def dependency_list(e):
    """
    Get the set of config identifiers referred to by an expression. A
    set is returned instead of a list as we don't need duplicates, and
    order doesn't matter.
    """
    if e is None:
        return set()
    assert type(e) == tuple

    if e[0] in ['and', 'or', '=', '!=', '<', '<=', '>', '>=', '+', '-']:
        return dependency_list(e[1]) | dependency_list(e[2])
    elif e[0] == 'not':
        return dependency_list(e[1])
    elif e[0] in ['string', 'number', 'boolean']:
        # Quoted string, number or boolean
        return set()
    elif e[0] == 'identifier':
        return {e[1]}
    raise Exception("Unexpected depend list: " + str(e))


OPERATOR_FORMAT_MAP = {
    "and": "&&",
    "or": "||",
}


def format_dependency_list(depends, skip_parens=False):
    assert depends, "Empty dependency list"
    assert type(depends) == tuple

    if len(depends) == 3:
        left = format_dependency_list(depends[1])
        right = format_dependency_list(depends[2])

        operator = OPERATOR_FORMAT_MAP.get(depends[0], depends[0])
        result = left + " " + operator + " " + right
        return result if skip_parens else "(" + result + ")"
    elif depends[0] == 'not':
        return "!" + format_dependency_list(depends[1])
    elif depends[0] == 'string':
        return '"' + depends[1] + '"'
    elif depends[0] == 'number':
        return str(depends[1])
    elif depends[0] == 'boolean':
        return 'y' if depends[1] else 'n'
    elif depends[0] == 'identifier':
        config = get_config(depends[1])
        value = config['value']
        if config['datatype'] == 'bool':
            value = 'y' if value else 'n'
        return depends[1] + "[=" + str(value) + "]"
