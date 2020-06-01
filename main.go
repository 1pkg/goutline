package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"

	"golang.org/x/tools/go/buildutil"
)

type Declaration struct {
	Label        string    `json:"label"`
	Type         string    `json:"type"`
	ReceiverType string    `json:"receiverType,omitempty"`
	Start        token.Pos `json:"start"`
	End          token.Pos `json:"end"`
}

var (
	file     = flag.String("f", "", "the path to the file to outline")
	mode     = flag.Uint("mode", uint(parser.ParseComments), "go parser mode")
	modified = flag.Bool("modified", false, "read an archive of the modified file from standard input")
)

func main() {
	flag.Parse()
	fset := token.NewFileSet()

	var fileAst *ast.File
	var err error

	if *modified == true {
		archive, err := buildutil.ParseOverlayArchive(os.Stdin)
		if err != nil {
			reportError(fmt.Errorf("failed to parse -modified archive: %v", err))
		}
		fc, ok := archive[*file]
		if !ok {
			reportError(fmt.Errorf("couldn't find %s in archive", *file))
		}
		fileAst, err = parser.ParseFile(fset, *file, fc, parser.Mode(*mode))
	} else {
		fileAst, err = parser.ParseFile(fset, *file, nil, parser.Mode(*mode))
	}

	if err != nil {
		reportError(fmt.Errorf("Could not parse file %s", *file))
	}

	decls := []Declaration{
		Declaration{
			fileAst.Name.String(),
			"package",
			"",
			fileAst.Pos(),
			fileAst.End(),
		},
	}

	ast.Inspect(fileAst, func(node ast.Node) bool {
		switch decl := node.(type) {
		case *ast.FuncDecl:
			receiverType, err := getReceiverType(fset, decl)
			if err != nil {
				reportError(fmt.Errorf("Failed to parse receiver type: %v", err))
			}
			decls = append(decls, Declaration{
				decl.Name.String(),
				"function",
				receiverType,
				decl.Pos(),
				decl.End(),
			})
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.ImportSpec:
					decls = append(decls, Declaration{
						spec.Path.Value,
						"import",
						"",
						spec.Pos(),
						spec.End(),
					})
				case *ast.TypeSpec:
					//TODO: Members if it's a struct or interface type?
					decls = append(decls, Declaration{
						spec.Name.String(),
						"type",
						"",
						spec.Pos(),
						spec.End(),
					})
				case *ast.ValueSpec:
					for _, id := range spec.Names {
						varOrConst := "variable"
						if decl.Tok == token.CONST {
							varOrConst = "constant"
						}
						decls = append(decls, Declaration{
							id.Name,
							varOrConst,
							"",
							id.Pos(),
							id.End(),
						})
					}
				default:
					reportError(fmt.Errorf("Unknown token type: %s", decl.Tok))
					return false
				}
			}
		default:
			reportError(fmt.Errorf("Unknown declaration @ %v", decl.Pos()))
			return false
		}
		return true
	})

	str, _ := json.Marshal(decls)
	fmt.Println(string(str))
}

func getReceiverType(fset *token.FileSet, decl *ast.FuncDecl) (string, error) {
	if decl.Recv == nil {
		return "", nil
	}

	buf := &bytes.Buffer{}
	if err := format.Node(buf, fset, decl.Recv.List[0].Type); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func reportError(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
}
