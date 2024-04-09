package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Moonlington/harmonia"
	"github.com/bwmarrin/discordgo"
)

var (
	validRoleName = regexp.MustCompile(`^(.+) \(#([0-9a-fA-F]{6})\)$`)
	validHexCode  = regexp.MustCompile(`^#?([0-9a-fA-F]{6})`)
)

func AddColorHandlers(h *harmonia.Harmonia) error {
	err := h.AddCommand(harmonia.NewSlashCommand("color").
		WithDescription("Gives yourself a color").
		WithCommand(func(h *harmonia.Harmonia, i *harmonia.Invocation) {
			// Check if valid hex
			inputHex := i.GetOption("hex").StringValue()
			if !validHexCode.MatchString(inputHex) {
				h.EphemeralRespond(i, "The color chosen must be in the form of a hex code. (e.g. #ffd700)")
				return
			}

			// Convert to int
			hexCode := validHexCode.FindStringSubmatch(inputHex)[1]
			roleColor, err := strconv.ParseInt(hexCode, 16, 0)
			if err != nil {
				h.EphemeralRespond(i, "Could not convert the hex code into an integer, something went wrong.")
				return
			}
			color := int(roleColor)

			// Check if user already has a color role given by the bot
			hasStarsColorRole := false
			var starsRole *discordgo.Role
			for _, role := range i.Author.Roles {
				if validRoleName.MatchString(role.Name) {
					hasStarsColorRole = true
					starsRole = role
					break
				}
			}

			roleName := fmt.Sprintf("%s (#%s)", i.GetOption("name").StringValue(), strings.ToUpper(hexCode))

			if hasStarsColorRole && starsRole != nil {
				_, err := h.GuildRoleEdit(i.GuildID, starsRole.ID, &discordgo.RoleParams{Name: roleName, Color: &color})
				if err != nil {
					h.EphemeralRespond(i, "There was an error editing your color role.")
					return
				}
				h.EphemeralRespond(i, "Edited your old color role.")
				return
			} else {
				starsRole, err := h.GuildRoleCreate(i.GuildID, &discordgo.RoleParams{Name: roleName, Color: &color})
				if err != nil {
					h.EphemeralRespond(i, "There was an error creating your color role.")
					return
				}

				err = h.GuildMemberRoleAdd(i.GuildID, i.Author.ID, starsRole.ID)
				if err != nil {
					h.EphemeralRespond(i, "There was an error giving you your color role.")
					return
				}
				h.EphemeralRespond(i, "Created your color role.")
				return
			}
		}).
		WithOptions(
			harmonia.NewOption("hex", discordgo.ApplicationCommandOptionString).
				WithDescription("The color you want your role to be").
				IsRequired(),
			harmonia.NewOption("name", discordgo.ApplicationCommandOptionString).
				WithDescription("The name you want your role to have").
				IsRequired()),
	)

	if err != nil {
		return err
	}

	return nil
}
