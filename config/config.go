package config

import (
	"encoding/json"
	"fmt"
	"forcebot/db"
	"os"
)

type Configuration struct {
	Token   string `json:"token"`
	GuildID string `json:"guild_id"`
}

var (
	Token   string
	GuildID string

	config   *Configuration
	database *db.Database
)

func LoadDB() error {
	fmt.Println("🤖 Loading database...")
	file, err := os.ReadFile("db.json")

	// Handle error
	if err != nil {
		fmt.Println("❌ Error loading database:", err)
		return err
	}

	// Parse JSON
	err = json.Unmarshal(file, &database)

	// Handle error
	if err != nil {
		fmt.Println("❌ Error parsing database:", err)
		return err
	}

	if database.Players != nil {
		db.Players = database.Players
	}

	if database.Duels != nil {
		db.Duels = database.Duels
	}

	fmt.Println("🤖 Database loaded.")

	return nil
}

func LoadConfig() error {
	fmt.Println("🤖 Loading config...")
	file, err := os.ReadFile("config.json")

	// Handle error
	if err != nil {
		fmt.Println("❌ Error loading config:", err)
		return err
	}

	// Parse JSON
	err = json.Unmarshal(file, &config)

	// Handle error
	if err != nil {
		fmt.Println("❌ Error parsing config:", err)
		return err
	}

	// Set global variables
	Token = config.Token
	GuildID = config.GuildID

	fmt.Println("🤖 Config loaded.")
	return nil
}
