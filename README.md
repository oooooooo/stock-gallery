# Stock Gallery

## About

Display a 5-year chart for a specified stock code to identify the best times to buy or sell.

This tool simply arranges and displays charts from Yahoo! Finance. It’s simple but surprisingly useful for me.

## Usage

```shell
go run cmd/fetch/main.go 9064.T,7203.T,6752.T,9501.T,8136.T,9064.T,2220.T,5253.T,5032.T,4661.T,9432.T,3778.T,3382.T > chart.json
cat chart.json | go run cmd/chart/main.go > docs/index.html
```

## Demo

<https://oooooooo.github.io/stock-gallery/>

## License

MIT
