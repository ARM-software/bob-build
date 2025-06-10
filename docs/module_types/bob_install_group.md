# `bob_install_group`

> This is a legacy target will not be supported by the [Gazelle plugin](../../gazelle/README.md).

```bp
bob_install_group {
    name, install_path
}
```

This target is used to identify a common directory in which to
copy outputs after the build completes.

Supports:

- [features](../features.md)

## Properties

|                                                |                                                                                                                                                                                                                                                                                                                                                                                                                     |
| ---------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [`name`](properties/common_properties.md#name) | String; required                                                                                                                                                                                                                                                                                                                                                                                                    |
| `install_path`                                 | String; default is `none`<br>Path to install output of aggregated targets.<br>Note that on the Android.bp backend, the first path element is treated specially, see [user guide](../user_guide/android.md#androidbp-backend-install-paths) for detail. The path does not reference the system or vendor partition, and the item will be installed in system or vendor based on the contents of the `tags` property. |
