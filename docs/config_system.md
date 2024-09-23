# Bob Configuration System

Bob's "features" allow module definitions to be changed depending on
configuration options, e.g.:

```
bob_binary {
    ...
    warnings_as_errors: {
        cflags: ["-Werror"],
    },
}
```

However, this is limited to one level deep, so isn't much use on its own.
Bob therefore uses a separate configuration system ("Mconfig") to allow more
complicated options and dependencies, which are translated into a set of
boolean options at build time. The language is based on the Linux kernel's
configuration mechanism, Kconfig, and is partially compatible with it.

## Configuring a build using Mconfig

After bootstrapping a build, there should be two scripts in the build
directory: `config` and `menuconfig`. Generally, at least one of
these needs to be run before starting a build for the first time.

### $BUILDDIR/config

This is a command-line tool to enable and disable options. It will attempt to
satisfy all the options passed to it within a single invocation, and all other
options will be reset to their default values:

```bash
$BUILDDIR/config ANDROID=y TARGET_TOOLCHAIN_CLANG=y ALLOW_HOST_EXPLORE=n
```

If incompatible options are requested, an error will be printed and the config
file will not be written:

```bash
$BUILDDIR/config TARGET_TOOLCHAIN_CLANG=y TARGET_TOOLCHAIN_GNU=y
ERROR: TARGET_TOOLCHAIN_CLANG=y was ignored or overridden. Value is n
```

User-set options are not preserved between invocations - i.e. specifying e.g.
`ENABLE_FOO=y`, where `ENABLE_FOO` is disabled by default, will not be
preserved if `$BUILDDIR/config` is run a subsequent time without specifying
`ENABLE_FOO=y`.

### $BUILDDIR/menuconfig

This is an ncurses-based graphical tool to enable and disable options, and is
invoked by simply running `$BUILDDIR/menuconfig`.

Unlike `config`, `menuconfig` _will_ remember previously user-set options, so
can be used to incrementally change the build configuration.

Note that running `config` after `menuconfig` will clear any any previously set
options.

## Configuring the config system

### The Mconfig file

Configuration options live in a file called `Mconfig` at the root of a project
directory.

Furthermore, `Mconfig` files can include other ones using the `source` or
`source_local` statements. For compatibility with kernel Kconfig files, the
argument to `source` is a directory _relative to the project root, not relative
to the Mconfig file containing the `source` statement_. `source_local` takes a
path relative to the current file. E.g., for the following layout:

```
project
|-- subcomponent
|   |-- subsubcomponent
|   |   `-- Mconfig
|   `-- Mconfig
`-- Mconfig
```

`project/Mconfig` would contain:

```
config WARNINGS_AS_ERRORS
	bool "Fail the build if the compiler issues a warning"
	default y

# ...More config options...

source "subcomponent/Mconfig"
```

And `subcomponent/Mconfig` would contain:

```
# ...
source "subcomponent/subsubcomponent/Mconfig"
# OR: source_local "subsubcomponent/Mconfig"
# NOT: source "subsubcomponent/Mconfig"
```

### Configuration options

The fundamental unit of Mconfig is the _config option_. Each one is mapped to a
value which can be used inside `build.bp` files during the build. They are
defined as follows:

```
config OPTION_NAME
	bool|int|string "User-visible option name"
	depends on (A && B) || C
	default n|y|"hello"|1234 if D || E
	default n
	bob_ignore n|y
	select ANOTHER_OPTION
	warning "warning text when option enabled"
	help
		This is a longer, possibly multiline help text
		describing OPTION_NAME.
```

#### Types

##### Booleans

In Mconfig, these have values `y` and `n`, which are translated into `1` and
`0` when used in a `build.bp`.

##### Strings

String processing is limited, but options can contain constant string values
chosen using defaults, or be overridden by the user if user-visible:

```
config INSTALL_PATH
	string "Location to install the binary"
	default "/usr/local/bin" if LINUX
	default "$(TARGET_OUT)/bin" if ANDROID
```

String values can also be compared in expressions (`default if` and `depends on`) using the `=` and `!=` operators.

##### Integers

Like strings, int options can contain constant values chosen using defaults, or
be overridden by the user if user-visible. They can also be compared in
expressions using the `=`, `!=`, `<`, `>`, `<=` and `>=` operators.

#### Hidden options

The "user visible option name" in the example above is optional:

```
config HIDDEN_DERIVED_VALUE
	bool
	default y if OPT_A && !OPT_B
	default n
```

Options like the above will not be visible in `menuconfig`, or settable on the
command-line using `config`. Instead, they are always given their default value
based on their `default if` conditions.

#### Dependencies

Options can have dependencies using the `depends on` construct. If the
expression is not true, it will not be possible to enable the option, and the
option will not be visible in `menuconfig`:

```
config BUILD_UNIT_TESTS
	bool "Build the unit tests"
	depends on DEBUG_SYMBOLS && UNIT_TEST_FRAMEWORK_FOUND
	default y
	help
		Build the unit tests. Disable if you are not a developer.
```

#### Default values

If no default value is specified, options will be `n`, `0`, or `""` by default
(for bools, ints, and strings, respectively). However, options can specify a
default value, which can change depending on other options using `default if`:

```
config OS_NAME
	string "Operating system name"
	default "Android" if ANDROID
	default "Linux" if LINUX
	default "Unknown"
```

#### Ignore configuration options by bob

There is a possibility to mark a config option as `bob_ignore y` to point
Bob that it should ignore such option while gathering parameters for templates
and features. This will prevent from accidentally exposing options by `cflags`.

```
config PLATFORM_VERBOSE_MODE
	bool "Enable verbose mode"
	default y
	bob_ignore y

config PLATFORM_VERBOSE_TYPE
	string "verbose mode type"
	default "all"
	bob_ignore y
```

Options are stored in `.bob.config.json` as:

```
{
	"platform_verbose_mode" : {
		"ignore" : true,
		"value" : true
	},
	"platform_verbose_type" : {
		"ignore" : true,
		"value" : "all"
	}
}
```

This way Bob will not be able to recognize at all of those options:

```
bob_defaults {
	...
	platform_verbose_mode: {
        cflags: [
			"-DVERBOSE_MODE=1",
			"-DVERBOSE_TYPE={{.platform_verbose_type}}",
		],
	},
}
```

#### Negated values

Mconfig can be used to negate config options:

```
config ASSERTIONS
	bool "Enable assertions"
	default n

config NO_ASSERTIONS
	bool
	default y if !ASSERTIONS

bob_defaults {
    ...
    no_assertions: {
        cflags: ["-DNDEBUG"],
    },
}
```

The canonical way for a negative option is `default y if !POSITIVE_OPTION`, as
shown above. It could also be done using `depends`, but this is not
recommended, because to be technically correct the dependency should be
circular:

```
config ASSERTIONS
	bool "Enable assertions"
	depends on !NO_ASSERTIONS     # Don't do this
	default y

config NO_ASSERTIONS:
	bool
	depends on !ASSERTIONS
	default n
```

#### Mutually-exclusive options (choice groups)

Choice groups are used to indicate mutually exclusive config options:

```
choice
	prompt "Build type"
	default DEBUG_BUILD # If this is missing, the first option whose
	                    # dependencies are satisfied will be the default.
	help
		Choose either an optimized build, for distribution, or an
		unoptimized build with debug symbols, for debugging and
		development.

config FAST_BUILD
	bool "Optimized build"
	depends on HAS_OPTIMIZING_COMPILER

config DEBUG_BUILD
	bool "Debug build"

endchoice
```

Only one of `FAST_BUILD` and `DEBUG_BUILD` can be enabled, and an attempt to
enable them both using `config` will result in an error.

The default value is chosen by adding a `default` clause to the top-level
`choice` section, instead of by putting one on a single `config` option.

#### Selecting other options

The `select` keyword means that, when a given option is enabled, it will also
enable other option(s). Adding `if` and a condition will make this conditional.

If those options cannot be enabled, for example if their dependencies are not
satisfied, an error will be raised.

For example, the `DEBUG_BUILD` option above could be modified as follows:

```
# ...inside "Build type" choice group

config DEBUG_BUILD
	bool "Debug build"
	select ASSERTIONS
	select BUILD_UNIT_TESTS if UNIT_TEST_FRAMEWORK_FOUND
```

#### Menus

Putting options between `menu` and `endmenu` keywords will make them appear in
a separate menu within `menuconfig`. Menus also support `visible if` and
`depends on`:

- `visible if`: If the associated condition is not satisfied, the menu will
  not appear in `menuconfig`. However, options within the menu may still
  become enabled.

- `depends on`: If the associated condition is not satisfied, none of the
  options within the menu will be able to be enabled.

##### `menuconfig`

Using `menuconfig` (instead of just `menu`) creates a config option with an
associated sub-menu, which contains the options which depend on it:

```
menuconfig ENABLE_NEW_FEATURE
	bool "Enable a new feature"
	depends on A_PREREQUISITE
	default y

config NEW_FEATURE_DOES_FOO
	bool "Enable the foo functionality of the new feature"
	default y
	depends on ENABLE_NEW_FEATURE

config NEW_FEATURE_NAME
	string "The unnecessarily-configurable name of the new feature"
	default "bar"
	depends on ENABLE_NEW_FEATURE
```

In the above case, `NEW_FEATURE_DOES_FOO` and `NEW_FEATURE_NAME` will not be
visible in the ncurses GUI menu unless `ENABLE_NEW_FEATURE` is enabled.

### Setting the menu title

The `mainmenu` construct sets the title of the `menuconfig` window:

```
mainmenu "Configuration for MyProgram"

config ...
```

### Dynamic configuration using Python

Not all options can be determined statically, or from user settings. The
configuration system allows Python plugins to inspect and change options.

A configuration plugin is a Python script containing the following:

```python
import config_system

def plugin_exec():
    # Read options using `get_config()`
    if config_system.get_config("TARGET_TOOLCHAIN_CLANG")["value"] == "y":
        ...
    # Write them using `set_config()`
    if find_unit_test_framework():
        config_system.set_config("UNIT_TEST_FRAMEWORK_FOUND", "y")
```

Config plugins are included in the build system by adding them to the
`BOB_CONFIG_PLUGINS` variable during bootstrap:

```bash
export BOB_CONFIG_PLUGINS="path/to/config_plugin:path/to/other_config_plugin" # No .py extension
... # Other exports for Bob bootstrap
bob/bootstrap_linux.bash # or bootstrap_androidbp.bash
```
