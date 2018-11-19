Module: bob_install_group
=========================

This target is used to identify a common directory in which to
copy outputs after the build completes.

`bob_install_group` supports [features](../features.md)

## Full specification of `bob_install_group` properties
```bp
bob_install_group {
    name: "custom_name",

    install_path: "{{.lib_path}}",

    // features available
}
```

----
### **bob_install_group.name** (required)
The unique identifier that can be used to refer to this module.

----
### **bob_install_group.install_path** (optional)
Path to install output of aggregated targets.
