package main

import (
	"fmt"
	"os"

	"govee-devices/govee"

	"github.com/olekukonko/tablewriter"
)

func main() {
	apiKey := os.Getenv("GOVEE_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: GOVEE_API_KEY environment variable is not set")
		fmt.Println()
		fmt.Println("Run this first:")
		fmt.Println("    export GOVEE_API_KEY=\"your-actual-key-here\"")
		fmt.Println("or on Windows:")
		fmt.Println("    set GOVEE_API_KEY=your-actual-key-here")
		os.Exit(1)
	}

	client := govee.NewClient(apiKey)
	resp, err := client.GetDevices()
	if err != nil {
		fmt.Printf("Failed to fetch devices: %v\n", err)
		os.Exit(1)
	}

	if len(resp.Data) == 0 {
		fmt.Println("No devices found.")
		return
	}

	fmt.Printf("Found %d Govee device(s):\n\n", len(resp.Data))

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Model", "MAC Address", "Type", "Features"})
	table.SetBorder(false)
	table.SetHeaderLine(true)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("│")
	table.SetColumnSeparator("│")
	table.SetRowSeparator("─")

	for _, d := range resp.Data {
		table.Append([]string{
			d.DeviceName,
			d.SKU,
			d.Device,
			d.Type,
			fmt.Sprintf("%d", len(d.Capabilities)),
		})
	}
	table.Render()
}
