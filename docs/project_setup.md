Project Setup
=============

In order to use Bob your project needs to be setup appropriately. The
easiest way to do this is to clone the Bob repository, copy in some
example files, and walk through the configuration.

```bash
cd $PROJECT
git clone https://github.com/ARM-software/bob-build
cd bob-build
git submodule update --init
cd -
cp bob-build/example/* -t .
```

The Bob repository includes the Blueprint repository as a submodule,
so be sure to update the submodule after checking out Bob. The above
snippet assumes you want Bob as a subdirectory in your project - this
does not have to be the case, and you can checkout to a sibling
directory if desired.

The `example` directory contains the basic files a project needs in
order to use Bob. It contains the following files:

|File|Description|
|---|---|
|bootstrap_android.bash   | Android bootstrap script |
|bootstrap_linux.bash     | Bootstrap script |
|bplist                   | Blueprint list file |
|Mconfig                  | Project configuration database |
|build.bp                 | Root build description |
|buildme.bash             | Build script |
|Android.mk.blueprint     | Template Android makefile |
|generate_android_inc.bash| Android script |

If you want to make this build, just add a `hello_world.cpp`, and, if
necessary, update the path in any `source` statements in `example/Mconfig` to
reflect the location of Bob inside the example project.

## Bootstrap scripts

The bootstrap scripts are the first thing called by the user, and
need to be tweaked for your project.

On Linux, the bootstrap script sets up the requested build output
directory, links it to the source directory, and creates some symlinks
so that the user can invoke the configuration tools and build.

On Android, the bootstrap script sets up the build output directory
that Android expects to use, and also does the initial
configuration. More about that later.

These are the main files that need to be tweaked for your project when
setting things up. Essentially the project bootstrap scripts need to
call the Bob bootstrap scripts telling them how the project is setup.

### Bob bootstrap inputs

For both Linux and Android, Bob requires the following environment
variables to be setup.

`SRCDIR` must be set to the directory under which all source files can
be found. At the moment, this must be an absolute path. Bob will not
modify this directory except to create an Android.mk (for Android), or
if a script executed by the build definitions creates something in
this directory.

`BUILDDIR` must be set to the directory where build output is
expected. This is expected to be relative to the working directory
(assumed to be the current directory).

`BLUEPRINT_LIST_FILE` is the path to the Blueprint list file.

`TOPNAME` is the path to the root build definition, relative to
`SRCDIR`.

`CONFIGNAME` is the path to the configuration file, relative to
`BUILDDIR`.

`BOB_CONFIG_OPTS` is a list of options passed to the configuration
system. This can usually be left empty.

`BOB_CONFIG_PLUGINS` is a space separated list of plugins that the
configuration system should run before saving the configuration file.

### Linux bootstrap

The Linux bootstrap script takes `BUILDDIR` as an input, and assumes
that the bootstrap script lives in `SRCDIR`. If `BUILDDIR` is
relative, it attempts to set things up so that relative directories
are used by Bob.

You should update the following:

* Update `BOB_DIR` to be a path relative to the bootstrap script at
  which to find the Bob repository.

* Set `SRCDIR` based on `SCRIPT_DIR`. If the bootstrap script is in
  the root directory, this can be set to `SCRIPT_DIR`.

* Tweak `TOPNAME`, `BLUEPRINT_LIST_FILE`, `CONFIGNAME` as needed.

* Tweak `BOB_CONFIG_OPTS` and `BOB_CONFIG_PLUGINS` if needed.

### Android

On Android the output directory is determined by the project name.

* Update `BOB_DIR` to be a path relative to the bootstrap script at
  which to find the Bob repository.

* Update `PROJ_NAME` to be a short string that is unique in the
  Android makefile namespace.

* Update `SRCDIR`, `TOPNAME`, `BLUEPRINT_LIST_FILE`, `CONFIG_NAME`,
  `BOB_CONFIG_OPTS` and `BOB_CONFIG_PLUGINS` as done for Linux.

## Blueprint file list (bplist)

Blueprint requires that all build definition files are provided to it
in a file list. The simplest way to do this is to supply the file. An
alternative is to point to a file generated at build time. The example
just lists the root `build.bp`, and points at the Bob and Blueprint
build files (both called `Blueprints`).

If you've renamed the Bob directory, update the path to the Bob and
Blueprint build definitions.

## Mconfig

The project configuration database file contains all the configuration
options for your project. Some configuration options are expected by
Bob, and are included in the example. For more information on the
configuration system see the [documentation](config_system.md).

You don't need to modify Mconfig to start with, but as you develop the
project you will probably add things as you go along.

## Root build definition (build.bp)

The root build definition file is the main file that defines the
build. You probably want it to contain commonly used compile flags in
some `bob_defaults` modules, at least one `bob_install_group` to
define where to put the output, and a definition for the main
binaries. These can be in separate files if desired - just ensure that
each file is mentioned in `bplist`.

In the example file we define a single binary, `hello_world` that uses
`build_defaults` to define warning flags and get installed to
`out/bin`. You will need to modify this file to describe how your
source code should be built.

## Build script (buildme.bash)

This build script is what the user will use to kick off a build (via a
symlink created during bootstrap). The example build script does
enough to kick off the build - you may want to customize it to do more.

If you want to generate the bplist file, you should do this here,
before calling `bob.bash`.

## Android.mk.blueprint

The Android makefile template is used to hook the project into the
Android build system. This should not require modification.

Note: when building for Android your project must be within the
Android tree. Generally you can't use a symlink from the Android tree
to your project (though you can use `bindfs` or `mount --bind`).

## generate_android_inc.bash

This script is called by the makefile template to cause the Bob
makefile fragments to be generated. This should not require modification.
