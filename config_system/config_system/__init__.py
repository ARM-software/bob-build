import os
import sys

# The config system depends on the `ply` parser generator. On Android, this may
# come as a prebuilt, but may _not_ automatically be added to PYTHONPATH. If
# we're on Android (tested by checking for `envsetup.mk`), then add the `ply`
# prebuilt to `sys.path`:
if os.path.isfile("build/make/core/envsetup.mk"):
    if os.path.isdir("external/ply/ply"):
        sys.path.insert(0, "external/ply/ply")


from .general import (
    can_enable,
    get_config_bool,
    get_config_int,
    get_config_string,
    get_options_depending_on,
    get_options_selecting,
    get_warning,
    init_config,
    read_config,
    set_config,
)  # nopep8: E402 module level import not at top of file

from .data import (
    get_config,
    get_config_list,
    get_mconfig_dir,
)  # nopep8: E402 module level import not at top of file

from .expr import (
    format_dependency_list,
)  # nopep8: E402 module level import not at top of file
