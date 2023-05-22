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
	defer file.Close()
	if err != nil {
		return err
	}

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
	motdCmd, err := h.AddSlashCommand("motd", "Set MOTD data", func(h *harmonia.Harmonia, i *harmonia.Invocation) {
		if !Contains(config.OwnerIDs, i.Author.ID) {
			h.EphemeralRespond(i, "You are not allowed to use this command.")
		}

		trigger := strings.ToLower(i.GetOption("trigger").StringValue())
		response := ""
		_, ok := i.GetOptionMap()["response"]
		if ok {
			response = i.GetOption("response").StringValue()
		}

		GuildMOTD, ok := MOTDDatabase[i.GuildID]
		if !ok {
			GuildMOTD = GuildMOTDData{}
			MOTDDatabase[i.GuildID] = GuildMOTD
		}

		_, ok = GuildMOTD[trigger]
		if response == "" {
			if !ok {
				h.EphemeralRespond(i, fmt.Sprintf("Trigger `%s` does not exist for this server and thus did not need to be cleared.", trigger))
				return
			}
			delete(GuildMOTD, trigger)
			h.EphemeralRespond(i, fmt.Sprintf("Trigger `%s` has been cleared.", trigger))
			SaveToMOTDJsonFile()
			return
		}

		GuildMOTD[trigger] = response
		h.EphemeralRespond(i, fmt.Sprintf("Trigger `%s` has been set to `%s`.", trigger, response))
		SaveToMOTDJsonFile()
	})
	if err != nil {
		return err
	}

	motdCmd.AddOption("trigger", "The trigger", true, discordgo.ApplicationCommandOptionString)
	motdCmd.AddOption("response", "The fish", false, discordgo.ApplicationCommandOptionString)
	return nil
}
