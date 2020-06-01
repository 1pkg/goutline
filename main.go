package main

import (
	"encoding/json"
	"flag"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"

	"golang.org/x/tools/go/buildutil"
)

type Declaration struct {
	Label string    `json:"label"`
	Type  string    `json:"type"`
	Start token.Pos `json:"start"`
	End   token.Pos `json:"end"`
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
			Label: file.Name.String(),
			Type:  "package",
			Start: file.Pos(),
			End:   file.End(),
		},
	}

	ast.Inspect(file, func(node ast.Node) bool {
		switch decl := node.(type) {
		case *ast.FuncDecl:
			decls = append(decls, Declaration{
				Label: decl.Name.String(),
				Type:  "function",
				Start: decl.Pos(),
				End:   decl.End(),
			})
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.ImportSpec:
					decls = append(decls, Declaration{
						Label: spec.Path.Value,
						Type:  "import",
						Start: spec.Pos(),
						End:   spec.End(),
					})
				case *ast.TypeSpec:
					decls = append(decls, Declaration{
						Label: spec.Name.String(),
						Type:  "type",
						Start: spec.Pos(),
						End:   spec.End(),
					})
				case *ast.ValueSpec:
					for _, id := range spec.Names {
						vc := "variable"
						if decl.Tok == token.CONST {
							vc = "constant"
						}
						decls = append(decls, Declaration{
							Label: id.Name,
							Type:  vc,
							Start: id.Pos(),
							End:   id.End(),
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
		print(string(r))
	} else {
		log.Fatal(err)
	}
}
