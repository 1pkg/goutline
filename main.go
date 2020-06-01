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

// Declaration defines DTO item
// to carry and serialize go code declaration
type Declaration struct {
	Label string    `json:"label"`
	Type  string    `json:"type"`
	Start token.Pos `json:"start"`
	End   token.Pos `json:"end"`
}

// collections of flags
var (
	fname    = flag.String("f", "", "the path to the file to outline")
	mode     = flag.Uint("mode", uint(parser.ParseComments), "go parser mode")
	modified = flag.Bool("modified", false, "read an archive of the modified file from standard input")
	_        = flag.Bool("imports-only", false, "DEPRECATED: does nothing, kept only for backward compatibility")
)

func main() {
	// parse flags
	flag.Parse()
	// set log output
	log.SetOutput(os.Stderr)
	// parse ast file
	file := parse(parser.Mode(*mode), *modified)
	// collect declarations
	decls := inspect(file)
	// print declarations to stdouts
	out(decls)
}

// parse parses ast file with mode and archive fallback
func parse(mode parser.Mode, archive bool) *ast.File {
	// try to get file source from build util archive
	var src interface{}
	if archive {
		archive, err := buildutil.ParseOverlayArchive(os.Stdin)
		if err != nil {
			log.Printf("failed to parse -modified archive %v", err)
		}
		asrc, ok := archive[*fname]
		if !ok {
			log.Printf("couldn't find %s in archive", *fname)
		}
		src = asrc
	}
	// parse file and handle errors
	file, err := parser.ParseFile(token.NewFileSet(), *fname, src, mode)
	if err != nil {
		log.Fatalf("could not parse file %s %v", *fname, err)
	}
	return file
}

// inspect collects ast file declarations
func inspect(file *ast.File) []Declaration {
	// inspect deep ast file for known declarations
	decls := make([]Declaration, 0)
	ast.Inspect(file, func(node ast.Node) bool {
		if node != nil {
			switch decl := node.(type) {
			case *ast.File:
				decls = append(decls, Declaration{
					Label: decl.Name.String(),
					Type:  "package",
					Start: file.Pos(),
					End:   file.End(),
				})
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
					}
				}
			default:
				log.Printf("unknown declaration at %v", decl.Pos())
			}
			return true
		}
		return false
	})
	return decls
}

// out writes declaration to std out
func out(decls []Declaration) {
	// serialize declarations to json
	r, err := json.Marshal(decls)
	if err != nil {
		log.Fatal(err)
	}
	// try to flush them to stdout
	if _, err := os.Stdout.Write(r); err != nil {
		log.Fatal(err)
	}
}
