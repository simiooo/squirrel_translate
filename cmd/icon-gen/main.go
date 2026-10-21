// icon-gen packages the OpenPencil PNG export into Windows ICO sizes.
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"os"

	"golang.org/x/image/draw"
)

func main() {
	f, err := os.Open("design/icon.png")
	must(err)
	src, err := png.Decode(f)
	must(err)
	must(f.Close())
	sizes := []int{16, 20, 24, 32, 40, 48, 64, 128, 256}
	var frames [][]byte
	for _, size := range sizes {
		dst := image.NewNRGBA(image.Rect(0, 0, size, size))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
		var frame bytes.Buffer
		must(png.Encode(&frame, dst))
		frames = append(frames, frame.Bytes())
	}
	var ico bytes.Buffer
	for _, n := range []uint16{0, 1, uint16(len(sizes))} {
		must(binary.Write(&ico, binary.LittleEndian, n))
	}
	offset := 6 + 16*len(sizes)
	for i, size := range sizes {
		ico.Write([]byte{byte(size), byte(size), 0, 0})
		for _, n := range []uint16{1, 32} {
			must(binary.Write(&ico, binary.LittleEndian, n))
		}
		for _, n := range []uint32{uint32(len(frames[i])), uint32(offset)} {
			must(binary.Write(&ico, binary.LittleEndian, n))
		}
		offset += len(frames[i])
	}
	for _, frame := range frames {
		ico.Write(frame)
	}
	must(os.WriteFile("design/icon.ico", ico.Bytes(), 0644))
}
func must(err error) {
	if err != nil {
		panic(err)
	}
}
