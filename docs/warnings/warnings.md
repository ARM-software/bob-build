# WarningLogger

# Preface

`Bob` is based on Google's [Blueprint](https://github.com/google/blueprint)
repository which has been archived some time ago.
Beside it implements few handy features it breaks some core rules
which new solid build system should have like hermeticity.

The main feature of `Bob` is to building for both Android and Linux.
Constant evolution of _Android_ makes things harder and forces `Bob`
to implement things which are not necessarily features but rather
workarounds. That causes more breakage of build system principles.

Apart of that an Android will be switching to
[Bazel](https://bazel.build) with incoming releases.

Thus `Bob` will be deprecated when Android will move to Bazel.

All above makes us to _adjust_ `Bob` to the form its `build.bp` files
will be easily convertible to Bazel.

`WarningLogger` is a first step to help with such transition and warn
about all the issues currently implemented in `build.bp` files that does
not conform to the new approach.
When all risen issues become fixed, particular warnings will be
promoted to error to prevent using all prohibited `Bob`'s functionality.

# WarningLogger

`WarningLogger` allows to write out the warnings to any `io.Writer`
with the CSV format of:

```
BpFile,BpModule,WarningAction,WarningMessage,WarningCategory
```

Additionally warnings set with `WarningAction` or `ErrorAction`
action will be printed to `os.Stderr`.

## Warnings categories

There are few types to categorize a warning:

- [DefaultSrcsWarning](default-srcs.md) - `[default-srcs]`
- [GenerateRuleWarning](generate-rule.md) - `[generate-rule]`
- [PropertyWarning](property.md) - `[property]`
- [RelativeUpLinkWarning](relative-up-link.md) - `[relative-up-link]`
- [UnmatchedNonCompileSrcsWarning](unmatched-non-compile-srcs.md) - `[unmatched-non-compile-srcs]`

## Warning actions

`WarningLogger` has a three actions to deal with warnings:

- `I` (ignore)
- `W` (warning)
- `E` (error)

By default all raised warnings are ignored coupled with action ignore (I).
It means all the issues will be written just to provided `io.Writer`
but not to `io.Stderr`.

## Warning filtering

_Filter expression_ allows to change the default behavior and change
the action for particular warning categories. It is a space separated
string with the format:

```
"WarningCategory:WarningAction"
```

E.g. to warn (`W`) all `RelativeUpLinkWarning`, _filter expression_ will be:

```
"RelativeUpLinkWarning:W"
```

It is possible to combine all warning categories with specific action using
a wildcard (`*`). E.g. to mark all warning categories as errors (`E`),
_filter expression_ will be:

```
"*:E"
```

Wildcard can be combined also with the other filters. In that case wildcard
will take an effect only with those categories which were not specified, e.g.
all but `RelativeUpLinkWarning` and `PropertyWarning` will be set as errors:

```
"PropertyWarning:I *:E RelativeUpLinkWarning:W"
```

**IMPORTANT:** Overriding category or a wildcard (`*`) is not possible.
Only the first occurrence will give an effect:

- `"*:W *:E"` - all as warning
- `"RelativeUpLinkWarning:E RelativeUpLinkWarning:W"` - RelativeUpLinkWarning as error

---

Current behavior of the logger is to emit all warnings before raising
an error. `WarningLogger` counts the number of errors occurred for
later processing:

```go
func (w *WarningLogger) ErrorWarnings() int
```
