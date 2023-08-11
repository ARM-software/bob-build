# Gendiffer test cases

These test cases are snapshot based testing to ensure Bob generates the expected output for given blueprints.

The tests are organized in the following manner:

```
	.
	├── example                   // Gendiffer boilerplate, copy this to add new test case
	├── config                    // Testing Mconfig variations, such as toolchain setup
	├── alias
	├── binary                    // Bob module name without `bob_` prefix
	│   ├── dependent             // Specific test case
	│   └── simple
	└── legacy                    // Legacy snapshots
	    ├── complex_srcs
	    ├── example
	    ├── ...
	    ├── strict_binary
	    ├── template
	    ├── transformsrcs
	    └── transformsrcs_new
```
