import os

from pathlib import Path

from config_system import lex


class LexWrapper:
    def __init__(self, ignore_missing, verbose=False):
        self.lexers = []
        self.sources = []
        self.root_dir = None
        self.ignore_missing = ignore_missing
        self.verbose = verbose

    def open(self, fname):
        """Open the named file."""
        if self.root_dir is None:
            self.root_dir = os.path.dirname(fname)

        if not os.path.exists(fname) and self.ignore_missing:
            return

        with open(fname, "rt") as fp:
            file_contents = fp.read()

        lexer = lex.create_mconfig_lexer(
            fname, verbose=self.verbose, root_dir=Path(self.root_dir)
        )

        self.push_lexer(lexer)
        self.input(file_contents)
        self.sources.append(fname)

    def source(self, fname):
        """Handle the source command, ensuring we open the file relative to
        the directory containing the first Mconfig."""
        if self.root_dir is not None:
            fname = os.path.join(self.root_dir, fname)

        self.open(fname)

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

        # Inject lexer for every "IDENTIFIER" token
        # to allow the parser read lexer's `relPath`
        if t is not None and t.type == "IDENTIFIER":
            t.lexer = self.current_lexer()

        return t

    def iterate_tokens(self):
        """Generator method to yield tokens"""
        while True:
            tok = self.current_lexer().token()
            if not tok:
                break
            yield tok
