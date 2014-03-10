package main

import (
	"flag"
	"fmt"
	"github.com/nilium/go-md3/md3"
	"io"
	"io/ioutil"
	"log"
	"os"
)

type appMode int

const (
	convertMode appMode = iota
	specMode
	viewMode
	defaultMode = specMode
)

func (m appMode) String() string {
	switch m {
	case convertMode:
		return "convert"
	case specMode:
		return "spec"
	case viewMode:
		return "view"
	default:
		return "invalid"
	}
}

type appModeVar struct {
	mode appMode
}

func (v *appModeVar) String() string {
	return v.mode.String()
}

func (v *appModeVar) Set(value string) error {
	switch value {
	case "spec":
		v.mode = specMode
	case "view":
		v.mode = viewMode
	case "convert":
		v.mode = convertMode
	default:
		return fmt.Errorf("Invalid mode flag: %q", value)
	}
	return nil
}

func dataForPath(path string) ([]byte, error) {
	var r io.Reader
	if path == "-" {
		r = os.Stdin
	} else {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		r = file
		defer file.Close()
	}

	return ioutil.ReadAll(r)
}

func stringOrEmpty(s, defval string) string {
	if len(s) == 0 {
		return defval
	}
	return s
}

const noName = "(no name)"

func logModelSpec(model *md3.Model) {
       log.Println("Unimpemented: spec")
}

func logModelSpecsProcess(models <-chan *md3.Model, done chan<- bool) {
	for model := range models {
		logModelSpec(model)
	}

	done <- true
}

func logModelSpecs() (chan<- *md3.Model, <-chan bool) {
	done := make(chan bool)
	input := make(chan *md3.Model)

	go logModelSpecsProcess(input, done)

	return input, done
}

func main() {
	mode := &appModeVar{defaultMode}

	flag.Var(mode, "mode", "One of convert, spec, or view. [default=convert]")
	flag.Parse()

	output := make(chan *md3.Model)

	for _, path := range flag.Args() {
		go func(path string, output chan<- *md3.Model) {
			var model *md3.Model
			var err error
			var data []byte
			data, err = dataForPath(path)

			if err != nil {
				log.Printf("Error reading data for path %q:\n%s", path, err)
				output <- nil
				return
			}

			model, err = md3.Read(data)
			if err != nil {
				log.Printf("Error reading MD3 header %q:\n%s", path, err)
			}

			output <- model
		}(path, output)
	}

	var modelOutput chan<- *md3.Model
	var doneProcessingModels <-chan bool

	switch mode.mode {
	case convertMode:
		panic("Unimplemented mode: convert")
	case viewMode:
		panic("Unimplemented mode: view")
	case specMode:
		modelOutput, doneProcessingModels = logModelSpecs()
	}

	nargs := flag.NArg()
	for i := 0; i < nargs; i++ {
		if model, ok := <-output; ok && model != nil {
			modelOutput <- model
		}
	}

	close(modelOutput)

	<-doneProcessingModels
}
