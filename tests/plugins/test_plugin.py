import logging

from config_system import get_config_bool, set_config

logger = logging.getLogger(__name__)


def plugin_exec():
    is_linux = get_config_bool("LINUX")
    if is_linux:
        set_config("USER_SETTING_1", "y")
    else:
        set_config("USER_SETTING_2", "y")

    set_config("NON_USER_SETTABLE", "y")  # should not be set

    configs = {
        "USER_SETTING_1": is_linux,
        "USER_SETTING_2": not is_linux,
        "NON_USER_SETTABLE": False,
    }

    for k, v in configs.items():
        assert get_config_bool(f"{k}") == v, f"'{k}' should be '{v}'"
