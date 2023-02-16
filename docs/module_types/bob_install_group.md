# Module: bob_install_group

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

---

### **bob_install_group.name** (required)

The unique identifier that can be used to refer to this module.

---

### **bob_install_group.install_path** (optional)

Path to install output of aggregated targets.

Note that on the Android.bp backend, the first path element is treated
specially, see
[user guide](../user_guide/android.md#androidbp-backend-install-paths)
for detail. The path does not reference the system or vendor
partition, and the item will be installed in system or vendor
based on whether the `owner` property has been set.
