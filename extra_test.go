package luaprovider

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestExtraChapterNumbers(t *testing.T) {
	const x = -1 // a chapter without a number
	for _, c := range []struct {
		numbers []float32
		want    []float32
	}{
		{[]float32{10, x}, []float32{10, 10.5}},
		{[]float32{x, 1}, []float32{0.5, 1}},
		{[]float32{10, 10.5, x}, []float32{10, 10.5, 10.6}},
		{[]float32{10, x, 10.5}, []float32{10, 10.6, 10.5}},
		{[]float32{10, x, x, x}, []float32{10, 10.5, 10.6, 10.7}},
		{[]float32{10.5, x}, []float32{10.5, 10.6}},
		{[]float32{5, x, 5.5, 5.6, 5.7, 5.8, 5.9}, []float32{5, 5.91, 5.5, 5.6, 5.7, 5.8, 5.9}},
		{[]float32{5.9, x, 6}, []float32{5.9, 5.91, 6}},
		{[]float32{5.99, x}, []float32{5.99, 5.991}},
		{[]float32{5.95, x}, []float32{5.95, 5.96}},
		{[]float32{1.25, x}, []float32{1.25, 1.35}},
	} {
		var chapters []string
		for _, n := range c.numbers {
			if n == x {
				chapters = append(chapters, `{title = "extra"}`)
			} else {
				chapters = append(chapters, fmt.Sprintf(`{title = "c", number = %v}`, n))
			}
		}
		p := loadTestProvider(t, strings.Replace(testScript, "function VolumeChapters(volume)", "function VolumeChapters(volume)\n\tdo return {"+strings.Join(chapters, ", ")+"} end", 1))
		ctx := context.Background()
		mangas, err := p.SearchMangas(ctx, "x")
		if err != nil {
			t.Fatal(err)
		}
		volumes, err := p.MangaVolumes(ctx, mangas[0])
		if err != nil {
			t.Fatal(err)
		}
		got, err := p.VolumeChapters(ctx, volumes[0])
		if err != nil {
			t.Fatal(err)
		}
		var numbers []float32
		for _, chapter := range got {
			numbers = append(numbers, chapter.Info().Number)
		}
		if !slices.Equal(numbers, c.want) {
			t.Errorf("%v: got %v, want %v", c.numbers, numbers, c.want)
		}
	}
}

func TestExtraChapterNumbersLongRun(t *testing.T) {
	// 40 extras in a row stay distinct (20 past 10000, float32), later ones never take the next
	// whole number
	for start, fit := range map[float32]int{81: 40, 12345: 20} {
		taken := map[int64]bool{scaledNumber(start): true, scaledNumber(start + 1): true}
		last := start
		for i := range 300 {
			n := nextExtraNumber(last, taken)
			if n <= start || n >= start+1 || (i < fit && taken[scaledNumber(n)]) {
				t.Fatalf("after %v, extra %d: %v", start, i, n)
			}
			taken[scaledNumber(n)], last = true, n
		}
	}
}

func TestNextExtraNumberPastFreeWholeNumbers(t *testing.T) {
	// with no chapter after it, a long run (mangapill's "Group 2" chapters) goes on past the
	// whole numbers without taking them
	start := float32(77)
	taken := map[int64]bool{scaledNumber(start): true}
	last := start
	for i := range 300 {
		n := nextExtraNumber(last, taken)
		if taken[scaledNumber(n)] || n <= last || n == float32(int(n)) {
			t.Fatalf("extra %d after %v: %v", i, last, n)
		}
		taken[scaledNumber(n)], last = true, n
	}
}
