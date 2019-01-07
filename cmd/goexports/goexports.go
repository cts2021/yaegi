// This program generates code to register binary program symbols to the interpreter.
// See stdlib.go for usage

package main

import (
	"bytes"
	"go/format"
	"go/importer"
	"go/types"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"text/template"
	"unicode"
)

const model = `package {{ .Dest }}

// CODE GENERATED AUTOMATICALLY BY 'goexports {{ .PkgName }}'.
// THIS FILE MUST NOT BE EDITED.

import (
	"{{ .PkgName }}"
	"reflect"
)

func init() {
	Value["{{ .PkgName }}"] = map[string]reflect.Value{
		{{range .Val -}}
			{{if forceUint $.Pkg . -}}
				"{{.}}": reflect.ValueOf(uint({{ $.Pkg }}.{{.}})),
			{{else -}}
				"{{.}}": reflect.ValueOf({{ $.Pkg }}.{{.}}),
			{{ end -}}
		{{end -}}
	}

	Type["{{ .PkgName }}"] = map[string]reflect.Type{
		{{range .Typ -}}
			"{{.}}": reflect.TypeOf((*{{ $.Pkg }}.{{.}})(nil)).Elem(),
		{{end -}}
	}
}
`

func genContent(dest, pkgName string) ([]byte, error) {
	p, err := importer.Default().Import(pkgName)
	if err != nil {
		return nil, err
	}

	var typ []string
	var val []string
	sc := p.Scope()
	for _, name := range sc.Names() {
		// Skip private symbols
		if r := []rune(name); unicode.IsLower(r[0]) || name[0] == '_' {
			continue
		}

		o := sc.Lookup(name)
		switch o.(type) {
		case *types.Const, *types.Func, *types.Var:
			val = append(val, name)
		case *types.TypeName:
			typ = append(typ, name)
		}
	}

	base := template.New("goexports")
	base.Funcs(template.FuncMap{
		"forceUint": forceUint,
	})

	parse, err := base.Parse(model)
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{
		"Dest":    dest,
		"PkgName": pkgName,
		"Pkg":     path.Base(pkgName),
		"Val":     val,
		"Typ":     typ,
	}

	b := &bytes.Buffer{}
	err = parse.Execute(b, data)
	if err != nil {
		return nil, err
	}

	// gofmt
	source, err := format.Source(b.Bytes())
	if err != nil {
		return nil, err
	}
	return source, nil
}

// go build will fail with overflow error if this const is untyped. Fix this.
func forceUint(pkgName, v string) bool {
	return (pkgName == "math" && v == "MaxUint64") ||
		(pkgName == "hash/crc64" && v == "ECMA") ||
		(pkgName == "hash/crc64" && v == "ISO")
}

func main() {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	dest := path.Base(dir)

	for _, pkg := range os.Args[1:] {
		content, err := genContent(dest, pkg)
		if err != nil {
			log.Fatal(err)
		}

		oFile := strings.Replace(pkg, "/", "_", -1) + ".go"

		err = ioutil.WriteFile(oFile, content, 0666)
		if err != nil {
			log.Fatal(err)
		}
	}
}
