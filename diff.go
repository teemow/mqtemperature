package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"github.com/spf13/cobra"
	"github.com/teemow/mqtemperature/onewire"
)

var (
	diffFlags struct {
		Host     string
		Port     int
		Topic    string
		Device   string
		Device2  string
		Interval int
	}

	diffCmd = &cobra.Command{
		Use:   "diff",
		Short: "Publish temperature difference via MQTT",
		Long:  "Send MQTT messages to a broker. Especially made for One-Wire (DS18S20) temperature sensors on a Raspberry Pi",
		Run:   diffRun,
	}
)

func init() {
	diffCmd.PersistentFlags().BoolVarP(&globalFlags.debug, "debug", "d", false, "Print debug output")
	diffCmd.PersistentFlags().BoolVarP(&globalFlags.verbose, "verbose", "v", false, "Print verbose output")
	diffCmd.PersistentFlags().StringVar(&diffFlags.Host, "host", "localhost", "MQTT host")
	diffCmd.PersistentFlags().IntVar(&diffFlags.Port, "port", 1883, "MQTT port")
	diffCmd.PersistentFlags().StringVar(&diffFlags.Topic, "topic", "", "MQTT topic")
	diffCmd.PersistentFlags().StringVar(&diffFlags.Device, "device", "", "DS18S20 device")
	diffCmd.PersistentFlags().StringVar(&diffFlags.Device2, "device2", "", "DS18S20 device")
	diffCmd.PersistentFlags().IntVar(&diffFlags.Interval, "interval", 60, "Interval in seconds")
}

func watchTemperatureDifference(client *mqtt.Client, device, device2 *onewire.DS18S20) {
	for {
		value, err := device.Read()
		if err != nil {
			fmt.Println("Could not read temperature ", err)
		} else {
			value2, err := device2.Read()
			if err != nil {
				fmt.Println("Could not read temperature ", err)
			} else {
				topic := fmt.Sprintf("mqtemperature/%s", diffFlags.Topic)
				payload := fmt.Sprintf("%d", value-value2)

				if token := client.Publish(topic, 1, false, payload); token.Wait() && token.Error() != nil {
					fmt.Printf("Failed to send message: %v\n", token.Error())
				}
			}
		}
		time.Sleep(time.Duration(diffFlags.Interval) * time.Second)
	}
}

func diffRun(cmd *cobra.Command, args []string) {
	// mqtt
	opts := mqtt.NewClientOptions().AddBroker(fmt.Sprintf("tcp://%s:%d", diffFlags.Host, diffFlags.Port))
	opts.SetClientID(fmt.Sprintf("mqtemperature-%s", diffFlags.Topic))

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}

	device, err := onewire.NewDS18S20(diffFlags.Device)
	if err != nil {
		log.Fatalf("Device %s not found", diffFlags.Device)
	}
	device2, err := onewire.NewDS18S20(diffFlags.Device2)
	if err != nil {
		log.Fatalf("Device %s not found", diffFlags.Device2)
	}

	go watchTemperatureDifference(client, device, device2)

	// Handle SIGINT and SIGTERM.
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Println(<-ch)

	client.Disconnect(250)
}
