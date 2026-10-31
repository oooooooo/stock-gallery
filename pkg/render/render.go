// Package render draws the stock gallery as one PNG-ready image,
// for places that cannot run the JavaScript charts of the web page, such as Slack.
package render

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"

	stock "stock/pkg/types"
)

// Series colors are the first two slots of a color-vision-deficiency-safe palette;
// the volume is a neutral gray because it is context, not a third identity.
var (
	HighColor   = color.RGBA{0x2a, 0x78, 0xd6, 0xff}
	LowColor    = color.RGBA{0xeb, 0x68, 0x34, 0xff}
	VolumeColor = color.RGBA{0xc3, 0xc2, 0xb7, 0xff}

	surface   = color.RGBA{0xfc, 0xfc, 0xfb, 0xff}
	primary   = color.RGBA{0x0b, 0x0b, 0x0b, 0xff}
	secondary = color.RGBA{0x52, 0x51, 0x4e, 0xff}
	grid      = color.RGBA{0xe6, 0xe5, 0xe0, 0xff}
)

const (
	columns     = 4
	panelWidth  = 640
	panelHeight = 360
	headerSize  = 72
	margin      = 24

	plotLeft     = 72 // room for the price labels
	plotRight    = 16
	titleHeight  = 36
	priceHeight  = 210
	bandGap      = 12
	volumeHeight = 56
	axisHeight   = 24

	tickCount  = 5
	lineWidth  = 2.0
	barGap     = 2
	legendLine = 24
)

var errNoStocks = errors.New("no stock data to render")

type faces struct {
	title, label, small font.Face
}

func loadFaces() (faces, error) {
	regular, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return faces{}, fmt.Errorf("parse regular font: %w", err)
	}

	bold, err := opentype.Parse(gobold.TTF)
	if err != nil {
		return faces{}, fmt.Errorf("parse bold font: %w", err)
	}

	face := func(f *opentype.Font, size float64) (font.Face, error) {
		return opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	}

	var result faces

	result.title, err = face(bold, 18)
	if err != nil {
		return faces{}, fmt.Errorf("title font: %w", err)
	}

	result.label, err = face(regular, 14)
	if err != nil {
		return faces{}, fmt.Errorf("label font: %w", err)
	}

	result.small, err = face(regular, 12)
	if err != nil {
		return faces{}, fmt.Errorf("small font: %w", err)
	}

	return result, nil
}

// Ticks returns round values between low and high, about count of them.
func Ticks(low, high float64, count int) []float64 {
	if high <= low {
		return []float64{low}
	}

	step := niceStep((high - low) / float64(count))

	var ticks []float64

	for value := math.Ceil(low/step) * step; value <= high; value += step {
		tick := math.Round(value/step) * step
		if tick == 0 {
			tick = 0 // rounding a value just below 0 gives -0, which formats as "-0"
		}

		ticks = append(ticks, tick)
	}

	return ticks
}

// niceStep rounds a raw step up to 1, 2, 5 or 10 times a power of ten.
func niceStep(raw float64) float64 {
	power := math.Pow(10, math.Floor(math.Log10(raw)))
	for _, factor := range []float64{1, 2, 5} {
		if raw <= factor*power {
			return factor * power
		}
	}

	return 10 * power
}

// PriceRange returns the lowest Low and the highest High.
// Months without a price (0, from null in the source) are skipped.
func PriceRange(data []stock.StockData) (float64, float64) {
	low, high := math.Inf(1), math.Inf(-1)

	for _, month := range data {
		if month.Low > 0 {
			low = min(low, month.Low)
		}

		if month.High > 0 {
			high = max(high, month.High)
		}
	}

	return low, high
}

// Render draws every stock in a grid of four columns, ordered by symbol.
// date is shown in the header so a reader knows when the data was taken.
func Render(stocks map[string][]stock.StockInfo, date string) (*image.RGBA, error) {
	symbols := make([]string, 0, len(stocks))

	for symbol, infos := range stocks {
		if len(infos) > 0 {
			symbols = append(symbols, symbol)
		}
	}

	if len(symbols) == 0 {
		return nil, errNoStocks
	}

	slices.Sort(symbols)

	faces, err := loadFaces()
	if err != nil {
		return nil, err
	}

	rows := (len(symbols) + columns - 1) / columns
	img := image.NewRGBA(image.Rect(0, 0, 2*margin+columns*panelWidth, headerSize+rows*panelHeight+margin))
	draw.Draw(img, img.Bounds(), image.NewUniform(surface), image.Point{}, draw.Src)

	drawHeader(img, faces, date)

	for i, symbol := range symbols {
		origin := image.Pt(margin+(i%columns)*panelWidth, headerSize+(i/columns)*panelHeight)
		drawPanel(img, faces, origin, symbol, stocks[symbol][0])
	}

	return img, nil
}

func drawHeader(img *image.RGBA, faces faces, date string) {
	drawText(img, faces.title, primary, margin, 32, "Stock Gallery: 5-year monthly range ("+date+")")

	x := margin

	for _, entry := range []struct {
		name  string
		color color.RGBA
		bar   bool
	}{{"High", HighColor, false}, {"Low", LowColor, false}, {"Volume", VolumeColor, true}} {
		if entry.bar {
			fillRect(img, image.Rect(x, 50, x+legendLine, 62), entry.color)
		} else {
			strokeLine(img, []point{{float32(x), 56}, {float32(x + legendLine), 56}}, entry.color)
		}

		x += legendLine + 8
		x += drawText(img, faces.label, secondary, x, 61, entry.name) + 24
	}
}

type point struct{ x, y float32 }

func drawPanel(img *image.RGBA, faces faces, origin image.Point, symbol string, info stock.StockInfo) {
	drawText(img, faces.label, primary, origin.X, origin.Y+22, fmt.Sprintf("[%s] %s", symbol, info.Name))

	plot := image.Rect(origin.X+plotLeft, origin.Y+titleHeight, origin.X+panelWidth-plotRight-margin, origin.Y+titleHeight+priceHeight)
	volume := image.Rect(plot.Min.X, plot.Max.Y+bandGap, plot.Max.X, plot.Max.Y+bandGap+volumeHeight)

	if len(info.Data) == 0 {
		drawText(img, faces.label, secondary, plot.Min.X, plot.Min.Y+20, "No data")

		return
	}

	low, high := PriceRange(info.Data)
	if math.IsInf(low, 0) || math.IsInf(high, 0) {
		drawText(img, faces.label, secondary, plot.Min.X, plot.Min.Y+20, "No price data")

		return
	}

	// a little room so the lines do not touch the frame
	padding := max((high-low)*0.05, high*0.01)
	scaleLow, scaleHigh := low-padding, high+padding
	priceY := func(value float64) float32 {
		return float32(plot.Max.Y) - float32((value-scaleLow)/(scaleHigh-scaleLow))*float32(plot.Dy())
	}

	for _, tick := range Ticks(scaleLow, scaleHigh, tickCount) {
		y := int(math.Round(float64(priceY(tick))))
		fillRect(img, image.Rect(plot.Min.X, y, plot.Max.X, y+1), grid)

		label := formatNumber(tick)
		drawText(img, faces.small, secondary, plot.Min.X-8-textWidth(faces.small, label), y+4, label)
	}

	step := float32(plot.Dx()) / float32(len(info.Data))
	centerX := func(i int) float32 { return float32(plot.Min.X) + step*(float32(i)+0.5) }

	drawVolume(img, faces, volume, info.Data, step)
	drawYears(img, faces, volume.Max.Y, info.Data, centerX)

	strokeLine(img, seriesPoints(info.Data, func(m stock.StockData) float64 { return m.High }, centerX, priceY), HighColor)
	strokeLine(img, seriesPoints(info.Data, func(m stock.StockData) float64 { return m.Low }, centerX, priceY), LowColor)
}

func drawVolume(img *image.RGBA, faces faces, band image.Rectangle, data []stock.StockData, step float32) {
	var peak int64
	for _, month := range data {
		peak = max(peak, month.Volume)
	}

	fillRect(img, image.Rect(band.Min.X, band.Max.Y, band.Max.X, band.Max.Y+1), grid)

	// the peak is the only scale a volume band needs
	for line, text := range []string{"Vol", formatNumber(float64(peak))} {
		drawText(img, faces.small, secondary, band.Min.X-8-textWidth(faces.small, text), band.Min.Y+10+14*line, text)
	}

	if peak == 0 {
		return
	}

	width := max(int(step)-barGap, 1)

	for i, month := range data {
		height := int(math.Round(float64(month.Volume) / float64(peak) * float64(band.Dy())))
		left := band.Min.X + int(step*float32(i)) + barGap/2
		fillRect(img, image.Rect(left, band.Max.Y-height, left+width, band.Max.Y), VolumeColor)
	}
}

// drawYears labels each January, so the five years read at a glance.
func drawYears(img *image.RGBA, faces faces, y int, data []stock.StockData, centerX func(int) float32) {
	for i, month := range data {
		year, found := strings.CutSuffix(month.Date, "-01")
		if !found {
			continue
		}

		x := int(centerX(i))
		fillRect(img, image.Rect(x, y, x+1, y+4), secondary)
		drawText(img, faces.small, secondary, x-textWidth(faces.small, year)/2, y+18, year)
	}
}

func seriesPoints(data []stock.StockData, value func(stock.StockData) float64,
	centerX func(int) float32, priceY func(float64) float32,
) []point {
	points := make([]point, 0, len(data))

	for i, month := range data {
		if v := value(month); v > 0 {
			points = append(points, point{centerX(i), priceY(v)})
		}
	}

	return points
}

// strokeLine draws an anti-aliased polyline with round joins.
// Each segment and each join is filled on its own: in one shared path, a join
// wound the other way than its segment would cancel it and leave a gap.
func strokeLine(img *image.RGBA, points []point, c color.RGBA) {
	half := float32(lineWidth / 2)

	for i := 1; i < len(points); i++ {
		a, b := points[i-1], points[i]
		dx, dy := b.x-a.x, b.y-a.y

		length := float32(math.Hypot(float64(dx), float64(dy)))
		if length == 0 {
			continue
		}

		nx, ny := -dy/length*half, dx/length*half
		fillPolygon(img, []point{{a.x + nx, a.y + ny}, {b.x + nx, b.y + ny}, {b.x - nx, b.y - ny}, {a.x - nx, a.y - ny}}, c)
	}

	for _, p := range points {
		fillPolygon(img, circle(p, half), c)
	}
}

func circle(center point, radius float32) []point {
	const sides = 12

	points := make([]point, 0, sides)

	for i := range sides {
		angle := 2 * math.Pi * float64(i) / sides
		points = append(points, point{
			center.x + radius*float32(math.Cos(angle)),
			center.y + radius*float32(math.Sin(angle)),
		})
	}

	return points
}

// fillPolygon rasterizes only the polygon's bounding box, not the whole image.
func fillPolygon(img *image.RGBA, polygon []point, c color.RGBA) {
	box := image.Rectangle{
		Min: image.Pt(math.MaxInt, math.MaxInt),
		Max: image.Pt(math.MinInt, math.MinInt),
	}
	for _, p := range polygon {
		box.Min.X = min(box.Min.X, int(math.Floor(float64(p.x))))
		box.Min.Y = min(box.Min.Y, int(math.Floor(float64(p.y))))
		box.Max.X = max(box.Max.X, int(math.Ceil(float64(p.x))))
		box.Max.Y = max(box.Max.Y, int(math.Ceil(float64(p.y))))
	}

	box = box.Intersect(img.Bounds())
	if box.Empty() {
		return
	}

	raster := vector.NewRasterizer(box.Dx(), box.Dy())

	for i, p := range polygon {
		x, y := p.x-float32(box.Min.X), p.y-float32(box.Min.Y)
		if i == 0 {
			raster.MoveTo(x, y)
		} else {
			raster.LineTo(x, y)
		}
	}

	raster.ClosePath()
	raster.DrawOp = draw.Over
	raster.Draw(img, box, image.NewUniform(c), image.Point{})
}

func fillRect(img *image.RGBA, rect image.Rectangle, c color.RGBA) {
	draw.Draw(img, rect, image.NewUniform(c), image.Point{}, draw.Src)
}

// drawText draws s with its baseline at y and returns its width.
func drawText(img *image.RGBA, face font.Face, c color.RGBA, x, y int, s string) int {
	drawer := font.Drawer{Dst: img, Src: image.NewUniform(c), Face: face, Dot: fixed.P(x, y)}
	drawer.DrawString(s)

	return textWidth(face, s)
}

func textWidth(face font.Face, s string) int {
	return font.MeasureString(face, s).Round()
}

// formatNumber writes 2500 as "2,500" and 668000000 as "668M".
func formatNumber(value float64) string {
	switch {
	case math.Abs(value) >= 1e9:
		return fmt.Sprintf("%.3gB", value/1e9)
	case math.Abs(value) >= 1e6:
		return fmt.Sprintf("%.3gM", value/1e6)
	case value != math.Trunc(value):
		return strconv.FormatFloat(value, 'f', -1, 64)
	}

	digits := strconv.FormatFloat(value, 'f', 0, 64)

	var grouped strings.Builder

	for i, digit := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 && digits[i-1] != '-' {
			grouped.WriteByte(',')
		}

		grouped.WriteRune(digit)
	}

	return grouped.String()
}
