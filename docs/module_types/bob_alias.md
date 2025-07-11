# `bob_alias`

```bp
bob_alias {
name, srcs, add_to_alias,
}
```

This target is used to trigger builds of multiple other targets.
This target itself does not build anything, it just ensures all
the sources that it mentions are built.

An alias can't be referenced by other module types
(for example to reference multiple libraries or source generation),
although aliases themselves can include other aliases.

An alias will not try to build a target that is disabled. This
simplifies the use of aliases, so you don't have to try and
replicate the enable conditions on the target to avoid errors.

Supports:

- [features](../features.md)

## Properties

|                                                |                                                                                                                         |
| ---------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| [`name`](properties/common_properties.md#name) | String; required                                                                                                        |
| `srcs`                                         | List of targets; default is `[]`<br>Modules that this alias will cause to build.                                        |
| `add_to_alias`                                 | Target; default is `none`<br>Allows this alias to add itself to another alias.<br>Should refer to existing `bob_alias`. |
