package main

import (
	"log"

	"github.com/nicholas-fedor/htmx-hardware-monitor/pkg/backend"
	"github.com/nicholas-fedor/htmx-hardware-monitor/pkg/frontend"
	"github.com/spf13/viper"
)

func main() {
	// Setup Viper for configuration
	viper.SetConfigName("config") // name of config file (without extension)
	viper.AddConfigPath("config") // path to look for the config file in
	viper.SetConfigType("yaml")   // REQUIRED if the config file does not have the extension in the name

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		log.Fatalf("Fatal error config file: %v \n", err)
	}

	// Start backend server
	go backend.Execute()

	// Start frontend server
	go frontend.Execute()

	// Keep the main goroutine running
	select {}
}