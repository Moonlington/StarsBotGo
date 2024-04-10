package main

import (
	"context"
	"fmt"

	"github.com/Moonlington/harmonia"
	"github.com/bwmarrin/discordgo"
	customsearch "google.golang.org/api/customsearch/v1"
	"google.golang.org/api/option"
)

func AddGoogleHandlers(h *harmonia.Harmonia) error {
	ctx := context.Background()

	svc, err := customsearch.NewService(ctx, option.WithAPIKey(config.GoogleAPIToken))
	if err != nil {
		return err
	}

	h.AddCommand(harmonia.NewSlashCommand("google").
		WithDescription("Googles something").
		WithOptions(
			harmonia.NewOption("query", discordgo.ApplicationCommandOptionString).
				WithDescription("Search query").
				IsRequired(),
		).
		WithCommand(func(h *harmonia.Harmonia, i *harmonia.Invocation) {
			query := i.GetOption("query").StringValue()

			resp, err := svc.Cse.List().Cx(config.GoogleSearchEngineID).Q(query).Do()
			if err != nil {
				h.EphemeralRespond(i, fmt.Sprintf("Failed querying google: %s", err))
				return
			}
			if len(resp.Items) == 0 {
				h.Respond(i, fmt.Sprintf("Found nothing for query `%s`", query))
				return
			}
			result := resp.Items[0]
			h.Respond(i, fmt.Sprintf("`%s`\n%s", query, result.Link))
		}))

	return nil
}
