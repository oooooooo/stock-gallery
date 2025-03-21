package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const yahooFinanceURL = "https://query1.finance.yahoo.com/v8/finance/chart/%s?range=5y&interval=1mo"
const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

type StockData struct {
	Date   string
	Low    float64
	High   float64
	Volume int64
}

type StockInfo struct {
	Name string
	Data []StockData
}

type YahooResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Symbol   string `json:"symbol"`
				LongName string `json:"longName"`
			} `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Low    []float64 `json:"low"`
					High   []float64 `json:"high"`
					Volume []int64   `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
	} `json:"chart"`
}

func fetchStockData(symbol string) ([]StockInfo, error) {
	url := fmt.Sprintf(yahooFinanceURL, symbol)
	client := &http.Client{}

	for attempt := 1; attempt <= 3; attempt++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %v", err)
		}
		req.Header.Set("User-Agent", userAgent)

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == 429 {
			fmt.Println("HTTP 429 Too Many Requests - Retrying in 5 seconds...")
			time.Sleep(5 * time.Second)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("error: HTTP Status %d\nResponse body: %s", resp.StatusCode, string(body))
		}

		var yahooResp YahooResponse
		if err := json.NewDecoder(resp.Body).Decode(&yahooResp); err != nil {
			return nil, fmt.Errorf("error decoding JSON: %v", err)
		}

		var data []StockData
		if len(yahooResp.Chart.Result) == 0 {
			return nil, fmt.Errorf("no data found for %s", symbol)
		}

		result := yahooResp.Chart.Result[0]
		longName := result.Meta.LongName
		if longName == "" {
			longName = symbol
		}
		quote := result.Indicators.Quote[0]

		for i, ts := range result.Timestamp {
			date := time.Unix(ts, 0).Format("2006-01")
			if len(quote.Low) <= i || len(quote.High) <= i || len(quote.Volume) <= i {
				continue
			}
			data = append(data, StockData{
				Date:   date,
				Low:    quote.Low[i],
				High:   quote.High[i],
				Volume: quote.Volume[i],
			})
		}
		return []StockInfo{{Name: longName, Data: data}}, nil
	}

	return nil, fmt.Errorf("failed to fetch data for %s after multiple retries", symbol)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run fetch_stock.go <symbol1,symbol2,symbol3>")
		os.Exit(1)
	}

	stockDataMap := make(map[string][]StockInfo)

	symbols := strings.Split(os.Args[1], ",")
	for _, symbol := range symbols {
		data, err := fetchStockData(symbol)
		if err != nil {
			fmt.Printf("Error fetching data for %s: %v\n", symbol, err)
			continue
		}

		// 1234.T では JavaScript で変数として扱えないため
		stockDataMap[strings.TrimSuffix(symbol, ".T")] = data
	}

	jsonData, err := json.MarshalIndent(stockDataMap, "", "  ")
	if err != nil {
		fmt.Printf("Failed to encode stockDataMap as JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonData))
}