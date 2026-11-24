package govee

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// RGB is a 24-bit color.
type RGB struct{ R, G, B uint8 }

// Int returns the color packed as 0xRRGGBB, the form the API uses.
func (c RGB) Int() int { return int(c.R)<<16 | int(c.G)<<8 | int(c.B) }

// String returns the color as "#rrggbb".
func (c RGB) String() string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

// RGBFromInt unpacks a 0xRRGGBB value, as returned in device state.
func RGBFromInt(v int) RGB { return RGB{uint8(v >> 16), uint8(v >> 8), uint8(v)} }

// ParseHex parses "#rrggbb", "rrggbb" or "#rgb".
func ParseHex(s string) (RGB, error) {
	h := strings.TrimPrefix(s, "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 {
		return RGB{}, fmt.Errorf("govee: invalid hex color %q", s)
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return RGB{}, fmt.Errorf("govee: invalid hex color %q", s)
	}
	return RGBFromInt(int(v)), nil
}

// TurnOn switches the device on.
func (c *Client) TurnOn(ctx context.Context, dev Device) error {
	return c.setPower(ctx, dev, 1)
}

// TurnOff switches the device off.
func (c *Client) TurnOff(ctx context.Context, dev Device) error {
	return c.setPower(ctx, dev, 0)
}

func (c *Client) setPower(ctx context.Context, dev Device, v int) error {
	typ, _, err := resolve(dev, InstancePowerSwitch, TypeOnOff)
	if err != nil {
		return err
	}
	return c.Control(ctx, dev, Command{Type: typ, Instance: InstancePowerSwitch, Value: v})
}

// SetBrightness sets brightness as a percentage, usually 1–100.
func (c *Client) SetBrightness(ctx context.Context, dev Device, percent int) error {
	return c.setInt(ctx, dev, InstanceBrightness, TypeRange, percent, Range{Min: 1, Max: 100})
}

// SetColor sets the whole device to one color.
func (c *Client) SetColor(ctx context.Context, dev Device, color RGB) error {
	return c.setInt(ctx, dev, InstanceColorRGB, TypeColorSetting, color.Int(), Range{Min: 0, Max: 0xFFFFFF})
}

// SetColorTemperature sets white light to a color temperature in kelvin.
// The allowed range depends on the device, commonly 2000–9000.
func (c *Client) SetColorTemperature(ctx context.Context, dev Device, kelvin int) error {
	return c.setInt(ctx, dev, InstanceColorTemperatureK, TypeColorSetting, kelvin, Range{Min: 2000, Max: 9000})
}

func (c *Client) setInt(ctx context.Context, dev Device, instance, defaultType string, v int, fallback Range) error {
	typ, capability, err := resolve(dev, instance, defaultType)
	if err != nil {
		return err
	}
	r := fallback
	if capability.Parameters.Range != nil {
		r = *capability.Parameters.Range
	}
	if !r.Contains(float64(v)) {
		return fmt.Errorf("govee: %s %d out of range %s for %s", instance, v, r, dev.SKU)
	}
	return c.Control(ctx, dev, Command{Type: typ, Instance: instance, Value: v})
}

// SetSegmentColor colors the given segments of an RGBIC device. Segments
// are numbered from 0; see the device's segmentedColorRgb capability for
// how many it has.
func (c *Client) SetSegmentColor(ctx context.Context, dev Device, segments []int, color RGB) error {
	typ, capability, err := resolve(dev, InstanceSegmentedColorRGB, TypeSegmentColorSetting)
	if err != nil {
		return err
	}
	if err := checkSegments(capability, segments); err != nil {
		return err
	}
	return c.Control(ctx, dev, Command{
		Type:     typ,
		Instance: InstanceSegmentedColorRGB,
		Value: struct {
			Segment []int `json:"segment"`
			RGB     int   `json:"rgb"`
		}{segments, color.Int()},
	})
}

// SetSegmentBrightness sets the brightness percentage of the given segments.
func (c *Client) SetSegmentBrightness(ctx context.Context, dev Device, segments []int, percent int) error {
	typ, capability, err := resolve(dev, InstanceSegmentedBrightness, TypeSegmentColorSetting)
	if err != nil {
		return err
	}
	if err := checkSegments(capability, segments); err != nil {
		return err
	}
	r := Range{Min: 0, Max: 100}
	if f, ok := capability.Parameters.Field("brightness"); ok && f.Range != nil {
		r = *f.Range
	}
	if !r.Contains(float64(percent)) {
		return fmt.Errorf("govee: segment brightness %d out of range %s", percent, r)
	}
	return c.Control(ctx, dev, Command{
		Type:     typ,
		Instance: InstanceSegmentedBrightness,
		Value: struct {
			Segment    []int `json:"segment"`
			Brightness int   `json:"brightness"`
		}{segments, percent},
	})
}

// resolve finds the capability a command needs and returns its type.
// If the device lists capabilities but not this one, it returns
// ErrUnsupported. If the device lists none (for example a Device built by
// hand from an ID and SKU), it falls back to defaultType and skips
// validation.
func resolve(dev Device, instance, defaultType string) (string, Capability, error) {
	capability, ok := dev.Capability(instance)
	if !ok {
		if len(dev.Capabilities) > 0 {
			return "", Capability{}, fmt.Errorf("%w: %s has no %s", ErrUnsupported, dev.SKU, instance)
		}
		capability = Capability{Type: defaultType, Instance: instance}
	}
	return capability.Type, capability, nil
}

func checkSegments(capability Capability, segments []int) error {
	if len(segments) == 0 {
		return fmt.Errorf("govee: no segments given")
	}
	f, ok := capability.Parameters.Field("segment")
	if !ok {
		return nil
	}
	if f.Size != nil && f.Size.Max > 0 && len(segments) > f.Size.Max {
		return fmt.Errorf("govee: %d segments given, device accepts at most %d", len(segments), f.Size.Max)
	}
	if f.ElementRange != nil {
		for _, s := range segments {
			if !f.ElementRange.Contains(float64(s)) {
				return fmt.Errorf("govee: segment %d out of range %s", s, f.ElementRange)
			}
		}
	}
	return nil
}
