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

import contextlib
import logging
import os

try:
    from StringIO import StringIO
except ImportError:
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
