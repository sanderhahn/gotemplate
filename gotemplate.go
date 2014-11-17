package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
)

func write(str string) string {
	if str[0] == '=' {
		// raw
		str = str[1:len(str)]
	} else if str[0] == 'u' {
		str = "url.QueryEscape(" + str[1:len(str)] + ")"
	} else if str[0] == 'h' {
		str = "html.EscapeString(" + str[1:len(str)] + ")"
	} else {
		str = "html.EscapeString(" + str + ")"
	}
	return "if err == nil { _, err = io.WriteString(writer, " + str + ") };"
}

var trim = regexp.MustCompile(`(?m)\n\s*(<%[^=]([^%]|%[^>])+%>)\s*\n`)
var erb = regexp.MustCompile(`<%=?([^%]|%[^>])+%>|([^<]|<[^%])+`)

func convertErb(content string) (code string) {
	content = trim.ReplaceAllString(content, "$1\n")

	return erb.ReplaceAllStringFunc(content, func(str string) string {
		if strings.HasPrefix(str, "<%") {
			str = strings.TrimPrefix(str, "<%")
			str = strings.TrimSuffix(str, "%>")
			if str[0] == '=' {
				return write(str[1:])
			} else {
				return str + ";"
			}
		} else {
			return write("=" + strconv.Quote(str))
		}
	})
}

func loadErb(infile string) string {
	content, err := ioutil.ReadFile(infile)
	if err != nil {
		log.Fatal(err)
	}
	return convertErb(string(content))
}

func hasAnnotation(comment *ast.CommentGroup, annotation string) bool {
	if comment != nil {
		for _, c := range comment.List {
			if c.Text == annotation {
				return true
			}
		}
	}
	return false
}

func printNode(fset *token.FileSet, node ast.Node) string {
	var doc bytes.Buffer
	err := printer.Fprint(&doc, fset, node)
	if err != nil {
		log.Fatal(err)
	}
	return doc.String()
}

func funcErb(filename, name, signature string) string {
	signature = strings.TrimPrefix(signature, "func")
	code := loadErb(path.Join(path.Dir(filename), name+".erb"))
	return fmt.Sprintf("func Write%s%s {%s return err\n}\n", name, signature, code)
}

func compileErb(fset *token.FileSet, filename string, file *ast.File) string {
	funcs := []string{}

	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.GenDecl:
			return hasAnnotation(n.Doc, "// +erb")
		case *ast.TypeSpec:
			funcs = append(funcs, funcErb(filename, n.Name.Name, printNode(fset, n.Type)))
			return false
		}
		return true
	})
	return strings.Join(funcs, "\n")
}

func cmd(arg string, args ...string) {
	cmd := exec.Command(arg, args...)
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	err = cmd.Wait()
	if err != nil {
		log.Fatal(err)
	}
}

func output(infile, pkgname, funcs string) {
	outfile := strings.TrimSuffix(infile, ".go") + "_gen.go"
	code := fmt.Sprintf("// Autogenerated\npackage %s\n%s", pkgname, funcs)
	ioutil.WriteFile(outfile, []byte(code), 0644)
	cmd("goimports", "-w", outfile)
}

func main() {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	for pkgname, pkg := range pkgs {
		for filename, file := range pkg.Files {
			funcs := compileErb(fset, filename, file)
			if len(funcs) > 0 {
				output(filename, pkgname, funcs)
			}
		}
	}
}
