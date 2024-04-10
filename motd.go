package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Moonlington/harmonia"
	"github.com/bwmarrin/discordgo"
)

type MOTDData map[string]GuildMOTDData
type GuildMOTDData map[string]string

var MOTDDatabase MOTDData

func ReadFromMOTDJsonFile() error {
	data, err := os.ReadFile("MOTD_data.json")
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &MOTDDatabase); err != nil {
		return err
	}
	return nil
}

func SaveToMOTDJsonFile() error {
	file, err := os.Create("MOTD_data.json")
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(&MOTDDatabase)
	if err != nil {
		return err
	}

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func AddMOTDHandlers(h *harmonia.Harmonia) error {
	if err := ReadFromMOTDJsonFile(); err != nil {
		return err
	}

	h.AddHandler(func(s *discordgo.Session, r *discordgo.MessageCreate) {
		if r.Author.Bot {
			return
		}

		if r.GuildID == "" {
			return
		}

		GuildMOTD, ok := MOTDDatabase[r.GuildID]
		if !ok {
			GuildMOTD = GuildMOTDData{}
			MOTDDatabase[r.GuildID] = GuildMOTD
		}

		for k, v := range GuildMOTD {
			if strings.Contains(strings.ToLower(r.Content), k) {
				h.ChannelMessageSend(r.ChannelID, v)
				break
			}
		}
	})

	err := h.AddCommand(harmonia.NewGroupSlashCommand("motd").
		WithDescription("MOTD commands").
		WithDefaultPermissions(discordgo.PermissionManageMessages).
		WithDMPermission(false).
		WithSubCommands(
			harmonia.NewSlashCommand("set").
				WithDescription("Sets the response for a trigger").
				WithOptions(
					harmonia.NewOption("trigger", discordgo.ApplicationCommandOptionString).
						WithDescription("The trigger").
						IsRequired(),
					harmonia.NewOption("response", discordgo.ApplicationCommandOptionString).
						IsRequired().
						WithDescription("The fish"),
				).
				WithCommand(func(h *harmonia.Harmonia, i *harmonia.Invocation) {
					trigger := strings.ToLower(i.GetOption("trigger").StringValue())
					response := i.GetOption("response").StringValue()

					GuildMOTD, ok := MOTDDatabase[i.GuildID]
					if !ok {
						GuildMOTD = GuildMOTDData{}
						MOTDDatabase[i.GuildID] = GuildMOTD
					}

					GuildMOTD[trigger] = response
					h.EphemeralRespond(i, fmt.Sprintf("Trigger `%s` has been set to `%s`.", trigger, response))
					err := SaveToMOTDJsonFile()
					if err != nil {
						h.EphemeralRespond(i, fmt.Sprintf("Something went wrong with saving the JSON file:\n```%v```", err))
					}
				}),
			harmonia.NewSlashCommand("remove").
				WithDescription("Removes the response for a trigger").
				WithOptions(
					harmonia.NewOption("trigger", discordgo.ApplicationCommandOptionString).
						WithDescription("The trigger").
						IsRequired(),
				).
				WithCommand(func(h *harmonia.Harmonia, i *harmonia.Invocation) {
					trigger := strings.ToLower(i.GetOption("trigger").StringValue())

					GuildMOTD, ok := MOTDDatabase[i.GuildID]
					if !ok {
						GuildMOTD = GuildMOTDData{}
						MOTDDatabase[i.GuildID] = GuildMOTD
					}

					if _, ok = GuildMOTD[trigger]; !ok {
						h.EphemeralRespond(i, fmt.Sprintf("Trigger `%s` does not exist for this server and thus did not need to be cleared.", trigger))
						return
					}
					delete(GuildMOTD, trigger)
					h.EphemeralRespond(i, fmt.Sprintf("Trigger `%s` has been cleared.", trigger))
					err := SaveToMOTDJsonFile()
					if err != nil {
						h.EphemeralRespond(i, fmt.Sprintf("Something went wrong with saving the JSON file:\n```%v```", err))
					}
				}),
		))

	if err != nil {
		return err
	}
	return nil
}
