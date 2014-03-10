package md3

type Frame struct {
	name   string
	min    Vec3
	max    Vec3
	origin Vec3
	radius float32
}

func (f *Frame) Name() string {
	return f.name
}

func (f *Frame) Min() Vec3 {
	return f.min
}

func (f *Frame) Max() Vec3 {
	return f.max
}

func (f *Frame) Origin() Vec3 {
	return f.origin
}

func (f *Frame) Radius() float32 {
	return f.radius
}

type TagFrame struct {
	Origin       Vec3
	XOrientation Vec3
	YOrientation Vec3
	ZOrientation Vec3
}

type Tag struct {
	name   string
	frames []TagFrame
}

func (t *Tag) Name() string {
	return t.name
}

func (t *Tag) NumFrames() int {
	return len(t.frames)
}

func (t *Tag) Frame(frame int) TagFrame {
	return t.frames[frame]
}

func (t *Tag) Frames() <-chan TagFrame {
	output := make(chan TagFrame)
	go func(t *Tag, output chan<- TagFrame) {
		for _, frame := range t.frames {
			output <- frame
		}
		close(output)
	}(t, output)
	return output
}

type Triangle struct {
	A, B, C int32
}

type Vertex struct {
	Origin Vec3
	Normal Vec3
}

type TexCoord struct {
	S, T float32
}

type Shader struct {
	Name  string
	Index int32
}

type Surface struct {
	name      string
	numFrames int
	shaders   []Shader
	triangles []Triangle
	texcoords []TexCoord
	vertices  [][]Vertex
}

func (s *Surface) Name() string {
	return s.name
}

// NumFrames returns the number of frames of vertex data held by the surface.
// This should be equal to its parent model's NumFrames result.
func (s *Surface) NumFrames() int {
	return s.numFrames
}

func (s *Surface) NumTriangles() int {
	return len(s.triangles)
}

func (s *Surface) NumVertices() int {
	return len(s.texcoords)
}

func (s *Surface) NumShaders() int {
	return len(s.shaders)
}

func (s *Surface) Triangle(index int) Triangle {
	return s.triangles[index]
}

func (s *Surface) Triangles() <-chan Triangle {
	output := make(chan Triangle)
	go func(s *Surface, output chan<- Triangle) {
		for _, tri := range s.triangles {
			output <- tri
		}
		close(output)
	}(s, output)
	return output
}

func (s *Surface) Vertex(frame, index int) Vertex {
	return s.vertices[frame][index]
}

func (s *Surface) Vertices(frame int) <-chan Vertex {
	output := make(chan Vertex)
	go func(s *Surface, output chan<- Vertex) {
		for _, vert := range s.vertices[frame] {
			output <- vert
		}
		close(output)
	}(s, output)
	return output
}

func (s *Surface) TexCoord(index int) TexCoord {
	return s.texcoords[index]
}

func (s *Surface) TexCoords() <-chan TexCoord {
	output := make(chan TexCoord)
	go func(s *Surface, output chan<- TexCoord) {
		for _, texcoord := range s.texcoords {
			output <- texcoord
		}
		close(output)
	}(s, output)
	return output
}

func (s *Surface) Shader(index int) Shader {
	return s.shaders[index]
}

func (s *Surface) Shaders() <-chan Shader {
	output := make(chan Shader)
	go func(s *Surface, output chan<- Shader) {
		for _, shader := range s.shaders {
			output <- shader
		}
		close(output)
	}(s, output)
	return output
}

type Model struct {
	name     string
	frames   []*Frame
	tags     []*Tag
	surfaces []*Surface
}

func (m *Model) Name() string {
	return m.name
}

func (m *Model) NumSurfaces() int {
	return len(m.surfaces)
}

func (m *Model) NumFrames() int {
	return len(m.frames)
}

func (m *Model) NumTags() int {
	return len(m.tags)
}

func (m *Model) Surface(index int) *Surface {
	return m.surfaces[index]
}

func (m *Model) Frame(index int) *Frame {
	return m.frames[index]
}

func (m *Model) Tag(index int) *Tag {
	return m.tags[index]
}

func (m *Model) Surfaces() <-chan *Surface {
	output := make(chan *Surface)
	go func(m *Model, output chan<- *Surface) {
		for _, surface := range m.surfaces {
			output <- surface
		}
		close(output)
	}(m, output)
	return output
}

func (m *Model) Frames() <-chan *Frame {
	output := make(chan *Frame)
	go func(m *Model, output chan<- *Frame) {
		for _, frame := range m.frames {
			output <- frame
		}
		close(output)
	}(m, output)
	return output
}

func (m *Model) Tags() <-chan *Tag {
	output := make(chan *Tag)
	go func(m *Model, output chan<- *Tag) {
		for _, tag := range m.tags {
			output <- tag
		}
		close(output)
	}(m, output)
	return output
}
