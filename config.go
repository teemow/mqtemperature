package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

func loadConfig(filePath string) (configuration, error) {
	conf := configuration{}

	f, err := os.Open(filePath)
	if err != nil {
		return conf, err
	}
	defer f.Close()

	confBytes, err := ioutil.ReadAll(f)
	if err != nil {
		return conf, err
	}

	err = yaml.Unmarshal(confBytes, &conf)

	return conf, err
}

type Diff struct {
	Topic   string `yaml:"topic"`
	Device1 string `yaml:"device1"`
	Device2 string `yaml:"device2"`
}

type configuration struct {
	Devices map[string]string `yaml:"devices"`
	Diffs   []Diff            `yaml:"diffs"`
}
