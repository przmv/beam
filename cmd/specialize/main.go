// specialize is a low-level tool to generate type-specialized code. It is a
// convenience wrapper over text/template suitable for go generate. Unlike
// many other template tools, it does not parse Go code and allows use of
// text/template control within the template itself.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var (
	x = flag.String("x", "", "Comma-separated list of X types")
	y = flag.String("y", "", "Comma-separated list of Y types (optional)")
	z = flag.String("z", "", "Comma-separated list of Z types (optional)")

	input  = flag.String("input", "", "Template file.")
	output = flag.String("output", "", "Filename for generated code. If not provided, a file next to the input is generated.")
)

// Top is the top-level struct to be passed to the template.
type Top struct {
	// Name is the base form of the filename: "foo/bar.go.templ" -> "bar".
	Name string
	// X is the list of X type values.
	X []*X
}

// X is the concrete type to be iterated over in the user template.
type X struct {
	// Name is the name of X for use as identifier: "int" -> "Int", "[]byte" -> "ByteSlice".
	Name string
	// Type is the textual type of X: "int", "float32", "foo.Baz".
	Type string
	// Y is the list of Y type values for this X.
	Y []*Y
}

// Y is the concrete type to be iterated over in the user template for each X.
// Each combination of X and Y will be present.
type Y struct {
	// Name is the name of Y for use as identifier: "int" -> "Int", "[]byte" -> "ByteSlice".
	Name string
	// Type is the textual type of Y: "int", "float32", "foo.Baz".
	Type string
	// Z is the list of Z type values for this Y.
	Z []*Z
}

// Z is the concrete type to be iterated over in the user template for each Y.
// Each combination of X, Y and Z will be present.
type Z struct {
	// Name is the name of Z for use as identifier: "int" -> "Int", "[]byte" -> "ByteSlice".
	Name string
	// Type is the textual type of Z: "int", "float32", "foo.Baz".
	Type string
}

var macros = map[string][]string{
	"integers": []string{"int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64"},
	"floats":   []string{"float32", "float64"},
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %v [options] --input=<filename.tmpl --x=<types>\n", filepath.Base(os.Args[0]))
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	log.SetFlags(0)
	log.SetPrefix("specialize: ")

	if *input == "" {
		flag.Usage()
		log.Fatalf("no template file")
	}
	if *x == "" {
		flag.Usage()
		log.Fatalf("no specialization types")
	}

	name := filepath.Base(*input)
	if index := strings.Index(name, "."); index > 0 {
		name = name[:index]
	}
	if *output == "" {
		*output = filepath.Join(filepath.Dir(*input), name+".go")
	}

	top := Top{name, nil}
	var ys []*Y
	if *y != "" {
		var zs []*Z
		if *z != "" {
			for _, zt := range expand(*z) {
				zs = append(zs, &Z{Name: makeName(zt), Type: zt})
			}
		}
		for _, yt := range expand(*y) {
			ys = append(ys, &Y{Name: makeName(yt), Type: yt, Z: zs})
		}
	}
	for _, xt := range expand(*x) {
		top.X = append(top.X, &X{Name: makeName(xt), Type: xt, Y: ys})
	}

	tmpl, err := template.ParseFiles(*input)
	if err != nil {
		log.Fatalf("template parse failed: %v", err)
	}
	var buf bytes.Buffer
	buf.WriteString("// File generated by specialize. Do not edit.\n\n")
	if err := tmpl.Execute(&buf, top); err != nil {
		log.Fatalf("specialization failed: %v", err)
	}
	if err := ioutil.WriteFile(*output, buf.Bytes(), 0644); err != nil {
		log.Fatalf("write failed: %v", err)
	}
}

// expand parses, cleans up and expands macros for a comma-separated list.
func expand(list string) []string {
	var ret []string
	for _, xt := range strings.Split(list, ",") {
		xt = strings.TrimSpace(xt)
		if xt == "" {
			continue
		}
		if exp, ok := macros[strings.ToLower(xt)]; ok {
			for _, t := range exp {
				ret = append(ret, t)
			}
			continue
		}
		ret = append(ret, xt)
	}
	return ret
}

// makeName creates a capitalized identifier from a type.
func makeName(t string) string {
	if strings.HasPrefix(t, "[]") {
		return makeName(t[2:] + "Slice")
	}

	t = strings.Replace(t, ".", "_", -1)
	t = strings.Replace(t, "[", "_", -1)
	t = strings.Replace(t, "]", "_", -1)
	return strings.Title(t)
}
