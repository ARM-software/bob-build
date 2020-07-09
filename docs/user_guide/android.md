Android Specifics
=================

There are a few properties that are specific to Android builds,
that map to information that the Android build system wants.

The `owner` property maps to the Android make variable
`LOCAL_MODULE_OWNER`. This needs to be set to the organisation
responsible for the module. When non-empty Bob will also set
`LOCAL_PROPRIETARY_MODULE=true` and the module will end up in the
vendor tree.

The `tags` property maps to the Android make variable
`LOCAL_MODULE_TAGS`. This can be used to control what gets built by
default on Android, based on the build type (`rel`, `eng`,
`userdebug`). From Android Q this is obsolete, and the product
makefile should be updated instead.

On Android, the `export_*` properties behave a bit differently. In
most cases Bob manually manages the propagation of the properties, and
they should behave the same. However the properties do not propagate
into Android-make-defined libraries. `export_local_include_dirs` is
handled by using Android make's `LOCAL_EXPORT_C_INCLUDE_DIRS` which
doesn't restrict the export to the module immediately above.

Installation on Android requires careful setup. The install paths must
be setup to match Android's install locations for the module
type. This includes whether the module is for host or target, and
whether it is in the vendor partition or not.

The Android make backend does not support [build
wrappers](wrappers.md) or [library versioning](versioning.md).

The Android.bp backend does not support post install actions.

Support for [forwarding libraries](forwarding.md) on Android is
minimal. Notably, if something links against a forwarding library,
`--copy-dt-needed-entries` is applied across the whole link and
the resultant binary will have `DT_NEEDED` symbols propagated from all
shared libraries it links against.

Android.bp Backend Install Paths
===

Soong restricts the install locations of its module types. In addition,
use of the system and vendor partitions is controlled via the
`proprietary`/`vendor` properties.

The Android.bp backend will recognize the identifiers below at the
beginning of the install path. The first three are chosen to agree
with what might be used in the Linux backend, to simplify configuration.

| Identifier | Description |
| --- | --- |
| `bin` | Executables to be installed in `<partition>/bin` |
| `lib` | Shared libraries to be installed in `<partition>/lib` |
| `etc` | Configuration to be installed in `<partition>/etc` |
| `firmware` | Firmware to be installed in `<partition>/etc/firmware` |
| `data` | Resources to be installed in `data` |
| `tests` | Test executables will be installed in an appropriate directory[1] |

[1] Ideally `tests` should be installed in `testcases`, which is not
in any of the Android images. This is not currently possible, so they
are installed to `data/nativetest` in `userdata.img`. The intention is
that tests are not flashed to the board, and they are copied to the
board when you want to run them. Please avoid relying on tests being
in the userdata image.

Android.mk Transition Support
===

Bob contains support to translate Android.mk install paths (using
Android make variables) to Android.bp backend install paths. This is
expected to be temporary.
