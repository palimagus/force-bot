package app

import (
	"fmt"
	"forcebot/config"
	"forcebot/db"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

type CustomChannel struct {
	DiscordChannel *discordgo.Channel
	NumberOfUsers  int
}

var (
	ID                string
	Session           *discordgo.Session
	CustomChannels    []*discordgo.Channel
	AllCustomChannels []CustomChannel

	Commands = []*discordgo.ApplicationCommand{
		{
			Name:        "scan",
			Description: "Effectue un scan pour obtenir des informations sur un membre.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "cible",
					Description: "Préciez le membre à scanner, laisser vide pour se scanner soi-même.",
					Type:        discordgo.ApplicationCommandOptionUser,
					Required:    false,
				},
			},
		},
		{
			Name:        "duel",
			Description: "Défie un autre membre en duel.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "adversaire",
					Description: "Le pseudo du membre que vous souhaitez défier.",
					Type:        discordgo.ApplicationCommandOptionUser,
					Required:    true,
				},
			},
		},
		{
			Name:        "xp",
			Description: "Donne de l'expérience à un membre.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "valeur",
					Description: "Nombre de points d'expérience à donner.",
					Type:        discordgo.ApplicationCommandOptionInteger,
					Required:    true,
				},
				{
					Name:        "cible",
					Description: "Membre à qui envoyer l'expérience. Laissez vide pour se sélectionner soi-même.",
					Type:        discordgo.ApplicationCommandOptionUser,
					Required:    false,
				},
			},
		},
	}
	CommandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"scan": func(s *discordgo.Session, i *discordgo.InteractionCreate) {

			options := i.ApplicationCommandData().Options
			optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
			for _, opt := range options {
				optionMap[opt.Name] = opt
			}

			targetUser := i.Member.User

			msgformat := "Scan en cours...\n"
			if opt, ok := optionMap["cible"]; ok {
				msgformat += "Vous scannez un autre membre.\n"

				targetUser = opt.UserValue(nil)
			}

			targetPlayer := db.GetPlayer(targetUser)
			embd := discordgo.MessageEmbed{
				Title:       "Scan",
				Description: fmt.Sprintf("Résultat du scan sur <@%s>", targetUser.ID),
				Color:       0x00ff00,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Niveau",
						Value:  fmt.Sprint(targetPlayer.Level),
						Inline: true,
					},
					{
						Name:   "Experience",
						Value:  fmt.Sprintf("%d/%d", targetPlayer.XP, db.XPNeededForLevel(targetPlayer.Level+1)*1000),
						Inline: true,
					},
					{
						Name:   "Force",
						Value:  fmt.Sprint(targetPlayer.Force),
						Inline: true,
					},
					{
						Name:   "Messages",
						Value:  fmt.Sprint(targetPlayer.MessagesCount),
						Inline: true,
					},
					{
						Name:   "Duels remportés",
						Value:  fmt.Sprintf("%d/%d", targetPlayer.Wins, targetPlayer.DuelsCount),
						Inline: true,
					},
				},
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:   discordgo.MessageFlagsEphemeral,
					Embeds:  []*discordgo.MessageEmbed{&embd},
					Content: fmt.Sprint(msgformat),
				},
			})
		},
		"duel": func(s *discordgo.Session, i *discordgo.InteractionCreate) {

			options := i.ApplicationCommandData().Options
			optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
			for _, opt := range options {
				optionMap[opt.Name] = opt
			}

			p1 := db.GetPlayer(i.Member.User)
			p2 := db.GetPlayer(optionMap["adversaire"].UserValue(s))

			duel := p1.NewDuel(p2)

			embd := discordgo.MessageEmbed{
				Title:       "Duel",
				Description: fmt.Sprintf("Un duel opposant <@%s> et <@%s> va commencer!", p1.DiscordUser.ID, p2.DiscordUser.ID),
				Color:       0x00ff00,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Code duel",
						Value:  duel.ID,
						Inline: true,
					},
					{
						Name:   "Attaquant",
						Value:  p1.DiscordUser.Username,
						Inline: true,
					},
					{
						Name:   "Défenseur",
						Value:  p2.DiscordUser.Username,
						Inline: true,
					},
				},
			}

			e := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					// Flags:  discordgo.MessageFlagsEphemeral,
					Embeds: []*discordgo.MessageEmbed{&embd},
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								&discordgo.Button{
									Label:    "✊ Pierre",
									Style:    discordgo.SecondaryButton,
									CustomID: fmt.Sprintf("duel|%s|rock", duel.ID),
								},
								&discordgo.Button{
									Label:    "✋ Feuille",
									Style:    discordgo.SecondaryButton,
									CustomID: fmt.Sprintf("duel|%s|leaf", duel.ID),
								},
								&discordgo.Button{
									Label:    "✌ Ciseaux",
									Style:    discordgo.SecondaryButton,
									CustomID: fmt.Sprintf("duel|%s|scis", duel.ID),
								},
							},
						},
					},
				},
			})
			if e != nil {
				fmt.Println(e)
				Notify(s, i)
			}
		},
		"xp": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Interfacing options
			options := i.ApplicationCommandData().Options
			optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
			for _, opt := range options {
				optionMap[opt.Name] = opt
			}

			// Default target
			targetUser := i.Member.User

			// Message
			msgformat := "Attribution d'expérience...\n"
			if opt, ok := optionMap["cible"]; ok {
				msgformat += "Envoi d'expérience à un autre membre.\n"

				targetUser = opt.UserValue(nil)
			}

			targetPlayer := db.GetPlayer(targetUser)
			if opt, ok := optionMap["valeur"]; ok {
				targetPlayer.AddXP(int(opt.IntValue()))
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:   discordgo.MessageFlagsEphemeral,
					Content: fmt.Sprint(msgformat),
				},
			})
		},
	}
)

func IsCustomChannel(channelID string) (bool, int, CustomChannel) {
	for index, customChannel := range AllCustomChannels {
		return customChannel.DiscordChannel.ID == channelID, index, customChannel
	}
	return false, -1, CustomChannel{}
}

func CheckError(e error) {
	if e != nil {
		fmt.Println("Error with Discord API")
		panic(e)
	}
}

func Start() {
	// Create Discord session
	Session, e := discordgo.New("Bot " + config.Token)
	CheckError(e)
	user, e := Session.User("@me")
	CheckError(e)
	ID = user.ID

	// Session handlers
	Session.AddHandler(OnVoiceStateUpdate)
	Session.AddHandler(OnInteraction)
	Session.AddHandler(OnMessage)

	e = Session.Open()
	CheckError(e)

	// Register commands
	fmt.Println("🤖 Registering commands.")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(Commands))
	for i, v := range Commands {
		cmd, e := Session.ApplicationCommandCreate(Session.State.User.ID, config.GuildID, v)
		if e != nil {
			fmt.Println("❌Error creating command:", e)
			return
		}
		registeredCommands[i] = cmd
	}

	// Initialize custom channels
	AllCustomChannels = make([]CustomChannel, 0)

	// Clean close
	defer Session.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Await interrupt
	fmt.Println("🤖 System is now running. Press CTRL+C to exit.")
	<-stop

	// Save all to database
	fmt.Println("🤖 Saving data.")
	err := db.Save()
	if err != nil {
		fmt.Println(err)
	}

	// Then, unregister commands
	fmt.Println("🤖 Unregistering commands.")
	for _, v := range registeredCommands {
		Session.ApplicationCommandDelete(
			Session.State.User.ID,
			config.GuildID,
			v.ID,
		)
	}

	// And exit program
	fmt.Println("🤖 Shutting down system.")
	os.Exit(0)
}

func OnInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	fmt.Println("🎫 OnInteraction")

	// Vérifier si l'interaction est une commande
	if i.Type == discordgo.InteractionApplicationCommand {
		if handler, ok := CommandHandlers[i.ApplicationCommandData().Name]; ok {
			handler(s, i)
		}
	}

	// Vérifier si l'interaction est un bouton
	if i.Type == discordgo.InteractionMessageComponent {
		p := db.GetPlayer(i.Member.User)

		// actionData := make(map[string]interface{})
		data := strings.Split(i.MessageComponentData().CustomID, "|")

		if data[0] == "duel" {
			duel := db.GetDuel(data[1])
			if duel != nil {
				if duel.Ended {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Flags:   discordgo.MessageFlagsEphemeral,
							Content: "Ce duel est terminé.",
						},
					})
				}
				if p.DiscordUser.ID == duel.Triggerer.DiscordUser.ID {
					duel.TChoice = data[2]
				}
				if p.DiscordUser.ID == duel.Ennemy.DiscordUser.ID {
					duel.EChoice = data[2]
				}

				if duel.TChoice != "" && duel.EChoice != "" {
					res := duel.Resolve()
					msg := "Résultat du duel:\n"
					switch res {
					case 1:
						msg += "Match nul."
					case 2:
						duel.Ennemy.Wins++
						msg += fmt.Sprintf("<@%s> a gagné!", duel.Ennemy.DiscordUser.ID)
					case 3:
						duel.Triggerer.Wins++
						msg += fmt.Sprintf("<@%s> a gagné!", duel.Triggerer.DiscordUser.ID)
					default:
						msg += "Erreur."
					}

					duel.Ended = true

					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							// Flags:   discordgo.MessageFlagsEphemeral,
							Content: msg,
						},
					})
				}
			}

			Notify(s, i)
			return
		}
	}

	// Dans tous les cas, notifier que l'interaction a été traitée
	Notify(s, i)
}

func OnMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == ID || m.Author.Bot {
		return
	}
	fmt.Println("🎫 OnMessage")

	p := db.GetPlayer(m.Author)
	p.MessagesCount++
	p.AddXP(2)

	// If it's 100th message, give a reward
	if p.MessagesCount == 100 {
		msg := "🎉 Bravo <@%s>, tu as atteint le 100ème message !"
		msg = fmt.Sprintf(msg, m.Author.ID)
		// Send a message to the player
		s.ChannelMessageSend(m.ChannelID, msg)
	}
}

func OnVoiceStateUpdate(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	// Get player and verify it's not a bot
	member, err := s.GuildMember(config.GuildID, v.UserID)
	if err != nil {
		fmt.Println("❌ Error getting user:", err)
		return
	}

	if member.User.Bot {
		return
	}

	// Get player action informations
	from := ""
	if v.BeforeUpdate != nil {
		from = v.BeforeUpdate.ChannelID
	}
	to := v.ChannelID

	// Player leaves a channel
	if from != "" && (from != to) {
		fmt.Println("🎫 OnChannelLeaved")
		custom, i, c := IsCustomChannel(from)

		if custom {
			c.NumberOfUsers--
			if c.NumberOfUsers <= 0 {
				writeDebugToChannel(s, "🗑️ Channel supprimé: "+c.DiscordChannel.Name)
				AllCustomChannels = append(AllCustomChannels[:i], AllCustomChannels[i+1:]...)
				s.ChannelDelete(from)
			}
		}
		return
	}

	// Player joins a channel
	if to != "" && (from != to) {
		fmt.Println("🎫 OnChannelJoined")
		writeDebugToChannel(s, fmt.Sprintf("🎫 OnChannelJoined: %s", to))
		writeDebugToChannel(s, fmt.Sprintf("📜 Liste des channels: %v", AllCustomChannels))
		custom, _, c := IsCustomChannel(to)

		if custom {
			c.NumberOfUsers++
			writeDebugToChannel(s, fmt.Sprintf("👥 %d utilisateurs dans le salon %s", c.NumberOfUsers, c.DiscordChannel.Name))

		} else {
			if to == "1026145931298619543" {
				// Create a new voice channel
				nc, err := s.GuildChannelCreate(
					config.GuildID,
					fmt.Sprintf("🔰 Salon de %s", member.User.Username),
					discordgo.ChannelTypeGuildVoice,
				)
				if err != nil {
					fmt.Println("❌ Error creating a custom channel")
					return
				}

				AllCustomChannels = append(AllCustomChannels, CustomChannel{
					// Init numOfUsers to 0 because we move the member next
					NumberOfUsers:  0,
					DiscordChannel: nc,
				})
				writeDebugToChannel(s, fmt.Sprintf("📜 Liste des channels: %v", AllCustomChannels))

				// Move member to new channel
				e := s.GuildMemberMove(config.GuildID, v.UserID, &nc.ID)
				if e != nil {
					fmt.Println("❌ Error moving member to it's custom channel")
				}
			}
		}
		return
	}
}

func Notify(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
}

func writeDebugToChannel(s *discordgo.Session, msg string) {
	// Create a debug embed message
	embed := &discordgo.MessageEmbed{
		Title:       "Debug",
		Description: msg,
		Color:       0xff1e00,
	}

	// Send the message
	_, e := s.ChannelMessageSendEmbed("803829358451359785", embed)
	if e != nil {
		fmt.Println("❌ Error sending debug message:", e)
	}
}
