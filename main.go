package main

import (
	"direbot/vercel"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var (
	GuildID     = flag.String("guild", "", "Test guild id")
	BotToken    = flag.String("bot-token", "", "Token for the bot")
	VercelToken = flag.String("vercel-token", "", "Token for vercel")
)

var (
	s  *discordgo.Session
	vc *vercel.VercelClient
)

func init() {
	flag.Parse()
}

func init() {
	var err error
	s, err = discordgo.New("Bot " + *BotToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}
}

func init() {
	vc = &vercel.VercelClient{
		Token: *VercelToken,
	}
}

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "maildequate",
			Description: "Get a cool email addresss",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "forward-to",
					Description: "the email this should forward to",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "address",
					Description: "the address you wants (defaults to your username)",
				},
			},
		},
	}
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"maildequate": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			forward := i.ApplicationCommandData().Options[0].StringValue()

			var address string
			if len(i.ApplicationCommandData().Options) >= 2 {
				address = i.ApplicationCommandData().Options[1].StringValue()
			} else if i.Member != nil {
				address = i.Member.User.Username
			} else if i.User != nil {
				address = i.User.Username
			} else {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Couldn't find your username for some reason, try again? :attempt:",
					},
				})
				return
			}

			log.Printf("recieved request: forward=%s, address=%s\n", forward, address)

			// Check for an existing name. -md
			res, err := vc.GetDomainRecords()
			if err != nil || res == nil {
				log.Printf("failed to list domains %s", err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Something went wrong :linkno:",
					},
				})
				return
			}
			for _, record := range res.Records {
				if strings.HasPrefix(record.Value, "forward-email=") &&
					strings.Contains(record.Value, "="+address+":") {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Unfortunately this name is taken :attempt:",
						},
					})
					return
				}
			}

			// Name hasn't been taken, create the record. -md
			err = vc.CreateDomainTXTRecord(fmt.Sprintf("forward-email=%s:%s", address, forward))
			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Failed to create an email forward :linkno:",
					},
				})
			} else {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Email forward has been created :linkyes:",
					},
				})
			}
		},
	}
)

func init() {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}

func main() {
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Println("Bot is up!")
	})
	err := s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}

	for _, v := range commands {
		_, err := s.ApplicationCommandCreate(s.State.User.ID, *GuildID, v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		} else {
			log.Printf("Registered command '%v'", v.Name)
		}
	}

	defer s.Close()

	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	<-stop
	log.Println("Gracefully shutdowning")
}
