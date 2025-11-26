package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/guptarohit/asciigraph"

	stock "stock/pkg/types"
)

const (
	requiredArgsForCode = 3
	graphHeight         = 15

	minArgs         = 2
	indexStockCodeArg = 2
)

// usage prints how to use this CLI.
func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <path-to-chart.json> [stock-code]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Example: %s chart.json 2220\n", os.Args[0])
}

// loadStockData reads JSON from r and decodes it into a map.
func loadStockData(r io.Reader) (map[string][]stock.StockInfo, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read json: %w", err)
	}

	var stockDataMap map[string][]stock.StockInfo
	if err := json.Unmarshal(data, &stockDataMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json: %w", err)
	}

	return stockDataMap, nil
}

// pickFirstKey returns the first key in the map, or empty string if none.
func pickFirstKey(m map[string][]stock.StockInfo) string {
	for k := range m {
		return k
	}

	return ""
}

// resolveCode decides which stock code to use based on args or map contents.
func resolveCode(args []string, m map[string][]stock.StockInfo) string {
	var code string
	if len(args) >= requiredArgsForCode {
		code = args[indexStockCodeArg]
	} else {
		code = pickFirstKey(m)
	}

	return code
}

// buildSeries builds high/low series from stock info.
func buildSeries(info stock.StockInfo) (highs, lows []float64, err error) {
	if len(info.Data) == 0 {
		return nil, nil, fmt.Errorf("no data")
	}

	highs = make([]float64, 0, len(info.Data))
	lows = make([]float64, 0, len(info.Data))

	for _, d := range info.Data {
		highs = append(highs, d.High)
		lows = append(lows, d.Low)
	}

	return highs, lows, nil
}

func main() {
	if len(os.Args) < minArgs {
		usage()
		os.Exit(1)
	}

	jsonPath := os.Args[1]

	f, err := os.Open(jsonPath)
	if err != nil {
		log.Fatalf("failed to open %s: %v", jsonPath, err)
	}
	defer f.Close()

	stockDataMap, err := loadStockData(f)
	if err != nil {
		log.Fatalf("failed to load stock data: %v", err)
	}

	code := resolveCode(os.Args, stockDataMap)
	if code == "" {
		log.Fatal("no stock code found in JSON")
	}

	infos, ok := stockDataMap[code]
	if !ok || len(infos) == 0 {
		log.Fatalf("stock code %s not found in JSON", code)
	}

	// Use the first StockInfo entry for the given code.
	info := infos[0]

	highs, lows, err := buildSeries(info)
	if err != nil {
		log.Fatalf("failed to build series for %s: %v", code, err)
	}

	// Plot High and Low on the same graph.
	graph := asciigraph.PlotMany(
		[][]float64{highs, lows},
		asciigraph.Height(graphHeight),
		asciigraph.Caption(fmt.Sprintf("%s (%s) High / Low", info.Name, code)),
	)

	fmt.Println(graph)

	// Optionally, print simple legend.
	fmt.Println()
	fmt.Println("Legend:")
	fmt.Println("  Line 1: High")
	fmt.Println("  Line 2: Low")
}
