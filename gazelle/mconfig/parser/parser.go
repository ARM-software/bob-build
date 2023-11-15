package parser

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/ARM-software/bob-build/gazelle/logic"
	"github.com/ARM-software/bob-build/gazelle/mapper"
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

// Parser implements a parser for Mconfig files that extracts configs
type Parser struct {
	m *mapper.Mapper
}

type Expression struct {
	Key        string
	Expression []interface{}
}

type ConditionalDefault struct {
	Condition  interface{}   `json:"cond"`
	Expression []interface{} `json:"expr"`
}

type ConfigData struct {
	Datatype            string               `json:"datatype"`
	RelPath             string               `json:"relPath"`
	Type                string               `json:"type"`
	Default             []interface{}        `json:"default"`
	ConditionalDefaults []ConditionalDefault `json:"default_cond"`
	Ignore              string               `json:"bob_ignore,omitempty"`
	Depends             interface{}          `json:"depends"`
	Position            uint32               `json:"position"`
	Name                string
}

func New(m *mapper.Mapper) *Parser {
	return &Parser{
		m: m,
	}
}

func (p *Parser) Parse(
	rootPath, pkgPath, fileName string) (*map[string]*ConfigData, error) {

	parserMutex.Lock()
	defer parserMutex.Unlock()

	var configs map[string]*ConfigData

	req := map[string]interface{}{
		"root_path":        rootPath,
		"rel_package_path": pkgPath,
		"file_name":        fileName,
		"ignore_source":    true,
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
		// fmt.Printf("%s\n", data)
		err = json.Unmarshal(data, &configs)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal: %w", err)
		}

		for target := range configs {
			configs[target].Name = target
			label := mapper.MakeLabel(target, pkgPath)
			p.m.Map(label, target)

			truthyExpr := &logic.Identifier{target}
			falsyExpr := &logic.Not{&logic.Identifier{target}}

			enabled := mapper.MakeLabel(truthyExpr.String(), pkgPath)
			disabled := mapper.MakeLabel(falsyExpr.String(), pkgPath)

			// Reserve enabled/disabled labels for future generation. This is important as any Mconfig can
			// refer to any other Mconfig
			p.m.Map(enabled, truthyExpr)
			p.m.Map(disabled, falsyExpr)
		}
	}

	return &configs, nil
}
