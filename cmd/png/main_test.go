package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"stock/pkg/render"
)

const sample = `{"7203": [{"Name": "Toyota Motor Corporation", "Data": [
	{"Date": "2025-12", "Low": 2000, "High": 2100, "Volume": 1000},
	{"Date": "2026-01", "Low": 2050, "High": 2200, "Volume": 2000}
]}]}`

func TestRunWritesAPNG(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer

	err := run([]string{"png", "-date", "2026-10-30"}, strings.NewReader(sample), &stdout)
	if err != nil {
		t.Fatal(err)
	}

	img, err := png.Decode(&stdout)
	if err != nil {
		t.Fatalf("stdout is not a PNG: %v", err)
	}

	if img.Bounds().Dx() == 0 || img.Bounds().Dy() == 0 {
		t.Fatalf("PNG is empty: %v", img.Bounds())
	}
}

const fourStocks = `{
	"7203": [{"Name": "Toyota", "Data": [{"Date": "2025-12", "Low": 2000, "High": 2100, "Volume": 1000}, {"Date": "2026-01", "Low": 2050, "High": 2180, "Volume": 2000}]}],
	"9432": [{"Name": "NTT", "Data": [{"Date": "2025-12", "Low": 150, "High": 160, "Volume": 1000}, {"Date": "2026-01", "Low": 200, "High": 240, "Volume": 2000}]}],
	"9501": [{"Name": "TEPCO", "Data": [{"Date": "2025-12", "Low": 500, "High": 600, "Volume": 1000}, {"Date": "2026-01", "Low": 550, "High": 680, "Volume": 2000}]}],
	"9064": [{"Name": "Yamato", "Data": [{"Date": "2025-12", "Low": 2000, "High": 2100, "Volume": 1000}, {"Date": "2026-01", "Low": 2050, "High": 2180, "Volume": 2000}]}]
}`

func runPNG(t *testing.T, args ...string) image.Rectangle {
	t.Helper()

	return decodePNG(t, args...).Bounds()
}

func decodePNG(t *testing.T, args ...string) image.Image {
	t.Helper()

	var stdout bytes.Buffer

	err := run(append([]string{"png", "-date", "2026-10-30"}, args...), strings.NewReader(fourStocks), &stdout)
	if err != nil {
		t.Fatal(err)
	}

	img, err := png.Decode(&stdout)
	if err != nil {
		t.Fatalf("stdout is not a PNG: %v", err)
	}

	return img
}

func highLinePixels(img image.Image, area image.Rectangle) int {
	count := 0

	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			if color.RGBAModel.Convert(img.At(x, y)) == render.HighColor {
				count++
			}
		}
	}

	return count
}

func TestRunDrawsEachOfFourStocksInItsOwnQuadrant(t *testing.T) {
	t.Parallel()

	img := decodePNG(t, "-columns", "2")
	width, height := img.Bounds().Dx(), img.Bounds().Dy()
	middleX, middleY := width/2, 72+360 // below the header of 72 px and one row of 360 px

	for name, quadrant := range map[string]image.Rectangle{
		"top left":     image.Rect(0, 72, middleX, middleY),
		"top right":    image.Rect(middleX, 72, width, middleY),
		"bottom left":  image.Rect(0, middleY, middleX, height),
		"bottom right": image.Rect(middleX, middleY, width, height),
	} {
		if got := highLinePixels(img, quadrant); got < 50 {
			t.Errorf("the %s quadrant has %d pixels of the high line, want a chart", name, got)
		}
	}
}

func TestRunDrawsFourStocksInTwoColumnsAsTwoRows(t *testing.T) {
	t.Parallel()

	inRow, inTwoColumns := runPNG(t), runPNG(t, "-columns", "2")

	// two panels of 640 px and two rows of 360 px, instead of one row of four panels
	if want := image.Rect(0, 0, 1328, 816); inTwoColumns != want {
		t.Fatalf("four stocks in two columns are %v, want %v; in one row they are %v", inTwoColumns, want, inRow)
	}
}

func TestRunDrawsFourColumnsByDefault(t *testing.T) {
	t.Parallel()

	if inRow, inFour := runPNG(t), runPNG(t, "-columns", "4"); inRow != inFour {
		t.Fatalf("the default is %v, want the four columns (%v) drawn before -columns existed", inRow, inFour)
	}
}

func TestRunRejectsColumnsBelowOne(t *testing.T) {
	t.Parallel()

	for _, columns := range []string{"0", "-1"} {
		var stdout bytes.Buffer

		err := run([]string{"png", "-columns", columns}, strings.NewReader(fourStocks), &stdout)
		if err == nil {
			t.Errorf("run returned nil for %s columns", columns)
		}

		if stdout.Len() != 0 {
			t.Errorf("stdout has %d bytes for %s columns, want nothing on failure", stdout.Len(), columns)
		}
	}
}

func TestRunRejectsInputThatIsNotJSON(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer

	err := run([]string{"png"}, strings.NewReader("Error fetching data"), &stdout)
	if err == nil {
		t.Fatal("run returned nil for input that is not JSON")
	}

	if stdout.Len() != 0 {
		t.Fatalf("stdout has %d bytes, want nothing on failure", stdout.Len())
	}
}
