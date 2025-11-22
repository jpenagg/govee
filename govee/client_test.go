package govee

import (
	"encoding/json"
	"testing"
)

const sampleResponse = `
{
  "code": 200,
  "message": "success",
  "data": [
    {
      "sku": "H706A",
      "device": "2F:FA:C6:24:9B:FA:ED:39",
      "deviceName": "52 York",
      "type": "devices.types.light",
      "capabilities": [
        {
          "type": "devices.capabilities.on_off",
          "instance": "powerSwitch",
          "parameters": { "dataType": "ENUM", "options": [{"name":"on","value":1},{"name":"off","value":0}] }
        },
        {
          "type": "devices.capabilities.range",
          "instance": "brightness",
          "parameters": { "dataType": "INTEGER", "range": {"min":1,"max":100,"precision":1} }
        },
        {
          "type": "devices.capabilities.segment_color_setting",
          "instance": "segmentedColorRgb",
          "parameters": {
            "dataType": "STRUCT",
            "fields": [
              {"fieldName": "segment", "dataType": "Array", "elementType": "INTEGER", "elementRange": {"min":0,"max":14}},
              {"fieldName": "rgb", "dataType": "INTEGER", "range": {"min":0,"max":16777215}}
            ]
          }
        }
      ]
    }
  ]
}
`

func TestUnmarshalSampleResponse(t *testing.T) {
	var resp DevicesResponse
	if err := json.Unmarshal([]byte(sampleResponse), &resp); err != nil {
		t.Fatalf("Failed to unmarshal sample response: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("Expected 1 device, got %d", len(resp.Data))
	}

	dev := resp.Data[0]
	if dev.DeviceName != "52 York" {
		t.Errorf("Expected name '52 York', got %s", dev.DeviceName)
	}
	if dev.SKU != "H706A" {
		t.Errorf("Expected SKU H706A, got %s", dev.SKU)
	}

	// Test segment parsing
	var segCap *Capability
	for _, c := range dev.Capabilities {
		if c.Instance == "segmentedColorRgb" {
			segCap = &c
			break
		}
	}
	if segCap == nil {
		t.Fatal("segmentedColorRgb capability not found")
	}

	segField := findField(segCap.Parameters.Fields, "segment")
	if segField == nil || segField.ElementRange == nil {
		t.Fatal("Failed to parse segment range")
	}
	if segField.ElementRange.Max != 14 {
		t.Errorf("Expected max segment 14, got %d", segField.ElementRange.Max)
	}
}

// Helper used in main.go too
func findField(fields []Field, name string) *Field {
	for _, f := range fields {
		if f.FieldName == name {
			return &f
		}
	}
	return nil
}
