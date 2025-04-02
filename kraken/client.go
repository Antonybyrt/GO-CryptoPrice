package kraken

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL = "https://api.kraken.com/0"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

func (c *Client) GetServerStatus() (map[string]any, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/public/Time", baseURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Client) GetTradingPairs() (map[string]any, error) {
	pairsResp, err := c.httpClient.Get(fmt.Sprintf("%s/public/AssetPairs", baseURL))
	if err != nil {
		return nil, err
	}
	defer pairsResp.Body.Close()

	pairsBody, err := io.ReadAll(pairsResp.Body)
	if err != nil {
		return nil, err
	}

	var pairsResult struct {
		Error  []string       `json:"error"`
		Result map[string]any `json:"result"`
	}

	if err := json.Unmarshal(pairsBody, &pairsResult); err != nil {
		return nil, err
	}

	if len(pairsResult.Error) > 0 {
		return nil, fmt.Errorf("API error: %v", pairsResult.Error)
	}

	tickerResp, err := c.httpClient.Get(fmt.Sprintf("%s/public/Ticker", baseURL))
	if err != nil {
		return nil, err
	}
	defer tickerResp.Body.Close()

	tickerBody, err := io.ReadAll(tickerResp.Body)
	if err != nil {
		return nil, err
	}

	var tickerResult struct {
		Error  []string       `json:"error"`
		Result map[string]any `json:"result"`
	}

	if err := json.Unmarshal(tickerBody, &tickerResult); err != nil {
		return nil, err
	}

	if len(tickerResult.Error) > 0 {
		return nil, fmt.Errorf("API error: %v", tickerResult.Error)
	}

	for pairName, pairData := range pairsResult.Result {
		if tickerData, ok := tickerResult.Result[pairName]; ok {
			if pairInfo, ok := pairData.(map[string]any); ok {
				pairInfo["v"] = tickerData.(map[string]any)["v"]
			}
		}
	}

	return pairsResult.Result, nil
}

func (c *Client) GetPairInfo(pair string) (map[string]any, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/public/Ticker?pair=%s", baseURL, pair))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Client) GetHistoricalData(pair string, interval int64, since int64) (map[string]any, error) {
	url := fmt.Sprintf("%s/public/OHLC?pair=%s&interval=%d", baseURL, pair, interval)
	if since > 0 {
		url += fmt.Sprintf("&since=%d", since)
	}

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}
