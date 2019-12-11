Overview
========

## Rationale

Why do we need another build system? What's special about Bob?

There are a number of build systems and meta-build systems around, so
why not just use one of them?

There are 2 issues that Bob solves that we think other build systems
don't address well.

* Building for both Android and Linux.

* Codebase configurability

For the primary codebase Bob is used with, there are roughly 5000
targets and 500 configuration options.

### Building on Android

On small projects it is easy enough to maintain a separate Android.mk
(or Android.bp) in addition to a set of makefiles. As the project gets
larger you might start to generate these using CMake.

If the project is tightly tied to Android, like a driver, then it
matters that the meta-build system keeps up with changes in Android.

By choosing to use Blueprint as a library we hope that it will be
easier to keep up with Android changes, and possibly integrate with
Soong in the future. Soong itself assumes a fair amount about the
build environment that we don't want imposed on Linux builds.

### Configurability

When the code base contains lots of optional behaviour, it's important that:

* users can identify what options there are and select them

  Bob's menuconfig is expected to solve this part of the problem,
  negating the need to document all the options somewhere else.

* maintainance of the options is 'simple'

  Features and templates in the build definitions make clear where the
  configurability exists.

* discourage superfluous configurability

  The fact that features can't be combined using boolean operators is
  a technical restriction based on how they are implemented (and the
  Blueprint file format). This has the benefit of forcing the
  developers to think about whether new configurations that cause
  combinatorial issues really should be configurable.

## Introduction

Bob builds happen in 3 phases:

* Bootstrap

  The user sets up a build output directory, and Bob records some
  information that it needs to retreive in subsequent phases.

* Configure

  The user sets up the configuration of the current build
  directory. This is expected to remain constant.

  Changes to the configuration can be made, and generally everything
  will work. However there are cases where the incremental build won't
  work correctly and the only fix is to delete ninja's stored
  dependency information.

* Build

  The build phase is where the main build is done.

  This also includes building Bob itself, which should only be done on
  the first build.

On Android the first 2 phases are merged, since the output directory
must match Android's expectation of where it needs to be.

At the moment Bob provides minimal scripts that projects can call for
each phase. It's expected that projects will implement their own
scripts that do project specific things in each phase.

## Supported Android versions

The Android build system changes with each Android release, which can
break some features in Bob. In general Bob will support the current
release and the one before that.

| Version | Status |
|---|---|
| Nougat | Some features may no longer work |
| Oreo | Supported |
| Pie | Supported |
| Q | In progress |
| earlier | not supported |

Note that not all Bob features are supported on Android. This includes:

* Aliases

* Versioned libraries

* Generated library modules only support a single target, so any
  library/binary that uses them will only build for the main target.
  i.e. Multilib is disabled on them.

## macOS support

Building on macOS is supported, however, some features are not available,
namely:

* Forwarding libraries
* Static libraries containing multiple objects with the same basename
* Building Linux Kernel modules
