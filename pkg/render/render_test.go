package render_test

import (
	"image"
	"image/color"
	"math"
	"slices"
	"strconv"
	"testing"

	"stock/pkg/render"
	stock "stock/pkg/types"
)

func months(lows ...float64) []stock.StockData {
	data := make([]stock.StockData, 0, len(lows))
	for i, low := range lows {
		data = append(data, stock.StockData{
			Date:   []string{"2025-11", "2025-12", "2026-01", "2026-02", "2026-03"}[i],
			Low:    low,
			High:   low + 100,
			Volume: int64(1000 * (i + 1)),
		})
	}

	return data
}

func countColor(img image.Image, want color.RGBA) int {
	count := 0

	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if color.RGBAModel.Convert(img.At(x, y)) == want {
				count++
			}
		}
	}

	return count
}

func TestTicksAreRoundNumbersInsideTheRange(t *testing.T) {
	t.Parallel()

	got := render.Ticks(1995, 3800, 5)
	want := []float64{2000, 2500, 3000, 3500}

	if !slices.Equal(got, want) {
		t.Fatalf("Ticks(1995, 3800, 5) = %v, want %v", got, want)
	}
}

func TestTicksHaveNoNegativeZero(t *testing.T) {
	t.Parallel()

	// a price padded below its lowest month can start the scale just under 0
	for _, tick := range render.Ticks(-400, 11000, 5) {
		if tick == 0 && math.Signbit(tick) {
			t.Fatalf("Ticks(-400, 11000, 5) = %v, has -0, which is labeled \"-0\"", render.Ticks(-400, 11000, 5))
		}
	}
}

func TestTicksOfAFlatRangeIsItsValue(t *testing.T) {
	t.Parallel()

	if got := render.Ticks(500, 500, 5); !slices.Equal(got, []float64{500}) {
		t.Fatalf("Ticks(500, 500, 5) = %v, want [500]", got)
	}
}

func TestPriceRangeSkipsMonthsWithoutPrices(t *testing.T) {
	t.Parallel()

	// Yahoo! Finance sends null for a month without trades, which decodes to 0
	low, high := render.PriceRange(months(1200, 0, 1100))

	if low != 1100 || high != 1300 {
		t.Fatalf("PriceRange = %v, %v, want 1100, 1300", low, high)
	}
}

func stocksOf(count int) map[string][]stock.StockInfo {
	stocks := make(map[string][]stock.StockInfo, count)
	for i := range count {
		stocks[strconv.Itoa(1000+i)] = []stock.StockInfo{{Name: "Company", Data: months(1000, 1100, 1050)}}
	}

	return stocks
}

func mustRender(t *testing.T, count int) image.Image {
	t.Helper()

	img, err := render.Render(stocksOf(count), "2026-10-30")
	if err != nil {
		t.Fatal(err)
	}

	return img
}

func TestRenderPutsFourPanelsInARow(t *testing.T) {
	t.Parallel()

	one, four, five := mustRender(t, 1), mustRender(t, 4), mustRender(t, 5)

	if one.Bounds() != four.Bounds() {
		t.Fatalf("four stocks (%v) do not fit the row of one (%v)", four.Bounds(), one.Bounds())
	}

	if five.Bounds().Dy() <= four.Bounds().Dy() {
		t.Fatalf("five stocks (%d px) are not taller than four (%d px)", five.Bounds().Dy(), four.Bounds().Dy())
	}
}

func TestRenderOfTwelveStocksIsLandscape(t *testing.T) {
	t.Parallel()

	// Slack shows a landscape image larger in the timeline than a portrait one
	if bounds := mustRender(t, 12).Bounds(); bounds.Dx() <= bounds.Dy() {
		t.Fatalf("12 stocks are %dx%d, want wider than tall", bounds.Dx(), bounds.Dy())
	}
}

func TestRenderDrawsHighLowAndVolume(t *testing.T) {
	t.Parallel()

	stocks := map[string][]stock.StockInfo{
		"7203": {{Name: "Toyota Motor Corporation", Data: months(2000, 2100, 2050, 2200, 2150)}},
	}

	img, err := render.Render(stocks, "2026-10-30")
	if err != nil {
		t.Fatal(err)
	}

	for name, want := range map[string]color.RGBA{
		"high": render.HighColor, "low": render.LowColor, "volume": render.VolumeColor,
	} {
		// the legend alone covers a few dozen pixels of each color; the chart adds hundreds
		if got := countColor(img, want); got < 200 {
			t.Errorf("%s color covers %d pixels, want the series drawn", name, got)
		}
	}
}

func TestRenderWithoutStocksFails(t *testing.T) {
	t.Parallel()

	_, err := render.Render(map[string][]stock.StockInfo{}, "2026-10-30")
	if err == nil {
		t.Fatal("Render of no stocks returned nil")
	}
}
