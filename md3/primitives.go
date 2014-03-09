package md3

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
)

const md3XYZFixedScale float32 = 1.0 / 64.0

type Vec3 struct {
	X, Y, Z float32
}

func readU8(r io.Reader) (uint8, error) {
	b := make([]byte, 1)
	_, err := r.Read(b)
	return b[0], err
}

func readS16(r io.Reader) (int16, error) {
	var err error
	var n int
	var result int16

	b := make([]byte, 2)

	n, err = r.Read(b)
	if err != nil {
		return result, err
	} else if n != 2 {
		return result, fmt.Errorf("Failed to read int16")
	}

	err = binary.Read(bytes.NewReader(b), binary.LittleEndian, &result)
	if err != nil {
		log.Println("binary.Read failed:", err)
	}
	return result, err
}

func readS32(r io.Reader) (int32, error) {
	var err error
	var n int
	var result int32

	b := make([]byte, 4)

	n, err = r.Read(b)
	if err != nil {
		return result, err
	} else if n != 4 {
		return result, fmt.Errorf("Failed to read int32")
	}

	err = binary.Read(bytes.NewReader(b), binary.LittleEndian, &result)
	if err != nil {
		log.Println("binary.Read failed:", err)
	}
	return result, err
}

func readF32(r io.Reader) (float32, error) {
	var err error
	var n int
	var result float32

	b := make([]byte, 4)

	n, err = r.Read(b)
	if err != nil {
		return result, err
	} else if n != 4 {
		return result, fmt.Errorf("Failed to read float32")
	}

	err = binary.Read(bytes.NewReader(b), binary.LittleEndian, &result)
	if err != nil {
		log.Println("binary.Read failed:", err)
	}
	return result, err
}

func readF16(r io.Reader) (float32, error) {
	s16, err := readS16(r)
	return float32(s16) * md3XYZFixedScale, err
}

func readF32Vec3(r io.Reader) (Vec3, error) {
	x, err := readF32(r)
	if err != nil {
		return Vec3{}, err
	}
	y, err := readF32(r)
	if err != nil {
		return Vec3{}, err
	}
	z, err := readF32(r)
	if err != nil {
		return Vec3{}, err
	}
	return Vec3{x, y, z}, nil
}

func readF16Vec3(r io.Reader) (Vec3, error) {
	x, err := readF16(r)
	if err != nil {
		return Vec3{}, err
	}
	y, err := readF16(r)
	if err != nil {
		return Vec3{}, err
	}
	z, err := readF16(r)
	if err != nil {
		return Vec3{}, err
	}
	return Vec3{x, y, z}, nil
}

func readSphereNormal(r io.Reader) (Vec3, error) {
	var result Vec3
	var zenith uint8
	var azimuth uint8
	var err error

	zenith, err = readU8(r)
	if err != nil {
		return result, err
	}

	azimuth, err = readU8(r)
	if err != nil {
		return result, err
	}

	latitude := float64(zenith) * (math.Pi * 2.0) / 255.0
	longitude := float64(azimuth) * (math.Pi * 2.0) / 255.0
	latsin := math.Sin(latitude)

	result.X = float32(math.Cos(longitude) * latsin)
	result.Y = float32(math.Sin(longitude) * latsin)
	result.Z = float32(math.Cos(latitude))

	return result, nil
}

func readNulString(r io.Reader, maxLen int) (string, error) {
	buf := make([]byte, maxLen)
	n, err := r.Read(buf)
	if err != nil {
		return "", err
	} else if n != maxLen {
		return "", fmt.Errorf("Couldn't read NUL string of length %d, only got %d bytes", maxLen, n)
	}

	if index := bytes.IndexByte(buf, 0); index != -1 {
		buf = buf[:index]
	}

	return bytes.NewBuffer(buf).String(), nil
}

func readFixedString(r io.Reader, length int) (string, error) {
	buf := make([]byte, length)
	n, err := r.Read(buf)
	if err != nil {
		return "", err
	} else if n != length {
		return "", fmt.Errorf("Couldn't read string of length %d, only got %d bytes", length, n)
	}

	return bytes.NewBuffer(buf).String(), nil
}
