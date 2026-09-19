package decorender

import (
	"io"
	"os"
	"sync"
	"testing"
)

func TestFull(t *testing.T) {
	d, err := NewRenderer("./test.yaml", &Options{LocalFiles: os.DirFS(".")})
	if err != nil {
		t.Errorf("unexpected error while yaml parse: %v", err)
		return
	}

	err = d.RenderToFile(nil, "test.png", &RenderOptions{
		UseSample: true,
	})

	if err != nil {
		t.Errorf("unexpected error while rendering: %v", err)
	}
}

// A renderer is documented as safe to use from several goroutines. It used to
// hand the same font.Face to all of them, which races on the face's glyph
// buffer and panics inside image/draw. Run with -race.
func TestConcurrentRender(t *testing.T) {
	d, err := NewRenderer("./test.yaml", &Options{LocalFiles: os.DirFS(".")})
	if err != nil {
		t.Fatalf("unexpected error while yaml parse: %v", err)
	}

	const goroutines, iterations = 8, 20

	errs := make(chan error, goroutines*iterations)

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if err := d.RenderAndWrite(nil, EncodeFormatNone, nil, &RenderOptions{UseSample: true}); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent render failed: %v", err)
	}
}

// Every other entry point tolerates a nil *RenderOptions, and JPEG encoding
// used to dereference it while picking a quality.
func TestEncodeWithNilRenderOptions(t *testing.T) {
	d, err := NewRendererWithTemplate([]byte("size: 50 50\nbkgColor: salmon\n"), nil)
	if err != nil {
		t.Fatalf("unexpected error while yaml parse: %v", err)
	}

	for _, format := range []EncodeFormat{EncodeFormatNone, EncodeFormatPNG, EncodeFormatJPG} {
		if err := d.RenderAndWrite(nil, format, io.Discard, nil); err != nil {
			t.Errorf("format %v with nil options: %v", format, err)
		}
	}
}

// Two renderers must not see each other's font faces.
func TestRenderersDoNotShareFonts(t *testing.T) {
	// Declares no faces, so "Inter" is unknown to it.
	plain, err := NewRendererWithTemplate([]byte("size: 50 50\ntext: hello\n"), nil)
	if err != nil {
		t.Fatalf("unexpected error while yaml parse: %v", err)
	}
	if err = plain.RenderAndWrite(nil, EncodeFormatNone, nil, nil); err != nil {
		t.Fatalf("rendering with the default font: %v", err)
	}

	// Asking for an undeclared family must fail rather than silently pick up
	// a face some other renderer happened to load first.
	missing, err := NewRendererWithTemplate([]byte("size: 50 50\nfont: Inter 20\ntext: hello\n"), nil)
	if err != nil {
		t.Fatalf("unexpected error while yaml parse: %v", err)
	}
	if err = missing.RenderAndWrite(nil, EncodeFormatNone, nil, nil); err == nil {
		t.Error("expected an error for a font family that was never declared")
	}
}

// With the image cache off, scaled images are owned by the caller and returned
// to the buffer pool after drawing. Getting that ownership wrong recycles a
// buffer another goroutine is still reading from.
func TestConcurrentRenderWithoutImageCache(t *testing.T) {
	d, err := NewRenderer("./test.yaml", &Options{
		LocalFiles:   os.DirFS("."),
		NoImageCache: true,
	})
	if err != nil {
		t.Fatalf("unexpected error while yaml parse: %v", err)
	}

	const goroutines, iterations = 4, 3

	errs := make(chan error, goroutines*iterations)

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if err := d.RenderAndWrite(nil, EncodeFormatNone, nil, &RenderOptions{UseSample: true}); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("render without image cache failed: %v", err)
	}
}

func BenchmarkRender(b *testing.B) {
	d, err := NewRenderer("./test.yaml", &Options{LocalFiles: os.DirFS(".")})
	if err != nil {
		b.Fatalf("unexpected error while yaml parse: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := d.RenderAndWrite(nil, EncodeFormatNone, nil, &RenderOptions{UseSample: true}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderParallel(b *testing.B) {
	d, err := NewRenderer("./test.yaml", &Options{LocalFiles: os.DirFS(".")})
	if err != nil {
		b.Fatalf("unexpected error while yaml parse: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := d.RenderAndWrite(nil, EncodeFormatNone, nil, &RenderOptions{UseSample: true}); err != nil {
				b.Error(err)
				return
			}
		}
	})
}
