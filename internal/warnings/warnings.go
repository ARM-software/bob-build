package warnings

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
)

const URL string = "https://github.com/ARM-software/bob-build/tree/master/docs/warnings/"

type Category string

const (
	DefaultSrcsWarning                Category = "default-srcs"
	GenerateRuleWarning               Category = "generate-rule"
	RelativeUpLinkWarning             Category = "relative-up-link"
	UnmatchedNonCompileSrcsWarning    Category = "unmatched-non-compile-srcs"
	AndroidOutOfTreeUnsupportedModule Category = "android-out-of-tree-unsupported-module"
)

var categoriesMap = map[string]Category{
	"DefaultSrcsWarning":                DefaultSrcsWarning,
	"GenerateRuleWarning":               GenerateRuleWarning,
	"RelativeUpLinkWarning":             RelativeUpLinkWarning,
	"UnmatchedNonCompileSrcsWarning":    UnmatchedNonCompileSrcsWarning,
	"AndroidOutOfTreeUnsupportedModule": AndroidOutOfTreeUnsupportedModule,
}

var categoriesMessages = map[Category]string{
	DefaultSrcsWarning:                "`srcs`/`exclude_srcs` property should not be used in defaults. Specify target sources explicitly or use `bob_filegroup`.",
	GenerateRuleWarning:               "`bob_generate_source` should not be used. Use `bob_genrule` instead.",
	RelativeUpLinkWarning:             "Relative up-links in `srcs` are not allowed. Use `bob_filegroup` instead.",
	UnmatchedNonCompileSrcsWarning:    "Non-compiled sources have not been matched fully.",
	AndroidOutOfTreeUnsupportedModule: "Android of out tree does not support all module types yet.",
}

type Action string

const (
	IgnoreAction  Action = "ignore"
	WarningAction Action = "warning"
	ErrorAction   Action = "error"
)

var actionsMap = map[string]Action{
	"I": IgnoreAction,
	"W": WarningAction,
	"E": ErrorAction,
}

type WarningLogger struct {
	out          *csv.Writer
	mu           sync.Mutex
	filters      map[Category]Action
	globalAction Action
	errors       int
	hypelinks    bool
}

func New(out io.Writer, filters string) *WarningLogger {
	w := csv.NewWriter(out)
	w.Write([]string{"BpFile", "BpModule", "WarningAction", "WarningMessage", "WarningCategory"})
	w.Flush()

	f, g := parseFilters(filters)

	return &WarningLogger{out: w, filters: f, globalAction: g, hypelinks: checkIfHyperlinks()}
}

func parseFilters(f string) (filters map[Category]Action, globalAction Action) {

	filters = make(map[Category]Action)

	fn := func(c rune) bool {
		return c == ' '
	}

	if f != "" {
		for _, subFilter := range strings.FieldsFunc(f, fn) {
			parts := strings.SplitN(subFilter, ":", 2)

			if len(parts) != 2 {
				fmt.Fprintf(os.Stderr, "Wrong warnings filter expression '%s'\n", subFilter)
				continue
			}

			c, a := parts[0], parts[1]

			if _, ok := actionsMap[a]; !ok {
				fmt.Fprintf(os.Stderr, "Wrong filter action '%s'\n", subFilter)
				continue
			}

			if c == "*" {
				if globalAction != "" {
					fmt.Fprintf(os.Stderr, "Overriding wildcard (*) not allowed: '%s'\n", subFilter)
				} else {
					globalAction = actionsMap[a]
				}

				continue
			}

			if category, ok := categoriesMap[c]; ok {
				if _, ok := filters[category]; ok {
					fmt.Fprintf(os.Stderr, "Overriding warning category not allowed: '%s'\n", subFilter)
					continue
				}

				filters[category] = actionsMap[a]
			} else {
				fmt.Fprintf(os.Stderr, "Wrong filter category '%s'\n", subFilter)
			}
		}
	}

	if globalAction == "" {
		globalAction = IgnoreAction
	}

	return
}

func (w *WarningLogger) getLink(category Category) string {
	if w.hypelinks {
		return fmt.Sprintf("\x1b]8;;%[1]s%[2]s.md\x07%[2]s\x1b]8;;\x07", URL, category)
	} else {
		return string(category)
	}
}

func (w *WarningLogger) InfoMessage() string {
	var head = "For more information on Bob warnings, see:"
	if w.hypelinks {
		return fmt.Sprintf("%[1]s [\x1b]8;;%[2]swarnings.md\x07%[2]swarnings.md\x1b]8;;\x07]", head, URL)
	} else {
		return fmt.Sprintf("%[1]s [%[2]swarnings.md]", head, URL)
	}
}

func (w *WarningLogger) ErrorWarnings() int {
	return w.errors
}

func (w *WarningLogger) getMessage(category Category, args ...interface{}) string {
	return fmt.Sprintf(categoriesMessages[category], args...)
}

func (w *WarningLogger) Warn(category Category, bpFile string, bpModule string, args ...interface{}) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	action, ok := w.filters[category]

	if !ok {
		action = w.globalAction
	}

	if action == ErrorAction {
		w.errors++
	}

	if action != IgnoreAction {
		io.WriteString(os.Stderr, fmt.Sprintf("%s:%s: %s: %s [%s]\n", bpFile, bpModule, action, w.getMessage(category, args...), w.getLink(category)))
	}

	w.out.Write([]string{bpFile, bpModule, string(action), w.getMessage(category, args...), string(category)})
	w.out.Flush()

	return w.out.Error()
}

func checkIfHyperlinks() bool {
	if _, ok := os.LookupEnv("DOMTERM"); ok {
		return true
	}

	if v, ok := os.LookupEnv("VTE_VERSION"); ok {
		ver, err := strconv.ParseInt(v, 10, 0)

		if err == nil && ver >= 5000 {
			return true
		}
	}

	if t, ok := os.LookupEnv("TERM"); ok {
		if strings.HasPrefix(t, "xterm") {
			return true
		}
	}

	return false
}
