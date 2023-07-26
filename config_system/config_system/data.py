import logging
import os

logger = logging.getLogger(__name__)
logger.addHandler(logging.NullHandler())


__mconfig_dir = ""
__mconfig_srcs = []


def get_mconfig_dir():
    """
    Retrieve the path to the input option database.
    """
    return __mconfig_dir


def get_mconfig_srcs():
    """
    Retrieve the list of sourced configuration files.
    """
    return __mconfig_srcs


def init(options_filename, ignore_missing=False):
    from config_system import lex, lex_wrapper, syntax

    global __mconfig_dir
    global __mconfig_srcs
    global configuration

    try:
        lexer = lex_wrapper.LexWrapper(ignore_missing)
        lexer.source(options_filename)
        configuration = syntax.parser.parse(None, debug=False, lexer=lexer)
        __mconfig_srcs = lexer.sources
    except lex.TokenizeError as _:  # noqa: F841
        logger.debug("Failed to tokenise input")
        exit(1)
    except syntax.ParseError as _:  # noqa: F841
        logger.debug("Parse error")
        exit(1)
    __mconfig_dir = os.path.dirname(options_filename)


def get_config(key):
    return configuration["config"][key]


def get_choice_group(key):
    return configuration["choice"][key]


def is_choice_group(key):
    return key in configuration["choice"]


def get_menu(key):
    return configuration["menu"][key]


def get_menu_list():
    return configuration["menu"]


def iter_symbols_menuorder():
    # return tuple of (type, symbol)
    for i in sorted(configuration["order"]):
        yield configuration["order"][i]


def get_menu_configitem(type, value):
    if type in ["config", "menuconfig"]:
        return get_config(value)
    elif type in ["choice"]:
        return get_choice_group(value)
    elif type in ["menu"]:
        return get_menu(value)
    elif type == "empty":
        return None


def get_config_list():
    return configuration["config"].keys()


def get_menu_title(number):
    if number in configuration["menu"]:
        if "title" in configuration["menu"][number]:
            return configuration["menu"][number]["title"]
    elif number in configuration["choice"]:
        if "title" in configuration["choice"][number]:
            return configuration["choice"][number]["title"]
    return "Configuration"


def get_title_bar():
    if "title_bar" in configuration:
        return configuration["title_bar"]
    return "Configuration System"
