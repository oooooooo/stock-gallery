package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestRunWritesErrorsToStderrSoStdoutStaysJSON(t *testing.T) {
	t.Parallel()

	fetch := func(symbol string) ([]StockInfo, error) {
		if symbol == "0000.T" {
			return nil, errors.New("no data found for 0000.T")
		}

		return []StockInfo{{Name: "Toyota Motor Corporation", Data: []StockData{{Date: "2026-10"}}}}, nil
	}

	var stdout, stderr bytes.Buffer

	err := run([]string{"fetch", "7203.T,0000.T"}, &stdout, &stderr, fetch)
	if err != nil {
		t.Fatalf("run returned %v", err)
	}

	var stocks map[string][]StockInfo
	if err := json.Unmarshal(stdout.Bytes(), &stocks); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}

	if len(stocks) != 1 || stocks["7203"] == nil {
		t.Fatalf("stdout = %v, want only 7203", stocks)
	}

	if !strings.Contains(stderr.String(), "0000.T") {
		t.Fatalf("stderr = %q, want the failed symbol", stderr.String())
	}
}

func TestRunRequiresSymbols(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer

	err := run([]string{"fetch"}, &stdout, &stderr, nil)
	if err == nil {
		t.Fatal("run without symbols returned nil")
	}
}
