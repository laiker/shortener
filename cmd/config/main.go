package config

import "flag"

var FlagRunAddr string
var FlagOutputURL string

func ParseFlags() {
	flag.StringVar(&FlagRunAddr, "a", "localhost:8080", "Initial webserver URL")
	flag.StringVar(&FlagOutputURL, "b", "https://localhost:8080", "Output short url host")
	flag.Parse()
}
