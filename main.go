package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/Moonlington/harmonia"
	"github.com/bwmarrin/discordgo"
	"gopkg.in/yaml.v3"
)

// Bot parameters
var (
	RemoveCommands = flag.Bool("rmcmd", true, "Remove all commands after shutdowning or not")
)

func Contains[T comparable](sl []T, name T) bool {
	for _, v := range sl {
		if v == name {
			return true
		}
	}
	return false
}

var h *harmonia.Harmonia
var config *Config

type Config struct {
	Token       string   `yaml:"token"`
	StarChannel string   `yaml:"starChannel"`
	OwnerIDs    []string `yaml:"ownerIDs"`
	ModRoleIDs  []string `yaml:"modRoleIDs"`
}

func parseConfig() error {
	data, err := os.ReadFile("config.yml")
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}
	return nil
}

func init() {
	flag.Parse()
	var err error

	err = parseConfig()

	h, err = harmonia.New(config.Token)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}
}

type Repeater struct {
	Content string
	UserIDs []string
}

type Check struct {
	Vibe      int
	Swag      int
	timestamp time.Time
}

func newCheck() *Check {
	return &Check{
		Vibe:      rand.Intn(101),
		Swag:      rand.Intn(101),
		timestamp: time.Now(),
	}
}

var checkMap map[string]*Check = map[string]*Check{}

func main() {
	h.AddSlashCommandInGuild("updatestars", "Update Stars from the GitHub repo.", "604286100181221395", func(h *harmonia.Harmonia, i *harmonia.Invocation) {
		if !Contains(config.OwnerIDs, i.Author.ID) {
			h.EphemeralRespond(i, "You are not allowed to use this command.")
		}
	})

	checkCmd, _ := h.AddSlashCommand("check", "Checks a user's vibe and swag.", func(h *harmonia.Harmonia, i *harmonia.Invocation) {
		var check *Check
		var user *harmonia.Author
		if i.GetOption("user") != nil {
			member, err := h.GuildMember(i.GuildID, i.GetOption("user").UserValue(h.Session).ID)
			if err != nil {
				h.EphemeralRespond(i, fmt.Sprintf("There was an error getting the user: %s", err))
				return
			}

			user, err = h.AuthorFromMember(member)
			if err != nil {
				h.EphemeralRespond(i, fmt.Sprintf("There was an error getting the user: %s", err))
				return
			}
		} else {
			user = i.Author
		}

		check, ok := checkMap[user.ID]
		if !ok {
			check = newCheck()
			checkMap[user.ID] = check
		}

		if time.Now().Sub(check.timestamp) > time.Hour {
			check = newCheck()
			checkMap[user.ID] = check
		}

		userName := user.Nick
		if userName == "" {
			userName = user.Username
		}

		h.Respond(i, fmt.Sprintf("%s's vibe: %d%%\n%s's swag: %d%%\nTime until next check: %s", userName, check.Vibe, userName, check.Swag, time.Now().Sub(check.timestamp.Add(time.Hour)).Abs().Truncate(time.Second).String()))
	})

	checkCmd.AddOption("user", "The user to check", false, discordgo.ApplicationCommandOptionUser)

	var repeaterMap map[string]*Repeater = map[string]*Repeater{}
	h.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		repeater, ok := repeaterMap[m.ChannelID]
		if !ok || repeater.Content != m.Content {
			repeaterMap[m.ChannelID] = &Repeater{m.Content, []string{m.Author.ID}}
			return
		}

		if Contains(repeater.UserIDs, m.Author.ID) || m.Author.Bot {
			return
		}

		repeaterMap[m.ChannelID].UserIDs = append(repeaterMap[m.ChannelID].UserIDs, m.Author.ID)

		if len(repeaterMap[m.ChannelID].UserIDs)%4 == 0 {
			h.ChannelMessageSend(m.ChannelID, m.Content)
		}
	})

	err := AddStarboardHandlers(h)
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}

	err = h.Run()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}

	defer h.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop

	if *RemoveCommands {
		err := h.RemoveAllCommands()
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Println("Gracefully shutting down.")
}
