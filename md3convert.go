package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/nilium/go-md3/md3"
	"io"
	"log"
	"os"
	"path"
)

const maxWriters = 8

var (
	outputPath = flag.String("o", ".", "Specify the output directory for converted OBJ file(s).")
	flipUVs    = flag.Bool("flipUVs", true, "Enables flipping UV coordinates vertically on output.")
	swapYZ     = flag.Bool("swapYZ", true, "Enables swapping Y and Z axes on output.")
)

type surfaceStringPair struct {
	surface *md3.Surface
	value   string
}

type surfaceMappingFunc func(surf *md3.Surface, output chan<- surfaceStringPair)

func surfaceTriangleList(surf *md3.Surface, baseVertex int, output chan<- surfaceStringPair) {
	var err error
	buffer := new(bytes.Buffer)
	triCount := surf.NumTriangles()
	for triIndex := 0; triIndex < triCount; triIndex++ {
		tri := surf.Triangle(triIndex)
		if *swapYZ {
			_, err = fmt.Fprintf(buffer, "f %[1]d/%[1]d/%[1]d %[2]d/%[2]d/%[2]d %[3]d/%[3]d/%[3]d\n",
				baseVertex+int(tri.A), baseVertex+int(tri.B), baseVertex+int(tri.C))
		} else {
			_, err = fmt.Fprintf(buffer, "f %[3]d/%[3]d/%[3]d %[2]d/%[2]d/%[2]d %[1]d/%[1]d/%[1]d\n",
				baseVertex+int(tri.A), baseVertex+int(tri.B), baseVertex+int(tri.C))
		}
		if err != nil {
			panic(err)
		}
	}

	output <- surfaceStringPair{surf, buffer.String()}
}

func surfacePosNormList(surf *md3.Surface, frame int, output chan<- surfaceStringPair) {
	buffer := new(bytes.Buffer)
	vertCount := surf.NumVertices()

	for vertIndex := 0; vertIndex < vertCount; vertIndex++ {
		posNorm := surf.Vertex(frame, vertIndex)
		if *swapYZ {
			var t float32
			t = posNorm.Origin.Y
			posNorm.Origin.Y = posNorm.Origin.Z
			posNorm.Origin.Z = t
			t = posNorm.Normal.Y
			posNorm.Normal.Y = posNorm.Normal.Z
			posNorm.Normal.Z = t
		}
		_, err := fmt.Fprintf(buffer,
			"v %f %f %f\nvn %f %f %f\n",
			posNorm.Origin.X, posNorm.Origin.Y, posNorm.Origin.Z,
			posNorm.Normal.X, posNorm.Normal.Y, posNorm.Normal.Z)
		if err != nil {
			panic(err)
		}
	}

	output <- surfaceStringPair{surf, buffer.String()}
}

func surfaceTexCoordList(surf *md3.Surface, output chan<- surfaceStringPair) {
	buffer := new(bytes.Buffer)
	vertCount := surf.NumVertices()

	for vertIndex := 0; vertIndex < vertCount; vertIndex++ {
		texCoord := surf.TexCoord(vertIndex)

		if *flipUVs {
			texCoord.T = 1.0 - texCoord.T
		}

		_, err := fmt.Fprintf(buffer, "vt %f %f\n",
			texCoord.S, texCoord.T)
		if err != nil {
			panic(err)
		}
	}

	output <- surfaceStringPair{surf, buffer.String()}
}

// forAllSurfaces loops over all surfaces of the given model and concurrently
// launches functions mapping the surfaces to string values. The results of the
// mapping operation are handled by a single goroutine to prevent writes from
// potentially parallel sources.
func forAllSurfaces(model *md3.Model, fn surfaceMappingFunc) map[*md3.Surface]string {

	numSurfs := model.NumSurfaces()

	pairs := make(map[*md3.Surface]string, numSurfs)
	builtPairs := make(chan surfaceStringPair)
	waitSignal := make(chan bool)

	for surfaceIndex := 0; surfaceIndex < numSurfs; surfaceIndex++ {
		surf := model.Surface(surfaceIndex)
		go fn(surf, builtPairs)
	}

	go func(count int, builtPairs <-chan surfaceStringPair, waitSignal chan<- bool) {
		for ; count > 0; count-- {
			pair := <-builtPairs
			pairs[pair.surface] = pair.value
		}
		waitSignal <- true
	}(numSurfs, builtPairs, waitSignal)

	<-waitSignal

	return pairs
}

// objTriangleLists produces a map of strings that can be used to write the
// triangle lists for all frames of an MD3's surface. Special variation on the
// forAllSurfaces function that handles base vertices (i.e., sequence is
// much more important).
func objTriangleLists(model *md3.Model) map[*md3.Surface]string {
	numSurfs := model.NumSurfaces()

	triStrings := make(map[*md3.Surface]string, numSurfs)
	builtPairs := make(chan surfaceStringPair)
	waitSignal := make(chan bool)

	baseVertex := 1
	for surfaceIndex := 0; surfaceIndex < numSurfs; surfaceIndex++ {
		surf := model.Surface(surfaceIndex)
		go surfaceTriangleList(surf, baseVertex, builtPairs)
		baseVertex += surf.NumVertices()
	}

	go func(count int, builtPairs <-chan surfaceStringPair, waitSignal chan<- bool) {
		for ; count > 0; count-- {
			pair := <-builtPairs
			triStrings[pair.surface] = pair.value
		}
		waitSignal <- true
	}(numSurfs, builtPairs, waitSignal)

	<-waitSignal

	return triStrings
}

func objTexCoordLists(model *md3.Model) map[*md3.Surface]string {
	return forAllSurfaces(model, surfaceTexCoordList)
}

func objPosNormLists(model *md3.Model, frame int) map[*md3.Surface]string {
	return forAllSurfaces(model, func(surf *md3.Surface, output chan<- surfaceStringPair) {
		surfacePosNormList(surf, frame, output)
	})
}

func writeOBJSurface(w io.Writer, surf *md3.Surface, posNorms, texCoords, triangles map[*md3.Surface]string) error {
	fmt.Fprintf(w, "g %s\n", surf.Name())
	n, err := io.WriteString(w, posNorms[surf])
	if err != nil {
		return err
	} else if n < len(posNorms[surf]) {
		return fmt.Errorf("Error writing positions and normals: only %d of %d bytes written")
	}
	n, err = io.WriteString(w, texCoords[surf])
	if err != nil {
		return err
	} else if n < len(texCoords[surf]) {
		return fmt.Errorf("Error writing texcoords: only %d of %d bytes written")
	}
	n, err = io.WriteString(w, triangles[surf])
	if n < len(triangles[surf]) {
		return fmt.Errorf("Error writing triangles: only %d of %d bytes written")
	}
	return err
}

func performConvertModel(modelPath string, model *md3.Model, signal chan<- bool, writeQueue chan<- func()) {

	waitSignal := make(chan bool)
	triLists := objTriangleLists(model)
	tcLists := objTexCoordLists(model)

	var _ = triLists
	var _ = tcLists

	name := path.Base(modelPath)
	name = name[:len(name)-len(path.Ext(name))]
	dir := path.Clean(*outputPath)
	os.MkdirAll(dir, 0755)

	numFrames := model.NumFrames()

	for frame := 0; frame < numFrames; frame++ {
		go func(frame int) {
			writeQueue <- func() {
				defer func(waitSignal chan<- bool) {
					go func() { waitSignal <- true }()
				}(waitSignal)

				var err error
				var file *os.File
				outName := fmt.Sprintf("%s+%d.obj", name, frame)
				outPath := path.Join(dir, outName)

				file, err = os.Create(outPath)
				if err != nil {
					log.Println("Error creating", outPath, "from", modelPath, "->", err)
					return
				}
				defer file.Close()

				_, err = fmt.Fprintf(file, "o %s\n", model.Name())
				if err != nil {
					log.Println("Error writing header for", outPath, "from", modelPath, "->", err)
				}

				numSurfaces := model.NumSurfaces()
				posNorms := objPosNormLists(model, frame)
				for surfIndex := 0; surfIndex < numSurfaces; surfIndex++ {
					surf := model.Surface(surfIndex)
					err = writeOBJSurface(file, surf, posNorms, tcLists, triLists)
					if err != nil {
						log.Println("Error writing surface for", outPath, "from", modelPath, "->", err)
						return
					}
				}
			}
		}(frame)
	}

	for frame := 0; frame < numFrames; frame++ {
		<-waitSignal
	}

	signal <- true
}

// writeFunnelProcess simply loops over the input channel and calls each
// function it receives. It's used only as a means of funneling file creation
// and writing through a limited number of ports to prevent exceeding the number
// of open files allowed per-process by the OS.
func writeFunnelProcess(input <-chan func()) {
	for fn := range input {
		fn()
	}
}

func convertModelsToOBJProcess(input <-chan *modelPathPair, done chan<- bool) {
	count := 0
	doneSignal := make(chan bool)
	writeQueue := make(chan func())

	for index := 0; index < maxWriters; index++ {
		go writeFunnelProcess(writeQueue)
	}

	for pair := range input {
		go performConvertModel(pair.path, pair.model, doneSignal, writeQueue)
		count++
	}

	for signals := 0; signals < count; signals++ {
		<-doneSignal
	}

	done <- true
}

func convertModelsToOBJ() (chan<- *modelPathPair, <-chan bool) {
	input := make(chan *modelPathPair)
	done := make(chan bool)

	go convertModelsToOBJProcess(input, done)

	return input, done
}
