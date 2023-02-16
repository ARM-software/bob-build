# Copyright 2018-2021, 2023 Arm Limited.
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
import os
import re

from config_system import data, expr, utils


logger = logging.getLogger(__name__)
logger.addHandler(logging.NullHandler())


def can_enable(config):
    # If there is no dependency expression, then the config can be enabled
    e = config.get("depends")
    if e is None:
        return True
    return expr.condexpr_value(e)


def is_visible(config):
    # If there is no visible_cond expression, then the config is visible
    e = config.get("visible_cond")
    if e is None:
        return True
    return expr.condexpr_value(e)


def enforce_dependent_values(stage, fix_bools=False, error_level=logging.WARNING):
    """
    Check that values are consistently set, specifically with respect
    to whether the dependencies are correct.

    This function is called when we expect the values to be
    consistent. i.e. after reading in a new config, or prior to
    writing it out. It is called prior to plugin execution, to try and
    ensure the plugins see consistent state.

    For non-boolean configs (string, int), set their value to default
    ("", 0) if the config's dependencies are not met. This can happen
    where a user sets the value of the config, and then changes
    another config resulting in the dependency being disabled.

    Boolean values are treated specially. Normally they will be kept
    in a consistent state (wrt dependencies) by set_config(). However
    they can be forced by a 'select' statement even when the
    dependencies are not met. This indicates there is a problem with
    the Mconfig that needs to be fixed. This function will only reset
    Boolean values to n if specifically requested by fix_bools=True.
    This is only expected to be used after reading in a config file.
    That file may have been written with a different Mconfig that
    allowed the selection.

    The error_level determines the log level at which bool
    inconsistencies are reported. When set to logging.ERROR this forces
    the user to fix the Mconfig.
    """
    for i in data.get_config_list():
        c = data.get_config(i)
        if can_enable(c):
            continue
        if c["datatype"] == "bool" and c["value"] is True:
            if len(c["selected_by"]) > 0:
                msg = (
                    "{0}unmet direct dependencies: {1} depends on {2}, "
                    "but is selected by [{3}].".format(
                        stage, i, c["depends"][1], ",".join(c["selected_by"])
                    )
                )
                if error_level is logging.ERROR:
                    msg += " Update the Mconfig so that this can't happen"
                logger.log(error_level, msg)
            else:
                raise Exception("Unmet direct dependencies without select")

            if fix_bools:
                set_config_internal(i, False)

        elif c["datatype"] == "string":
            set_config_internal(i, "")
        elif c["datatype"] == "int":
            set_config_internal(i, 0)


def check_type(key, config, actual):
    actual = expr.expr_type(actual)
    expected = config["datatype"]
    if expected != actual:
        logger.error(
            "Type mismatch in config %s: expected %s but got %s"
            % (key, expected, actual)
        )


def validate_configs():
    """
    Ensure that the types in 'default' statements are correct,
    and that the configs referred to in 'select' statements are boolean.
    """
    config_list = data.get_config_list()
    for k in config_list:
        c = data.get_config(k)
        if "default_cond" in c:
            for i in c["default_cond"]:
                check_type(k, c, i["expr"])
        if "default" in c:
            check_type(k, c, c["default"])
        if "select_if" in c:
            for k in c["select_if"]:
                try:
                    c1 = data.get_config(k[0])
                except KeyError:
                    logger.warning("Ignoring unknown configuration option %s" % k[0])
                    continue
                if c1["datatype"] != "bool":
                    logger.error(
                        "Select option must have type bool but got type %s instead"
                        % c1["datatype"]
                    )
        if "select" in c:
            for k in c["select"]:
                try:
                    c1 = data.get_config(k)
                except KeyError:
                    logger.warning("Ignoring unknown configuration option %s" % k)
                    continue
                if c1["datatype"] != "bool":
                    logger.error(
                        "Select option must have type bool but got type %s instead"
                        % c1["datatype"]
                    )


def init_config(options_filename, ignore_missing=False):
    global menu_data

    data.init(options_filename, ignore_missing)

    validate_configs()
    menu_data = menu_parse()
    set_initial_values()


def read_profile_file(profile_filename):
    try:
        with open(profile_filename, "rt") as f:
            source = os.path.basename(profile_filename)
            for line in f:
                line = line.strip()
                if line == "":
                    continue  # Ignore blank lines
                elif line.startswith("#"):
                    continue  # ignore comment
                else:
                    m = re.match(r"([^=]+)=(\"?)(.*)\2", line)
                    if m:
                        (key, quoted, value) = (m.group(1), m.group(2), m.group(3))
                        if quoted == '"':
                            value = re.sub(r"\\(.)", r"\1", value)
                        set_config_if_prompt(key, value, True, source=source)
                    else:
                        raise Exception("Couldn't parse ", line)
    except IOError as _:  # noqa: F841
        logger.warning('Failed to load non-existing "%s" profile' % profile_filename)


def read_config_file(config_filename):
    try:
        with open(config_filename, "rt") as f:
            for line in f:
                source = ""
                is_user_set = False
                line = line.strip()
                if line == "":
                    continue  # Ignore blank lines
                elif line.startswith("#"):
                    # match config name together with optional '[by user]' and '(source)' parts
                    # eg. "# CONFIG_RELEASE is not set [by user] (source)"
                    m = re.match(
                        r"# CONFIG_([^=]+) is not set( \[by user\](?: \((.+)\))?)?",
                        line,
                    )
                    if m:
                        key = m.group(1)
                        is_user_set = True if m.group(2) else False
                        source = m.group(3) if m.group(3) else ""
                        value = "n"
                        set_config_if_prompt(key, value, is_user_set, source)
                    # else ignore comment
                else:
                    line = line.split(
                        "#", 1
                    )  # Strip comment from line eg. "xyz # comment"
                    comment = line[1] if len(line) > 1 else ""
                    if comment:
                        # match the source if present eg. " set by user (source)"
                        m = re.match(r" set by user(?: \((.+)\))?", comment)
                        if m:
                            is_user_set = True
                            source = m.group(1) if m.group(1) else ""
                    line = line[0].rstrip()  # Remove extra space before comment
                    m = re.match(r"CONFIG_([^=]+)=(\"?)(.*)\2", line)
                    if m:
                        (key, quoted, value) = (m.group(1), m.group(2), m.group(3))
                        if quoted == '"':
                            value = re.sub(r"\\(.)", r"\1", value)
                        set_config_if_prompt(key, value, is_user_set, source)
                    else:
                        raise Exception("Couldn't parse ", line)
    except IOError as _:  # noqa: F841
        logger.warning(
            'Failed to load non-existing "%s" configuration' % config_filename
        )


def read_config(options_filename, config_filename, ignore_missing):
    init_config(options_filename, ignore_missing)
    read_config_file(config_filename)
    enforce_dependent_values("Inconsistent input, correcting: ", fix_bools=True)


def write_config(config_filename):
    with utils.open_and_write_if_changed(config_filename) as f:
        for i_type, i_symbol in data.iter_symbols_menuorder():
            if i_type in ["config", "menuconfig"]:
                c = data.get_config(i_symbol)
                mark_set_by_user = " # set by user"
                if not can_enable(c):
                    # Don't output this option because it cannot be enabled
                    continue
                elif c["datatype"] == "bool":
                    if c["value"] is True:
                        f.write("CONFIG_%s=y" % i_symbol)
                    else:
                        f.write("# CONFIG_%s is not set" % i_symbol)
                        mark_set_by_user = " [by user]"
                elif c["datatype"] == "string":
                    quoted_str = re.sub(r'(["\\])', r"\\\1", c["value"])
                    f.write('CONFIG_%s="%s"' % (i_symbol, quoted_str))
                else:
                    f.write("CONFIG_%s=%s" % (i_symbol, c["value"]))
                # Save meta data as user explicit mark this
                if c["is_user_set"]:
                    if "source" in c and c["source"]:
                        mark_set_by_user += " (" + c["source"] + ")"
                    f.write("%s" % mark_set_by_user)
                f.write("\n")
            elif i_type == "menu":
                f.write("\n#\n# %s\n#\n" % data.get_menu_title(i_symbol))
    logger.info("Written configuration to '%s'" % config_filename)


def write_depfile(depfile, target_name):
    with utils.open_and_write_if_changed(depfile) as fp:
        fp.write(target_name + ": \\\n    ")
        fp.write(" \\\n    ".join(data.get_mconfig_srcs()) + "\n")


def get_user_set_options():
    """
    Return all the options which have been set by the user
    """
    user_set_options = []
    for i_type, i_symbol in data.iter_symbols_menuorder():
        if i_type in ["config", "menuconfig"]:
            c = data.get_config(i_symbol)
            if c["is_user_set"]:
                value = c["value"]
                if c["datatype"] == "bool":
                    value = "y" if c["value"] else "n"
                source = c["source"] if "source" in c else ""
                user_set_options.append((i_symbol, value, source))
    return user_set_options


def get_options_selecting(selected):
    """Return the options which select `selected`"""
    opt = data.get_config(selected)
    return opt.get("selected_by", [])


def get_options_depending_on(dependent):
    """Return the options which depend on `dependent`"""
    opt = data.get_config(dependent)
    rdeps = opt.get("rdepends", [])
    enabled_options = []
    for rdep in rdeps:
        rdep_val = data.get_config(rdep)
        if rdep_val["datatype"] == "bool" and get_config_bool(rdep):
            enabled_options.append(rdep)
    return enabled_options


def get_warning(key):
    """
    Returns the warning associated with the given config option.
    Returns None if there isn't an associated warning.
    """
    opt = data.get_config(key)
    return opt.get("warning", None)


def set_initial_values():  # noqa: C901
    "Set all configuration objects to their default value"

    config_list = data.get_config_list()

    # Set up initial values, and set up reverse dependencies
    for k in config_list:
        c = data.get_config(k)
        c["selected_by"] = set()
        c["is_user_set"] = False
        c["is_new"] = True
        if c["datatype"] == "bool":
            c["value"] = False
        elif c["datatype"] == "int":
            c["value"] = 0
        else:
            c["value"] = ""

        for i in expr.dependency_list(c.get("depends")):
            data.get_config(i).setdefault("rdepends", []).append(k)

        if "default_cond" in c:
            for j in c["default_cond"]:
                for i in expr.dependency_list(j["cond"]):
                    data.get_config(i).setdefault("rdefault", []).append(k)
                value_deps = expr.dependency_list(j["expr"])
                for d in value_deps:
                    data.get_config(d).setdefault("rdefault", []).append(k)

        if "default" in c:
            value_deps = expr.dependency_list(c["default"])
            for d in value_deps:
                data.get_config(d).setdefault("rdefault", []).append(k)

        if "select_if" in c:
            for j in c["select_if"]:
                for i in expr.dependency_list(j[1]):
                    data.get_config(i).setdefault("rselect_if", []).append(k)

    # Check for dependency cycles
    for k in config_list:
        c = data.get_config(k)
        if k in c.get("rdepends", []):
            logger.error("%s depends on itself" % k)
            c["rdepends"].remove(k)
        if k in c.get("rdefault", []):
            logger.error("The default value of %s depends on itself" % k)
            c["rdefault"].remove(k)

    # Set values using defaults and taking into account dependencies
    for k in config_list:
        update_defaults(k)

    for k in config_list:
        update_bob_ignore(k)


def update_choice_default(c):
    """Update a choice group to select the best default"""

    choice_group = data.get_choice_group(c)

    def choice_rank(k, r):
        """Produces a ranking value for each choice entry, lower values are
        higher priority"""

        config = data.get_config(k)
        rank = 5

        if not can_enable(config):
            # Lowest rank - cannot be enabled
            return 6, config["position"]

        if len(config.get("selected_by")) > 0:
            rank = 1
        if config.get("is_user_set"):
            if k == r:
                rank = 2
        else:
            for i in config.get("default_cond", []):
                if (
                    expr.condexpr_value(i["cond"])
                    and expr.expr_value(i["expr"]) is True
                ):
                    rank = 3
                    break
            else:
                def_expr = config.get("default", expr.NO)
                if expr.expr_value(def_expr) is True:
                    rank = 4

        return rank, config["position"]

    selection = choice_group.get("selected")
    requested = choice_group.get("requested_value")
    s_rank = (100, 0)  # Higher than any other
    if selection is not None:
        s_rank = choice_rank(selection, requested)

    for k in choice_group["configs"]:
        if k == selection:
            continue
        k_rank = choice_rank(k, requested)

        if k_rank < s_rank:
            # A better choice
            selection = k
            s_rank = k_rank

    if selection != choice_group.get("selected"):
        set_config_internal(selection, True)


def update_bob_ignore(k):
    """Checks whether an option should be ignored by bob"""
    c = data.get_config(k)

    if "bob_ignore" not in c:
        c["bob_ignore"] = "n"

    if c["bob_ignore"] not in ["y", "n"]:
        logger.error("bob_ignore for %s needs to be boolean ('y' or 'n')" % k)


def update_defaults(k):
    """Set a configuration option to the correct default value if it hasn't
    been manually set"""
    c = data.get_config(k)

    if c.get("is_user_set"):
        # Has been manually set, so ignore
        return

    if "choice_group" in c:
        update_choice_default(c["choice_group"])
        return

    if "default_cond" in c:
        for i in c["default_cond"]:
            if expr.condexpr_value(i["cond"]):
                set_config_internal(k, expr.expr_value(i["expr"]))
                return

    # None of the conditioned defaults match
    if "default" in c:
        set_config_internal(k, expr.expr_value(c["default"]))
        return

    # No default - so set bools to 'n'
    if c["datatype"] == "bool":
        set_config_internal(k, False)
    elif c["datatype"] == "string":
        set_config_internal(k, "")
    elif c["datatype"] == "int":
        set_config_internal(k, 0)


def set_config_if_prompt(key, value, is_user_set=True, source="cmd_line"):
    """
    Used to set the option value from the command line, and through
    profile files. Only options that have prompts (indicating they are
    user-settable) can be set.
    """

    logger.debug("Trying to set %s %s" % (key, value))
    try:
        c = data.get_config(key)
    except KeyError:
        logger.warning("Ignoring unknown configuration option %s" % key)
        return
    c["is_new"] = False
    logger.debug(c)
    if "title" in c:
        logger.debug("Setting %s : %s " % (key, value))
        if c["datatype"] == "bool":
            value = True if value == "y" else False
        if is_user_set:
            c["source"] = source
        set_config(key, value, is_user_set)


def set_config_internal(key, value):
    # Internally most calls to set_config are not (directly) the result of the
    # user specifying the value, so this helper function avoids the need to
    # explicitly set is_user_set=False on every call
    set_config(key, value, is_user_set=False)


def set_config_selectifs(key):
    c = data.get_config(key)
    value = c["value"]
    if "select_if" in c:
        for k in c["select_if"]:
            if expr.condexpr_value(k[1]):
                force_config(k[0], value, key)
            else:
                # No longer forced, so turn off if previously forced
                force_config(k[0], False, key)


def set_config(key, value, is_user_set=True):  # noqa: C901
    try:
        c = data.get_config(key)
    except KeyError:
        logger.warning("Ignoring unknown configuration option %s" % key)
        return

    # Validate input
    if c["datatype"] == "bool" and value not in [True, False]:
        # This interface should always receive True/False.
        logger.warning(
            "Ignoring boolean configuration option %s with non-boolean value '%s'."
            % (key, value)
        )
        return
    elif c["datatype"] == "int":
        # We get a string from menus and file reading. Always convert
        # to an integer. Output a warning if we can't convert to an integer
        try:
            value = int(value)
        except ValueError:
            logger.warning(
                "Ignoring integer configuration option %s with non-integer value '%s'"
                % (key, value)
            )
            return

    # Record user specified value even if it is (currently) impossible
    c["is_user_set"] |= is_user_set
    if is_user_set:
        if "choice_group" in c:
            group = c["choice_group"]
            if value is True:
                # Record the selection for this group
                data.get_choice_group(group)["requested_value"] = key
        else:
            c["requested_value"] = value

    if c["datatype"] == "bool":
        if value is False and len(c["selected_by"]) > 0:
            # Option is forced, so cannot be turned off
            return
        if value is True and not can_enable(c):
            # Option is unavailable, so cannot be turned on. However if the
            # option is selected by another we force it on regardless
            if len(c["selected_by"]) == 0:
                return

    c["value"] = value
    if is_user_set:
        c["is_new"] = False

    if "choice_group" in c:
        group = c["choice_group"]
        cg = data.get_choice_group(group)

        if value is True:
            # Record the selection for this group
            cg["selected"] = key
            # Member of a choice group - unset all other items
            for k in cg["configs"]:
                if k != key:
                    set_config(k, False, is_user_set=is_user_set)
                    if data.get_config(k)["value"] is True:
                        # Failed to turn the other option off, so set this to n
                        c["value"] = False
                        return
        else:
            # Check if this is the last entry in a choice being unset.
            # If there is no other option set then either this entry will be
            # set back to 'y' - or if this entry cannot be set, the best default
            # entry in the choice will be picked
            for k in cg["configs"]:
                if k != key and data.get_config(k)["value"] is True:
                    break
            else:
                if can_enable(c):
                    cg["selected"] = key
                    c["value"] = True
                else:
                    # Reset current selection.
                    cg["selected"] = None
                    update_choice_default(group)

    if "select" in c:
        for k in c["select"]:
            force_config(k, value, key)

    set_config_selectifs(key)

    if "rdepends" in c:
        # Check any reverse dependencies to see if they need updating
        for k in c["rdepends"]:
            c2 = data.get_config(k)
            if c2["value"] is True and not can_enable(c2):
                set_config_internal(k, False)
            elif not c2["is_user_set"]:
                update_defaults(k)
            elif "choice_group" in c2:
                update_choice_default(c2["choice_group"])
            elif "requested_value" in c2 and c2["value"] != c2["requested_value"]:
                set_config(k, c2["requested_value"])

    if "rdefault" in c:
        # Check whether any default values need updating
        for k in c["rdefault"]:
            update_defaults(k)

    if "rselect_if" in c:
        # Update any select_ifs that might now be valid
        for k in c["rselect_if"]:
            set_config_selectifs(k)


# For 'select' options
def force_config(key, value, source):
    try:
        c = data.get_config(key)
    except KeyError:
        # validate_configs must have already output the warning that the
        # unknown configuration option was ignored, so we just return here
        return

    assert (
        type(value) == bool
    ), "force_config value argument must be boolean, got %s" % str(value)

    if value is True:
        if source in c["selected_by"]:
            return
        c["selected_by"].add(source)
    elif source in c["selected_by"]:
        c["selected_by"].remove(source)
    else:
        # Option wasn't previously forced, so don't change it
        return

    if len(c["selected_by"]) > 0:
        set_config_internal(key, True)
    elif "requested_value" in c:
        set_config(key, c["requested_value"])
    else:
        update_defaults(key)


def get_config_bool(key):
    c = data.get_config(key)
    assert c["datatype"] == "bool", "Config option %s is not a bool (has type %s)" % (
        key,
        c["datatype"],
    )
    if c["value"] is True:
        return True
    else:
        return False


def get_config_int(key):
    c = data.get_config(key)
    assert c["datatype"] == "int", "Config option %s is not an int (has type %s)" % (
        key,
        c["datatype"],
    )
    return int(c["value"])


def get_config_string(key):
    c = data.get_config(key)
    assert (
        c["datatype"] == "string"
    ), "Config option %s is not a string (has type %s)" % (key, c["datatype"])
    return c["value"]


class Menu(object):
    def __init__(self, menu_number):
        self.items = menu_data[menu_number]
        self.title_bar = data.get_title_bar()
        self.selection = 0
        self.top = 0
        self.title = data.get_menu_title(menu_number)

        if len(self.items) == 0:
            # empty menu
            self.items = [MenuItem("empty", None)]
        elif data.is_choice_group(menu_number):
            # Find currently selected entry
            while self.items[self.selection].get_value() is not True:
                if self.selection < len(self.items) - 1:
                    self.selection += 1
                else:
                    break

        while not self.items[self.selection].can_enable():
            if self.selection < len(self.items) - 1:
                self.selection += 1
            else:
                break

    def __getitem__(self, key):
        return self.items[key]

    def get_selection(self):
        return self[self.selection]


def display_value(value, datatype):
    if datatype == "int":
        return str(value)
    if datatype != "bool":
        return value
    if value is True:
        return "*"
    return " "


def get_default_style():
    return "window"  # default style


class StyledText(object):
    def __init__(self, text):
        self.style = get_default_style()
        self.text = text


class MenuItem(object):
    def __init__(self, type, value):
        self.type = type
        self.value = value

    # Return a list of StyledText objects
    def get_styled_text(self, is_selected, max_width):
        text_parts = []
        if self.type == "config":
            config = data.get_config(self.value)

            indent = config.get("depends_indent") or 0
            # Display "(new)" next to menu options that have no previously selected value
            new_text = " (new)" if config.get("is_new") else ""

            show_value = display_value(config["value"], config["datatype"])
            if len(config["selected_by"]) > 0:
                text_parts.append(StyledText("-"))
                text_parts.append(StyledText("%s" % show_value))
                text_parts.append(StyledText("-"))
            elif "choice_group" in config or config["datatype"] != "bool":
                text_parts.append(StyledText("("))
                trim_to = max_width - len(config["title"]) - indent - len(new_text) - 3
                trim_to = max(trim_to, 8)  # we want to display something
                if trim_to >= len(show_value):
                    text_parts.append(StyledText("%s" % show_value))
                else:
                    text_parts.append(StyledText("%s..." % show_value[: (trim_to - 3)]))
                text_parts.append(StyledText(")"))
            else:
                text_parts.append(StyledText("["))
                text_parts.append(StyledText("%s" % show_value))
                text_parts.append(StyledText("]"))

            if config["is_user_set"]:
                text_parts[1].style = "option_set_by_user"

            text_parts.append(
                StyledText(" %s%s%s" % ("  " * (indent), config["title"], new_text))
            )
        elif self.type == "menu":
            text_parts.append(StyledText("   "))
            text_parts.append(StyledText(" %s --->" % data.get_menu_title(self.value)))
        elif self.type == "menuconfig":
            config = data.get_config(self.value)
            is_menu_enabled = ">"
            if config["value"] is False:
                # Submenu is empty
                is_menu_enabled = "-"

            text_parts.append(StyledText("["))
            show_value = display_value(config["value"], config["datatype"])
            trim_to = max_width - len(config["title"]) - 5
            trim_to = max(trim_to, 8)  # we want to display something
            if trim_to >= len(show_value):
                text_parts.append(StyledText("%s" % show_value))
            else:
                text_parts.append(StyledText("%s..." % show_value[: (trim_to - 5)]))
            text_parts.append(StyledText("]"))
            text_parts.append(
                StyledText(" %s ---%s" % (config["title"], is_menu_enabled))
            )

            if config["is_user_set"]:
                text_parts[1].style = "option_set_by_user"
        elif self.type == "choice":
            choice = data.get_choice_group(self.value)
            current_value = ""
            for i in choice["configs"]:
                if data.get_config(i)["value"] is True:
                    current_value = data.get_config(i).get("title")
                    break
            text_parts.append(StyledText("   "))
            text_parts.append(StyledText(" %s (%s)" % (choice["title"], current_value)))
        elif self.type == "empty":
            text_parts.append(StyledText("***"))
            text_parts.append(StyledText(" Empty Menu ***"))
        else:
            raise Exception("Unknown type (%s)" % self.type)

        text_parts[-1].style = "highlight" if is_selected else get_default_style()
        return text_parts

    def get_menu(self):
        if self.type in ["menu", "menuconfig", "choice"]:
            return Menu(self.value)
        else:
            raise Exception("Not a menu!")

    def is_menu(self):
        if self.type == "menuconfig" and data.get_config(self.value)["value"] is False:
            # Prevent entry to a menuconfig that is disabled
            return False
        return self.type in ["menu", "menuconfig", "choice"]

    def needs_inputbox(self):
        if self.type == "config":
            config = data.get_config(self.value)
            return config["datatype"] in ["string", "int"]
        return False

    def __clear_config_source(self, key):
        """
        Clear 'source' property of a config and all its
        counterparts if it's a member of a choice group
        """
        config = data.get_config(key)
        if "choice_group" in config:
            group = config["choice_group"]
            cg = data.get_choice_group(group)
            for k in cg["configs"]:
                c2 = data.get_config(k)
                c2.pop("source", None)
        else:
            config.pop("source", None)

    def set(self):
        """Sets a boolean option to true"""
        if self.type in ["config", "menuconfig"]:
            config = data.get_config(self.value)
            if len(config["selected_by"]) > 0:
                # menuconfig shouldn't mark set by user if option can't be changed
                return

            if config["datatype"] == "bool":
                set_config(self.value, True)

    def clear(self):
        """Sets a boolean option to false"""
        if self.type in ["config", "menuconfig"]:
            config = data.get_config(self.value)
            if len(config["selected_by"]) > 0:
                # menuconfig shouldn't mark set by user if option can't be changed
                return

            if config["datatype"] == "bool":
                set_config(self.value, False)

    def toggle(self):
        if self.type in ["config", "menuconfig"]:
            config = data.get_config(self.value)
            if config["datatype"] != "bool":
                pass  # Ignore
            elif config["value"] is True:
                self.clear()
            else:
                self.set()

    def get_value(self):
        """Always return a string representation"""
        if self.type == "empty":
            return ""
        else:
            value = data.get_config(self.value)["value"]
            if data.get_config(self.value)["datatype"] == "int":
                value = str(value)
            return value

    def set_value(self, new_value):
        """
        set_value operates on any kind of option. new_value will always be a
        string, and set_config will do the appropriate conversions.
        """
        set_config(self.value, new_value)

    def can_enable(self):
        if self.type == "config":
            config = data.get_config(self.value)
            if "choice_group" in config:
                # Check if any member of the group is forced by a select
                group = config["choice_group"]
                for k in data.get_choice_group(group)["configs"]:
                    if k != self.value and len(data.get_config(k)["selected_by"]) > 0:
                        return False
        elif self.type == "empty":
            return False
        return can_enable(data.get_menu_configitem(self.type, self.value))

    def is_visible(self):
        if self.type == "empty":
            return False
        return is_visible(data.get_menu_configitem(self.type, self.value))

    def select(self):
        # Choice menus can use select
        if self.type in ["config", "menuconfig"] and "choice_group" in data.get_config(
            self.value
        ):
            self.set()
            return True
        return False

    def get_help(self):
        if self.type in ["config", "menuconfig"]:
            config = data.get_config(self.value)
            text = self.value + ": " + config["title"] + "\n\n"
            if "help" in config:
                text += config["help"]
            else:
                text += "No help available"
            return text
        elif self.type in ["choice"]:
            choice = data.get_choice_group(self.value)
            text = choice["title"] + "\n\n"
            if "help" in choice:
                text += choice["help"]
            else:
                text += "No help available"
            return text
        elif self.type in ["menu"]:
            config = data.get_menu(self.value)
            if "help" in config:
                return config["help"]
        return "No help available"

    def get_title(self):
        return data.get_config(self.value).get("title")

    def reset(self):
        if self.type not in ["config", "menuconfig"]:
            return

        config = data.get_config(self.value)
        if not config["is_user_set"]:
            logger.info(
                "Option '%s' = %s is default value" % (self.value, self.get_value())
            )
            return

        logger.info(
            "Reset option '%s' = %s to default value" % (self.value, self.get_value())
        )
        if "choice_group" in config:
            group = config["choice_group"]
            cg = data.get_choice_group(group)
            for k in cg["configs"]:
                choice_config = data.get_config(k)
                # We need to set it to False because update_defaults will ignore otherwise
                choice_config["is_user_set"] = False
            cg.pop("requested_value", None)  # pop if available
        else:
            config.pop("requested_value", None)  # pop if available

        # We need to set it to False because update_defaults will ignore if user set
        config["is_user_set"] = False

        # Clear 'source' for config
        self.__clear_config_source(self.value)

        update_defaults(self.value)
        logger.info("After reset: %s" % self.get_value())


def get_root_menu():
    return Menu(None)


def menu_parse():
    menus = {None: []}
    for i in data.get_menu_list():
        menus[i] = []

    menuconfig_stack = []
    depends_stack = []

    for i_type, i_symbol in data.iter_symbols_menuorder():
        if i_type == "config":
            config = data.get_config(i_symbol)
            inmenu = config.get("inmenu")

            if "title" not in config:
                # No title, so we don't display it
                continue

            while len(depends_stack) > 0:
                if expr.check_depends(config.get("depends"), depends_stack[-1]):
                    break
                depends_stack.pop()

            while len(menuconfig_stack) > 0:
                if expr.check_depends(config.get("depends"), menuconfig_stack[-1]):
                    inmenu = menuconfig_stack[-1]
                    break
                menuconfig_stack.pop()
                depends_stack = []

            if "choice_group" in config:
                inmenu = config["choice_group"]

            config["depends_indent"] = len(depends_stack)

            menus[inmenu].append(MenuItem("config", i_symbol))

            depends_stack.append(i_symbol)
        elif i_type == "menuconfig":
            config = data.get_config(i_symbol)
            inmenu = config.get("inmenu")

            while len(menuconfig_stack) > 0:
                if expr.check_depends(config.get("depends"), menuconfig_stack[-1]):
                    inmenu = menuconfig_stack[-1]
                    break
                menuconfig_stack.pop()

            menuconfig_stack.append(i_symbol)
            menus[i_symbol] = []

            menus[inmenu].append(MenuItem("menuconfig", i_symbol))
        elif i_type == "menu":
            menu = data.get_menu(i_symbol)
            inmenu = menu.get("inmenu")

            menuconfig_stack = []

            menus[inmenu].append(MenuItem("menu", i_symbol))
        elif i_type == "choice":
            inmenu = data.get_choice_group(i_symbol).get("inmenu")

            menus[i_symbol] = []

            menus[inmenu].append(MenuItem("choice", i_symbol))
        else:
            raise Exception(
                "Unexpected menu item: type {}, symbol {}".format(i_type, i_symbol)
            )
    return menus
