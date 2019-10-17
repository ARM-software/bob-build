# Copyright 2018-2019 Arm Limited.
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

import os.path
import ply.yacc as yacc

from config_system import expr
from config_system.lex import tokens, report_error


class ParseError(Exception):
    pass


def merge(a, b):
    c = a.copy()
    for k in b:
        if k in c:
            if isinstance(c[k], list):
                assert(isinstance(b[k], list))
                c[k] += b[k]
            elif isinstance(c[k], dict):
                c[k] = merge(c[k], b[k])
            else:
                raise Exception("Duplicate definition for " +
                                k+" when merging %r and %r" % (a, b))
        else:
            c[k] = b[k]
    return c


def p_input(p):
    "input : input stmt"
    p[0] = merge(p[1], p[2])


def p_null_input(p):
    "input :"
    p[0] = {"menu": {}, "order": {}, "config": {}, "choice": {}}


def p_stmt(p):
    """stmt : menuconfig_stmt
            | menu_stmt
            | config_stmt
            | choice_stmt
            | source_stmt
            | source_local_stmt
            | mainmenu_stmt
    """
    p[0] = p[1]


order_count = 0


def p_menuconfig_stmt(p):
    "menuconfig_stmt : MENUCONFIG IDENTIFIER EOL config_options"

    global order_count
    order_count += 1

    title = p[4].get("title")

    p[0] = {"config": {p[2]: merge(p[4], {"type": "menuconfig",
                                          "position": order_count})},
            "order": {order_count: ("menuconfig", p[2])},
            "menu": {p[2]: {"title": title}}}


def add_inmenu(data, i, depends):
    for k in data:
        if "inmenu" not in data[k]:
            data[k]["inmenu"] = i
        # Merge in any dependencies from the menu
        if depends is not None:
            if "depends" in data[k]:
                data[k]["depends"] = ("and", data[k]["depends"], depends)
            else:
                data[k]["depends"] = depends


def p_menu_stmt(p):
    "menu_stmt : menu_stmt_begin input ENDMENU EOL"

    menu_number = p[1]["id"]

    depends = p[1].get("depends")

    if "config" in p[2]:
        add_inmenu(p[2]["config"], menu_number, depends)
    if "menu" in p[2]:
        add_inmenu(p[2]["menu"], menu_number, depends)
    if "choice" in p[2]:
        add_inmenu(p[2]["choice"], menu_number, depends)

    menu_data = {"menu": {menu_number: p[1]},
                 "order": {menu_number: ("menu", menu_number)}}

    p[0] = merge(menu_data, p[2])


def p_menu_stmt_begin(p):
    "menu_stmt_begin : MENU QUOTED_STRING EOL menu_options"

    global order_count
    order_count += 1

    p[0] = merge(p[4], {"title": p[2], "id": order_count})


def p_menu_options(p):
    """menu_options : menu_options menu_visible
                    | menu_options config_depends
                    | menu_options config_help"""
    p[0] = merge(p[1], p[2])


def p_menu_options_empty(p):
    "menu_options :"
    p[0] = {}


def p_menu_visible(p):
    "menu_visible : VISIBLE IF condexpr EOL"
    p[0] = {"visible_cond": p[3]}


def p_config_stmt(p):
    "config_stmt : CONFIG IDENTIFIER EOL config_options"
    global order_count
    order_count += 1
    config_options = merge(p[4], {"type": "config", "position": order_count})
    p[0] = {"config": {p[2]: config_options},
            "order": {order_count: ("config", p[2])}}


def p_choice_stmt_begin(p):
    "choice_stmt_begin : CHOICE EOL choice_options"
    global order_count
    order_count += 1
    p[0] = merge({"id": order_count}, p[3])


def p_choice_stmt(p):
    "choice_stmt : choice_stmt_begin config_stmts ENDCHOICE EOL"

    config = p[1]
    choice_id = config["id"]
    p[0] = {"choice": {choice_id: config}, "config": {},
            "order": {choice_id: ("choice", choice_id)}}
    p[0]["choice"][choice_id]["configs"] = []
    for k in p[2]["config"]:
        p[0]["config"][k] = {"choice_group": choice_id}
        p[0]["choice"][choice_id]["configs"].append(k)

    p[0] = merge(p[0], p[2])

    # Merge in the dependency list from the "choice" block to each configuration
    # option underneath
    if "depends" in config:
        for k in p[0]["config"]:
            if "depends" in p[0]["config"][k]:
                old_depends = p[0]["config"][k]["depends"]
                p[0]["config"][k]["depends"] = ("and", old_depends,
                                                config["depends"])
            else:
                p[0]["config"][k]["depends"] = config["depends"]

    for d in config.get("default_cond", []):
        assert d["expr"][0] == "identifier", "Expressions not supported in choice default"
        for k in p[0]["config"]:
            if k == d["expr"][1]:
                p[0]["config"][k] = merge(p[0]["config"][k],
                                          {"default_cond": [{"cond": d["cond"], "expr": expr.YES}]})

    if "default" in config:
        assert config["default"][0] == "identifier", "Expressions not supported in choice default"
        for k in p[0]["config"]:
            if k == config["default"][1]:
                d = {"default": expr.YES}
                p[0]["config"][k] = merge(p[0]["config"][k], d)


def p_choice_options(p):
    """choice_options : choice_options config_type
                      | choice_options choice_default
                      | choice_options config_depends
                      | choice_options config_help
                      | choice_options config_prompt
                      """
    p[0] = merge(p[1], p[2])


def p_choice_options_empty(p):
    "choice_options :"
    p[0] = {}


def p_choice_default(p):
    """choice_default : DEFAULT identifier EOL
                      | DEFAULT identifier IF condexpr EOL
    """
    p[0] = {"default": p[2]}
    if len(p) > 4:
        p[0] = {"default_cond": [{"cond": p[4], "expr": p[2]}]}


def p_source_stmt_first(p):
    """source_stmt_first : SOURCE QUOTED_STRING dummy"""
    p.lexer.source(p[2])
    p[0] = {}


def p_source_local_stmt_first(p):
    """source_local_stmt_first : SOURCE_LOCAL QUOTED_STRING dummy"""
    fname = p.lexer.current_lexer().fname
    dname = os.path.dirname(fname)
    mname = os.path.join(dname, p[2])
    p.lexer.open(mname)
    p[0] = {}


def p_mainmenu_stmt_first(p):
    """mainmenu_stmt_first : MAINMENU QUOTED_STRING dummy"""
    p[0] = p[2]

# Force the parser to fetch the next token (even though the lexer will never
# actually return it). This stops it applying the "default reduction"
# optimization, which could cause it to delay lookahead until after the rule
# has been processed.


def p_dummy(p):
    """dummy :
             | DUMMY
             | COMMENT"""
    p[0] = {}


def p_source_stmt(p):
    "source_stmt : source_stmt_first EOL"
    p[0] = {}


def p_source_local_stmt(p):
    'source_local_stmt : source_local_stmt_first EOL'
    p[0] = {}


def p_mainmenu_stmt(p):
    "mainmenu_stmt : mainmenu_stmt_first EOL"
    p[0] = {"title_bar": p[1]}


def p_config_stmts(p):
    """config_stmts :
                    | config_stmts config_stmt"""
    if len(p) > 1:
        p[0] = merge(p[1], p[2])
    else:
        p[0] = {}


def p_type(p):
    """type : BOOL
            | INT
            | STRING"""
    p[0] = p[1]


def p_config_options(p):
    """config_options : config_options config_type
                      | config_options config_select
                      | config_options config_default
                      | config_options config_depends
                      | config_options config_help
                      | config_options config_prompt
                      """
    p[0] = merge(p[1], p[2])


def p_config_option_empty(p):
    "config_options :"
    p[0] = {}


def p_config_type(p):
    """config_type : type QUOTED_STRING EOL
                   | type EOL"""
    if len(p) == 4:
        p[0] = {"datatype": p[1], "title": p[2]}
    else:
        p[0] = {"datatype": p[1]}


def p_config_select(p):
    "config_select : SELECT IDENTIFIER EOL"
    p[0] = {"select": [p[2]]}


def p_config_select_if(p):
    "config_select : SELECT IDENTIFIER IF condexpr EOL"
    p[0] = {"select_if": [(p[2], p[4])]}


def p_config_default(p):
    """config_default : DEFAULT expr EOL
                      | DEFAULT expr IF condexpr EOL
    """
    p[0] = {"default": p[2]}
    if len(p) > 4:
        p[0] = {"default_cond": [{"cond": p[4], "expr": p[2]}]}


def p_config_depends(p):
    "config_depends : DEPENDS ON condexpr EOL"
    p[0] = {"depends": p[3]}


def p_config_prompt(p):
    """config_prompt : PROMPT QUOTED_STRING EOL
                     | PROMPT QUOTED_STRING IF condexpr EOL"""
    if len(p) > 4:
        p[0] = {"title": p[2], "visible_cond": p[4]}
    else:
        p[0] = {"title": p[2]}


def p_condexpr(p):
    """condexpr : condexpr2
                | condexpr OROR condexpr2"""
    if len(p) > 2:
        p[0] = ("or", p[1], p[3])
    else:
        p[0] = p[1]


def p_condexpr2(p):
    """condexpr2 : condexpr3
                 | condexpr2 ANDAND condexpr3"""
    if len(p) > 2:
        p[0] = ("and", p[1], p[3])
    else:
        p[0] = p[1]


def p_condexpr3(p):
    """condexpr3 : condexpr4
                 | condexpr4 comparison condexpr4"""
    if len(p) > 2:
        p[0] = (p[2], p[1], p[3])
    else:
        p[0] = p[1]


def p_comparison(p):
    """comparison : EQUAL
                  | UNEQUAL
                  | LESS
                  | LESS_EQUAL
                  | GREATER
                  | GREATER_EQUAL"""
    p[0] = p[1]


def p_condexpr4_not(p):
    "condexpr4 : NOT condexpr4"
    p[0] = ("not", p[2])


def p_condexpr4_brackets(p):
    "condexpr4 : LBRACKET condexpr RBRACKET"
    p[0] = p[2]


def p_condexpr4_expr(p):
    "condexpr4 : expr"
    p[0] = p[1]


def p_expr_op(p):
    "expr : expr combination expr2"
    p[0] = (p[2], p[1], p[3])


def p_expr_term(p):
    "expr : expr2"
    p[0] = p[1]


def p_expr2_brackets(p):
    "expr2 : LBRACKET expr RBRACKET"
    p[0] = p[2]


def p_expr2_term(p):
    "expr2 : literal_or_identifier"
    p[0] = p[1]


def p_combination(p):
    """combination : PLUS
                   | MINUS"""
    p[0] = p[1]


def p_lit_or_ident_string(p):
    "literal_or_identifier : QUOTED_STRING"
    p[0] = ("string", p[1])


def p_lit_or_ident_number(p):
    "literal_or_identifier : NUMBER"
    p[0] = ("number", p[1])


def p_lit_or_ident_boolean(p):
    """literal_or_identifier : YES
                             | NO"""
    value = False
    if p[1] == 'y':
        value = True
    p[0] = ("boolean", value)


def p_lit_or_ident_identifier(p):
    "literal_or_identifier : identifier"
    p[0] = p[1]


def p_identifier(p):
    "identifier : IDENTIFIER"
    p[0] = ("identifier", p[1])


def p_config_help(p):
    "config_help : HELP helptext"
    p[0] = {"help": p[2].strip()}


def p_helptext(p):
    """helptext :
                | helptext HELPTEXT"""
    if len(p) == 1:
        p[0] = ""
    else:
        p[0] = p[1] + "\n" + p[2]


def p_error(p):
    report_error("Parse error on token: {}".format(str(p)), p, ParseError)


parser = yacc.yacc(debug=False, write_tables=False)
