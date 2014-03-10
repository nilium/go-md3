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

	model := new(Model)
	surfaces := make([]*Surface, 0, 1)

	surfaceOutput := readSurfaceList(data[header.ofs_surfaces:], int(header.num_surfaces))
	tagOutput := readTagList(data[header.ofs_tags:], int(header.num_tags))

	for completions := header.num_surfaces + 1; completions > 0; completions-- {
		select {
		case surface := <-surfaceOutput:
			if surface != nil {
				surfaces = append(surfaces, surface)
			}
		case tags := <-tagOutput:
			model.tags = tags
		}
	}

	model.surfaces = surfaces

	return model, nil
}

func readTagList(data []byte, count int) <-chan []Tag {
	output := make(chan []Tag)

	go func() {
		r := bytes.NewReader(data)
		tags := make([]Tag, count)

		for x := range tags {
			var err error
			tags[x], err = readTag(r)
			if err != nil {
				log.Println("Error reading tag list:", err)
				break
			}
		}

		output <- tags
	}()

	return output
}

func readTag(r io.Reader) (Tag, error) {
	var err error
	tag := Tag{}

	tag.Name, err = readNulString(r, maxQPath)
	if err != nil {
		return tag, err
	}

	vecPointers := []*Vec3{
		&tag.Origin,
		&tag.XOrientation,
		&tag.YOrientation,
		&tag.ZOrientation,
	}

	for _, ptr := range vecPointers {
		*ptr, err = readF32Vec3(r)
		if err != nil {
			return tag, err
		}
	}

	return tag, nil
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

	s32Fields := []*int32{
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

	s32Fields := []*int32{
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

func readTriangleList(data []byte, count int) <-chan []Triangle {
	output := make(chan []Triangle)
	go func() {
		var err error
		tris := make([]Triangle, count)
		r := bytes.NewReader(data)
		for index := range tris {
			tri := Triangle{}

			tri.A, err = readS32(r)
			if err != nil {
				break
			}

			tri.B, err = readS32(r)
			if err != nil {
				break
			}

			tri.C, err = readS32(r)
			if err != nil {
				break
			}

			tris[index] = tri
		}

		if err != nil {
			log.Println("Error reading triangles:", err)
		}

		output <- tris
	}()
	return output
}

func readTexCoordList(data []byte, count int) <-chan []TexCoord {
	output := make(chan []TexCoord)
	go func() {
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
	}()
	return output
}

func readShaderList(data []byte, count int) <-chan []Shader {
	output := make(chan []Shader)
	go func() {
		var err error
		tcs := make([]Shader, count)
		r := bytes.NewReader(data)
		for index := range tcs {
			shader := Shader{}

			shader.Name, err = readNulString(r, maxQPath)
			if err != nil {
				break
			}

			shader.Index, err = readS32(r)
			if err != nil {
				break
			}

			tcs[index] = shader
		}

		if err != nil {
			log.Println("Error reading shaders:", err)
		}

		output <- tcs
	}()
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
	var err error
	surface := new(Surface)
	completions := h.num_frames + 3

	_ = err

	triangleOutput := readTriangleList(data[h.ofs_triangles:], int(h.num_triangles))
	shaderOutput := readShaderList(data[h.ofs_shaders:], int(h.num_shaders))
	texcoordOutput := readTexCoordList(data[h.ofs_st:], int(h.num_verts))

	vertexCompletion := make(chan func(frames [][]Vertex))
	surface.vertices = make([][]Vertex, h.num_frames)
	vdataStart := data[h.ofs_xyznormal:]
	vdataSize := int(h.num_verts) * md3VertexSize

	for frame := range surface.vertices {
		from := frame * vdataSize
		to := (frame + 1) * vdataSize
		vdata := vdataStart[from:to]

		go func(index int, data []byte) {
			var localErr error
			var vertices []Vertex
			vertReader := bytes.NewReader(data)
			vertices, localErr = readXYZNormals(vertReader, int(h.num_verts))

			if localErr != nil {
				log.Println("Error reading vertex data: ", localErr)
				vertexCompletion <- nil
				return
			}

			vertexCompletion <- func(frames [][]Vertex) { frames[index] = vertices }
		}(frame, vdata)
	}

	for completions > 0 {
		select {
		case vfunc := <-vertexCompletion:
			vfunc(surface.vertices)
		case tris := <-triangleOutput:
			surface.triangles = tris
		case tcs := <-texcoordOutput:
			surface.texcoords = tcs
		case shaders := <-shaderOutput:
			surface.shaders = shaders
		}
		completions--
	}

	return surface, nil
}
