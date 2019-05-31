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
import os
import re

logger = logging.getLogger(__name__)
logger.addHandler(logging.NullHandler())


def config_options_in_menu(data, menu):
    output = {}
    configs = data['config']
    for k in configs:
        if configs[k].get('inmenu') == menu:
            output[k] = configs[k]
    return output


def check_depends(depends, value):
    """Check if the config identified in value is a simple dependency
    listed in the depends expression.
    A simple expression consists of just && and || boolean operators.
    If the expression uses any other operator, return False.

    This is used by the menu_parse below to indent dependent configs.
    """
    if depends is None:
        return False
    if type(depends) == tuple:
        if depends[0] == 'and':
            return (check_depends(depends[1], value) or
                    check_depends(depends[2], value))
        elif depends[0] == 'or':
            return (check_depends(depends[1], value) and
                    check_depends(depends[2], value))
        else:
            return False
    return depends == value


def value_of(depends):
    if depends is None:
        return "y"
    if type(depends) == tuple:
        if len(depends) == 3:
            left = value_of(depends[1])
            right = value_of(depends[2])
            if type(left) != type(right):
                return 'n'
            elif depends[0] == 'and':
                if left == 'y' and right == 'y':
                    return 'y'
                return 'n'
            elif depends[0] == 'or':
                if left == 'y' or right == 'y':
                    return 'y'
                return 'n'
            elif depends[0] == '=':
                if left == right:
                    return 'y'
                return 'n'
            elif depends[0] == '!=':
                if left != right:
                    return 'y'
                return 'n'
            elif depends[0] == '>':
                if left > right:
                    return 'y'
                return 'n'
            elif depends[0] == '>=':
                if left >= right:
                    return 'y'
                return 'n'
            elif depends[0] == '<':
                if left < right:
                    return 'y'
                return 'n'
            elif depends[0] == '<=':
                if left <= right:
                    return 'y'
                return 'n'
        elif depends[0] == 'not':
            if value_of(depends[1]) == 'y':
                return 'n'
            return 'y'
        elif depends[0] == 'string':
            return depends[1]
        elif depends[0] == 'number':
            return depends[1]
        else:
            raise Exception("Unexpected depend list: " + str(depends))
    return configuration['config'][depends]['value']


def can_enable(depends):
    value = value_of(depends)
    if value == 'y':
        return True
    return False


def is_visible(cond):
    value = value_of(cond)
    if value == 'n':
        return False
    return True


def dependency_list(depends):
    if depends is None:
        return []
    if type(depends) == tuple:
        if depends[0] in ['and', 'or', '=', '!=', '<', '<=', '>', '>=']:
            return dependency_list(depends[1]) + dependency_list(depends[2])
        elif depends[0] == "not":
            return dependency_list(depends[1])
        elif depends[0] in ["string", "number"]:
            # Quoted string or number
            return []
        raise Exception("Unexpected depend list: " + str(depends))

    return [depends]


OPERATOR_FORMAT_MAP = {
    "and": "&&",
    "or": "||",
}


def format_dependency_list(depends, skip_parens=False):
    assert depends, "Empty dependency list"

    if type(depends) == tuple:
        if len(depends) == 3:
            left = format_dependency_list(depends[1])
            right = format_dependency_list(depends[2])

            operator = OPERATOR_FORMAT_MAP.get(depends[0], depends[0])
            expr = left + " " + operator + " " + right
            return expr if skip_parens else "(" + expr + ")"
        elif depends[0] == "not":
            return "!" + format_dependency_list(depends[1])
        elif depends[0] == 'string':
            return '"' + depends[1] + '"'
        elif depends[0] == 'number':
            return str(depends[1])
    elif type(depends) == str:
        return depends + "[=" + str(get_config(depends)["value"]) + "]"


def enforce_dependent_values(auto_fix=False):
    """
    Check that values are consistently set, specifically with respect
    to whether the dependencies are correct.

    For boolean values we only reset to n if auto_fix is
    True. Otherwise we raise an exception (the Mconfig needs to be
    fixed).

    For string and int values, if the configuration value is not
    enabled, reset them to "" and 0 respectively. This is different to
    the bool case as we need to reset default values.
    """
    for i in configuration['config']:
        c = get_config(i)
        if can_enable(c.get('depends')):
            continue
        if c['datatype'] == 'bool' and c['value'] == 'y':
            logger.warn("unmet direct dependencies: " +
                        "%s depends on %s" % (i, c['depends']))
            if auto_fix:
                set_config_internal(i, 'n')
            else:
                raise Exception("Unmet direct dependencies")
        elif c['datatype'] == 'string':
            set_config_internal(i, '')
        elif c['datatype'] == 'int':
            set_config_internal(i, 0)


__mconfig_dir = ""


def get_mconfig_dir():
    """
    Retrieve the path to the input option database.
    """
    return __mconfig_dir


def init_config(options_filename, ignore_missing=False):
    from config_system import lex, lex_wrapper, syntax

    global __mconfig_dir
    global configuration
    global menu_data
    try:
        lexer = lex_wrapper.LexWrapper(ignore_missing)
        lexer.source(options_filename)
        configuration = syntax.parser.parse(None, debug=False, lexer=lexer)
    except lex.TokenizeError as e:
        logger.debug("Failed to tokenise input")
        exit(1)
    except syntax.ParseError as e:
        logger.debug("Parse error")
        exit(1)
    __mconfig_dir = os.path.dirname(options_filename)
    menu_data = menu_parse(configuration)

    set_initial_values()


def read_profile_file(profile_filename):
    try:
        with open(profile_filename, "rt") as f:
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
                        set_config_if_prompt(key, value, True)
                    else:
                        raise Exception("Couldn't parse ", line)
    except IOError as e:
        logger.warn("Failed to load non-existing \"%s\" profile" % profile_filename)


def read_config_file(config_filename):
    try:
        with open(config_filename, "rt") as f:
            for line in f:
                line = line.strip()
                if line == "":
                    continue  # Ignore blank lines
                elif line.startswith("#"):
                    m = re.match(r"^#\s*CONFIG_([^=]+) is not set", line)
                    if m:
                        key = m.group(1)
                        value = "n"
                        set_config_if_prompt(key, value, ("[by user]" in line))
                    # else ignore comment
                else:
                    line = line.split("#", 1)  # Strip comment from line eg. "xyz # comment"
                    comment = line[1] if len(line) > 1 else ""
                    line = line[0].rstrip()  # Remove extra space before comment
                    m = re.match(r"CONFIG_([^=]+)=(\"?)(.*)\2", line)
                    if m:
                        (key, quoted, value) = (m.group(1), m.group(2), m.group(3))
                        if quoted == '"':
                            value = re.sub(r"\\(.)", r"\1", value)
                        set_config_if_prompt(key, value, ("by user" in comment))
                    else:
                        raise Exception("Couldn't parse ", line)
    except IOError as e:
        logger.warn("Failed to load non-existing \"%s\" configuration" % config_filename)


def read_config(options_filename, config_filename, ignore_missing):
    init_config(options_filename, ignore_missing)
    read_config_file(config_filename)
    enforce_dependent_values(True)


def write_config(config_filename):
    enforce_dependent_values()
    with open(config_filename, "wt") as f:
        for i in sorted(configuration['order']):
            (i_type, i_symbol) = configuration['order'][i]
            if i_type in ["config", "menuconfig"]:
                c = get_config(i_symbol)
                mark_set_by_user = " # set by user"
                if not can_enable(c.get('depends')):
                    # Don't output this option because it cannot be enabled
                    continue
                elif c['datatype'] == "bool":
                    if c['value'] == 'y':
                        f.write("CONFIG_%s=y" % i_symbol)
                    else:
                        f.write("# CONFIG_%s is not set" % i_symbol)
                        mark_set_by_user = " [by user]"
                elif c['datatype'] == "string":
                    quoted_str = re.sub(r'(["\\])', r'\\\1', c['value'])
                    f.write('CONFIG_%s="%s"' % (i_symbol, quoted_str))
                else:
                    f.write("CONFIG_%s=%s" % (i_symbol, c['value']))
                # Save meta data as user explicit mark this
                if c['is_user_set']:
                    f.write("%s" % mark_set_by_user)
                f.write("\n")
            elif i_type == "menu":
                f.write("\n#\n# %s\n#\n" %
                        configuration['menu'][i_symbol]['title'])
    logger.info("Written configuration to '%s'" % config_filename)


def get_options_selecting(selected):
    """Return the options which select `selected`"""
    opt = get_config(selected)
    return opt.get("selected_by", [])


def get_options_depending_on(dependent):
    """Return the options which depend on `dependent`"""
    opt = get_config(dependent)
    rdeps = opt.get("rdepends", [])
    enabled_options = []
    for rdep in rdeps:
        rdep_val = get_config(rdep)
        if rdep_val["datatype"] == "bool" and get_config_bool(rdep):
            enabled_options.append(rdep)
    return enabled_options


def set_initial_values():
    "Set all configuration objects to their default value"
    config = configuration['config']

    # Set up initial values, and set up reverse dependencies
    for k in config:
        config[k]['selected_by'] = set()
        config[k]['is_user_set'] = False
        config[k]['is_new'] = True
        if config[k]['datatype'] == "bool":
            config[k]['value'] = 'n'
        else:
            config[k]['value'] = ''

        for i in dependency_list(config[k].get("depends")):
            config[i].setdefault('rdepends', []).append(k)

        if "default_cond" in config[k]:
            for j in config[k]['default_cond']:
                for i in dependency_list(j['cond']):
                    config[i].setdefault('rdefault', []).append(k)
                if config[k]['datatype'] == "string":
                    if j['word'] is not None:
                        config[j['word']].setdefault('rdefault', []).append(k)

        if "default" in config[k]:
            if config[k]['datatype'] == "string":
                if config[k]['word'] is not None:
                    config[config[k]['word']].setdefault('rdefault', []).append(k)

        if "select_if" in config[k]:
            for j in config[k]['select_if']:
                for i in dependency_list(j[1]):
                    config[i].setdefault('rselect_if', []).append(k)

    # Check for dependency cycles
    for k in config:
        if k in config[k].get('rdepends', []):
            logger.error("%s depends on itself" % k)
            config[k]['rdepends'].remove(k)
        if k in config[k].get('rdefault', []):
            logger.error("The default value of %s depends on itself" % k)
            config[k]['rdefault'].remove(k)

    # Set values using defaults and taking into account dependencies
    for k in config:
        update_defaults(k)


def update_choice_default(c):
    """Update a choice group to select the best default"""

    choice_group = configuration['choice'][c]

    def choice_rank(k, r):
        """Produces a ranking value for each choice entry, lower values are
        higher priority"""

        config = get_config(k)
        rank = 5

        if not can_enable(config.get('depends')):
            # Lowest rank - cannot be enabled
            return 6, config['position']

        if len(config.get("selected_by")) > 0:
            rank = 1
        if config.get("is_user_set"):
            if k == r:
                rank = 2
        else:
            for i in config.get("default_cond", []):
                if can_enable(i['cond']) and i['val'] == 'y':
                    rank = 3
                    break
            else:
                if config.get("default", "n") == "y":
                    rank = 4

        return rank, config['position']

    selection = choice_group.get("selected")
    requested = choice_group.get("requested_value")
    s_rank = (100, 0)  # Higher than any other
    if selection is not None:
        s_rank = choice_rank(selection, requested)

    for k in choice_group['configs']:
        if k == selection:
            continue
        k_rank = choice_rank(k, requested)

        if k_rank < s_rank:
            # A better choice
            selection = k
            s_rank = k_rank

    if selection != choice_group.get("selected"):
        set_config_internal(selection, 'y')


def update_defaults(k):
    """Set a configuration option to the correct default value if it hasn't
    been manually set"""
    config = configuration['config']

    c = config[k]

    if c.get("is_user_set"):
        # Has been manually set, so ignore
        return

    if "choice_group" in c:
        update_choice_default(c['choice_group'])
        return

    if "default_cond" in c:
        for i in c['default_cond']:
            if can_enable(i['cond']):
                if c['datatype'] == "string" and i['word'] is not None:
                    found_config = config[i['word']]
                    set_config_internal(k, found_config['value'])
                else:
                    set_config_internal(k, i['val'])
                return

    # None of the conditioned defaults match
    if "default" in c:
        if c['datatype'] == "string" and c['word'] is not None:
            found_config = config[c['word']]
            set_config_internal(k, found_config['value'])
        else:
            set_config_internal(k, config[k]['default'])
        return

    # No default - so set bools to 'n'
    if c['datatype'] == "bool":
        set_config_internal(k, 'n')
    elif c['datatype'] == "string":
        set_config_internal(k, '')
    elif c['datatype'] == "int":
        set_config_internal(k, 0)


def set_config_if_prompt(key, value, is_user_set=True):
    config = configuration['config']
    logger.debug("Trying to set %s %s" % (key, value))
    if key not in config:
        logger.warn("Ignoring unknown configuration option %s" % key)
        return
    config[key]['is_new'] = False
    logger.debug(config[key])
    if 'title' in config[key]:
        logger.debug("Setting %s : %s " % (key, value))
        set_config(key, value, is_user_set)


def set_config_internal(key, value):
    # Internally most calls to set_config are not (directly) the result of the
    # user specifying the value, so this helper function avoids the need to
    # explictly set is_user_set=False on every call
    set_config(key, value, is_user_set=False)


def set_config_selectifs(key):
    config = configuration['config']
    value = config[key]['value']
    if "select_if" in config[key]:
        for k in config[key]["select_if"]:
            if can_enable(k[1]):
                force_config(k[0], value, key)
            else:
                # No longer forced, so turn off if previously forced
                force_config(k[0], 'n', key)


def set_config(key, value, is_user_set=True):
    config = configuration['config']
    if key not in config:
        logger.warn("Ignoring unknown configuration option %s" % key)
        return

    if is_user_set:
        # Validate user input
        if config[key]['datatype'] == 'bool':
            # Must be y or n
            if value not in ['y', 'n']:
                logger.warn("Ignoring boolean configuration option %s with non-boolean value '%s'. "
                            "Please use 'y' or 'n'" % (key, value))
                return
        elif config[key]['datatype'] == 'int':
            # Must convert to an integer
            try:
                value = int(value)
            except ValueError:
                logger.warn(
                    "Ignoring integer configuration option %s with non-integer value '%s'" % (
                        key, value))
                return

    # Record user specified value even if it is (currently) impossible
    config[key]['is_user_set'] |= is_user_set
    if is_user_set:
        if "choice_group" in config[key]:
            group = config[key]['choice_group']
            if value == 'y':
                # Record the selection for this group
                configuration['choice'][group]['requested_value'] = key
        else:
            config[key]['requested_value'] = value

    if config[key]['datatype'] == 'bool':
        if value == 'n' and len(config[key]['selected_by']) > 0:
            # Option is forced, so cannot be turned off
            return
        if value == 'y' and not can_enable(config[key].get('depends')):
            # Option is unavailable, so cannot be turned on. However if the
            # option is selected by another we force it on regardless
            if len(config[key]['selected_by']) == 0:
                return

    config[key]['value'] = value
    if is_user_set:
        config[key]['is_new'] = False

    if "choice_group" in config[key]:
        group = config[key]['choice_group']
        if value == 'y':
            # Record the selection for this group
            configuration['choice'][group]['selected'] = key
            # Member of a choice group - unset all other items
            for k in configuration['choice'][group]['configs']:
                if k != key:
                    set_config(k, 'n', is_user_set=is_user_set)
                    if get_config(k)['value'] == 'y':
                        # Failed to turn the other option off, so set this to n
                        config[key]['value'] = 'n'
                        return
        else:
            # Check if this is the last entry in a choice being unset.
            # If there is no other option set then either this entry will be
            # set back to 'y' - or if this entry cannot be set, the best default
            # entry in the choice will be picked
            for k in configuration['choice'][group]['configs']:
                if k != key and get_config(k)['value'] == 'y':
                    break
            else:
                if can_enable(config[key].get('depends')):
                    configuration['choice'][group]['selected'] = key
                    config[key]['value'] = 'y'
                else:
                    # Reset current selection.
                    configuration['choice'][group]['selected'] = None
                    update_choice_default(group)

    if "select" in config[key]:
        for k in config[key]["select"]:
            force_config(k, value, key)

    set_config_selectifs(key)

    if "rdepends" in config[key]:
        # Check any reverse dependencies to see if they need updating
        for k in config[key]['rdepends']:
            c = config[k]
            if c['value'] == 'y' and not can_enable(c.get('depends')):
                set_config_internal(k, 'n')
            elif not c['is_user_set']:
                update_defaults(k)
            elif "choice_group" in c:
                update_choice_default(c['choice_group'])
            elif 'requested_value' in c and c['value'] != c['requested_value']:
                set_config(k, c['requested_value'])

    if "rdefault" in config[key]:
        # Check whether any default values need updating
        for k in config[key]['rdefault']:
            update_defaults(k)

    if "rselect_if" in config[key]:
        # Update any select_ifs that might now be valid
        for k in config[key]['rselect_if']:
            set_config_selectifs(k)


# For 'select' options
def force_config(key, value, source):
    config = configuration['config']
    if key not in config:
        logger.warn("Ignoring unknown configuration option %s" % key)
        return

    if value == "y":
        if source in config[key]['selected_by']:
            return
        config[key]['selected_by'].add(source)
    elif source in config[key]['selected_by']:
        config[key]['selected_by'].remove(source)
    else:
        # Option wasn't previously forced, so don't change it
        return

    if len(config[key]['selected_by']) > 0:
        set_config_internal(key, 'y')
    elif 'requested_value' in config[key]:
        set_config(key, config[key]['requested_value'])
    else:
        update_defaults(key)


def get_config(key):
    return configuration['config'][key]


def get_config_bool(key):
    c = get_config(key)
    assert c['datatype'] == 'bool', \
        'Config option %s is not a bool (has type %s)' % (key, c['datatype'])
    if c['value'] == 'y':
        return True
    else:
        return False


def get_config_int(key):
    c = get_config(key)
    assert c['datatype'] == 'int', \
        'Config option %s is not an int (has type %s)' % (key, c['datatype'])
    return int(c['value'])


def get_config_string(key):
    c = get_config(key)
    assert c['datatype'] == 'string', \
        'Config option %s is not a string (has type %s)' % (key, c['datatype'])
    return c['value']


def get_menu_depends(type, value):
    if type in ["config", "menuconfig"]:
        return configuration['config'][value].get('depends')
    elif type in ["choice"]:
        return configuration['choice'][value].get('depends')
    elif type in ["menu"]:
        return configuration['menu'][value].get('depends')


def get_menu_visible(type, value):
    if type in ["config", "menuconfig"]:
        return configuration['config'][value].get('visible_cond')
    elif type in ["choice"]:
        return configuration['choice'][value].get('visible_cond')
    elif type in ["menu"]:
        return configuration['menu'][value].get('visible_cond')


def get_config_list():
    return configuration['config'].keys()


def get_menu_title(number):
    if number in configuration['menu']:
        if 'title' in configuration['menu'][number]:
            return configuration['menu'][number]['title']
    elif number in configuration['choice']:
        if 'title' in configuration['choice'][number]:
            return configuration['choice'][number]['title']
    return 'Configuration'


def get_title_bar():
    if 'title_bar' in configuration:
        return configuration['title_bar']
    return "Configuration System"


class Menu(object):
    def __init__(self, menu_number):
        self.items = menu_data[menu_number]
        self.title_bar = get_title_bar()
        self.selection = 0
        self.top = 0
        self.title = get_menu_title(menu_number)

        if len(self.items) == 0:
            # Empty menu
            self.items = [MenuItem("Empty Menu", None)]
        elif menu_number in configuration['choice']:
            # Find currently selected entry
            while self.items[self.selection].get_value() != 'y':
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
    if value == 'y':
        return '*'
    return ' '


def get_default_style():
    return 'window'  # default style


class StyledText(object):
    def __init__(self):
        self.style = get_default_style()
        self.text = ""

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
            config = get_config(self.value)

            indent = config.get("depends_indent") or 0
            # Display "(new)" next to menu options that have no previously selected value
            new_text = " (new)" if config.get('is_new') else ""

            show_value = display_value(config['value'], config['datatype'])
            if len(config['selected_by']) > 0:
                text_parts.append(StyledText("-"))
                text_parts.append(StyledText("%s" % show_value))
                text_parts.append(StyledText("-"))
            elif 'choice_group' in config or config['datatype'] != "bool":
                text_parts.append(StyledText("("))
                trim_to = max_width - len(config['title']) - indent - len(new_text) - 3
                trim_to = max(trim_to, 8)  # we want to display something
                if trim_to >= len(show_value):
                    text_parts.append(StyledText("%s" % show_value))
                else:
                    text_parts.append(StyledText("%s..." % show_value[:(trim_to - 3)]))
                text_parts.append(StyledText(")"))
            else:
                text_parts.append(StyledText("["))
                text_parts.append(StyledText("%s" % show_value))
                text_parts.append(StyledText("]"))

            if config['is_user_set']:
                text_parts[1].style = 'option_set_by_user'

            text_parts.append(StyledText(" %s%s%s" % ("  " * (indent), config['title'], new_text)))
        elif self.type == "menu":
            text_parts.append(StyledText("   "))
            text_parts.append(StyledText(" %s --->" % configuration['menu'][self.value]['title']))
        elif self.type == "menuconfig":
            config = get_config(self.value)
            is_menu_enabled = '>'
            if config['value'] == 'n':
                # Submenu is empty
                is_menu_enabled = '-'

            text_parts.append(StyledText("["))
            show_value = display_value(config['value'], config['datatype'])
            trim_to = max_width - len(config['title']) - 5
            trim_to = max(trim_to, 8)  # we want to display something
            if trim_to >= len(show_value):
                text_parts.append(StyledText("%s" % show_value))
            else:
                text_parts.append(StyledText("%s..." % show_value[:(trim_to - 5)]))
            text_parts.append(StyledText("]"))
            text_parts.append(StyledText(" %s ---%s" % (config['title'], is_menu_enabled)))

            if config['is_user_set']:
                text_parts[1].style = 'option_set_by_user'
        elif self.type == "choice":
            choice = configuration['choice'][self.value]
            current_value = ''
            for i in choice['configs']:
                if get_config(i)['value'] == 'y':
                    current_value = get_config(i).get('title')
                    break
            text_parts.append(StyledText("   "))
            text_parts.append(StyledText(" %s (%s)" % (choice['title'], current_value)))
        else:
            text_parts.append(StyledText("***"))
            text_parts.append(StyledText(" %s ***" % self.type))

        text_parts[-1].style = 'highlight' if is_selected else get_default_style()
        return text_parts

    def get_menu(self):
        if self.type in ["menu", "menuconfig", "choice"]:
            return Menu(self.value)
        else:
            raise Exception("Not a menu!")

    def is_menu(self):
        if self.type == "menuconfig" and get_config(self.value)['value'] == "n":
            # Prevent entry to a menuconfig that is disabled
            return False
        return self.type in ["menu", "menuconfig", "choice"]

    def needs_inputbox(self):
        if self.type == "config":
            config = get_config(self.value)
            return config['datatype'] in ["string", "int", "hex"]
        return False

    def set(self):
        """Sets a boolean option to true"""
        if self.type in ["config", "menuconfig"]:
            config = get_config(self.value)
            if len(config['selected_by']) > 0:
                # menuconfig shouldn't mark set by user if option can't be changed
                return

            if config['datatype'] == 'bool':
                set_config(self.value, 'y')

    def clear(self):
        """Sets a boolean option to false"""
        if self.type in ["config", "menuconfig"]:
            config = get_config(self.value)
            if len(config['selected_by']) > 0:
                # menuconfig shouldn't mark set by user if option can't be changed
                return

            if config['datatype'] == 'bool':
                set_config(self.value, 'n')

    def toggle(self):
        if self.type in ["config", "menuconfig"]:
            config = get_config(self.value)
            if config['datatype'] != 'bool':
                pass  # Ignore
            elif config['value'] == 'y':
                self.clear()
            else:
                self.set()

    def get_value(self):
        """Always return a string representation"""
        value = get_config(self.value)['value']
        if get_config(self.value)['datatype'] == 'int':
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
            config = get_config(self.value)
            if 'choice_group' in config:
                # Check if any member of the group is forced by a select
                group = config['choice_group']
                for k in configuration['choice'][group]['configs']:
                    if k != self.value and len(get_config(k)['selected_by']) > 0:
                        return False
        return can_enable(get_menu_depends(self.type, self.value))

    def is_visible(self):
        return is_visible(get_menu_visible(self.type, self.value))

    def select(self):
        # Choice menus can use select
        if (self.type in ["config", "menuconfig"] and
                "choice_group" in get_config(self.value)):
            self.set()
            return True
        return False

    def get_help(self):
        if self.type in ["config", "menuconfig"]:
            config = get_config(self.value)
            text = self.value + ": " + config['title'] + "\n\n"
            if "help" in config:
                text += config['help']
            else:
                text += "No help available"
            return text
        elif self.type in ["choice"]:
            choice = configuration['choice'][self.value]
            text = choice['title'] + "\n\n"
            if "help" in choice:
                text += choice['help']
            else:
                text += "No help available"
            return text
        elif self.type in ["menu"]:
            if 'help' in configuration['menu'][self.value]:
                return configuration['menu'][self.value]['help']
        return "No help available"

    def get_title(self):
        return get_config(self.value).get('title')

    def reset(self):
        if self.type not in ["config", "menuconfig"]:
            return

        config = get_config(self.value)
        if not config['is_user_set']:
            logging.info("Option '%s' = %s is default value" % (self.value, self.get_value()))
            return

        logging.info("Reset option '%s' = %s to default value" % (self.value, self.get_value()))
        if 'choice_group' in config:
            group = config['choice_group']
            for k in configuration['choice'][group]['configs']:
                choice_config = get_config(k)
                # We need to set it to False because update_defaults will ignore otherwise
                choice_config['is_user_set'] = False
            configuration['choice'][group].pop('requested_value', None)  # pop if available
        else:
            config.pop('requested_value', None)  # pop if available

        # We need to set it to False because update_defaults will ignore if user set
        config['is_user_set'] = False
        update_defaults(self.value)
        logging.info("After reset: %s" % self.get_value())


def get_root_menu():
    return Menu(None)


def menu_parse(data):
    menus = {None: []}
    for i in data['menu']:
        menus[i] = []

    menuconfig_stack = []
    depends_stack = []

    for i in sorted(data['order']):
        (i_type, i_symbol) = data['order'][i]

        if i_type == "config":
            config = data['config'][i_symbol]
            inmenu = config.get('inmenu')

            if 'title' not in config:
                # No title, so we don't display it
                continue

            while len(depends_stack) > 0:
                if check_depends(config.get('depends'), depends_stack[-1]):
                    break
                depends_stack.pop()

            while len(menuconfig_stack) > 0:
                if check_depends(config.get('depends'), menuconfig_stack[-1]):
                    inmenu = menuconfig_stack[-1]
                    break
                menuconfig_stack.pop()
                depends_stack = []

            if 'choice_group' in config:
                inmenu = config['choice_group']

            config['depends_indent'] = len(depends_stack)

            menus[inmenu].append(MenuItem('config', i_symbol))

            depends_stack.append(i_symbol)
        elif i_type == "menuconfig":
            config = data['config'][i_symbol]
            inmenu = config.get('inmenu')

            while len(menuconfig_stack) > 0:
                if check_depends(config.get('depends'), menuconfig_stack[-1]):
                    inmenu = menuconfig_stack[-1]
                    break
                menuconfig_stack.pop()

            menuconfig_stack.append(i_symbol)
            menus[i_symbol] = []

            menus[inmenu].append(MenuItem('menuconfig', i_symbol))
        elif i_type == "menu":
            menu = data['menu'][i_symbol]
            inmenu = menu.get('inmenu')

            menuconfig_stack = []

            menus[inmenu].append(MenuItem('menu', i_symbol))
        elif i_type == "choice":
            inmenu = data['choice'][i_symbol].get('inmenu')

            menus[i_symbol] = []

            menus[inmenu].append(MenuItem('choice', i_symbol))
        else:
            raise Exception("Unexpected item: ", data['order'][i])
    return menus
