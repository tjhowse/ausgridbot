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
	MastodonTootInterval int64  `env:"MASTODON_TOOT_INTERVAL" envDefault:"1800"`
	TestMode             bool   `env:"TEST_MODE" envDefault:"false"`
}

func main() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		fmt.Printf("%+v\n", err)
	}

	aemo := NewAEMO()
	var m *Mastodon

	nextTootTime := time.Now().Add(-time.Second)

	slog.Info("Starting up")
	for {
		slog.Info("Getting data")
		aemoData, err := aemo.GetAEMOData("")
		if err != nil {
			slog.Error("failed to get data from AEMO:", err)
		}
		slog.Info("Got data")
		print(aemoData.Intervals[0].RegionID)
		if time.Now().After(nextTootTime) {
			// Calculate the next toot time.
			nextTootTime = nextTootTime.Add(time.Duration(cfg.MastodonTootInterval) * time.Second)

			// If the mastodon link is down, bring it back up.
			if m == nil && !cfg.TestMode {
				m, err = NewMastodon(cfg.MastodonURL, cfg.MastodonClientID, cfg.MastodonClientSecret)
				if err != nil {
					slog.Error("Failed to connect to mastodon: " + err.Error())
					time.Sleep(10 * time.Second)
					continue
				}
			}

			err = m.PostStatus("Tap tap. This thing on?")
			if err != nil {
				slog.Error(err.Error())
				m = nil
			}
		}

		time.Sleep(10 * time.Second)
	}
}
