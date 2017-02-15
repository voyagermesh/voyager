package framework

import "flag"

func Initialize() {
	SetTestFlags()
	flag.Parse()
}

var TestRealToken bool

func SetTestFlags() {
	flag.BoolVar(&TestRealToken, "real-token", false, "")
}
