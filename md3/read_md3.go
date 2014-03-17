package md3

import (
	"bytes"
	"fmt"
	"io"
	"log"
)

const (
	maxQPath       = 64
	maxFrameLength = 16

	md3HeaderIdent  = "IDP3"
	md3SurfaceIdent = md3HeaderIdent
	md3MaxVersion   = 15
	md3VertexSize   = 8
)

type surfaceHeader struct {
	name string

	flags int32 // Unused.

	num_frames    int32
	num_shaders   int32
	num_verts     int32
	num_triangles int32

	ofs_triangles int32
	ofs_shaders   int32
	ofs_st        int32
	ofs_xyznormal int32
	ofs_end       int32
}

type fileHeader struct {
	name string

	version int32
	flags   int32 // Unused.

	num_frames   int32
	num_tags     int32
	num_surfaces int32
	num_skins    int32 // Unused.

	ofs_frames   int32
	ofs_tags     int32
	ofs_surfaces int32
	ofs_eof      int32
}

func Read(data []byte) (*Model, error) {
	var (
		header *fileHeader
		err    error
	)

	r := bytes.NewReader(data)
	header, err = readMD3Header(r)
	if err != nil {
		log.Println("Error reading header:", err)
		return nil, err
	}

	numSurfaces := int(header.num_surfaces)
	model := new(Model)
	surfaces := make([]*Surface, 0, numSurfaces)

	model.name = header.name

	surfaceOutput := readSurfaceList(data[header.ofs_surfaces:], int(header.num_surfaces))
	tagOutput := readTagList(data[header.ofs_tags:], int(header.num_tags), int(header.num_frames))
	frameOutput := readFrameList(data[header.ofs_frames:], int(header.num_frames))

	for surfIndex := 0; surfIndex < numSurfaces; surfIndex++ {
		if surface := <-surfaceOutput; surface != nil {
			surfaces = append(surfaces, surface)
		}
	}

	model.tags = <-tagOutput
	model.frames = <-frameOutput
	model.surfaces = surfaces

	return model, nil
}

func readTagList(data []byte, count int, numFrames int) <-chan []*Tag {
	output := make(chan []*Tag)

	go func(output chan<- []*Tag) {
		// defer close(output)

		r := bytes.NewReader(data)
		tagMap := make(map[string]*Tag)
		tags := make([]*Tag, 0, count)
		var ok bool
		var tag *Tag
		var frame TagFrame
		var name string
		var err error
		var numTagsToRead = count * numFrames

		for i := 0; i < numTagsToRead; i++ {
			name, frame, err = readTag(r)

			if err != nil {
				log.Println("Error reading tag list:", err)
				break
			}

			if tag, ok = tagMap[name]; !ok {
				tag = new(Tag)
				tag.name = name
				tags = append(tags, tag)
				tagMap[name] = tag
			}

			tag.frames = append(tag.frames, frame)
		}

		output <- tags
	}(output)

	return output
}

func readTag(r io.Reader) (string, TagFrame, error) {
	var err error
	var name string
	var frame TagFrame

	name, err = readNulString(r, maxQPath)
	if err != nil {
		return name, frame, err
	}

	vecPointers := [...]*Vec3{
		&frame.Origin,
		&frame.XOrientation,
		&frame.YOrientation,
		&frame.ZOrientation,
	}

	for _, ptr := range vecPointers {
		*ptr, err = readF32Vec3(r)
		if err != nil {
			return name, frame, err
		}
	}

	return name, frame, nil
}

func readMD3Header(r io.Reader) (*fileHeader, error) {
	header := new(fileHeader)
	var ident string
	var err error

	ident, err = readFixedString(r, 4)
	switch {
	case err != nil:
		log.Println("Error reading header identifier", err)
		return nil, err
	case ident != md3HeaderIdent:
		return nil, fmt.Errorf("MD3 header identifier is %q, should be %q", ident, md3HeaderIdent)
	}

	header.version, err = readS32(r)
	switch {
	case err != nil:
		log.Println("Error reading header version", err)
		return nil, err
	case header.version > md3MaxVersion:
		return nil, fmt.Errorf("MD3 header version (%d) exceeds max version (%d)", header.version, md3MaxVersion)
	}

	header.name, err = readNulString(r, maxQPath)
	if err != nil {
		log.Println("Error reading header model name", err)
		return nil, err
	}

	s32Fields := [...]*int32{
		&header.flags,
		&header.num_frames,
		&header.num_tags,
		&header.num_surfaces,
		&header.num_skins,
		&header.ofs_frames,
		&header.ofs_tags,
		&header.ofs_surfaces,
		&header.ofs_eof,
	}

	for _, x := range s32Fields {
		*x, err = readS32(r)
		if err != nil {
			return nil, err
		}
	}

	return header, nil
}

func readSurfaceHeader(r io.Reader) (*surfaceHeader, error) {
	var err error
	var ident string

	header := new(surfaceHeader)

	ident, err = readFixedString(r, 4)
	if err != nil {
		return nil, err
	} else if ident != md3SurfaceIdent {
		return nil, fmt.Errorf("Surface header identifier is %q, should be %q", ident, md3SurfaceIdent)
	}

	header.name, err = readNulString(r, maxQPath)
	if err != nil {
		return nil, err
	}

	s32Fields := [...]*int32{
		&header.flags,
		&header.num_frames,
		&header.num_shaders,
		&header.num_verts,
		&header.num_triangles,
		&header.ofs_triangles,
		&header.ofs_shaders,
		&header.ofs_st,
		&header.ofs_xyznormal,
		&header.ofs_end,
	}

	for _, x := range s32Fields {
		*x, err = readS32(r)
		if err != nil {
			return nil, err
		}
	}

	return header, nil
}

func readVertex(r io.Reader) (Vertex, error) {
	var result Vertex
	var err error

	result.Origin, err = readF16Vec3(r)
	if err != nil {
		return result, err
	}

	result.Normal, err = readSphereNormal(r)

	return result, err
}

func readXYZNormals(r io.Reader, count int) ([]Vertex, error) {
	vertices := make([]Vertex, count)
	for index := range vertices {
		var err error
		vertices[index], err = readVertex(r)
		if err != nil {
			return nil, err
		}
	}
	return vertices, nil
}

func readTriangle(r io.Reader) (Triangle, error) {
	var err error
	var tri Triangle

	for _, ptr := range []*int32{&tri.A, &tri.B, &tri.C} {
		if *ptr, err = readS32(r); err != nil {
			break
		}
	}

	return tri, err
}

func readTriangleList(data []byte, count int) <-chan []Triangle {
	output := make(chan []Triangle)
	go func(output chan<- []Triangle) {
		var err error
		tris := make([]Triangle, count)
		r := bytes.NewReader(data)
		for index := range tris {
			tris[index], err = readTriangle(r)

			if err != nil {
				output <- nil
				log.Println("Error reading triangles:", err)
				return
			}
		}

		output <- tris
	}(output)
	return output
}

func readTexCoordList(data []byte, count int) <-chan []TexCoord {
	output := make(chan []TexCoord)
	go func(output chan<- []TexCoord) {
		var err error
		tcs := make([]TexCoord, count)
		r := bytes.NewReader(data)
		for index := range tcs {
			tc := TexCoord{}

			tc.S, err = readF32(r)
			if err != nil {
				break
			}

			tc.T, err = readF32(r)
			if err != nil {
				break
			}

			tcs[index] = tc
		}

		if err != nil {
			log.Println("Error reading texcoords:", err)
		}

		output <- tcs
	}(output)
	return output
}

func readShaderList(data []byte, count int) <-chan []Shader {
	output := make(chan []Shader)
	go func(output chan<- []Shader) {
		var err error
		shaders := make([]Shader, count)
		r := bytes.NewReader(data)
		for index := range shaders {
			shader := Shader{}

			shader.Name, err = readNulString(r, maxQPath)
			if err != nil {
				break
			}

			shader.Index, err = readS32(r)
			if err != nil {
				break
			}

			shaders[index] = shader
		}

		if err != nil {
			log.Println("Error reading shaders:", err)
		}

		output <- shaders
	}(output)
	return output
}

func readSurfaceList(data []byte, count int) <-chan *Surface {
	output := make(chan *Surface)
	go func(data []byte, output chan<- *Surface) {
		for index := 0; index < count; index++ {
			reader := bytes.NewReader(data[:])
			header, err := readSurfaceHeader(reader)
			if err != nil {
				log.Println("Error reading surface header:", err)
				break
			}

			go func(data []byte) {
				surf, err := readSurface(header, data)
				if err != nil {
					log.Printf("Error reading surface %q: %s\n", header.name, err)
				}

				surf.name = header.name
				surf.numFrames = int(header.num_frames)

				output <- surf
			}(data)

			data = data[header.ofs_end:]
		}
	}(data, output)
	return output
}

func readSurface(h *surfaceHeader, data []byte) (*Surface, error) {
	surface := new(Surface)

	triangleOutput := readTriangleList(data[h.ofs_triangles:], int(h.num_triangles))
	shaderOutput := readShaderList(data[h.ofs_shaders:], int(h.num_shaders))
	texcoordOutput := readTexCoordList(data[h.ofs_st:], int(h.num_verts))
	verticesOutput := readVertexFrames(data[h.ofs_xyznormal:], int(h.num_verts), int(h.num_frames))

	surface.vertices = <-verticesOutput
	surface.triangles = <-triangleOutput
	surface.texcoords = <-texcoordOutput
	surface.shaders = <-shaderOutput

	return surface, nil
}

type frameAndVertices struct {
	index    int
	vertices []Vertex
}

func readVertexFrames(data []byte, numVertices, numFrames int) <-chan [][]Vertex {
	output := make(chan [][]Vertex)

	go func(data []byte, output chan<- [][]Vertex) {
		var (
			frameVertices = make([][]Vertex, numFrames)
			frameReceiver = make(chan frameAndVertices)
			frameSize     = numVertices * md3VertexSize
		)

		for frame := range frameVertices {
			go func(frame int, data []byte) {
				reader := bytes.NewReader(data)
				vertices, err := readXYZNormals(reader, numVertices)
				if err != nil {
					log.Println("Error reading vertices:", err)
				}
				frameReceiver <- frameAndVertices{frame, vertices}
			}(frame, data[:frameSize])
			data = data[frameSize:]
		}

		for _ = range frameVertices {
			pack := <-frameReceiver
			frameVertices[pack.index] = pack.vertices
		}

		output <- frameVertices
	}(data, output)

	return output
}

func readFrame(r io.Reader) (*Frame, error) {
	var err error
	frame := new(Frame)

	frame.name, err = readNulString(r, maxFrameLength)
	if err != nil {
		return nil, err
	}

	for _, ptr := range []*Vec3{&frame.min, &frame.max, &frame.origin} {
		*ptr, err = readF32Vec3(r)
		if err != nil {
			return nil, err
		}
	}

	frame.radius, err = readF32(r)
	if err != nil {
		return nil, err
	}

	return frame, nil
}

func readFrameList(data []byte, count int) <-chan []*Frame {
	output := make(chan []*Frame)

	go func(output chan<- []*Frame) {
		// defer close(output)

		var err error
		reader := bytes.NewReader(data)
		frames := make([]*Frame, count)

		for index := range frames {
			frames[index], err = readFrame(reader)

			if err != nil {
				log.Println("Error reading frame list:", err)
				frames = nil
				break
			}
		}

		output <- frames
	}(output)

	return output
}
