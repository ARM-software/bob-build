# Copyright 2018 Arm Limited.
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

import os

from . import lex

class LexWrapper:
    def __init__(self, ignore_missing):
        self.lexers = []
        self.root_dir = None
        self.ignore_missing = ignore_missing

    def source(self, fname):
        if self.root_dir is None:
            self.root_dir = os.path.dirname(fname)
        else:
            # All paths are relative to the dir containing the first Mconfig
            fname = os.path.join(self.root_dir, fname)

        if not os.path.exists(fname) and self.ignore_missing:
           return

        with open(fname, "rt") as fp:
            file_contents = fp.read()

        lexer = lex.create_mconfig_lexer(fname)

        self.push_lexer(lexer)
        self.input(file_contents)

    def current_lexer(self):
        return self.lexers[-1]

    def push_lexer(self, lexer):
        self.lexers.append(lexer)

    def pop_lexer(self):
        self.lexers = self.lexers[0:-1]

    def input(self, input):
        assert self.lexers

        self.current_lexer().input(input)

    def token(self):
        if not self.lexers:
            return None

        t = self.current_lexer().token()

        if t is None:
            self.pop_lexer()
            t = self.token()

        return t
