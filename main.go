package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/caarlos0/env/v9"
)

type config struct {
	MastodonURL          string `env:"MASTODON_SERVER" envDefault:"https://howse.social"`
	MastodonClientID     string `env:"MASTODON_CLIENT_ID"`
	MastodonClientSecret string `env:"MASTODON_CLIENT_SECRET"`
	MastodonUserEmail    string `env:"MASTODON_USER_EMAIL"`
	MastodonUserPassword string `env:"MASTODON_USER_PASSWORD"`
	MastodonTootInterval int64  `env:"MASTODON_TOOT_INTERVAL" envDefault:"1"`
	TestMode             bool   `env:"TEST_MODE" envDefault:"true"`
}

func main() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		fmt.Printf("%+v\n", err)
	}

	aemo := NewAEMO()

	// This is a map of regions to the GridBot handling that region.
	gridBots := make(map[string]*GridBot)

	gridBots["QLD1"] = NewGridBot(cfg, "Queensland")

	// Start the main loop for each GridBot
	for _, gb := range gridBots {
		go gb.Mainloop()
	}

	slog.Info("Starting up")
	for {
		slog.Info("Getting data")
		aemoData, err := aemo.GetAEMOData("")
		if err != nil {
			slog.Error("failed to get data from AEMO:", err)
		}
		slog.Info("Got data")
		// print(aemoData.Intervals[0].RegionID)

		for _, i := range aemoData.Intervals {
			// Send the interval to the appropriate GridBot
			if gb, ok := gridBots[i.RegionID]; ok {
				gb.GetIntervalChannel() <- i
			}
		}

		// Kick off processing for each GridBot
		for _, gb := range gridBots {
			close(gb.GetIntervalChannel())
		}

		time.Sleep(30 * time.Second)
	}
}
