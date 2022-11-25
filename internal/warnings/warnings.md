WarningLogger
==================

`WarningLogger` allows to write out the warnings to any `io.Writer`
with the CSV format of:
```
BpFile,BpModule,WarningAction,WarningMessage,WarningCategory
```
Additionally warnings set with `WarningAction` or `ErrorAction`
action will be printed to `os.Stderr`.


## Warnings categories

There are few types to categorize a warning:

- `DirectPathsWarning`
- `GenerateRuleWarning`
- `PropertyWarning`
- `RelativeUpLinkWarning`
- `UserWarning`


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
