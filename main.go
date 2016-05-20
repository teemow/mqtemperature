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
	globalFlags struct {
		debug   bool
		verbose bool
	}

	mainFlags struct {
		Host     string
		Port     int
		Topic    string
		Device   string
		Interval int
	}

	mainCmd = &cobra.Command{
		Use:   "mqtemperature",
		Short: "Publish temperature via MQTT",
		Long:  "Send MQTT messages to a broker. Especially made for One-Wire (DS18S20) temperature sensors on a Raspberry Pi",
		Run:   mainRun,
	}

	projectVersion string
	projectBuild   string

	isMQTTconnected = false
)

func init() {
	mainCmd.PersistentFlags().BoolVarP(&globalFlags.debug, "debug", "d", false, "Print debug output")
	mainCmd.PersistentFlags().BoolVarP(&globalFlags.verbose, "verbose", "v", false, "Print verbose output")
	mainCmd.PersistentFlags().StringVar(&mainFlags.Host, "host", "localhost", "MQTT host")
	mainCmd.PersistentFlags().IntVar(&mainFlags.Port, "port", 1883, "MQTT port")
	mainCmd.PersistentFlags().StringVar(&mainFlags.Topic, "topic", "", "MQTT topic")
	mainCmd.PersistentFlags().StringVar(&mainFlags.Device, "device", "", "DS18S20 device")
	mainCmd.PersistentFlags().IntVar(&mainFlags.Interval, "interval", 60, "Interval in seconds")
}

func assert(err error) {
	if err != nil {
		if globalFlags.debug {
			fmt.Printf("%#v\n", err)
			os.Exit(1)
		} else {
			log.Fatal(err)
		}
	}
}

func watchTemperature(client *mqtt.Client, device *onewire.DS18S20) {
	for {
		if !isMQTTconnected {
			reconnect(client)
		}

		value, err := device.Read()
		if err == nil {
			topic := fmt.Sprintf("mqtemperature/%s", mainFlags.Topic)
			payload := fmt.Sprintf("%d", value)

			if token := client.Publish(topic, 1, false, payload); token.Wait() && token.Error() != nil {
				fmt.Printf("Failed to send message: %v\n", token.Error())
			}
		} else {
			fmt.Println("Could not read temperature")
		}
		time.Sleep(time.Duration(mainFlags.Interval) * time.Second)
	}
}

func onMQTTConnectionLost(client *mqtt.Client, err error) {
	if isMQTTconnected {
		log.Println("MQTT connection lost: ", err)
		isMQTTconnected = false
	}
}

func reconnect(client *mqtt.Client) {
	if !isMQTTconnected {
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			log.Println(token.Error())
		} else {
			log.Println("MQTT reconnected")
			isMQTTconnected = true
		}
	}
}

func mainRun(cmd *cobra.Command, args []string) {
	// mqtt
	opts := mqtt.NewClientOptions().AddBroker(fmt.Sprintf("tcp://%s:%d", mainFlags.Host, mainFlags.Port))
	opts.SetClientID(fmt.Sprintf("mqtemperature-%s", mainFlags.Topic))
	opts.SetConnectionLostHandler(onMQTTConnectionLost)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	} else {
		isMQTTconnected = true
	}

	device, err := onewire.NewDS18S20(mainFlags.Device)
	if err != nil {
		log.Fatalf("Device %s not found", mainFlags.Device)
	}

	go watchTemperature(client, device)

	// Handle SIGINT and SIGTERM.
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Println(<-ch)

	client.Disconnect(250)
}

func main() {
	mainCmd.AddCommand(versionCmd)
	mainCmd.AddCommand(diffCmd)

	mainCmd.Execute()
}
