// Command png draws the JSON of cmd/fetch as one PNG image, for places that
// cannot run the JavaScript charts of the web page, such as Slack.
//
//	go run ./cmd/fetch 7203.T,9432.T | go run ./cmd/png > chart.png
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"image/png"
	"io"
	"os"
	"time"

	"stock/pkg/render"
	stock "stock/pkg/types"
)

func run(args []string, stdin io.Reader, stdout io.Writer) error {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	date := flags.String("date", time.Now().Format(time.DateOnly), "the date shown in the header")

	err := flags.Parse(args[1:])
	if err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	var stocks map[string][]stock.StockInfo

	err = json.NewDecoder(stdin).Decode(&stocks)
	if err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	img, err := render.Render(stocks, *date)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	// encoded in memory first, so a failure never leaves half a PNG on stdout
	var encoded bytes.Buffer

	err = png.Encode(&encoded, img)
	if err != nil {
		return fmt.Errorf("failed to encode PNG: %w", err)
	}

	_, err = encoded.WriteTo(stdout)
	if err != nil {
		return fmt.Errorf("failed to write PNG: %w", err)
	}

	return nil
}

func main() {
	err := run(os.Args, os.Stdin, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
