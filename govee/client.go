package govee

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	APIKey     string
	HTTPClient *http.Client
}

type Device struct {
	SKU         string `json:"sku"`
	Device      string `json:"device"`
	DeviceName  string `json:"deviceName"`
	Type        string `json:"type"`
	Capabilities []Capability `json:"capabilities"`
}

type Capability struct {
	Type     string          `json:"type"`
	Instance string          `json:"instance"`
	Parameters json.RawMessage `json:"parameters"`
}

type DevicesResponse struct {
	Code    int      `json:"code"`
	Message string   `json:"message"`
	Data    []Device `json:"data"`
}

func NewClient(apiKey string) *Client {
	return &Client{
		APIKey: apiKey,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) GetDevices() (*DevicesResponse, error) {
	url := "https://openapi.api.govee.com/router/api/v1/user/devices"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Govee-API-Key", c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var result DevicesResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Code != 200 {
		return nil, fmt.Errorf("API error %d: %s", result.Code, result.Message)
	}
	return &result, nil
}
