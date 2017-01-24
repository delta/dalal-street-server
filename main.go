package main

import (
	"github.com/thakkarparth007/dalal-street-server/utils"
)

func main() {
	utils.InitConfiguration("config.json")
	utils.InitLogger()

	// start http server
	// do cool stuff
}
