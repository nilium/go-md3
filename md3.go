package main

import (
	"flag"
	"github.com/nilium/go-md3/md3"
	"io"
	"log"
	"os"
)

func readerForPath(path string) (io.Reader, error) {
	if path == "-" {
		return os.Stdin, nil
	} else {
		return os.Open(path)
	}
}

func main() {
	flag.Parse()

	output := make(chan *md3.Model)
	models := make([]*md3.Model, 0)

	for _, path := range flag.Args() {
		go func(path string, output chan<- *md3.Model) {
			var model *md3.Model
			var err error
			var r io.Reader
			r, err = readerForPath(path)

			if err != nil {
				log.Printf("Error creating reader for path %q:\n%s", path, err)
				output <- nil
				return
			}

			model, err = md3.Read(r)
			if err != nil {
				log.Printf("Error reading MD3 header %q:\n%s", path, err)
			}

			output <- model
		}(path, output)
	}

	nargs := flag.NArg()
	for i := 0; i < nargs; i++ {
		if model, ok := <-output; ok && model != nil {
			models = append(models, model)
			for tag := range model.Tags() {
				log.Println(tag)
			}
			for surf := range model.Surfaces() {
				log.Println(surf.Name())
			}
		}
	}
}
