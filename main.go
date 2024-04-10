package main

import (
	"flag"
	"fmt"
	"log/slog"
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
	if err != nil {
		slog.Error("Error parsing config", err)
		os.Exit(-1)
	}

	h, err = harmonia.New(config.Token)
	if err != nil {
		slog.Error("Invalid bot parameters", err)
		os.Exit(-1)
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
	slog.Debug("adding updatestars command")
	h.AddCommand(harmonia.NewSlashCommand("updatestars").
		WithDescription("Update Stars from the GitHub repo.").
		WithGuildID("604286100181221395").
		WithCommand(func(h *harmonia.Harmonia, i *harmonia.Invocation) {
			if !Contains(config.OwnerIDs, i.Author.ID) {
				h.EphemeralRespond(i, "You are not allowed to use this command.")
				return
			}

			h.Respond(i, "Not supported yet :)")
		}))

	slog.Debug("adding check command")
	h.AddCommand(harmonia.NewSlashCommand("check").
		WithDescription("Checks a user's vibe and swag.").
		WithCommand(func(h *harmonia.Harmonia, i *harmonia.Invocation) {
			var check *Check
			var user *harmonia.Author
			if i.GetOption("user") != nil {
				member, err := h.GuildMember(i.GuildID, i.GetOption("user").UserValue(h.Session).ID)
				if err != nil {
					h.EphemeralRespond(i, fmt.Sprintf("There was an error getting the user: %s", err))
					return
				}

				user, err = harmonia.AuthorFromMember(h, member)
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

			if time.Since(check.timestamp) > time.Hour {
				check = newCheck()
				checkMap[user.ID] = check
			}

			userName := user.Nick
			if userName == "" {
				userName = user.Username
			}

			h.Respond(i, fmt.Sprintf("%s's vibe: %d%%\n%s's swag: %d%%\nTime until next check: %s", userName, check.Vibe, userName, check.Swag, time.Since(check.timestamp.Add(time.Hour)).Abs().Truncate(time.Second).String()))
		}).
		WithOptions(
			harmonia.NewOption("user", discordgo.ApplicationCommandOptionUser).
				WithDescription("The user to check"),
		))

	slog.Debug("adding check user command")
	h.AddCommand(harmonia.NewUserCommand("Check vibe and swag").
		WithCommand(func(h *harmonia.Harmonia, i *harmonia.Invocation) {
			var check *Check

			user, err := i.TargetAuthor(h)
			if err != nil {
				h.EphemeralRespond(i, fmt.Sprintf("There was an error getting the user: %s", err))
				return
			}

			check, ok := checkMap[user.ID]
			if !ok {
				check = newCheck()
				checkMap[user.ID] = check
			}

			if time.Since(check.timestamp) > time.Hour {
				check = newCheck()
				checkMap[user.ID] = check
			}

			userName := user.Nick
			if userName == "" {
				userName = user.Username
			}

			h.Respond(i, fmt.Sprintf("%s's vibe: %d%%\n%s's swag: %d%%\nTime until next check: %s", userName, check.Vibe, userName, check.Swag, time.Since(check.timestamp.Add(time.Hour)).Abs().Truncate(time.Second).String()))
		}))

	slog.Debug("adding repeater handler")
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

	slog.Debug("adding starboard handlers")
	err := AddStarboardHandlers(h)
	if err != nil {
		slog.Error("Cannot add Starboard handlers", err)
	}

	slog.Debug("adding motd handlers")
	err = AddMOTDHandlers(h)
	if err != nil {
		slog.Error("Cannot add MOTD handlers", err)
	}

	slog.Debug("adding color handlers")
	err = AddColorHandlers(h)
	if err != nil {
		slog.Error("Cannot add Color handlers", err)
	}

	slog.Debug("running the session")
	err = h.Run()
	if err != nil {
		slog.Error("Cannot open the session", err)
		os.Exit(-1)
	}

	defer h.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	slog.Info("Press Ctrl+C to exit")
	<-stop

	if *RemoveCommands {
		slog.Info("Removing all commands")
		err := h.RemoveAllCommands()
		if err != nil {
			slog.Error("Error removing all commands", err)
			os.Exit(-1)
		}
	}

	slog.Info("Gracefully shutting down.")
}
