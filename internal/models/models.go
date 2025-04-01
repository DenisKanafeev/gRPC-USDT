package models

import (
	"time"
)

type Rate struct {
	AskAmount float64   `json:"askamount"` // Объем по цене ask
	BidAmount float64   `json:"bidamount"` // Объем по цене bid
	Ask       float64   `json:"ask"`       // Цена ask
	Bid       float64   `json:"bid"`       // Цена bid
	Time      time.Time `json:"timestamp"` // Время получения курса
}

type BinanceDepthResponse struct {
	LastUpdateID int64      `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}
