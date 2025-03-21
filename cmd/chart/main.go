package main

import (
		_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"text/template"

	"stock/pkg/types"
)

//go:embed chart.go.tmpl
var htmlTemplate string

func processStockData() (map[string][]stock.StockInfo, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	var stockDataMap map[string][]stock.StockInfo
	if err := json.Unmarshal(data, &stockDataMap); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return stockDataMap, nil
}

func renderTemplate(stockDataMap map[string][]stock.StockInfo) error {
	tmpl, err := template.New("chart").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	if err := tmpl.Execute(os.Stdout, stockDataMap); err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	return nil
}

func main() {
	stockDataMap, err := processStockData()
	if err != nil {
		log.Fatalf("Error processing stock data: %v", err)
	}

	if err := renderTemplate(stockDataMap); err != nil {
		log.Fatalf("Error rendering template: %v", err)
	}
}
