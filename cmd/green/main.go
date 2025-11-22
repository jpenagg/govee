package main

import (
	"log"
	"os"

	"github.com/jpenagg/govee/govee"
)

func main() {
	client := govee.NewClient(os.Getenv("GOVEE_API_KEY"))
	dev, err := client.FindDeviceByName("52 York")
	if err != nil {
		log.Fatal(err)
	}
	if err := client.SetSolidColor(dev, 0x00FF00); err != nil {
		log.Fatal(err)
	}
	println("52 York → GREEN")
}
