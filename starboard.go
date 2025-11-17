package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"time"

	"github.com/Moonlington/harmonia"
	"github.com/bwmarrin/discordgo"
)

type StarboardData map[string]GuildStarboardData
type GuildStarboardData map[string]string

var StarboardDatabase StarboardData

func ReadFromStarboardJsonFile() error {
	data, err := os.ReadFile("starboard_data.json")
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &StarboardDatabase); err != nil {
		return err
	}
	return nil
}

func SaveToStarboardJsonFile() error {
	file, err := os.Create("starboard_data.json")
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(&StarboardDatabase)
	if err != nil {
		return err
	}

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func AddStarboardHandlers(h *harmonia.Harmonia) error {
	if err := ReadFromStarboardJsonFile(); err != nil {
		return err
	}

	h.AddHandler(func(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
		handleReactionEvent(h, r.MessageReaction)
	})
	h.AddHandler(func(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
		handleReactionEvent(h, r.MessageReaction)
	})

	return nil
}

func handleReactionEvent(h *harmonia.Harmonia, r *discordgo.MessageReaction) {
	if r.Emoji.Name != "⭐" {
		return
	}

	if r.ChannelID == config.StarChannel {
		return
	}

	if r.GuildID == "" {
		return
	}

	GuildStarboard, ok := StarboardDatabase[r.GuildID]
	if !ok {
		GuildStarboard = GuildStarboardData{}
		StarboardDatabase[r.GuildID] = GuildStarboard
	}

	message, err := h.ChannelMessage(r.ChannelID, r.MessageID)
	if err != nil {
		fmt.Printf("Unable to get message: %v", err)
		return
	}

	starCount := 0
	for _, reaction := range message.Reactions {
		if reaction.Emoji.Name == r.Emoji.Name {
			starCount = reaction.Count
			break
		}
	}

	existingStarboardMessageID, ok := GuildStarboard[r.MessageID]

	if starCount < 3 {
		if ok {
			h.ChannelMessageDelete(config.StarChannel, existingStarboardMessageID)
			delete(GuildStarboard, r.MessageID)
			SaveToStarboardJsonFile()
		}
		return
	}

	starboardEmbed := &discordgo.MessageEmbed{}
	member, err := h.GuildMember(r.GuildID, message.Author.ID)
	if err != nil {
		fmt.Println(r.GuildID, message.Author.ID)
		fmt.Printf("Unable to get member: %v", err)
		return
	}
	roles, err := harmonia.RolesFromMember(h, member)
	if err != nil {
		fmt.Printf("Unable to get roles: %v", err)
		return
	}
	sort.SliceStable(roles, func(i, j int) bool {
		return roles[i].Position > roles[j].Position
	})

	starboardEmbed.Color = 0
	for _, role := range roles {
		if role.Color != 0 {
			starboardEmbed.Color = role.Color
			break
		}
	}

	messageContentReplaced, err := message.ContentWithMoreMentionsReplaced(h.Session)
	if err != nil {
		return
	}
	starboardEmbed.Description = messageContentReplaced

	if message.MessageReference != nil {
		referencedMessage, err := h.ChannelMessage(message.MessageReference.ChannelID, message.MessageReference.MessageID)
		if err != nil {
			return
		}
		starboardEmbed.Description = fmt.Sprintf("\n**[Replying](https://discord.com/channels/%s/%s/%s) to %s**\n", r.GuildID, referencedMessage.ChannelID, referencedMessage.ID, referencedMessage.Author.Mention()) + starboardEmbed.Description
	}

	starboardEmbed.Fields = append(starboardEmbed.Fields, &discordgo.MessageEmbedField{
		Name:   "Source",
		Value:  fmt.Sprintf("[Jump!](https://discord.com/channels/%s/%s/%s)", r.GuildID, message.ChannelID, message.ID),
		Inline: false,
	})

	imgregex := regexp.MustCompile(`(?i)https?:\/\/(.+\/)+.+(\.(gif|png|jpg|jpeg|webp|svg|psd|bmp|tif|jfif))`)
	imgspoilerregex := regexp.MustCompile(`(?i)https?:\/\/(.+\/)+(SPOILER_.+)(\.(gif|png|jpg|jpeg|webp|svg|psd|bmp|tif|jfif))`)

	image := imgregex.FindString(messageContentReplaced)
	if image != "" {
		starboardEmbed.Image = &discordgo.MessageEmbedImage{
			URL: image,
		}
	}

	if len(message.Attachments) > 0 {
		image = imgregex.FindString(message.Attachments[0].URL)
		spoileredimage := imgspoilerregex.FindString(message.Attachments[0].URL)
		if image != "" && spoileredimage == "" {
			starboardEmbed.Image = &discordgo.MessageEmbedImage{
				URL: image,
			}
		}
		if spoileredimage != "" {
			starboardEmbed.Description += "\n**Spoilered Image**"
		}
	}

	name := member.User.Username
	if member.Nick != "" {
		name = member.Nick
	}

	starboardEmbed.Author = &discordgo.MessageEmbedAuthor{
		Name:    name,
		IconURL: member.AvatarURL(""),
	}

	starboardEmbed.Timestamp = message.Timestamp.Format(time.RFC3339)

	star := "⭐"
	if starCount >= 6 {
		star = "🌟"
	}
	if starCount >= 10 {
		star = "💫"
	}

	content := fmt.Sprintf("%s **%d** <#%s>", star, starCount, message.ChannelID)
	if !ok {
		starboardMessage, err := h.ChannelMessageSendComplex(config.StarChannel, &discordgo.MessageSend{
			Content: content,
			Embeds:  []*discordgo.MessageEmbed{starboardEmbed},
		})
		if err != nil {
			fmt.Printf("Unable to send message: %v", err)
			return
		}
		GuildStarboard[r.MessageID] = starboardMessage.ID
	} else {
		_, err := h.ChannelMessageEditComplex(&discordgo.MessageEdit{
			ID:      existingStarboardMessageID,
			Content: &content,
			Embeds:  &[]*discordgo.MessageEmbed{starboardEmbed},
			Channel: config.StarChannel,
		})
		if err != nil {
			fmt.Printf("Unable to edit message: %v", err)
			return
		}
	}
	SaveToStarboardJsonFile()
}
