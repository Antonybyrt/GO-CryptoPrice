package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/antonyloussararian/Go-CryptoPrice/database"
	"github.com/antonyloussararian/Go-CryptoPrice/kraken"
	"github.com/antonyloussararian/Go-CryptoPrice/models"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	db     *database.DB
	client *kraken.Client
}

func NewHandler(db *database.DB, client *kraken.Client) *Handler {
	return &Handler{
		db:     db,
		client: client,
	}
}

func (h *Handler) GetServerStatus(c *gin.Context) {
	status, err := h.client.GetServerStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h *Handler) GetTradingPairs(c *gin.Context) {
	pairs, err := h.client.GetTradingPairs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type pairVolume struct {
		name   string
		volume float64
	}

	var pairVolumes []pairVolume

	for pairName, pairData := range pairs {
		if pairInfo, ok := pairData.(map[string]any); ok {
			if volumeData, ok := pairInfo["v"].([]any); ok && len(volumeData) > 1 {
				if volume24h, ok := volumeData[1].(string); ok {
					if vol, err := strconv.ParseFloat(volume24h, 64); err == nil {
						pairVolumes = append(pairVolumes, pairVolume{
							name:   pairName,
							volume: vol,
						})
					}
				}
			}
		}
	}

	sort.Slice(pairVolumes, func(i, j int) bool {
		return pairVolumes[i].volume > pairVolumes[j].volume
	})

	topPairs := make(map[string]any)
	for i := 0; i < len(pairVolumes) && i < 10; i++ {
		topPairs[pairVolumes[i].name] = pairs[pairVolumes[i].name]
	}

	c.JSON(http.StatusOK, gin.H{
		"pairs": topPairs,
		"count": len(topPairs),
	})
}

func (h *Handler) GetPairInfo(c *gin.Context) {
	pair := c.Param("pair")
	if pair == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pair parameter is required"})
		return
	}

	info, err := h.client.GetPairInfo(pair)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

func (h *Handler) createCSV(targetTime time.Time) error {
	lastCandleTime := targetTime.Truncate(5 * time.Minute)
	lastCandleTimestamp := lastCandleTime.Unix()

	if err := os.MkdirAll("csv", 0755); err != nil {
		return fmt.Errorf("erreur lors de la création du dossier csv: %v", err)
	}

	filename := fmt.Sprintf("csv/top10_5min_highlow_%s_%s.csv",
		lastCandleTime.Format("20060102"),
		lastCandleTime.Format("150405"))
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("erreur lors de la création du fichier CSV: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{"Pair", "Timestamp", "Open", "High", "Low", "Close", "Volume"}); err != nil {
		return fmt.Errorf("erreur lors de l'écriture de l'en-tête CSV: %v", err)
	}

	pairs, err := h.client.GetTradingPairs()
	if err != nil {
		return fmt.Errorf("erreur lors de la récupération des paires: %v", err)
	}

	type pairVolume struct {
		name   string
		volume float64
	}

	var pairVolumes []pairVolume

	for pairName, pairData := range pairs {
		if pairInfo, ok := pairData.(map[string]any); ok {
			if volumeData, ok := pairInfo["v"].([]any); ok && len(volumeData) > 1 {
				if volume24h, ok := volumeData[1].(string); ok {
					if vol, err := strconv.ParseFloat(volume24h, 64); err == nil {
						pairVolumes = append(pairVolumes, pairVolume{
							name:   pairName,
							volume: vol,
						})
					}
				}
			}
		}
	}

	sort.Slice(pairVolumes, func(i, j int) bool {
		return pairVolumes[i].volume > pairVolumes[j].volume
	})

	topPairs := make([]string, 0)
	for i := 0; i < len(pairVolumes) && i < 10; i++ {
		topPairs = append(topPairs, pairVolumes[i].name)
	}

	for _, pair := range topPairs {
		data, err := h.client.GetHistoricalData(pair, 5, lastCandleTimestamp)
		if err != nil {
			continue
		}

		if result, ok := data["result"].(map[string]any); ok {
			if ohlc, ok := result[pair].([]any); ok {
				if len(ohlc) > 0 {
					for _, candle := range ohlc {
						if candleData, ok := candle.([]any); ok {
							candleTimestamp := time.Unix(int64(candleData[0].(float64)), 0)
							if candleTimestamp.Equal(lastCandleTime) {
								record := []string{
									pair,
									candleTimestamp.Format("2006-01-02 15:04:05"),
									fmt.Sprintf("%v", candleData[1]),
									fmt.Sprintf("%v", candleData[2]),
									fmt.Sprintf("%v", candleData[3]),
									fmt.Sprintf("%v", candleData[4]),
									fmt.Sprintf("%v", candleData[6]),
								}
								if err := writer.Write(record); err != nil {
									return fmt.Errorf("erreur lors de l'écriture des données CSV: %v", err)
								}
								break
							}
						}
					}
				}
			}
		}
	}

	return nil
}

func (h *Handler) getLatestCSV() (string, error) {
	files, err := os.ReadDir("csv")
	if err != nil {
		return "", err
	}

	var latestFile string
	var latestTime time.Time

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".csv") {
			parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(file.Name(), "top10_5min_highlow_"), ".csv"), "_")
			if len(parts) != 2 {
				continue
			}
			dateStr := parts[0]
			timeStr := parts[1]
			fileTime, err := time.Parse("20060102150405", dateStr+timeStr)
			if err != nil {
				continue
			}

			if latestFile == "" || fileTime.After(latestTime) {
				latestFile = file.Name()
				latestTime = fileTime
			}
		}
	}

	if latestFile == "" {
		return "", fmt.Errorf("aucun fichier CSV trouvé")
	}

	return filepath.Join("csv", latestFile), nil
}

func (h *Handler) getCSVForDate(date time.Time) (string, error) {
	files, err := os.ReadDir("csv")
	if err != nil {
		return "", err
	}

	var latestFile string
	var latestTime time.Time
	targetDate := date.Format("20060102")

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".csv") {
			parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(file.Name(), "top10_5min_highlow_"), ".csv"), "_")
			if len(parts) != 2 {
				continue
			}
			fileDate := parts[0]
			if fileDate == targetDate {
				timeStr := parts[1]
				fileTime, err := time.Parse("20060102150405", fileDate+timeStr)
				if err != nil {
					continue
				}

				if latestFile == "" || fileTime.After(latestTime) {
					latestFile = file.Name()
					latestTime = fileTime
				}
			}
		}
	}

	if latestFile == "" {
		return "", fmt.Errorf("aucun fichier CSV trouvé pour la date %s", date.Format("2006-01-02"))
	}

	return filepath.Join("csv", latestFile), nil
}

func (h *Handler) DownloadHistoricalData(c *gin.Context) {
	dateStr := c.Query("date")
	var targetTime time.Time
	var err error

	if dateStr != "" {
		targetTime, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format de date invalide. Utilisez le format YYYY-MM-DD"})
			return
		}
	}

	var csvPath string
	if dateStr == "" {
		csvPath, err = h.getLatestCSV()
	} else {
		csvPath, err = h.getCSVForDate(targetTime)
	}

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.File(csvPath)
}

func (h *Handler) GetDBData(c *gin.Context) {
	pairs, err := h.db.GetTradingPairsFromDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération des paires"})
		return
	}

	result := make(map[string]any)
	for _, pair := range pairs {
		infos, err := h.db.GetPairInfoFromDB(pair.ID)
		if err != nil {
			continue
		}

		historical, err := h.db.GetHistoricalDataFromDB(pair.ID)
		if err != nil {
			continue
		}

		result[pair.Name] = gin.H{
			"pair_info":  pair,
			"info":       infos,
			"historical": historical,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"pairs": result,
		"count": len(pairs),
	})
}

func (h *Handler) SaveDataToDB() error {
	_, err := h.client.GetServerStatus()
	if err != nil {
		return err
	}

	errorJSON := "[]"
	if err != nil {
		errorJSON = fmt.Sprintf(`["%s"]`, err.Error())
	}

	serverStatus := &models.ServerStatus{
		Timestamp: time.Now(),
		Status:    "online",
		Error:     errorJSON,
	}
	if err := h.db.SaveServerStatus(serverStatus); err != nil {
		return err
	}

	pairs, err := h.client.GetTradingPairs()
	if err != nil {
		return err
	}

	type pairVolume struct {
		name   string
		volume float64
		data   map[string]any
	}

	var pairVolumes []pairVolume

	for pairName, pairData := range pairs {
		if pairInfo, ok := pairData.(map[string]any); ok {
			if volumeData, ok := pairInfo["v"].([]any); ok && len(volumeData) > 1 {
				if volume24h, ok := volumeData[1].(string); ok {
					if vol, err := strconv.ParseFloat(volume24h, 64); err == nil {
						pairVolumes = append(pairVolumes, pairVolume{
							name:   pairName,
							volume: vol,
							data:   pairInfo,
						})
					}
				}
			}
		}
	}

	sort.Slice(pairVolumes, func(i, j int) bool {
		return pairVolumes[i].volume > pairVolumes[j].volume
	})

	now := time.Now()
	lastCandleTime := now.Truncate(5 * time.Minute)
	lastCandleTimestamp := lastCandleTime.Unix()

	lastCSV, err := h.getLatestCSV()
	if err == nil {
		lastCSVDate := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(lastCSV), "top10_5min_highlow_"), ".csv")
		lastCSVTime, err := time.Parse("20060102150405", lastCSVDate)
		if err == nil && lastCSVTime.Equal(lastCandleTime) {
			fmt.Printf("Pas de nouveau CSV à créer, nous sommes dans la même bougie de 5 minutes\n")
		} else {
			if err := h.createCSV(now); err != nil {
				fmt.Printf("Erreur lors de la création du CSV: %v\n", err)
			} else {
				fmt.Printf("Nouveau CSV créé pour la bougie de %s\n", lastCandleTime.Format("2006-01-02 15:04:05"))
			}
		}
	} else {
		if err := h.createCSV(now); err != nil {
			fmt.Printf("Erreur lors de la création du CSV: %v\n", err)
		} else {
			fmt.Printf("Premier CSV créé pour la bougie de %s\n", lastCandleTime.Format("2006-01-02 15:04:05"))
		}
	}

	for i := 0; i < len(pairVolumes) && i < 10; i++ {
		pair := &models.TradingPair{
			Name:        pairVolumes[i].name,
			Base:        pairVolumes[i].data["base"].(string),
			Quote:       pairVolumes[i].data["quote"].(string),
			LastUpdated: lastCandleTime,
		}

		if err := h.db.SaveTradingPair(pair); err != nil {
			continue
		}

		info, err := h.client.GetPairInfo(pair.Name)
		if err != nil {
			continue
		}

		if pairInfo, ok := info[pair.Name].(map[string]any); ok {
			price, _ := strconv.ParseFloat(pairInfo["c"].([]any)[0].(string), 64)
			volume24h, _ := strconv.ParseFloat(pairInfo["v"].([]any)[1].(string), 64)
			high24h, _ := strconv.ParseFloat(pairInfo["h"].([]any)[1].(string), 64)
			low24h, _ := strconv.ParseFloat(pairInfo["l"].([]any)[1].(string), 64)

			info := &models.PairInfo{
				PairID:    pair.ID,
				Price:     price,
				Volume24h: volume24h,
				High24h:   high24h,
				Low24h:    low24h,
				Timestamp: lastCandleTime,
			}
			if err := h.db.SavePairInfo(info); err != nil {
				continue
			}
		}

		historical, err := h.client.GetHistoricalData(pair.Name, 5, lastCandleTimestamp)
		if err != nil {
			continue
		}

		if result, ok := historical["result"].(map[string]any); ok {
			if ohlc, ok := result[pair.Name].([]any); ok && len(ohlc) > 0 {
				for _, candle := range ohlc {
					if candleData, ok := candle.([]any); ok {
						candleTimestamp := time.Unix(int64(candleData[0].(float64)), 0)
						if candleTimestamp.Equal(lastCandleTime) {
							open, _ := strconv.ParseFloat(candleData[1].(string), 64)
							high, _ := strconv.ParseFloat(candleData[2].(string), 64)
							low, _ := strconv.ParseFloat(candleData[3].(string), 64)
							close, _ := strconv.ParseFloat(candleData[4].(string), 64)
							volume, _ := strconv.ParseFloat(candleData[6].(string), 64)

							data := &models.HistoricalData{
								PairID:    pair.ID,
								Timestamp: candleTimestamp,
								Open:      open,
								High:      high,
								Low:       low,
								Close:     close,
								Volume:    volume,
							}
							if err := h.db.SaveHistoricalData(data); err != nil {
								continue
							}
							break
						}
					}
				}
			}
		}
	}

	return nil
}

func (h *Handler) SaveDataNow(c *gin.Context) {
	if err := h.SaveDataToDB(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Erreur lors de la sauvegarde des données",
			"details": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Données sauvegardées avec succès",
	})
}

func (h *Handler) StartAutoSave() {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := h.SaveDataToDB(); err != nil {
					fmt.Printf("Erreur lors de la sauvegarde automatique: %v\n", err)
				} else {
					fmt.Printf("Données sauvegardées automatiquement à %v\n", time.Now().Format("2006-01-02 15:04:05"))
				}
			}
		}
	}()
}
