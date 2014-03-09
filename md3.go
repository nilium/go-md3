package main

import (
	"flag"
	"github.com/nilium/go-md3/md3"
	"io/ioutil"
	"log"
	"os"
)

func dataForPath(path string) ([]byte, error) {
	if path == "-" {
		return ioutil.ReadAll(os.Stdin)
	} else {
		var buf []byte
		var file *os.File
		var err error

		file, err = os.Open(path)
		if err != nil {
			return nil, err
		}

		buf, err = ioutil.ReadAll(file)
		if err != nil {
			if closeErr := file.Close(); closeErr != nil {
				log.Println("Error closing file %q:", path, closeErr)
			}
			return buf, err
		}

		return buf, file.Close()
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
