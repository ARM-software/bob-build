# Module: bob_glob

Glob is a helper module that finds all files that match certain path patterns
and returns a list of their paths.

## Full specification of `bob_glob` properties

```bp
bob_glob {
    name: "glob_lib_srcs",
    srcs: ["src/**/*.cpp"],
    exclude: ["src/**/exclude_*.cpp"],
    exclude_directories: True
    allow_empty: False
}
```

---

### **bob_glob.name** (required)

The unique identifier that can be used to refer to this module.

---

### **bob_glob.srcs** (required)

Path patterns that are relative to the current module.

---

### **bob_glob.exclude** (optional)

Path patterns that are relative to the current module
to exclude from `srcs`.

---

### **bob_glob.exclude_directories** (optional)

If the `exclude_directories` argument is set to `true` (default),
the directories will be omitted from the results.

---

### **bob_glob.allow_empty** (optional)

If the `allow_empty` argument is set to `false`, the glob function will
error-out if the result is the empty list. By default it is set to `true`.
