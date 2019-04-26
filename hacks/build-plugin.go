package main

import (
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/alecthomas/kingpin"
)

var packagePath = kingpin.Arg("package", "Go package path").String()
var packageSymbol = kingpin.Arg("symbol", "Symbol exported by the plugin package").Default("Plugin").String()
var outputPath = kingpin.Flag("output", "Directory where the file should be written to").Short('o').Default(".").String()

var tmpl = `
package main

import (
	p "{{.Package}}"
)

var Plugin = p.{{.Symbol}}
`

func main() {
	kingpin.Parse()

	t, err := template.New("template").Parse(tmpl)
	if err != nil {
		log.Fatal(err)
	}

	tmpfile, err := ioutil.TempFile("", "envel-plugin.*.go")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up
	defer tmpfile.Close()

	if err := t.Execute(tmpfile, map[string]string{
		"Package": *packagePath,
		"Symbol":  *packageSymbol,
	}); err != nil {
		log.Fatal(err)
	}

	name := path.Base(*packagePath)
	output := filepath.Join(*outputPath, name+".so")

	cmd := exec.Command("go", "build", "-o", output, "-buildmode=plugin", "-ldflags", "-s -w", tmpfile.Name())
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
