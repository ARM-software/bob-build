# Shared Library Versioning

When the API of a shared library changes in a backwards incompatible
way, it is useful to be have both versions of the library installed on
the system so that old and new clients can be supported
simultaneously.

In order to do this, version information must be added to shared
libraries. At link time, the libraries user will record the version
that it needs. When installed a few symlinks also need to be setup.

Bob supports creation of versioned libraries. Simply set
`library_version` to the required version string. The expected version
format is "MAJOR.MINOR.PATCH(-PRERELEASE)", where MAJOR, MINOR and
PATCH are integers, and PRERELEASE is a string. The version must
change for every public release.

The "-PRERELEASE" string should only be used to identify early
releases, and indicates an unstable release.

The MAJOR number must increase whenever a backwards incompatible API
change is made. A backwards incompatible API change will break
anything compiled against the older version of the library. A running
system can expect to have a library installed for each MAJOR version
of the library. MINOR and PATCH are expected to reset to 0.

The MINOR number must increase when new APIs are added to the library,
but it is otherwise backwards compatible. In a running system the
existing library of the same MAJOR version can be replaced with the
MINOR release. PATCH is expected to reset to 0.

The PATCH number must increment on all other releases, which must be
backwards compatible. The new library can replace an existing library
with the same MAJOR.

Examples:

| Version     | Description                                                        |
| ----------- | ------------------------------------------------------------------ |
| 1.0.0       | First production release                                           |
| 1.0.1       | Bugfix release                                                     |
| 1.1.0-alpha | New functionality added (compared to 1.0.0), early test            |
| 1.1.0-rc3   | New functionality added (compared to 1.0.0), 3rd release candidate |
| 1.1.0       | Production release, new functionality compared to 1.0.0            |
| 2.0.0       | Production release, incompatible with 1.x series                   |

```
bob_shared_library {
    name: "libdrm",
    srcs: ["libdrm.c"],
    export_local_include_dirs: ["include"],

    library_version: "1.4.2",
}
```
