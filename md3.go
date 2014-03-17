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

func stringOrEmpty(s, defval string) string {
	if len(s) == 0 {
		return defval
	}
	return s
}

const noName = "(no name)"

func logModelSpec(model *md3.Model) {
	fmt.Printf("MD3(%s):\n", stringOrEmpty(model.Name(), "no name"))
	fmt.Printf("  Frames: %d\n", model.NumFrames())
	fmt.Printf("  Tags(%d):\n", model.NumTags())
	for tag := range model.Tags() {
		fmt.Printf("    %s\n", tag.Name())
	}
	fmt.Printf("  Surfaces(%d):\n", model.NumSurfaces())
	for surf := range model.Surfaces() {
		fmt.Printf("    %s:\n", stringOrEmpty(surf.Name(), noName))
		if surf.NumFrames() != model.NumFrames() {
			fmt.Printf("      Frames:    %d\n", surf.NumFrames())
		}
		fmt.Printf("      Vertices:  %d\n", surf.NumVertices())
		fmt.Printf("      Triangles: %d\n", surf.NumTriangles())
		fmt.Printf("      Shaders[%d]:\n", surf.NumShaders())
		for shader := range surf.Shaders() {
			fmt.Printf("        Shader[%d]: %s\n", shader.Index, stringOrEmpty(shader.Name, noName))
		}
	}
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
