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

import logging
from copy import deepcopy
from logging.handlers import BufferingHandler


# Like MemoryHandler, but only flush on close
class InfBufferHandler(BufferingHandler):
    __slots__ = "target", "buffer"

    def __init__(self, capacity, target):
        super(BufferingHandler, self).__init__(capacity)
        self.setLevel(logging.NOTSET)
        self.target = target
        self.buffer = []

    def shouldFlush(self, record):
        return False

    def flush(self):
        while self.buffer:
            rec = self.buffer.pop(0)
            self.target.emit(rec)

    def close(self):
        self.flush()
        self.target = None
        self.buffer = []


# Count the number of each error type
class ErrorCounterHandler(logging.Handler):
    __slots__ = "counts"

    def __init__(self, *args, **kwargs):
        super(ErrorCounterHandler, self).__init__(*args, **kwargs)
        self.counts = {
            "NOTSETS": 0,
            "DEBUG": 0,
            "INFO": 0,
            "WARNING": 0,
            "ERROR": 0,
            "CRITICAL": 0,
        }

    def emit(self, record):
        lvl = record.levelname
        self.counts[lvl] += 1

    def debugs(self):
        return self.counts["DEBUG"]

    def infos(self):
        return self.counts["INFO"]

    def warnings(self):
        return self.counts["WARNING"]

    def errors(self):
        return self.counts["ERROR"]

    def criticals(self):
        return self.counts["CRITICAL"]


class ColorFormatter(logging.Formatter):
    """Formatter that provide colored messages for console logging when possible"""
    color_fmt = "\033[1;3{}m"
    reset = "\033[0m"

    def __init__(self, fmt, enabled=False):
        super(ColorFormatter, self).__init__(fmt)
        self.enabled = enabled
        self.colors = {
            "WARNING": 3,
            "CRITICAL": 1,
            "ERROR": 1,
            "DEBUG": 6,
            "INFO": 2
        }

    def format(self, log_msg):
        paint = self.colors.get(log_msg.levelname)
        if paint and self.enabled:
            log_msg = deepcopy(log_msg)
            log_msg.levelname = ColorFormatter.color_fmt.format(paint) + log_msg.levelname + ColorFormatter.reset
        return logging.Formatter.format(self, log_msg)
