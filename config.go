package main

import (
	"os"

	"gopkg.in/yaml.v2"
)

// Config ...
type Config struct {
	Cookie   string `yaml:"cookie"`
	ItemID   uint   `yaml:"itemID"`
	Price    uint   `yaml:"price"`
	SellerID uint   `yaml:"sellerID"`
	UAID     uint   `yaml:"uaid"`
}

// ParseConfig Opens and decodes a config file into a 'Config' object and returns it
func ParseConfig(file string) (Config, error) {
	f, err := os.Open(file)
	if err != nil {
		return Config{}, err
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
}
