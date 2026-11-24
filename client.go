// Package govee is a client for the Govee Open API
// (https://developer.govee.com), the router API that lists devices,
// reads their state and controls them by capability.
package govee

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultBaseURL is the production endpoint of the Govee router API.
const DefaultBaseURL = "https://openapi.api.govee.com/router/api/v1"

// maxResponseSize caps how much of a response body is read.
const maxResponseSize = 4 << 20

// Client talks to the Govee Open API. It is safe for concurrent use.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets the HTTP client used for requests.
// The default has a 15 second timeout.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.httpClient = h }
}

// WithBaseURL points the client at another endpoint, such as a test server.
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") }
}

// NewClient returns a client that authenticates with apiKey.
// Request a key in the Govee Home app under Settings → Apply for API Key.
func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

var (
	// ErrNotFound is returned when no device matches a lookup.
	ErrNotFound = errors.New("govee: device not found")
	// ErrUnsupported is returned when a device does not advertise the
	// capability a command needs.
	ErrUnsupported = errors.New("govee: capability not supported by device")
)

// APIError is returned when the API rejects a request, either with a
// non-2xx HTTP status or with a non-200 code in the response body.
// Govee often answers HTTP 200 and reports the failure in the body.
type APIError struct {
	StatusCode int    // HTTP status code
	Code       int    // code from the response body, if any
	Message    string // message from the response body, if any
}

func (e *APIError) Error() string {
	code := e.Code
	if code == 0 {
		code = e.StatusCode
	}
	if e.Message == "" {
		return fmt.Sprintf("govee: API error %d", code)
	}
	return fmt.Sprintf("govee: API error %d: %s", code, e.Message)
}

// RateLimited reports whether the request was refused for exceeding
// Govee's rate limits. Callers should back off before retrying.
func (e *APIError) RateLimited() bool {
	return e.StatusCode == http.StatusTooManyRequests || e.Code == http.StatusTooManyRequests
}

// envelope is the common response wrapper. The devices endpoint uses
// "message" and "data"; the state and control endpoints use "msg" and
// "payload".
type envelope struct {
	RequestID string          `json:"requestId"`
	Code      int             `json:"code"`
	Message   string          `json:"message"`
	Msg       string          `json:"msg"`
	Data      json.RawMessage `json:"data"`
	Payload   json.RawMessage `json:"payload"`
}

func (e *envelope) message() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Msg
}

// request is the body shape for the state and control endpoints.
type request struct {
	RequestID string `json:"requestId"`
	Payload   any    `json:"payload"`
}

func (c *Client) do(ctx context.Context, method, path string, body any) (*envelope, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("govee: encode request: %w", err)
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("govee: build request: %w", err)
	}
	req.Header.Set("Govee-API-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("govee: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("govee: read response: %w", err)
	}

	var env envelope
	jsonErr := json.Unmarshal(raw, &env)

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if jsonErr == nil {
			apiErr.Code = env.Code
			apiErr.Message = env.message()
		} else {
			apiErr.Message = strings.TrimSpace(string(raw))
		}
		return nil, apiErr
	}
	if jsonErr != nil {
		return nil, fmt.Errorf("govee: decode response: %w", jsonErr)
	}
	if env.Code != 0 && env.Code != http.StatusOK {
		return nil, &APIError{StatusCode: resp.StatusCode, Code: env.Code, Message: env.message()}
	}
	return &env, nil
}

// newRequestID returns a random UUID (version 4).
func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:]) // crypto/rand.Read never fails on supported platforms
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Devices lists every device on the account.
func (c *Client) Devices(ctx context.Context) ([]Device, error) {
	env, err := c.do(ctx, http.MethodGet, "/user/devices", nil)
	if err != nil {
		return nil, err
	}
	var devices []Device
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &devices); err != nil {
			return nil, fmt.Errorf("govee: decode devices: %w", err)
		}
	}
	return devices, nil
}

// Device finds a device by its name in the Govee Home app (ignoring case)
// or by its device ID. It returns ErrNotFound if nothing matches.
//
// Each call lists all devices, and that endpoint is limited to 30 requests
// per minute, so long-running programs should call Devices once and use
// FindDevice on the result.
func (c *Client) Device(ctx context.Context, nameOrID string) (Device, error) {
	devices, err := c.Devices(ctx)
	if err != nil {
		return Device{}, err
	}
	return FindDevice(devices, nameOrID)
}

// FindDevice returns the device whose ID matches nameOrID exactly or whose
// name matches it ignoring case.
func FindDevice(devices []Device, nameOrID string) (Device, error) {
	for _, d := range devices {
		if d.ID == nameOrID || strings.EqualFold(d.Name, nameOrID) {
			return d, nil
		}
	}
	return Device{}, fmt.Errorf("%w: %q", ErrNotFound, nameOrID)
}

// State fetches the current value of each capability of a device.
func (c *Client) State(ctx context.Context, dev Device) (*State, error) {
	env, err := c.do(ctx, http.MethodPost, "/device/state", request{
		RequestID: newRequestID(),
		Payload:   devicePayload{SKU: dev.SKU, Device: dev.ID},
	})
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(env.Payload, &s); err != nil {
		return nil, fmt.Errorf("govee: decode state: %w", err)
	}
	return &s, nil
}

type devicePayload struct {
	SKU        string   `json:"sku"`
	Device     string   `json:"device"`
	Capability *Command `json:"capability,omitempty"`
}

// Command is a single capability change, the unit the control endpoint
// accepts. Use it directly for capabilities without a helper method, such
// as scenes, toggles or work modes. Value is encoded as JSON.
type Command struct {
	Type     string `json:"type"`
	Instance string `json:"instance"`
	Value    any    `json:"value"`
}

// Control sends a command to a device. It does no validation; the helper
// methods such as SetBrightness check values against the device's
// advertised capabilities first.
func (c *Client) Control(ctx context.Context, dev Device, cmd Command) error {
	_, err := c.do(ctx, http.MethodPost, "/device/control", request{
		RequestID: newRequestID(),
		Payload:   devicePayload{SKU: dev.SKU, Device: dev.ID, Capability: &cmd},
	})
	return err
}
