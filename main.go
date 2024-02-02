package main

import (
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/caarlos0/env/v9"
)

type config struct {
	MastodonURL          string `env:"MASTODON_SERVER"`
	MastodonClientID     string `env:"MASTODON_CLIENT_ID"`
	MastodonClientSecret string `env:"MASTODON_CLIENT_SECRET"`
	MastodonUserEmail    string `env:"MASTODON_USER_EMAIL"`
	MastodonUserPassword string `env:"MASTODON_USER_PASSWORD"`
	MastodonTootInterval int64  `env:"MASTODON_TOOT_INTERVAL" envDefault:"1800"`
	ImageURL             string `env:"IMAGE_URL"`
	ImageURLParsed       *url.URL
	ImageUpdateInterval  int64 `env:"IMAGE_UPDATE_INTERVAL" envDefault:"300"`
	ImageFrameCount      int64 `env:"IMAGE_FRAME_COUNT" envDefault:"12"`
	ImageFrameDelay      int64 `env:"IMAGE_FRAME_DELAY" envDefault:"50"`
	ImageMinDuration     int64 `env:"IMAGE_MINIMUM_DURATION" envDefault:"1"`
	TestMode             bool  `env:"TEST_MODE" envDefault:"false"`
}

func main() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		fmt.Printf("%+v\n", err)
	}
	for {
		slog.Error("Hi!")
		time.Sleep(1 * time.Second)
	}
}
