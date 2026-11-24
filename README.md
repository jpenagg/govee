# govee

[![Go](https://github.com/jpenagg/govee/actions/workflows/go.yml/badge.svg)](https://github.com/jpenagg/govee/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/jpenagg/govee.svg)](https://pkg.go.dev/github.com/jpenagg/govee)

A Go client and command-line tool for the [Govee Open API](https://developer.govee.com), the cloud API for Govee lights, plugs and appliances. No dependencies beyond the standard library.

- List devices and the capabilities each one supports
- Turn devices on and off, and set brightness, color, color temperature and per-segment color
- Read device state: power, brightness, color, sensor readings
- Values are checked against the device's own ranges before anything is sent
- Send any other capability (scenes, toggles, work modes) with `Control`

## Get an API key

In the Govee Home app, open your profile, go to **Settings → Apply for API Key**, and fill in the form. Govee emails the key to you. Then:

```bash
export GOVEE_API_KEY="your-key"
```

## Command-line tool

```bash
go install github.com/jpenagg/govee/cmd/govee@latest
```

```
$ govee list
NAME         MODEL  DEVICE ID                TYPE   CAPABILITIES
Living Room  H6159  AA:BB:CC:DD:EE:FF:00:11  light  powerSwitch, brightness, colorRgb, colorTemperatureK, ...
Desk Strip   H706A  11:22:33:44:55:66:77:88  light  powerSwitch, brightness, segmentedColorRgb, ...

$ govee on "living room"
$ govee brightness "Living Room" 40
$ govee color "Living Room" ff8800
$ govee temp "Living Room" 2700
$ govee segment "Desk Strip" 0-4,10 00ff00
$ govee state "Living Room"
Living Room (H6159)
  online       true
  powerSwitch  1
  brightness   40
  colorRgb     #ff8800
```

Devices can be named by their name in the Govee Home app (any case) or by device ID. `govee list -json` prints the full capability data.

## Library

```bash
go get github.com/jpenagg/govee
```

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/jpenagg/govee"
)

func main() {
	ctx := context.Background()
	client := govee.NewClient(os.Getenv("GOVEE_API_KEY"))

	dev, err := client.Device(ctx, "Living Room")
	if err != nil {
		log.Fatal(err)
	}

	if err := client.TurnOn(ctx, dev); err != nil {
		log.Fatal(err)
	}
	if err := client.SetColor(ctx, dev, govee.RGB{R: 255, G: 136}); err != nil {
		log.Fatal(err)
	}

	state, err := client.State(ctx, dev)
	if err != nil {
		log.Fatal(err)
	}
	brightness, _ := state.Int(govee.InstanceBrightness)
	log.Printf("on=%v brightness=%d", state.On(), brightness)
}
```

### Methods

| Method | What it does |
| --- | --- |
| `Devices(ctx)` | List every device with its capabilities |
| `Device(ctx, nameOrID)` | Fetch the list and return one device (`FindDevice` does the same on a list you already have) |
| `State(ctx, dev)` | Current values; read them with `On`, `Online`, `Int`, `Float` or `Value` |
| `TurnOn`, `TurnOff` | Power |
| `SetBrightness(ctx, dev, percent)` | Whole-device brightness |
| `SetColor(ctx, dev, rgb)` | Whole-device color |
| `SetColorTemperature(ctx, dev, kelvin)` | White color temperature |
| `SetSegmentColor(ctx, dev, segments, rgb)` | Color of individual segments on RGBIC strips |
| `SetSegmentBrightness(ctx, dev, segments, percent)` | Brightness of individual segments |
| `Control(ctx, dev, cmd)` | Send any capability, unvalidated |

### Other capabilities

For anything without a helper, find the capability on the device and send its value with `Control`. For example, a scene listed in the device's `lightScene` options:

```go
scene, ok := dev.Capability(govee.InstanceLightScene)
if !ok {
	log.Fatal("no scenes")
}
for _, opt := range scene.Parameters.Options {
	if opt.Name == "Sunset" {
		cmd := govee.Command{Type: scene.Type, Instance: scene.Instance, Value: opt.Value}
		if err := client.Control(ctx, dev, cmd); err != nil {
			log.Fatal(err)
		}
	}
}
```

### Validation

The helper methods check values against the ranges the device reports, so a light that supports 2700–6500 K rejects 2000 K before any request is made. A device that doesn't list the capability at all returns `govee.ErrUnsupported`. If you build a `Device` yourself with only `SKU` and `ID`, validation falls back to Govee's documented defaults.

### Errors

Govee often answers HTTP 200 and puts the real result in the response body. The client checks both and returns a `*govee.APIError` when either reports a failure:

```go
var apiErr *govee.APIError
if errors.As(err, &apiErr) && apiErr.RateLimited() {
	// back off and retry later
}
```

### Rate limits

Govee limits each account. As of the current docs:

| Endpoint | Limit |
| --- | --- |
| List devices | 30 per minute per account |
| Control | 2 per second per device, 12 per second per account |
| State | 30 per minute per device |

`Device(ctx, name)` lists all devices on each call. In a long-running program, call `Devices` once and look devices up with `govee.FindDevice`.

### Options

```go
client := govee.NewClient(key,
	govee.WithHTTPClient(&http.Client{Timeout: 5 * time.Second}),
	govee.WithBaseURL(testServer.URL), // for tests
)
```

## Development

```bash
go test -race ./...
go vet ./...
```

The tests run against a local fake server and need no API key.
