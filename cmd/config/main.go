package config

import "flag"

var FlagRunAddr string
var FlagOutputUrl string

func ParseFlags() {
	flag.StringVar(&FlagRunAddr, "a", "localhost:8080", "Initial webserver URL")
	flag.StringVar(&FlagOutputUrl, "b", "https://localhost:8080", "Output short url host")
	flag.Parse()
}
