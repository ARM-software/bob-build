// Common definitions and vars
package common

import (
	"fmt"
	"strings"
)

const (
	ConditionDefault string = "//conditions:default"
)

// TODO: resolve feature names properly depending on
// the location in `build.bp`
func GetFeatureCondition(f string) string {

	if f == ConditionDefault {
		return f
	} else {
		return fmt.Sprintf(":config_%s", strings.ToLower(f))
	}
}
