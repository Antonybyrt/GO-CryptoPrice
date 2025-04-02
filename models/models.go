package models

import "time"

type ServerStatus struct {
	ID        int64     `json:"id" db:"id"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
	Status    string    `json:"status" db:"status"`
	Error     string    `json:"error" db:"error"`
}

type TradingPair struct {
	ID          int64     `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Base        string    `json:"base" db:"base"`
	Quote       string    `json:"quote" db:"quote"`
	LastUpdated time.Time `json:"last_updated" db:"last_updated"`
}

type PairInfo struct {
	ID        int64     `json:"id" db:"id"`
	PairID    int64     `json:"pair_id" db:"pair_id"`
	Price     float64   `json:"price" db:"price"`
	Volume24h float64   `json:"volume_24h" db:"volume_24h"`
	High24h   float64   `json:"high_24h" db:"high_24h"`
	Low24h    float64   `json:"low_24h" db:"low_24h"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
}

type HistoricalData struct {
	ID        int64     `json:"id" db:"id"`
	PairID    int64     `json:"pair_id" db:"pair_id"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
	Open      float64   `json:"open" db:"open"`
	High      float64   `json:"high" db:"high"`
	Low       float64   `json:"low" db:"low"`
	Close     float64   `json:"close" db:"close"`
	Volume    float64   `json:"volume" db:"volume"`
}
