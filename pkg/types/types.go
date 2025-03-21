package stock

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