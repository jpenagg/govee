package govee

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	APIKey     string
	HTTPClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		APIKey: apiKey,
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// ==============================
// Full structs (unchanged)
// ==============================

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

type Range struct{ Min, Max, Precision int }
type Option struct{ Name string; Value int }
type Field struct {
	FieldName    string  `json:"fieldName"`
	DataType     string  `json:"dataType"`
	Required     bool    `json:"required,omitempty"`
	Range        *Range  `json:"range,omitempty"`
	ElementRange *Range  `json:"elementRange,omitempty"`
	ElementType  string  `json:"elementType,omitempty"`
	Size         *Size   `json:"size,omitempty"`
}
type Size struct{ Min, Max int }

// ==============================
// Public API
// ==============================

func (c *Client) GetDevices() (*DevicesResponse, error) {
	req, _ := http.NewRequest("GET", "https://openapi.api.govee.com/router/api/v1/user/devices", nil)
	req.Header.Set("Govee-API-Key", c.APIKey)
	resp, err := c.HTTPClient.Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()
	if resp.StatusCode != 200 { return nil, fmt.Errorf("HTTP %d", resp.StatusCode) }
	var r DevicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil { return nil, err }
	if r.Code != 200 { return nil, fmt.Errorf("API error %d: %s", r.Code, r.Message) }
	return &r, nil
}

func (c *Client) FindDeviceByName(name string) (*Device, error) {
	resp, err := c.GetDevices()
	if err != nil { return nil, err }
	for _, d := range resp.Data {
		if d.DeviceName == name { return &d, nil }
	}
	return nil, fmt.Errorf("device %q not found", name)
}

func (c *Client) TurnOn(dev *Device) error      { return c.sendCommand(dev, "powerSwitch", 1) }
func (c *Client) TurnOff(dev *Device) error     { return c.sendCommand(dev, "powerSwitch", 0) }
func (c *Client) SetBrightness(dev *Device, v int) error {
	if v < 1 || v > 100 { return fmt.Errorf("brightness 1–100") }
	return c.sendCommand(dev, "brightness", v)
}
func (c *Client) SetColorTemperature(dev *Device, k int) error {
	if k < 2000 || k > 9000 { return fmt.Errorf("kelvin 2000–9000") }
	return c.sendCommand(dev, "colorTemperatureK", k)
}
func (c *Client) SetSolidColor(dev *Device, rgb int) error {
	if rgb < 0 || rgb > 16777215 { return fmt.Errorf("rgb 0–16777215") }
	return c.sendCommand(dev, "colorRgb", rgb)
}

// ==============================
// Internal sender
// ==============================

func (c *Client) sendCommand(dev *Device, name string, value any) error {
	payload := map[string]any{
		"device": dev.Device,
		"sku":    dev.SKU,
		"name":   name,
		"value":  value,
	}
	body := map[string]any{"request": payload}
	data, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "https://openapi.api.govee.com/router/api/v1/device/control", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Govee-API-Key", c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("control failed (%d): %s", resp.StatusCode, string(b))
	}
	return nil
}
