Module: bob_alias
=================

This target is used to trigger builds of multiple other targets.
This target itself does not build anything, it just ensures all
the sources that it mentions are built.

An alias can't be referenced by other module types
(for example to reference multiple libraries or source generation),
although aliases themselves can include other aliases.

An alias will not try to build a target that is disabled. This
simplifies the use of aliases, so you don't have to try and
replicate the enable conditions on the target to avoid errors.

`bob_alias` supports [features](../features.md)

## Full specification of `bob_alias` properties
```bp
bob_alias {
    name: "custom_name",
    srcs: ["module_name_foo", "module_name_bar"],

    add_to_alias: ["bob_alias_module_name"],

    // features available
}
```

----
### **bob_alias.name** (required)
The unique identifier that can be used to refer to this module.

----
### **bob_alias.srcs** (optional)
Modules that this alias will cause to build.

----
### **bob_alias.add_to_alias** (optional)
Allows this alias to add itself to another alias.
`bob_alias_module_name` should refer to existing `bob_alias`.
