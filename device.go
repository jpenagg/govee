package govee

import (
	"encoding/json"
	"strconv"
)

// Capability types used by Govee devices.
const (
	TypeOnOff               = "devices.capabilities.on_off"
	TypeToggle              = "devices.capabilities.toggle"
	TypeRange               = "devices.capabilities.range"
	TypeMode                = "devices.capabilities.mode"
	TypeColorSetting        = "devices.capabilities.color_setting"
	TypeSegmentColorSetting = "devices.capabilities.segment_color_setting"
	TypeMusicSetting        = "devices.capabilities.music_setting"
	TypeDynamicScene        = "devices.capabilities.dynamic_scene"
	TypeWorkMode            = "devices.capabilities.work_mode"
	TypeTemperatureSetting  = "devices.capabilities.temperature_setting"
	TypeOnline              = "devices.capabilities.online"
	TypeProperty            = "devices.capabilities.property"
	TypeEvent               = "devices.capabilities.event"
)

// Common capability instances.
const (
	InstancePowerSwitch         = "powerSwitch"
	InstanceBrightness          = "brightness"
	InstanceColorRGB            = "colorRgb"
	InstanceColorTemperatureK   = "colorTemperatureK"
	InstanceSegmentedColorRGB   = "segmentedColorRgb"
	InstanceSegmentedBrightness = "segmentedBrightness"
	InstanceLightScene          = "lightScene"
	InstanceDIYScene            = "diyScene"
	InstanceSnapshot            = "snapshot"
	InstanceMusicMode           = "musicMode"
	InstanceOnline              = "online"
)

// Device is a device on the account, with the capabilities it supports.
type Device struct {
	SKU          string       `json:"sku"`        // model, e.g. "H6159"
	ID           string       `json:"device"`     // device ID, a MAC-like string
	Name         string       `json:"deviceName"` // name set in the Govee Home app
	Type         string       `json:"type"`       // e.g. "devices.types.light"
	Capabilities []Capability `json:"capabilities"`
}

// Capability returns the capability with the given instance name.
func (d Device) Capability(instance string) (Capability, bool) {
	for _, c := range d.Capabilities {
		if c.Instance == instance {
			return c, true
		}
	}
	return Capability{}, false
}

// Supports reports whether the device advertises the given instance.
func (d Device) Supports(instance string) bool {
	_, ok := d.Capability(instance)
	return ok
}

// Capability describes one thing a device can do and the values it takes.
type Capability struct {
	Type       string     `json:"type"`
	Instance   string     `json:"instance"`
	Parameters Parameters `json:"parameters"`
}

// Parameters describes the values a capability accepts. Which fields are
// set depends on DataType: ENUM uses Options, INTEGER uses Range, and
// STRUCT uses Fields.
type Parameters struct {
	DataType string       `json:"dataType"`
	Unit     string       `json:"unit,omitempty"`
	Range    *Range       `json:"range,omitempty"`
	Options  []EnumOption `json:"options,omitempty"`
	Fields   []Field      `json:"fields,omitempty"`
}

// Range is an inclusive numeric range.
type Range struct {
	Min       float64 `json:"min"`
	Max       float64 `json:"max"`
	Precision float64 `json:"precision,omitempty"`
}

// Contains reports whether v lies within the range.
func (r Range) Contains(v float64) bool { return v >= r.Min && v <= r.Max }

func (r Range) String() string {
	return strconv.FormatFloat(r.Min, 'f', -1, 64) + "–" + strconv.FormatFloat(r.Max, 'f', -1, 64)
}

// EnumOption is one allowed value of an ENUM parameter, such as a scene.
// Value is usually a number but can be a string or an object.
type EnumOption struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

// Field is one member of a STRUCT parameter.
type Field struct {
	FieldName    string       `json:"fieldName"`
	DataType     string       `json:"dataType"`
	Required     bool         `json:"required,omitempty"`
	Range        *Range       `json:"range,omitempty"`
	Options      []EnumOption `json:"options,omitempty"`
	ElementRange *Range       `json:"elementRange,omitempty"`
	ElementType  string       `json:"elementType,omitempty"`
	Size         *Size        `json:"size,omitempty"`
	DefaultValue any          `json:"defaultValue,omitempty"`
}

// Size bounds the length of an array field.
type Size struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// Field returns the struct field with the given name.
func (p Parameters) Field(name string) (Field, bool) {
	for _, f := range p.Fields {
		if f.FieldName == name {
			return f, true
		}
	}
	return Field{}, false
}

// State is a snapshot of a device's capability values.
type State struct {
	SKU          string            `json:"sku"`
	ID           string            `json:"device"`
	Capabilities []CapabilityState `json:"capabilities"`
}

// CapabilityState is the current value of one capability.
type CapabilityState struct {
	Type     string `json:"type"`
	Instance string `json:"instance"`
	State    struct {
		Value json.RawMessage `json:"value"`
	} `json:"state"`
}

// Value returns the raw JSON value of an instance. It returns false if the
// instance is missing or its value is empty, which Govee uses for
// instances that cannot be queried.
func (s *State) Value(instance string) (json.RawMessage, bool) {
	for _, c := range s.Capabilities {
		if c.Instance == instance {
			v := c.State.Value
			if len(v) == 0 || string(v) == `""` || string(v) == "null" {
				return nil, false
			}
			return v, true
		}
	}
	return nil, false
}

// Float returns a numeric value, such as a sensor reading.
func (s *State) Float(instance string) (float64, bool) {
	raw, ok := s.Value(instance)
	if !ok {
		return 0, false
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err != nil {
		return 0, false
	}
	return f, true
}

// Int returns an integer value, such as brightness or colorRgb.
func (s *State) Int(instance string) (int, bool) {
	f, ok := s.Float(instance)
	if !ok || f != float64(int(f)) {
		return 0, false
	}
	return int(f), true
}

// Online reports whether the device is reachable. When it is not, the
// other values are the last ones Govee saw.
func (s *State) Online() bool {
	raw, ok := s.Value(InstanceOnline)
	if !ok {
		return false
	}
	var b bool
	return json.Unmarshal(raw, &b) == nil && b
}

// On reports whether the power switch is on.
func (s *State) On() bool {
	v, ok := s.Int(InstancePowerSwitch)
	return ok && v == 1
}
