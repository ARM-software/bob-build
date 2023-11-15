package parser

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ARM-software/bob-build/gazelle/mapper"
	"github.com/google/blueprint/parser"
)

type Parser struct {
	m     *mapper.Mapper
	scope *parser.Scope
}

func (p *Parser) GetScope() *parser.Scope {
	return p.scope
}

func New(
	m *mapper.Mapper,
	parent *parser.Scope) *Parser {

	return &Parser{
		m:     m,
		scope: parser.NewScope(parent),
	}
}

func (p *Parser) Parse(rootPath, pkgPath, fileName string) (*parser.File, error) {
	absolute := filepath.Join(rootPath, pkgPath, fileName)

	f, err := os.OpenFile(absolute, os.O_RDONLY, 0400)
	if err != nil {
		log.Fatalf("Failed to read file:'%v' with error:'%v'\n", absolute, err)
		return nil, err
	}

	ast, errs := parser.ParseAndEval(filepath.Join(pkgPath, "build.bp"), f, p.scope)
	if len(errs) != 0 {
		return nil, fmt.Errorf("Failed to parse blueprint file %v with %v", absolute, errs)
	}

	// Given a valid AST, register all module targets in mapper
	for _, def := range ast.Defs {
		switch def := def.(type) {
		case *parser.Module:
			if prop, ok := def.GetProperty("name"); ok {
				target := prop.Value.(*parser.String).Value
				label := mapper.MakeLabel(target, pkgPath)
				p.m.Map(label, target)
			}
		}
	}

	return ast, nil
}
