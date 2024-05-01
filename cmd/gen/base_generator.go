package gen

import (
	"fmt"
	"go/ast"
	ps "go/parser"
	"go/token"
	"strings"

	"strconv"

	"bytes"
	"go/format"

	"go-kit-demo/cmd/fs"
	"go-kit-demo/cmd/parser"
	"go-kit-demo/cmd/utils"

	"github.com/dave/jennifer/jen"
	"github.com/sirupsen/logrus"
)

// Gen represents a generator.
type Gen interface {
	Generate() error
}

// BaseGenerator implements some basic generator functionality used by all generators.
type BaseGenerator struct {
	srcFile *jen.File
	code    *PartialGenerator
	fs      *fs.KitFs
}

// InitPg initiates the partial generator (used when we don't want to generate the full source only portions)
func (b *BaseGenerator) InitPg() {
	b.code = NewPartialGenerator(b.srcFile.Empty())
}
func (b *BaseGenerator) getMissingImports(imp []parser.NamedTypeValue, f *parser.File) ([]parser.NamedTypeValue, error) {
	var n []parser.NamedTypeValue
	for _, v := range imp {
		for i, vo := range f.Imports {
			if vo.Name == "" {
				tp, err := strconv.Unquote(vo.Type)
				if err != nil {
					return n, err
				}
				if v.Type == vo.Type && strings.HasSuffix(tp, v.Name) {
					break
				}
			}
			if v.Type == vo.Type && v.Name == vo.Name {
				break
			} else if i == len(f.Imports)-1 {
				n = append(n, v)
			}
		}
	}
	if len(f.Imports) == 0 {
		n = imp
	}
	return n, nil
}

// CreateFolderStructure create folder structure of path
func (b *BaseGenerator) CreateFolderStructure(path string) error {
	e, err := b.fs.Exists(path)

	if err != nil {
		return err
	}
	if !e {
		logrus.Debug(fmt.Sprintf("Creating missing folder structure : %s", path))
		return b.fs.MkdirAll(path)
	}
	return nil
}

// GenerateNameBySample is used to generate a variable name using a sample.
//
// The exclude parameter represents the names that it can not use.
//
// E.x  sample = "hello" this will return the name "h" if it is not in any NamedTypeValue name.
func (b *BaseGenerator) GenerateNameBySample(sample string, exclude []parser.NamedTypeValue) string {
	sn := 1
	name := utils.ToLowerFirstCamelCase(sample)[:sn]
	for _, v := range exclude {
		if v.Name == name {
			sn++
			if sn > len(sample) {
				sample = strconv.Itoa(len(sample) - sn)
			}
			name = utils.ToLowerFirstCamelCase(sample)[:sn]
		}
	}
	return name
}

// EnsureThatWeUseQualifierIfNeeded is used to see if we need to import a path of a given type.
func (b *BaseGenerator) EnsureThatWeUseQualifierIfNeeded(tp string, imp []parser.NamedTypeValue) string {
	if bytes.HasPrefix([]byte(tp), []byte("...")) {
		return ""
	}
	if t := strings.Split(tp, "."); len(t) > 0 {
		s := t[0]
		for _, v := range imp {
			i, _ := strconv.Unquote(v.Type)
			if strings.HasSuffix(i, s) || v.Name == s {
				return i
			}
		}
		return ""
	}
	return ""
}

// AddImportsToFile adds missing imports toa file that we edit with the generator
func (b *BaseGenerator) AddImportsToFile(imp []parser.NamedTypeValue, src string) (string, error) {
	// Create the AST by parsing src
	fset := token.NewFileSet()
	f, err := ps.ParseFile(fset, "", src, ps.ParseComments)
	if err != nil {
		return "", err
	}
	found := false
	// Add the imports
	for i := 0; i < len(f.Decls); i++ {
		d := f.Decls[i]
		switch d.(type) {
		case *ast.FuncDecl:
			// No action
		case *ast.GenDecl:
			dd := d.(*ast.GenDecl)

			// IMPORT Declarations
			if dd.Tok == token.IMPORT {
				if dd.Rparen == 0 || dd.Lparen == 0 {
					dd.Rparen = f.Package
					dd.Lparen = f.Package
				}
				found = true
				// Add the new import
				for _, v := range imp {
					iSpec := &ast.ImportSpec{
						Name: &ast.Ident{Name: v.Name},
						Path: &ast.BasicLit{Value: v.Type},
					}
					dd.Specs = append(dd.Specs, iSpec)
				}
			}
		}
	}
	if !found {
		dd := ast.GenDecl{
			TokPos: f.Package + 1,
			Tok:    token.IMPORT,
			Specs:  []ast.Spec{},
			Lparen: f.Package,
			Rparen: f.Package,
		}
		lastPos := 0
		for _, v := range imp {
			lastPos += len(v.Name) + len(v.Type) + 1
			iSpec := &ast.ImportSpec{
				Name:   &ast.Ident{Name: v.Name},
				Path:   &ast.BasicLit{Value: v.Type},
				EndPos: token.Pos(lastPos),
			}
			dd.Specs = append(dd.Specs, iSpec)

		}
		f.Decls = append([]ast.Decl{&dd}, f.Decls...)
	}

	// Sort the imports
	ast.SortImports(fset, f)
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s", buf.Bytes()), nil
}
