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

type modelPathPair struct {
	model *md3.Model
	path  string
}

const (
	convertMode = "convert"
	specMode    = "spec"
	viewMode    = "view"
	defaultMode = specMode
)

var (
	appMode = flag.String("mode", defaultMode, "One of convert, spec, or view. [default=spec]")
	flipUVs = flag.Bool("flipUVs", true, "Enables flipping UV coordinates vertically on output.")
	swapYZ  = flag.Bool("swapYZ", true, "Enables swapping Y and Z axes on output.")
)

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

func main() {
	flag.Parse()

	output := make(chan *modelPathPair)

	for _, path := range flag.Args() {
		go func(path string, output chan<- *modelPathPair) {
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

			output <- &modelPathPair{model, path}
		}(path, output)
	}

	var modelOutput chan<- *modelPathPair
	var doneProcessingModels <-chan bool

	switch *appMode {
	case convertMode:
		panic("Unimplemented mode: convert")
	case viewMode:
		panic("Unimplemented mode: view")
	case specMode:
		modelOutput, doneProcessingModels = logModelSpecs()
	default:
		panic(fmt.Errorf("Invalid mode: %q", *appMode))
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
