// chris 071415

package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"unicode"

	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"unicode/utf8"
)

// Program name variables.  Set by init.
var prog string
var progUpper string

func validateString(b *buf, fldname string) {
	b.writef("\tret.%s = data[\"%s\"]\n", fldname, fldname)
}

func validateBool(b *buf, fldname string) {
	b.writef("\tret.%s, err = strconv.ParseBool(data[\"%s\"])\n", fldname, fldname)
	b.writef("\tif err != nil {\n")
	b.writef("\t\treturn nil, err\n")
	b.writef("\t}\n")
}

func validator(b *buf, name string, s *ast.StructType) {
	first, _ := utf8.DecodeRune([]byte(name))
	isPublic := unicode.IsUpper(first)
	var fname string
	if isPublic {
		fname = fmt.Sprintf("Validate%s", name)
	} else {
		fname = fmt.Sprintf("validate%s", strings.Title(name))
	}

	b.writef("func %s(data map[string]string) (*%s, error) {\n", fname, name)
	b.writef("\tret := new(%s)\n", name)

	for _, fld := range s.Fields.List {
		nam := fld.Names[0].Name
		typ, ok := fld.Type.(*ast.Ident)
		if !ok {
			continue
		}
		b.writef("\t// %s %s\n", nam, typ)
		switch typ.Name {
		case "string":
			validateString(b, nam)
		case "bool":
			validateBool(b, nam)
			b.needsStrconv = true
		}
	}

	b.writef("\t\n")
	b.writef("\treturn ret, nil\n")
	b.writef("}\n")
}

func prependImport(astfile *ast.File, name string) {
	nopos := token.Pos(0)
	comment := fmt.Sprintf("// *** %s IMPORT ADDED BY %s ***", name, progUpper)
	litvalue := fmt.Sprintf("\"%s\"", name)

	decl := &ast.GenDecl{
		Doc: &ast.CommentGroup{
			List: []*ast.Comment{
				&ast.Comment{
					Slash: nopos,
					Text:  comment,
				},
			},
		},
		TokPos: nopos,
		Tok:    token.IMPORT,
		Lparen: nopos,
		Specs: []ast.Spec{
			&ast.ImportSpec{
				Doc:  nil,
				Name: nil,
				Path: &ast.BasicLit{
					ValuePos: nopos,
					Kind:     token.STRING,
					Value:    litvalue,
				},
				Comment: nil,
				EndPos:  nopos,
			},
		},
		Rparen: nopos,
	}

	astfile.Decls = append([]ast.Decl{decl}, astfile.Decls...)
}

func process(filename string, file *os.File) error {
	dst := os.Stdout // Destination.

	// Parse first before outputting anything.
	fset := token.NewFileSet()
	mode := parser.AllErrors
	astfile, err := parser.ParseFile(fset, filename, file, mode)
	if err != nil {
		return err
	}

	// Buffer validator function code before outputting anything.
	// We do this because we need to know whether we need to augment
	// the import list before outputting any declarations (imports
	// must precede declarations).
	fi, err := file.Stat()
	if err != nil {
		return err
	}
	b := newBuf(fi.Size())

	// Isolate the struct types--the things for which we want to
	// generate validator functions.
	for _, obj := range astfile.Scope.Objects {
		if obj.Kind != ast.Typ {
			continue
		}
		ts, ok := obj.Decl.(*ast.TypeSpec)
		if !ok {
			continue
		}
		s, ok := ts.Type.(*ast.StructType)
		if !ok {
			continue
		}
		if s.Fields == nil {
			return fmt.Errorf("type %s struct has empty field list %v", ts.Name, ts)
		}

		// Ok, we isolated the struct type, now output a
		// validator for it.
		validator(b, ts.Name.Name, s)
	}

	// Add strconv import if needed.
	if b.needsStrconv {
		prependImport(astfile, "strconv")
	}

	// Output header comment.
	_, err = fmt.Printf("// *** GENERATED BY %s; DO NOT EDIT ***\n\n", progUpper)
	if err != nil {
		return err
	}

	// Next, output original file.
	err = printer.Fprint(dst, fset, astfile)
	if err != nil {
		return err
	}

	// Newline to separate things.
	_, err = fmt.Println()
	if err != nil {
		return err
	}

	io.Copy(dst, b)

	return nil
}

func usage() {
	log.Printf("usage: %s file.v", path.Base(os.Args[0]))
	os.Exit(2)
}

func init() {
	log.SetFlags(0)
	prog = path.Base(os.Args[0])
	progUpper = strings.ToUpper(prog)
}

func main() {
	if len(os.Args) != 2 {
		usage()
	}

	filename := os.Args[1]
	if filename == "" {
		usage()
	}

	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	err = process(filename, file)
	if err != nil {
		log.Fatal(err)
	}
}
