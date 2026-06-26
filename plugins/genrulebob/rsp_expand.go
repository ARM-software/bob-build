//go:build soong
// +build soong

package genrulebob

import "regexp"

var rspVarRegex = regexp.MustCompile(`\$\{([A-Za-z0-9_.-]+)\}`)

// expandRspContent replaces ${var} with args[var] when present.
// Unknown variables are left intact for later expansion.
func expandRspContent(rsp string, args map[string]string) string {
	return rspVarRegex.ReplaceAllStringFunc(rsp, func(m string) string {
		sub := rspVarRegex.FindStringSubmatch(m)
		if len(sub) != 2 {
			return m
		}
		if v, ok := args[sub[1]]; ok {
			return v
		}
		return m
	})
}
