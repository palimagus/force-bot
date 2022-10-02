package db

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Database struct {
	Players map[string]*Player `json:"players"`
	Duels   map[string]*Duel   `json:"duels"`
}

type Player struct {
	DiscordUser   *discordgo.User `json:"discord_user"`
	Dark          bool            `json:"dark"`
	DuelsCount    uint            `json:"duels_count"`
	Wins          uint            `json:"duels_wins"`
	MessagesCount uint            `json:"message_count"`
	XP            uint            `json:"xp"`
	Level         uint            `json:"level"`
	Energy        uint            `json:"energy"`
	Power         uint            `json:"power"`
	Force         int             `json:"force"`
}

type Duel struct {
	ID        string  `json:"id"`
	Triggerer *Player `json:"triggerer"`
	Ennemy    *Player `json:"ennemy"`
	Winner    bool    `json:"winner"`
	Ended     bool    `json:"ended"`
	TChoice   string  `json:"tchoice"`
	EChoice   string  `json:"echoice"`
}

const charset = "abcdefghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var (
	seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

	Players map[string]*Player = make(map[string]*Player)
	Duels   map[string]*Duel   = make(map[string]*Duel)
)

func Save() error {
	data, err := json.MarshalIndent(&Database{
		Players: Players,
		Duels:   Duels,
	}, "", " ")
	if err != nil {
		fmt.Println("❌ Error parsing data to bytes.")
		return err
	}

	err = os.WriteFile("db.json", data, 0644)
	if err != nil {
		fmt.Println("❌ Error writing data to file.")
		return err
	}

	return nil
}

func (p *Player) NewDuel(target *Player) *Duel {
	duel := &Duel{
		ID:        RandomString(8),
		Triggerer: p,
		Ennemy:    target,
		Winner:    false,
		TChoice:   "",
		EChoice:   "",
		Ended:     false,
	}
	Duels[duel.ID] = duel
	return duel
}

func GetDuel(id string) *Duel {
	return Duels[id]
}

func (d *Duel) Resolve() int {
	var result int = 0
	d.Ennemy.DuelsCount++
	d.Triggerer.DuelsCount++
	if d.EChoice == d.TChoice {
		result = 1
	}

	switch d.TChoice {
	case "rock":
		if d.EChoice == "scis" {
			result = 3
		}
		result = 2
	case "leaf":
		if d.EChoice == "rock" {
			result = 3
		}
		result = 2
	case "scis":
		if d.EChoice == "leaf" {
			result = 3
		}
		result = 2
	default:
		result = 0
	}

	switch result {
	case 1:
		// Draw
		d.Ennemy.AddXP(50)
		d.Triggerer.AddXP(50)
	case 2:
		// Ennemy wins the duel
		d.Ennemy.AddXP(100)
		d.Triggerer.AddXP(-(int(d.Triggerer.XP) * 10) / 100)
	case 3:
		d.Ennemy.AddXP(-(int(d.Ennemy.XP) * 10) / 100)
		d.Triggerer.AddXP(100)
	}

	return result
}

func GetPlayer(u *discordgo.User) *Player {
	if Players[u.ID] == nil {
		Players[u.ID] = &Player{
			DiscordUser:   u,
			DuelsCount:    0,
			Wins:          0,
			MessagesCount: 0,
			XP:            0,
			Level:         1,
			Force:         0,
			Power:         100,
			Energy:        100,
			Dark:          false,
		}
	}

	return Players[u.ID]
}

// Leveling

func XPNeededForLevel(level uint) uint {
	if level == 0 {
		return 0
	}
	return level + XPNeededForLevel(level-1)
}

func (p *Player) AddXP(value int) {
	if value > 0 {
		p.XP += uint(value)
		if p.XP >= XPNeededForLevel(p.Level+1)*1000 {
			p.Levelup()
		}
	}
	if value < 0 {
		p.XP -= uint(value)
	}
}

func (p *Player) Levelup() {
	p.Level++
	fmt.Printf("%s gained a level!\n", p.DiscordUser.Username)
}

// Random utils

func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func RandomString(length int) string {
	return StringWithCharset(length, charset)
}
