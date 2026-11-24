package govee

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const devicesResponse = `{
  "code": 200,
  "message": "success",
  "data": [{
    "sku": "H706A",
    "device": "AA:BB:CC:DD:EE:FF:00:11",
    "deviceName": "Living Room",
    "type": "devices.types.light",
    "capabilities": [
      {
        "type": "devices.capabilities.on_off",
        "instance": "powerSwitch",
        "parameters": {"dataType": "ENUM", "options": [{"name": "on", "value": 1}, {"name": "off", "value": 0}]}
      },
      {
        "type": "devices.capabilities.range",
        "instance": "brightness",
        "parameters": {"unit": "unit.percent", "dataType": "INTEGER", "range": {"min": 1, "max": 100, "precision": 1}}
      },
      {
        "type": "devices.capabilities.color_setting",
        "instance": "colorTemperatureK",
        "parameters": {"dataType": "INTEGER", "range": {"min": 2700, "max": 6500, "precision": 1}}
      },
      {
        "type": "devices.capabilities.segment_color_setting",
        "instance": "segmentedColorRgb",
        "parameters": {
          "dataType": "STRUCT",
          "fields": [
            {"fieldName": "segment", "dataType": "Array", "elementType": "INTEGER", "elementRange": {"min": 0, "max": 14}, "size": {"min": 1, "max": 15}, "required": true},
            {"fieldName": "rgb", "dataType": "INTEGER", "range": {"min": 0, "max": 16777215, "precision": 1}, "required": true}
          ]
        }
      },
      {
        "type": "devices.capabilities.dynamic_scene",
        "instance": "lightScene",
        "parameters": {"dataType": "ENUM", "options": [{"name": "Party", "value": 3055}]}
      }
    ]
  }]
}`

const stateResponse = `{
  "requestId": "x",
  "msg": "success",
  "code": 200,
  "payload": {
    "sku": "H706A",
    "device": "AA:BB:CC:DD:EE:FF:00:11",
    "capabilities": [
      {"type": "devices.capabilities.online", "instance": "online", "state": {"value": true}},
      {"type": "devices.capabilities.on_off", "instance": "powerSwitch", "state": {"value": 1}},
      {"type": "devices.capabilities.range", "instance": "brightness", "state": {"value": 42}},
      {"type": "devices.capabilities.color_setting", "instance": "colorRgb", "state": {"value": 65280}},
      {"type": "devices.capabilities.property", "instance": "sensorTemperature", "state": {"value": 79.52}},
      {"type": "devices.capabilities.property", "instance": "sensorHumidity", "state": {"value": ""}}
    ]
  }
}`

// server records the last request and answers with a fixed status and body.
type server struct {
	t      *testing.T
	status int
	body   string

	calls  int
	method string
	path   string
	header http.Header
	sent   []byte
}

func newTestClient(t *testing.T, status int, body string) (*Client, *server) {
	t.Helper()
	s := &server{t: t, status: status, body: body}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.calls++
		s.method, s.path, s.header = r.Method, r.URL.Path, r.Header
		s.sent, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(s.status)
		_, _ = io.WriteString(w, s.body)
	}))
	t.Cleanup(ts.Close)
	return NewClient("test-key", WithBaseURL(ts.URL)), s
}

func testDevice(t *testing.T) Device {
	t.Helper()
	var env struct{ Data []Device }
	if err := json.Unmarshal([]byte(devicesResponse), &env); err != nil {
		t.Fatal(err)
	}
	return env.Data[0]
}

// controlBody decodes what the client sent to the control endpoint.
func controlBody(t *testing.T, s *server) (requestID string, sku, device string, cmd map[string]any) {
	t.Helper()
	var body struct {
		RequestID string `json:"requestId"`
		Payload   struct {
			SKU        string         `json:"sku"`
			Device     string         `json:"device"`
			Capability map[string]any `json:"capability"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(s.sent, &body); err != nil {
		t.Fatalf("decode sent body %s: %v", s.sent, err)
	}
	return body.RequestID, body.Payload.SKU, body.Payload.Device, body.Payload.Capability
}

func TestDevices(t *testing.T) {
	c, s := newTestClient(t, 200, devicesResponse)

	devices, err := c.Devices(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if s.method != http.MethodGet || s.path != "/user/devices" {
		t.Errorf("request = %s %s", s.method, s.path)
	}
	if got := s.header.Get("Govee-API-Key"); got != "test-key" {
		t.Errorf("Govee-API-Key = %q", got)
	}
	if len(devices) != 1 {
		t.Fatalf("got %d devices, want 1", len(devices))
	}

	d := devices[0]
	if d.Name != "Living Room" || d.SKU != "H706A" || d.ID != "AA:BB:CC:DD:EE:FF:00:11" {
		t.Errorf("device = %+v", d)
	}
	seg, ok := d.Capability(InstanceSegmentedColorRGB)
	if !ok {
		t.Fatal("segmentedColorRgb missing")
	}
	f, ok := seg.Parameters.Field("segment")
	if !ok || f.ElementRange == nil || f.ElementRange.Max != 14 {
		t.Errorf("segment field = %+v", f)
	}
	scene, _ := d.Capability(InstanceLightScene)
	if len(scene.Parameters.Options) != 1 || scene.Parameters.Options[0].Name != "Party" {
		t.Errorf("scene options = %+v", scene.Parameters.Options)
	}
}

func TestFindDevice(t *testing.T) {
	devices := []Device{testDevice(t)}

	for _, q := range []string{"living room", "Living Room", "AA:BB:CC:DD:EE:FF:00:11"} {
		if _, err := FindDevice(devices, q); err != nil {
			t.Errorf("FindDevice(%q): %v", q, err)
		}
	}
	if _, err := FindDevice(devices, "Kitchen"); !errors.Is(err, ErrNotFound) {
		t.Errorf("FindDevice(Kitchen) err = %v, want ErrNotFound", err)
	}
}

func TestControlRequestShape(t *testing.T) {
	c, s := newTestClient(t, 200, `{"requestId":"x","msg":"success","code":200}`)
	dev := testDevice(t)

	if err := c.TurnOn(context.Background(), dev); err != nil {
		t.Fatal(err)
	}
	if s.method != http.MethodPost || s.path != "/device/control" {
		t.Errorf("request = %s %s", s.method, s.path)
	}
	if got := s.header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q", got)
	}

	id, sku, device, cmd := controlBody(t, s)
	if id == "" {
		t.Error("requestId is empty")
	}
	if sku != dev.SKU || device != dev.ID {
		t.Errorf("payload sku/device = %q/%q", sku, device)
	}
	want := map[string]any{"type": TypeOnOff, "instance": "powerSwitch", "value": float64(1)}
	for k, v := range want {
		if cmd[k] != v {
			t.Errorf("capability.%s = %v, want %v", k, cmd[k], v)
		}
	}
}

func TestSetSegmentColor(t *testing.T) {
	c, s := newTestClient(t, 200, `{"code":200}`)

	err := c.SetSegmentColor(context.Background(), testDevice(t), []int{0, 1, 2}, RGB{0, 0, 255})
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, cmd := controlBody(t, s)
	got, _ := json.Marshal(cmd["value"])
	if string(got) != `{"rgb":255,"segment":[0,1,2]}` {
		t.Errorf("value = %s", got)
	}
	if cmd["type"] != TypeSegmentColorSetting {
		t.Errorf("type = %v", cmd["type"])
	}
}

func TestValidationUsesDeviceCapabilities(t *testing.T) {
	c, s := newTestClient(t, 200, `{"code":200}`)
	ctx := context.Background()
	dev := testDevice(t)

	tests := []struct {
		name    string
		call    func() error
		wantErr error // nil means any error
	}{
		{"brightness too high", func() error { return c.SetBrightness(ctx, dev, 101) }, nil},
		{"kelvin below device min", func() error { return c.SetColorTemperature(ctx, dev, 2000) }, nil},
		{"segment out of range", func() error { return c.SetSegmentColor(ctx, dev, []int{15}, RGB{}) }, nil},
		{"no segments", func() error { return c.SetSegmentColor(ctx, dev, nil, RGB{}) }, nil},
		{"unsupported colorRgb", func() error { return c.SetColor(ctx, dev, RGB{R: 255}) }, ErrUnsupported},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil {
				t.Fatal("expected an error")
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
	if s.calls != 0 {
		t.Errorf("%d requests sent; invalid commands should not reach the network", s.calls)
	}

	if err := c.SetColorTemperature(ctx, dev, 4000); err != nil {
		t.Errorf("valid kelvin rejected: %v", err)
	}
}

func TestBareDeviceSkipsValidation(t *testing.T) {
	c, s := newTestClient(t, 200, `{"code":200}`)
	dev := Device{SKU: "H6159", ID: "11:22:33:44:55:66:77:88"}

	if err := c.SetColor(context.Background(), dev, RGB{G: 255}); err != nil {
		t.Fatal(err)
	}
	_, _, _, cmd := controlBody(t, s)
	if cmd["type"] != TypeColorSetting || cmd["value"] != float64(0x00FF00) {
		t.Errorf("capability = %v", cmd)
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		body        string
		wantCode    int
		rateLimited bool
	}{
		{"error code in body with HTTP 200", 200, `{"code":400,"msg":"Parameter value out of range"}`, 400, false},
		{"HTTP 401", 401, `{"code":401,"message":"Invalid API Key"}`, 401, false},
		{"HTTP 429 plain text", 429, `rate limit exceeded`, 0, true},
		{"HTTP 500 empty", 500, ``, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := newTestClient(t, tt.status, tt.body)
			err := c.TurnOff(context.Background(), Device{SKU: "H1", ID: "x"})

			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("err = %v, want *APIError", err)
			}
			if apiErr.Code != tt.wantCode || apiErr.StatusCode != tt.status {
				t.Errorf("APIError = %+v", apiErr)
			}
			if apiErr.RateLimited() != tt.rateLimited {
				t.Errorf("RateLimited() = %v", apiErr.RateLimited())
			}
		})
	}
}

func TestState(t *testing.T) {
	c, s := newTestClient(t, 200, stateResponse)
	dev := testDevice(t)

	st, err := c.State(context.Background(), dev)
	if err != nil {
		t.Fatal(err)
	}
	if s.path != "/device/state" || !strings.Contains(string(s.sent), `"device":"AA:BB:CC:DD:EE:FF:00:11"`) {
		t.Errorf("request = %s %s", s.path, s.sent)
	}
	if !st.Online() || !st.On() {
		t.Errorf("Online=%v On=%v", st.Online(), st.On())
	}
	if b, ok := st.Int(InstanceBrightness); !ok || b != 42 {
		t.Errorf("brightness = %d, %v", b, ok)
	}
	if rgb, _ := st.Int(InstanceColorRGB); RGBFromInt(rgb) != (RGB{0, 255, 0}) {
		t.Errorf("color = %v", RGBFromInt(rgb))
	}
	if f, ok := st.Float("sensorTemperature"); !ok || f != 79.52 {
		t.Errorf("temperature = %v, %v", f, ok)
	}
	if _, ok := st.Value("sensorHumidity"); ok {
		t.Error("empty value should report not available")
	}
}

func TestParseHex(t *testing.T) {
	tests := map[string]RGB{
		"#00ff00": {0, 255, 0},
		"FF8800":  {255, 136, 0},
		"#abc":    {0xaa, 0xbb, 0xcc},
	}
	for in, want := range tests {
		got, err := ParseHex(in)
		if err != nil || got != want {
			t.Errorf("ParseHex(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "#12345", "zzzzzz", "#1234567"} {
		if _, err := ParseHex(bad); err == nil {
			t.Errorf("ParseHex(%q) succeeded", bad)
		}
	}
	if got := (RGB{255, 136, 0}).String(); got != "#ff8800" {
		t.Errorf("String() = %q", got)
	}
}

func TestRequestIDIsUUID(t *testing.T) {
	id := newRequestID()
	if len(id) != 36 || id[14] != '4' || strings.Count(id, "-") != 4 {
		t.Errorf("newRequestID() = %q", id)
	}
	if id == newRequestID() {
		t.Error("request IDs repeat")
	}
}
