# Goutline

Simple utility for extracting a JSON representation of the declarations,
for any code scope nesting in a Go source file.

## Origin

Based on [go-outline](https://github.com/ramya-rao-a/go-outline) but adds:

- nesting scopes support
- go parser modes support

## Installing

```bash
go get -u github.com/1pkg/goutline
```

## Using

Optionally you can provide _uint_ mode for go parser the `-mode` flag,
by default mode equals to (4) _ParseComments_, see
[go parser Mode](https://golang.org/pkg/go/parser/#Mode) for more options and details.

```bash
> goutline -f file.go -mode 32
[{"label":"proc","type":"package",<...>}]
```

To parse unsaved file contents, use the `-modified` flag along with the `-f` flag and write an archive to stdin.  
File in the archive will be preferred over the one on disk.

The archive entry consists of:

- the file name, followed by a newline
- the (decimal) file size, followed by a newline
- the contents of the file

### Schema

Declarations are provided as flatten list, with
artificial package declaration at list head.

```go
type Declaration struct {
	Label        string        `json:"label"`
	Type         string        `json:"type"`
	Start        token.Pos     `json:"start"`
	End          token.Pos     `json:"end"`
}
```
