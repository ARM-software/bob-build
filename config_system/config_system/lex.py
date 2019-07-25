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

from __future__ import print_function

import ply.lex as lex


verbose_flag = False


class TokenizeError(Exception):
    pass


tokens = (
    "ANDAND", "OROR", "NOT",
    "BOOL",
    "CHOICE", "ENDCHOICE",
    "CONFIG",
    "DEFAULT",
    "DEPENDS",
    "DUMMY",
    "EOL",
    "EQUAL", "UNEQUAL", "LESS", "LESS_EQUAL", "GREATER", "GREATER_EQUAL",
    "HELP", "HELPTEXT",
    "IF", "ON",
    "INT",
    "LBRACKET", "RBRACKET",
    "MENU", "ENDMENU", "MAINMENU",
    "MENUCONFIG",
    "NUMBER",
    "PROMPT",
    "QUOTED_STRING",
    "SELECT",
    "SOURCE",
    "STRING",
    "VISIBLE",
    "WORD",
    "COMMENT",
)

states = (
    ("PARAM", "exclusive"),
    ("HELP", "exclusive"),
)

commands = (
    "bool",
    "choice",
    "config",
    "default",
    "depends",
    "endchoice",
    "endmenu",
    "int",
    "menu",
    "mainmenu",
    "menuconfig",
    "prompt",
    "select",
    "source",
    "string",
    "visible",
)

params = ("if", "on")


help_indent = 0


def t_newline(t):
    r"\n+"
    t.lexer.lineno += len(t.value)
    t.type = "EOL"
    return t if verbose_flag else None


def t_ANY_comment(t):
    r"[ \t]*\#.*\n"
    t.lexer.lineno += 1

    # Backtrack on newlines so that an EOL token is generated.
    t.lexer.lexpos -= 1

    t.type = "COMMENT"
    return t if verbose_flag else None


def t_space(t):
    r"[\t ]+"
    return None


def t_commandhelp(t):
    r"help[ \t]*\n"
    t.lexer.begin("HELP")
    t.type = "HELP"
    global help_indent
    help_indent = 0

    t.lexer.lineno += 1
    return t


def t_command(t):
    r"[A-Za-z0-9_-]+"
    t.lexer.begin("PARAM")
    if t.value in commands:
        t.type = t.value.upper()
    else:
        report_error("Unknown identifier %s" % t.value, t)
    return t


t_PARAM_ANDAND = r"&&"
t_PARAM_OROR = r"\|\|"
t_PARAM_NOT = r"!"
t_PARAM_LBRACKET = r"\("
t_PARAM_RBRACKET = r"\)"
t_PARAM_EQUAL = r"="
t_PARAM_UNEQUAL = r"!="
t_PARAM_LESS = r"<"
t_PARAM_LESS_EQUAL = r"<="
t_PARAM_GREATER = r">"
t_PARAM_GREATER_EQUAL = r">="


def t_PARAM_space(t):
    r"[ \t]+"
    return None


def t_PARAM_word(t):
    r"[A-Za-z][A-Za-z0-9_-]*"
    if t.value in params:
        t.type = t.value.upper()
    else:
        t.type = "WORD"
    return t


def t_PARAM_number(t):
    r"[0-9]+"
    t.type = "NUMBER"
    t.value = int(t.value)
    return t


def t_PARAM_string(t):
    r'"[^"]*"'
    t.value = t.value[1:-1]
    t.type = "QUOTED_STRING"
    return t


def t_PARAM_newline(t):
    r"\n"
    t.lexer.begin("INITIAL")
    t.type = "EOL"

    t.lexer.lineno += 1
    return t


def t_HELP_text(t):
    r"(?P<indent>[ \t]+)(?P<text>.+)\n"
    global help_indent

    m = t.lexer.lexmatch
    indent = len(m.group("indent").expandtabs())
    text = m.group("text")

    if help_indent == 0:
        help_indent = indent
    elif indent < help_indent:
        report_error("Unexpected indent in help text", t)
    indent -= help_indent
    t.type = "HELPTEXT"
    t.value = text
    t.lexer.lineno += 1
    return t


def t_HELP_blankline(t):
    r"[ \t]*\n"
    t.value = ""
    t.type = "HELPTEXT"
    t.lexer.lineno += 1
    return t


def t_HELP_end(t):
    r"[^ \t]"
    t.lexer.lexpos -= 1  # Push the character back
    t.lexer.begin("INITIAL")


def t_ANY_error(t):
    report_error("Illegal character '%s'" % t.value[0], t)


def report_error(msg, t, err_type=TokenizeError):
    if t is None:
        print("%s at end of file" % msg)
        raise err_type()
    lexer = t.lexer

    from .lex_wrapper import LexWrapper
    if isinstance(lexer, LexWrapper):
        lexer = lexer.current_lexer()
    print("%s:%d: %s" % (lexer.fname, lexer.lineno, msg))
    last_cr = lexer.lexdata.rfind("\n", 0, t.lexpos) + 1
    next_cr = lexer.lexdata.find("\n", t.lexpos)
    if last_cr < 0:
        last_cr = 0
    column = len(lexer.lexdata[last_cr:t.lexpos].expandtabs())
    print(lexer.lexdata[last_cr: next_cr])
    print((" " * (column)) + "^")
    raise err_type()


def create_mconfig_lexer(fname, verbose=False):
    lexer = lex.lex()
    global verbose_flag
    verbose_flag = verbose
    lexer.lineno = 1
    lexer.fname = fname
    return lexer
