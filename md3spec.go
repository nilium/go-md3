package main

import (
	"fmt"
	"github.com/nilium/go-md3/md3"
)

func stringOrEmpty(s, defval string) string {
	if len(s) == 0 {
		return defval
	}
	return s
}

func logModelSpec(model *md3.Model) {
	fmt.Printf("MD3(%s):\n", stringOrEmpty(model.Name(), "no name"))
	fmt.Printf("  Frames: %d\n", model.NumFrames())
	fmt.Printf("  Tags(%d):\n", model.NumTags())
	for tag := range model.Tags() {
		fmt.Printf("    %s\n", tag.Name())
	}
	fmt.Printf("  Surfaces(%d):\n", model.NumSurfaces())
	for surf := range model.Surfaces() {
		fmt.Printf("    %s:\n", stringOrEmpty(surf.Name(), "(no name)"))
		if surf.NumFrames() != model.NumFrames() {
			fmt.Printf("      Frames:    %d\n", surf.NumFrames())
		}
		fmt.Printf("      Vertices:  %d\n", surf.NumVertices())
		fmt.Printf("      Triangles: %d\n", surf.NumTriangles())
		fmt.Printf("      Shaders[%d]:\n", surf.NumShaders())
		for shader := range surf.Shaders() {
			fmt.Printf("        Shader[%d]: %s\n", shader.Index, stringOrEmpty(shader.Name, "(no name)"))
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
