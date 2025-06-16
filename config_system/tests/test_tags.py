import os
import sys
import pytest

TEST_DIR = os.path.dirname(os.path.abspath(__file__))
CFG_DIR = os.path.dirname(TEST_DIR)
sys.path.append(CFG_DIR)

from config_system import general

tagged_test_data = [
    (
        """
config ONE_TAG
    bool
    tag ONE
    default y

config TWO_TAGS
    string
    tag ONE
    tag TWO
    default "text"

config NO_TAGS
    bool
""",
        [("ONE_TAG", "ONE"), ("TWO_TAGS", "ONE"), ("TWO_TAGS", "TWO")],
        [("NO_TAGS", "ONE"), ("NO_TAGS", "TWO")],
    )
]


@pytest.mark.parametrize(
    "mconfig,expected_tagged,expected_not_tagged", tagged_test_data
)
def test_tagged(tmpdir, mconfig, expected_tagged, expected_not_tagged):
    mconfig_file = tmpdir.join("Mconfig")

    mconfig_file.write(mconfig, "wt")

    general.init_config(str(mconfig_file), False)
    for option, tag in expected_tagged:
        assert general.tagged(option, tag)

    for option, tag in expected_not_tagged:
        assert not general.tagged(option, tag)


get_tags_test_data = [
    (
        """
config ONE_TAG
    bool
    tag ONE
    default y

config TWO_TAGS
    string
    tag ONE
    tag TWO
    default "text"

config NO_TAGS
    bool
""",
        ["ONE", "TWO"],
    ),
    (
        """
config NO_TAGS
    bool
""",
        [],
    ),
]


@pytest.mark.parametrize("mconfig,expected_tags", get_tags_test_data)
def test_get_tags(tmpdir, mconfig, expected_tags):
    mconfig_file = tmpdir.join("Mconfig")

    mconfig_file.write(mconfig, "wt")

    general.init_config(str(mconfig_file), False)
    assert sorted(general.get_tags()) == sorted(expected_tags)


get_options_tagged_test_data = [
    (
        """
config ONE_TAG
    bool
    tag ONE
    default y

config TWO_TAGS
    string
    tag ONE
    tag TWO
    default "text"

config NO_TAGS
    bool
""",
        [
            ("ONE", ["ONE_TAG", "TWO_TAGS"]),
            ("NOT_PRESENT", []),
            ("", []),
            ("TWO", ["TWO_TAGS"]),
        ],
    )
]


@pytest.mark.parametrize("mconfig,exected_options_tagged", get_options_tagged_test_data)
def test_get_options_tagged(tmpdir, mconfig, exected_options_tagged):
    mconfig_file = tmpdir.join("Mconfig")

    mconfig_file.write(mconfig, "wt")

    general.init_config(str(mconfig_file), False)
    for tag, expected_options in exected_options_tagged:
        assert general.get_options_tagged(tag) == expected_options


if __name__ == "__main__":
    raise SystemExit(pytest.main(sys.argv))
