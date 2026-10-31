package main

import (
	"bytes"
	"image/png"
	"strings"
	"testing"
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
