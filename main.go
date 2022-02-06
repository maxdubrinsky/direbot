package main

import (
	"direbot/vercel"
	"fmt"
	"log"
	"net/mail"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/bwmarrin/discordgo"
)

var (
	s  *discordgo.Session
	vc *vercel.VercelClient
)

var cli struct {
	BotToken    string `required:"" env:"DIREBOT_BOT_TOKEN"`
	VercelToken string `required:"" env:"DIREBOT_VERCEL_TOKEN"`
	Domain      string `required:"" env:"DIREBOT_DOMAIN"`
}

func init() {
	kong.Parse(&cli)
}

func init() {
	var err error
	s, err = discordgo.New("Bot " + cli.BotToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}
}

func init() {
	vc = &vercel.VercelClient{
		Token: cli.VercelToken,
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

			if _, err := mail.ParseAddress(forward); err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Invalid forwarding address, Mx. Tables :attempt:",
					},
				})
				return
			}

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

			if _, err := mail.ParseAddress(fmt.Sprintf("%s@%s", address, cli.Domain)); err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "The provided address will be invalid :attempt:",
					},
				})
				return
			}

			log.Printf("recieved request: forward=%s, address=%s\n", forward, address)

			// Check for an existing name. -md
			res, err := vc.GetDomainRecords(cli.Domain)
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
			record := fmt.Sprintf("forward-email=%s:%s", address, forward)
			err = vc.CreateDomainTXTRecord(cli.Domain, record)
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
		_, err := s.ApplicationCommandCreate(s.State.User.ID, "", v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		} else {
			log.Printf("Registered command '%v'", v.Name)
		}
	}

	defer s.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("Gracefully shutdowning")
}
