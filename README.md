# govee-go

**A clean, type-safe Go client for the official Govee Open API**

[![Go](https://github.com/jpenagg/govee/actions/workflows/go.yml/badge.svg)](https://github.com/jpenagg/govee/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/jpenagg/govee.svg)](https://pkg.go.dev/github.com/jpenagg/govee)

Control your Govee lights from Go — fully typed and automatically validated.

### Features
- Full device discovery with human-readable names
- Complete capability parsing (brightness range, color temp, segment count, music modes, etc.)
- Safe command builders — invalid values are rejected **before** hitting the network
- Ready for CLI tools, Home Assistant, servers, or automation

Tested with:
- H706A / M1 Matter RGBIC strip (15 segments)
- H619x, H615x, H61xx lights
- Plugs, bulbs, and more

### Quick Start

```bash
export GOVEE_API_KEY="your-api-key-here"
go run main.go
