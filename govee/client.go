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

type DevicesResponse struct {
	Code    int      `json:"code"`
	Message string   `json:"message"`
	Data    []Device `json:"data"`
}

type Device struct {
	SKU        string       `json:"sku"`
	Device     string       `json:"device"`
	DeviceName string       `json:"deviceName"`
	Type       string       `json:"type"`
	Capabilities []Capability `json:"capabilities"`
}

type Capability struct {
	Type       string               `json:"type"`
	Instance   string               `json:"instance"`
	Parameters CapabilityParameters `json:"parameters"`
}

type CapabilityParameters struct {
	DataType  string   `json:"dataType"`
	Unit      string   `json:"unit,omitempty"`
	Range     *Range   `json:"range,omitempty"`
	Options   []Option `json:"options,omitempty"`
	Fields    []Field  `json:"fields,omitempty"`
}

type Range struct {
	Min       int `json:"min"`
	Max       int `json:"max"`
	Precision int `json:"precision,omitempty"`
}

type Option struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type Field struct {
	FieldName    string  `json:"fieldName"`
	DataType     string  `json:"dataType"`
	Required     bool    `json:"required,omitempty"`
	Range        *Range  `json:"range,omitempty"`
	ElementRange *Range  `json:"elementRange,omitempty"`
	ElementType  string  `json:"elementType,omitempty"`
	Size         *Size   `json:"size,omitempty"`
}

type Size struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

func NewClient(apiKey string) *Client {
	return &Client{
		APIKey: apiKey,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) GetDevices() (*DevicesResponse, error) {
	req, _ := http.NewRequest("GET", "https://openapi.api.govee.com/router/api/v1/user/devices", nil)
	req.Header.Set("Govee-API-Key", c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var result DevicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Code != 200 {
		return nil, fmt.Errorf("API error %d: %s", result.Code, result.Message)
	}
	return &result, nil
}
