package database

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/antonyloussararian/Go-CryptoPrice/models"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	db *sql.DB
}

func NewDB(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &DB{db: db}, nil
}

func (d *DB) InitSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS server_status (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME NOT NULL,
			status TEXT NOT NULL,
			error TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS trading_pairs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			base TEXT NOT NULL,
			quote TEXT NOT NULL,
			last_updated DATETIME NOT NULL,
			UNIQUE(name, last_updated)
		)`,
		`CREATE TABLE IF NOT EXISTS pair_info (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pair_id INTEGER NOT NULL,
			price REAL NOT NULL,
			volume_24h REAL NOT NULL,
			high_24h REAL NOT NULL,
			low_24h REAL NOT NULL,
			timestamp DATETIME NOT NULL,
			FOREIGN KEY (pair_id) REFERENCES trading_pairs(id)
		)`,
		`CREATE TABLE IF NOT EXISTS historical_data (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pair_id INTEGER NOT NULL,
			timestamp DATETIME NOT NULL,
			open REAL NOT NULL,
			high REAL NOT NULL,
			low REAL NOT NULL,
			close REAL NOT NULL,
			volume REAL NOT NULL,
			FOREIGN KEY (pair_id) REFERENCES trading_pairs(id)
		)`,
	}

	for _, query := range queries {
		_, err := d.db.Exec(query)
		if err != nil {
			return fmt.Errorf("erreur lors de l'exécution de la requête %s: %v", query, err)
		}
	}

	log.Println("Schéma de la base de données initialisé avec succès")
	return nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) GetTradingPairsFromDB() ([]models.TradingPair, error) {
	query := `SELECT id, name, base, quote, last_updated FROM trading_pairs ORDER BY last_updated DESC`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pairs []models.TradingPair
	for rows.Next() {
		var pair models.TradingPair
		err := rows.Scan(&pair.ID, &pair.Name, &pair.Base, &pair.Quote, &pair.LastUpdated)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, pair)
	}
	return pairs, nil
}

func (d *DB) GetPairInfoFromDB(pairID int64) ([]models.PairInfo, error) {
	query := `SELECT id, pair_id, price, volume_24h, high_24h, low_24h, timestamp FROM pair_info WHERE pair_id = ? ORDER BY timestamp DESC`
	rows, err := d.db.Query(query, pairID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var infos []models.PairInfo
	for rows.Next() {
		var info models.PairInfo
		err := rows.Scan(&info.ID, &info.PairID, &info.Price, &info.Volume24h, &info.High24h, &info.Low24h, &info.Timestamp)
		if err != nil {
			return nil, err
		}
		infos = append(infos, info)
	}
	return infos, nil
}

func (d *DB) GetHistoricalDataFromDB(pairID int64) ([]models.HistoricalData, error) {
	query := `SELECT id, pair_id, timestamp, open, high, low, close, volume FROM historical_data WHERE pair_id = ? ORDER BY timestamp DESC`
	rows, err := d.db.Query(query, pairID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []models.HistoricalData
	for rows.Next() {
		var h models.HistoricalData
		err := rows.Scan(&h.ID, &h.PairID, &h.Timestamp, &h.Open, &h.High, &h.Low, &h.Close, &h.Volume)
		if err != nil {
			return nil, err
		}
		data = append(data, h)
	}
	return data, nil
}

func (d *DB) SaveServerStatus(status *models.ServerStatus) error {
	query := `INSERT INTO server_status (timestamp, status, error) VALUES (?, ?, ?)`
	_, err := d.db.Exec(query, status.Timestamp, status.Status, status.Error)
	return err
}

func (d *DB) SaveTradingPair(pair *models.TradingPair) error {
	query := `INSERT INTO trading_pairs (name, base, quote, last_updated) VALUES (?, ?, ?, ?)`
	result, err := d.db.Exec(query, pair.Name, pair.Base, pair.Quote, pair.LastUpdated)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	pair.ID = id
	return nil
}

func (d *DB) SavePairInfo(info *models.PairInfo) error {
	query := `INSERT INTO pair_info (pair_id, price, volume_24h, high_24h, low_24h, timestamp) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := d.db.Exec(query, info.PairID, info.Price, info.Volume24h, info.High24h, info.Low24h, info.Timestamp)
	return err
}

func (d *DB) SaveHistoricalData(data *models.HistoricalData) error {
	query := `INSERT INTO historical_data (pair_id, timestamp, open, high, low, close, volume) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := d.db.Exec(query, data.PairID, data.Timestamp, data.Open, data.High, data.Low, data.Close, data.Volume)
	return err
}

func (d *DB) SaveTradingPairBatch(pairs []models.TradingPair) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO trading_pairs (name, base, quote, last_updated)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, pair := range pairs {
		_, err := stmt.Exec(pair.Name, pair.Base, pair.Quote, pair.LastUpdated)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (d *DB) SavePairInfoBatch(infos []models.PairInfo) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO pair_info (pair_id, price, volume_24h, high_24h, low_24h, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, info := range infos {
		_, err := stmt.Exec(info.PairID, info.Price, info.Volume24h, info.High24h, info.Low24h, info.Timestamp)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (d *DB) SaveHistoricalDataBatch(data []models.HistoricalData) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO historical_data (pair_id, timestamp, open, high, low, close, volume)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, d := range data {
		_, err := stmt.Exec(d.PairID, d.Timestamp, d.Open, d.High, d.Low, d.Close, d.Volume)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
