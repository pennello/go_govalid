// chris 071615 Validator code.

package main

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"

	"go/ast"
	"unicode/utf8"
)

// write uses fmt.Sprintf on its arguments and writes the resultant
// string into the given buffer.
func write(buf *bytes.Buffer, format string, a ...interface{}) {
	buf.WriteString(fmt.Sprintf(format, a...))
}

// validateString writes validator code for a string to the given
// buffer.
func validateString(buf *bytes.Buffer, fldname string) {
	write(buf, "\tret.%s = data[\"%s\"]\n", fldname, fldname)
}

// validateBool writes validator code for a bool to the given buffer.
func validateBool(buf *bytes.Buffer, fldname string) {
	write(buf, "\tret.%s, err = strconv.ParseBool(data[\"%s\"])\n", fldname, fldname)
	write(buf, "\tif err != nil {\n")
	write(buf, "\t\treturn nil, err\n")
	write(buf, "\t}\n")
}

// It would be nice if we didn't have so much duplication of generated
// code between the numeric validators.

// validateUint writes validator code for a uint of the given bitSize to
// the given buffer.
func validateUint(buf *bytes.Buffer, fldname string, bitSize int) {
	write(buf, "\t%stmp, err = strconv.ParseUint(data[\"%s\"], 0, %d)\n", fldname, fldname, bitSize)
	write(buf, "\tif err != nil {\n")
	write(buf, "\t\treturn nil, err\n")
	write(buf, "\t}\n")
	// Have to cast since ParseUint returns a uint64.  Superfluous
	// if bitSize is 64, but whatever.
	write(buf, "\tret.%s = uint%d(%stmp)\n", fldname, bitSize, fldname)
}

// validateUint writes validator code for a uint of
// implementation-specific size to the given buffer.
func validateUintBare(buf *bytes.Buffer, fldname string) {
	write(buf, "\tret.%s, err = strconv.ParseUint(data[\"%s\"], 0, 0)\n", fldname, fldname)
	write(buf, "\tif err != nil {\n")
	write(buf, "\t\treturn nil, err\n")
	write(buf, "\t}\n")
}

// validateInt writes validator code for an int of the given bitSize to
// the given buffer.
func validateInt(buf *bytes.Buffer, fldname string, bitSize int) {
	write(buf, "\t%stmp, err = strconv.ParseInt(data[\"%s\"], 0, %d)\n", fldname, fldname, bitSize)
	write(buf, "\tif err != nil {\n")
	write(buf, "\t\treturn nil, err\n")
	write(buf, "\t}\n")
	// Have to cast since ParseInt returns an int64.  Superfluous
	// if bitSize is 64, but whatever.
	write(buf, "\tret.%s = int%d(%stmp)\n", fldname, bitSize, fldname)
}

// validateInt writes validator code for an int of
// implementation-specific size to the given buffer.
func validateIntBare(buf *bytes.Buffer, fldname string) {
	write(buf, "\tret.%s, err = strconv.ParseInt(data[\"%s\"], 0, 0)\n", fldname, fldname)
	write(buf, "\tif err != nil {\n")
	write(buf, "\t\treturn nil, err\n")
	write(buf, "\t}\n")
}

// validateFloat writes validator code for a float of the given bitSize to
// the given buffer.
func validateFloat(buf *bytes.Buffer, fldname string, bitSize int) {
	write(buf, "\t%stmp, err = strconv.ParseFloat(data[\"%s\"], 0, %d)\n", fldname, fldname, bitSize)
	write(buf, "\tif err != nil {\n")
	write(buf, "\t\treturn nil, err\n")
	write(buf, "\t}\n")
	// Have to cast since ParseFloat returns a float64.  Superfluous
	// if bitSize is 64, but whatever.
	write(buf, "\tret.%s = float%d(%stmp)\n", fldname, bitSize, fldname)
}

// validator writes validator code for the given struct to the given
// buffer.  It iterates through the struct fields, and for those for
// which it can generate validator code, it does so.  It returns whether
// or not the strconv package is needed by the generated code.
func validator(buf *bytes.Buffer, name string, s *ast.StructType) (needsStrconv bool) {
	first, _ := utf8.DecodeRune([]byte(name))
	isPublic := unicode.IsUpper(first)
	var funcname string
	if isPublic {
		funcname = fmt.Sprintf("Validate%s", name)
	} else {
		funcname = fmt.Sprintf("validate%s", strings.Title(name))
	}

	write(buf, "\n") // Newline to separate from above content.
	write(buf, "func %s(data map[string]string) (*%s, error) {\n", funcname, name)
	write(buf, "\tret := new(%s)\n", name)

	for _, fld := range s.Fields.List {
		nam := fld.Names[0].Name
		typ, ok := fld.Type.(*ast.Ident)
		if !ok {
			continue
		}
		write(buf, "\t// %s %s\n", nam, typ)
		switch typ.Name {
		case "string":
			validateString(buf, nam)

		case "bool":
			validateBool(buf, nam)
			needsStrconv = true

		case "uint":
			validateUintBare(buf, nam)
			needsStrconv = true
		case "uint8":
			validateUint(buf, nam, 8)
			needsStrconv = true
		case "uint16":
			validateUint(buf, nam, 16)
			needsStrconv = true
		case "uint32":
			validateUint(buf, nam, 32)
			needsStrconv = true
		case "uint64":
			validateUint(buf, nam, 64)
			needsStrconv = true

		case "int":
			validateIntBare(buf, nam)
			needsStrconv = true
		case "int8":
			validateInt(buf, nam, 8)
			needsStrconv = true
		case "int16":
			validateInt(buf, nam, 16)
			needsStrconv = true
		case "int32":
			validateInt(buf, nam, 32)
			needsStrconv = true
		case "int64":
			validateInt(buf, nam, 64)
			needsStrconv = true

		case "float32":
			validateFloat(buf, nam, 32)
			needsStrconv = true
		case "float64":
			validateFloat(buf, nam, 64)
			needsStrconv = true
		}
	}

	write(buf, "\t\n")
	write(buf, "\treturn ret, nil\n")
	write(buf, "}\n")

	return needsStrconv
}
