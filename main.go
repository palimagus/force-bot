package main

import (
	"fmt"
	"forcebot/app"
	"forcebot/config"
)

// $env:GOOS='linux';  $env:GOARCH='amd64'; go build
func main() {
	fmt.Print("\033[H\033[2J")
	fmt.Println("🤖 Booting system.")

	// Load config
	err := config.LoadConfig()
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	// Load database
	err = config.LoadDB()
	if err != nil {
		fmt.Println("Error loading database", err)
		return
	}

	app.Start()
}
