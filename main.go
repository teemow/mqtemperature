package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"github.com/pborman/uuid"
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
		Config   string
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
)

func init() {
	mainCmd.PersistentFlags().BoolVarP(&globalFlags.debug, "debug", "d", false, "Print debug output")
	mainCmd.PersistentFlags().BoolVarP(&globalFlags.verbose, "verbose", "v", false, "Print verbose output")
	mainCmd.PersistentFlags().StringVar(&mainFlags.Host, "host", "localhost", "MQTT host")
	mainCmd.PersistentFlags().IntVar(&mainFlags.Port, "port", 1883, "MQTT port")
	mainCmd.PersistentFlags().StringVar(&mainFlags.Config, "config", "/etc/mqtemperature/config.yml", "Configuration file")
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

func watchTemperature(client *mqtt.Client) {
	conf, err := loadConfig(mainFlags.Config)
	if err != nil {
		log.Fatalf("Couldn't read config: %s", err)
	}

	for {
		temperatures, err := readTemperatures(conf.Devices)
		if err != nil {
			log.Printf("Failed to read temperatures: %v\n", err)
		}

		for d, topic := range conf.Devices {
			err := sendMsg(client, topic, temperatures[d])

			if err != nil {
				log.Printf("Failed to send message: %v\n", err)
			}
		}

		for _, diff := range conf.Diffs {
			err := sendMsg(client, diff.Topic, temperatures[diff.Device1]-temperatures[diff.Device2])

			if err != nil {
				log.Printf("Failed to send message: %v\n", err)
			}
		}

		time.Sleep(time.Duration(mainFlags.Interval) * time.Second)
	}
}

func sendMsg(client *mqtt.Client, topic string, temperature int64) error {
	topic = fmt.Sprintf("mqtemperature/%s", topic)
	payload := fmt.Sprintf("%d", temperature)

	token := client.Publish(topic, 1, false, payload)

	token.WaitTimeout(500)
	return token.Error()
}

func readTemperatures(devices map[string]string) (map[string]int64, error) {
	temperatures := map[string]int64{}

	for d := range devices {
		device, err := onewire.NewDS18S20(d)
		if err != nil {
			log.Printf("Device %s not found\n", d)
		}

		temperatures[d], err = device.Read()
		if err != nil {
			log.Printf("Couldn't read value of %s\n", d)
		}
	}
	return temperatures, nil
}

func mainRun(cmd *cobra.Command, args []string) {
	// mqtt
	opts := mqtt.NewClientOptions().AddBroker(fmt.Sprintf("tcp://%s:%d", mainFlags.Host, mainFlags.Port))
	opts.SetClientID(fmt.Sprintf("mqtemperature-%s", uuid.New()))

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}

	go watchTemperature(client)

	// Handle SIGINT and SIGTERM.
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Println(<-ch)

	client.Disconnect(250)
}

func main() {
	mainCmd.AddCommand(versionCmd)

	mainCmd.Execute()
}
