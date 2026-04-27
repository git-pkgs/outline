package outline

import (
	"io"
	"os"
	"testing"
)

func readSibling(b *testing.B, path string) []byte {
	b.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		b.Skipf("sibling fixture unavailable: %v", err)
	}
	return src
}

func BenchmarkOutlineGoSmall(b *testing.B) {
	src := []byte(goFixture)
	Outline(src, "warm.go")
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	for b.Loop() {
		Outline(src, "sample.go")
	}
}

func BenchmarkOutlineRubySmall(b *testing.B) {
	src := []byte(rubyFixture)
	Outline(src, "warm.rb")
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	for b.Loop() {
		Outline(src, "sample.rb")
	}
}

func BenchmarkOutlineGoSelf(b *testing.B) {
	src, err := os.ReadFile("outline.go")
	if err != nil {
		b.Fatal(err)
	}
	Outline(src, "warm.go")
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	for b.Loop() {
		Outline(src, "outline.go")
	}
}

func BenchmarkOutlineGoLarge(b *testing.B) {
	src := readSibling(b, "../brief/detect/detect.go")
	Outline(src, "warm.go")
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	for b.Loop() {
		Outline(src, "detect.go")
	}
}

func BenchmarkOutlineGoParallel(b *testing.B) {
	src := readSibling(b, "../brief/detect/detect.go")
	Outline(src, "warm.go")
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Outline(src, "detect.go")
		}
	})
}

func BenchmarkPack(b *testing.B) {
	if _, err := os.Stat("../brief"); err != nil {
		b.Skip("../brief not available")
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Pack("../brief", Options{Compress: true}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPackRender(b *testing.B) {
	if _, err := os.Stat("../brief"); err != nil {
		b.Skip("../brief not available")
	}
	r, err := Pack("../brief", Options{Compress: true})
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if err := r.Markdown(io.Discard); err != nil {
			b.Fatal(err)
		}
	}
}
