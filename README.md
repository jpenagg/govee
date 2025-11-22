# govee-go

**A clean, type-safe Go client for the official Govee Open API**

[![Go](https://github.com/jpenagg/govee-go/actions/workflows/go.yml/badge.svg)](https://github.com/jpenagg/govee-go/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/jpenagg/govee-go.svg)](https://pkg.go.dev/github.com/jpenagg/govee-go)

Control your Govee lights from Go — fully typed, automatically validated, no hardcoded keys.

### Features
- Full device discovery with human-readable names
- Complete capability parsing (brightness range, color temp, segment count, music modes, etc.)
- Safe command builders — invalid values are rejected **before** hitting the network
- No API key in source code (uses `GOVEE_API_KEY` env var)
- Ready for CLI tools, Home Assistant, servers, or automation

Tested with:
- H706A / M1 Matter RGBIC strip (15 segments)
- H619x, H615x, H61xx lights
- Plugs, bulbs, and more

### Quick Start

```bash
export GOVEE_API_KEY="your-api-key-here"
go run main.go
