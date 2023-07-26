import contextlib
import logging
import os
from io import StringIO


logger = logging.getLogger(__name__)


@contextlib.contextmanager
def open_and_write_if_changed(fname):
    """Return a file-like object which buffers whatever is written to it. When
    closing, compare the new content with the existing file, and avoid writing
    to the file if no changes were made. This prevents the mtime being updated
    unnecessarily.
    """
    buf = StringIO()
    yield buf

    same_content = False
    try:
        if os.path.isfile(fname):
            with open(fname, "rt") as fp:
                original_content = fp.read()
                same_content = buf.getvalue() == original_content
    finally:
        if not same_content:
            logger.debug("Updating {}".format(fname))
            with open(fname, "wt") as fp:
                fp.write(buf.getvalue())
