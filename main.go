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
	"log"
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
	fname    = flag.String("f", "", "the path to the file to outline")
	mode     = flag.Uint("mode", uint(parser.ParseComments), "go parser mode")
	modified = flag.Bool("modified", false, "read an archive of the modified file from standard input")
)

func main() {
	flag.Parse()
	fset := token.NewFileSet()

	var src []byte
	if *modified {
		archive, err := buildutil.ParseOverlayArchive(os.Stdin)
		if err != nil {
			log.Printf("failed to parse -modified archive %v", err)
		}
		fc, ok := archive[*fname]
		if !ok {
			log.Printf("couldn't find %s in archive", *fname)
		}
		src = fc
	}

	file, err := parser.ParseFile(fset, *fname, src, parser.Mode(*mode))
	if err != nil {
		log.Fatalf("could not parse file %s", *fname)
	}

	decls := []Declaration{
		Declaration{
			file.Name.String(),
			"package",
			"",
			file.Pos(),
			file.End(),
		},
	}

	ast.Inspect(file, func(node ast.Node) bool {
		switch decl := node.(type) {
		case *ast.FuncDecl:
			receiverType, err := getReceiverType(fset, decl)
			if err != nil {
				log.Printf("failed to parse receiver type %v", err)
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
					log.Printf("unknown token type %s", decl.Tok)
					return false
				}
			}
		default:
			log.Printf("unknown declaration at %v", decl.Pos())
			return false
		}
		return true
	})

	if r, err := json.Marshal(decls); err == nil {
		fmt.Println(string(r))
	} else {
		log.Fatal(err)
	}
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
