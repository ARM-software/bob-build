package plugin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ARM-software/bob-build/gazelle/registry"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

var (
	parserStdin  io.Writer
	parserStdout io.Reader
	parserMutex  sync.Mutex
)

func init() {
	// TODO Use github.com/bazelbuild/rules_go/go/runfiles instead `bazel.Runfile`
	parseScriptRunfile, err := bazel.Runfile("config_system/get_configs_gazelle")
	if err != nil {
		log.Printf("failed to initialize parser: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	ctx, parserCancel := context.WithTimeout(ctx, time.Minute*10)
	cmd := exec.CommandContext(ctx, parseScriptRunfile)

	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("failed to initialize parser: %v\n", err)
		os.Exit(1)
	}
	parserStdin = stdin

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("failed to initialize parser: %v\n", err)
		os.Exit(1)
	}

	parserStdout = stdout

	if err := cmd.Start(); err != nil {
		log.Printf("failed to initialize parser: %v\n", err)
		os.Exit(1)
	}

	go func() {
		defer parserCancel()
		if err := cmd.Wait(); err != nil {
			log.Printf("failed to wait for parser: %v\n", err)
			os.Exit(1)
		}
	}()
}

// mconfigParser implements a parser for Mconfig files that extracts configs
type mconfigParser struct {
	// The value of `language.GenerateArgs.Config.RepoRoot`.
	repoRoot string

	// The value of `language.GenerateArgs.Rel`.
	relPackagePath string
}

type configData struct {
	Datatype   string      `json:"datatype"`
	RelPath    string      `json:"relPath"`
	Type       string      `json:"type"`
	Default    interface{} `json:"default"`
	Condition  interface{} `json:"default_cond"`
	Ignore     string      `json:"bob_ignore,omitempty"`
	Depends    interface{} `json:"depends"`
	Position   uint32      `json:"position"`
	BazelLabel label.Label
	Name       string
}

var _ registry.Registrable = (*configData)(nil)

func (c *configData) GetName() string {
	return c.Name
}

func (c *configData) GetRelativePath() string {
	return c.RelPath
}

func (c *configData) GetLabel() label.Label {
	return c.BazelLabel
}

// Constructs a new `mconfigParser`
func newMconfigParser(repoRoot string, relPackagePath string) *mconfigParser {
	return &mconfigParser{
		repoRoot:       repoRoot,
		relPackagePath: relPackagePath,
	}
}

func (p *mconfigParser) parse(fileNames *[]string) (*map[string]*configData, error) {
	parserMutex.Lock()
	defer parserMutex.Unlock()

	var configs map[string]*configData

	for _, f := range *fileNames {

		req := map[string]interface{}{
			"root_path":        p.repoRoot,
			"rel_package_path": p.relPackagePath,
			"file_name":        f,
			"ignore_source":    false,
		}

		encoder := json.NewEncoder(parserStdin)
		if err := encoder.Encode(&req); err != nil {
			return nil, fmt.Errorf("failed to encode: %w", err)
		}

		reader := bufio.NewReader(parserStdout)
		data, err := reader.ReadBytes(0)
		if err != nil {
			return nil, fmt.Errorf("failed to read: %w", err)
		}

		if len(data) > 0 {
			// remove delimiter
			data = data[:len(data)-1]

			err = json.Unmarshal(data, &configs)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal: %w", err)
			}
		}
	}

	resolveConfigLabels(&configs, p.repoRoot)

	return &configs, nil
}

func resolveConfigLabels(c *map[string]*configData, root string) {
	for k, v := range *c {
		relPath := filepath.Clean(v.RelPath)

		if relPath == "." {
			relPath = ""
		}

		v.Name = strings.ToLower(k)
		v.BazelLabel = label.Label{Pkg: relPath, Name: v.Name}

		(*c)[k] = v
	}
}
