// Command govee controls Govee lights from the terminal.
//
//	govee list
//	govee off "Living Room"
//	govee color "Living Room" ff8800
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jpenagg/govee"
)

const usage = `Usage: govee <command> [arguments]

Commands:
  list [-json]                       list devices on the account
  state <device>                     show a device's current state
  on <device>                        turn a device on
  off <device>                       turn a device off
  brightness <device> <percent>      set brightness (1-100)
  color <device> <hex>               set a color, e.g. ff8800 or #0f0
  temp <device> <kelvin>             set white color temperature
  segment <device> <segments> <hex>  color segments, e.g. 0-4,9

<device> is the name from the Govee Home app (any case) or the device ID.
The API key is read from GOVEE_API_KEY.
`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		if errors.Is(err, errUsage) {
			fmt.Fprint(os.Stderr, usage)
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, "govee:", strings.TrimPrefix(err.Error(), "govee: "))
		os.Exit(1)
	}
}

var errUsage = errors.New("usage")

func run(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		return errUsage
	}

	apiKey := os.Getenv("GOVEE_API_KEY")
	if apiKey == "" {
		return errors.New("GOVEE_API_KEY is not set; request a key in the Govee Home app under Settings → Apply for API Key")
	}
	client := govee.NewClient(apiKey)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd, args := args[0], args[1:]
	if cmd == "list" {
		return list(ctx, client, args, out)
	}

	want := map[string]int{"state": 1, "on": 1, "off": 1, "brightness": 2, "color": 2, "temp": 2, "segment": 3}
	n, ok := want[cmd]
	if !ok {
		return fmt.Errorf("unknown command %q (run 'govee help')", cmd)
	}
	if len(args) != n {
		return errUsage
	}

	dev, err := client.Device(ctx, args[0])
	if err != nil {
		return err
	}

	switch cmd {
	case "state":
		return state(ctx, client, dev, out)
	case "on":
		return client.TurnOn(ctx, dev)
	case "off":
		return client.TurnOff(ctx, dev)
	case "brightness":
		v, err := strconv.Atoi(strings.TrimSuffix(args[1], "%"))
		if err != nil {
			return fmt.Errorf("brightness must be a number: %q", args[1])
		}
		return client.SetBrightness(ctx, dev, v)
	case "color":
		c, err := govee.ParseHex(args[1])
		if err != nil {
			return err
		}
		return client.SetColor(ctx, dev, c)
	case "temp":
		k, err := strconv.Atoi(strings.TrimSuffix(strings.ToUpper(args[1]), "K"))
		if err != nil {
			return fmt.Errorf("temperature must be a number of kelvin: %q", args[1])
		}
		return client.SetColorTemperature(ctx, dev, k)
	case "segment":
		segs, err := parseSegments(args[1])
		if err != nil {
			return err
		}
		c, err := govee.ParseHex(args[2])
		if err != nil {
			return err
		}
		return client.SetSegmentColor(ctx, dev, segs, c)
	}
	return nil
}

func list(ctx context.Context, client *govee.Client, args []string, out io.Writer) error {
	asJSON := len(args) == 1 && (args[0] == "-json" || args[0] == "--json")
	if len(args) > 0 && !asJSON {
		return errUsage
	}

	devices, err := client.Devices(ctx)
	if err != nil {
		return err
	}
	if asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(devices)
	}
	if len(devices) == 0 {
		fmt.Fprintln(out, "No devices found.")
		return nil
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tMODEL\tDEVICE ID\tTYPE\tCAPABILITIES")
	for _, d := range devices {
		instances := make([]string, 0, len(d.Capabilities))
		for _, c := range d.Capabilities {
			instances = append(instances, c.Instance)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			d.Name, d.SKU, d.ID, strings.TrimPrefix(d.Type, "devices.types."), strings.Join(instances, ", "))
	}
	return w.Flush()
}

func state(ctx context.Context, client *govee.Client, dev govee.Device, out io.Writer) error {
	st, err := client.State(ctx, dev)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "%s (%s)\n", dev.Name, dev.SKU)
	for _, c := range st.Capabilities {
		value := string(c.State.Value)
		if c.Instance == govee.InstanceColorRGB {
			if v, ok := st.Int(c.Instance); ok {
				value = govee.RGBFromInt(v).String()
			}
		}
		if value == "" || value == `""` {
			value = "-"
		}
		fmt.Fprintf(w, "  %s\t%s\n", c.Instance, value)
	}
	return w.Flush()
}

// parseSegments parses a list like "0-4,7,9" into segment numbers.
func parseSegments(s string) ([]int, error) {
	var segs []int
	for part := range strings.SplitSeq(s, ",") {
		lo, hi, isRange := strings.Cut(strings.TrimSpace(part), "-")
		a, err := strconv.Atoi(lo)
		if err != nil {
			return nil, fmt.Errorf("invalid segment list %q", s)
		}
		b := a
		if isRange {
			if b, err = strconv.Atoi(hi); err != nil || b < a {
				return nil, fmt.Errorf("invalid segment range %q", part)
			}
		}
		for i := a; i <= b; i++ {
			segs = append(segs, i)
		}
	}
	return segs, nil
}
