package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"time"
)

const INTERESTING_PEAK_RRP = 500

// This amounts to 5 cents /kWh
const UNINTERESTING_DELTA_RRP = 50
const PEAK_TOOT_FORMAT = "A new %s wholesale electricity price peak of $%.2f/kWh is predicted at %s: https://aemo.com.au/aemo/apps/visualisations/elec-nem-priceanddemand.html"
const PEAK_DOWNGRADE_TOOT_FORMAT = "The %s predicted wholesale electricity price peak of $%.2f/kWh has been downgraded to a peak of $%.2f/kWh at %s: https://aemo.com.au/aemo/apps/visualisations/elec-nem-priceanddemand.html"
const PEAK_CANCELLED_TOOT_FORMAT = "The %s wholesale electricity price peak of $%.2f/kWh at %s has been averted. Thanks AEMO! https://aemo.com.au/aemo/apps/visualisations/elec-nem-priceanddemand.html"

const INTRO_TOOT = "Testing, testing, 1, 2, 3. This is a test toot from the gridbot. If you see this, it's working."

type GridBot struct {
	m                  *Mastodon
	input              chan Interval
	cfg                GridBotCfg
	regionString       string
	lastTootedPeakRRP  float64
	lastTootedPeakTime time.Time
	lastToot           string
	testTootSent       bool

	peakRRP  float64
	peakTime time.Time
}

func BuildGridBots(cfg config) (gridBotMap, error) {
	var err error
	// This is a map of RegionID to string
	gridBots := make(gridBotMap)

	// Deserialise the credentials envar
	var credentials []GridBotCfg
	if err := json.Unmarshal([]byte(cfg.GridBotCredentials), &credentials); err != nil {
		slog.Error("Failed to deserialise credentials:", err)
	}

	if len(credentials) == 0 {
		slog.Info("Falling back to old credential envars")
		// Fall back to old operation
		gbCfg := GridBotCfg{
			RegionID:             "QLD1",
			MastodonClientID:     cfg.MastodonClientID,
			MastodonClientSecret: cfg.MastodonClientSecret,
			MastodonUserEmail:    cfg.MastodonUserEmail,
			MastodonUserPassword: cfg.MastodonUserPassword,
			TestMode:             cfg.TestMode,
			MastodonURL:          cfg.MastodonURL,
		}
		if gridBots["QLD1"], err = NewGridBot(gbCfg); err != nil {
			return nil, fmt.Errorf("failed to create GridBot: %s", err)
		}
	} else {
		slog.Info("Using credentials from JSON envar.")
		for _, c := range credentials {
			c.TestMode = cfg.TestMode
			c.MastodonURL = cfg.MastodonURL
			if gridBots[c.RegionID], err = NewGridBot(c); err != nil {
				return nil, fmt.Errorf("failed to create GridBot: %s", err)
			}
		}
	}

	return gridBots, nil
}

func FloatEquals(a, b float64) bool {
	return math.Abs(a-b) < 0.00000001
}

func RegionIDToRegionString(regionID RegionID) (string, error) {
	switch regionID {
	case "QLD1":
		return "Queensland", nil
	case "NSW1":
		return "New South Wales", nil
	case "SA1":
		return "South Australia", nil
	case "TAS1":
		return "Tasmania", nil
	case "VIC1":
		return "Victoria", nil
	default:
		return "", fmt.Errorf("unknown region ID: %s", regionID)
	}
}

func NewGridBot(cfg GridBotCfg) (*GridBot, error) {
	gb := &GridBot{}
	gb.cfg = cfg
	if s, err := RegionIDToRegionString(cfg.RegionID); err != nil {
		return nil, fmt.Errorf("failed to convert region ID \"%s\" to string: %s", cfg.RegionID, err)
	} else {
		gb.regionString = s
	}
	gb.resetIntervalChannel()
	gb.SendTestToot()
	return gb, nil
}

func (gb *GridBot) GetIntervalChannel() chan Interval {
	return gb.input
}
func (gb *GridBot) SendTestToot() {
	if !gb.cfg.TestMode && !gb.testTootSent {
		if err := gb.sendToot(INTRO_TOOT); err != nil {
			slog.Error("Failed to send test toot:", err)
		}
		gb.testTootSent = true
	}
}

func (gb *GridBot) sendToot(toot string) error {
	var err error
	if gb.m == nil {
		gb.m, err = NewMastodon(gb.cfg.MastodonURL, gb.cfg.MastodonClientID, gb.cfg.MastodonClientSecret)
		if err != nil {
			return fmt.Errorf("failed to connect to mastodon: %s", err)
		}
	}
	err = gb.m.PostStatus(INTRO_TOOT)
	if err != nil {
		gb.m = nil
		return fmt.Errorf("failed to toot: %s", err)
	}
	return nil
}

func (gb *GridBot) resetIntervalChannel() {
	gb.input = make(chan Interval)
	gb.peakRRP = -20000
}

func (gb *GridBot) Mainloop() {
	slog.Info("Launching gridbot", "region", gb.regionString)
	for {
		for i := range gb.input {
			gb.processInterval(i)
			slog.Debug("Processed interval", "rrp", i.RRP, "time", i.SettlementDate.Time)
		}
		gb.considerPostingToot()
		gb.resetIntervalChannel()
	}
}

func (gb *GridBot) considerPostingToot() {

	// We've already tooted about this peak.
	if FloatEquals(gb.lastTootedPeakRRP, gb.peakRRP) && gb.lastTootedPeakTime.Equal(gb.peakTime) {
		return
	}

	// If the change in peak RRP is uninterestingly small, ignore it.
	if math.Abs(gb.peakRRP-gb.lastTootedPeakRRP) < UNINTERESTING_DELTA_RRP && gb.lastTootedPeakTime.Equal(gb.peakTime) {
		return
	}

	var toot string

	if gb.peakRRP < INTERESTING_PEAK_RRP && gb.lastTootedPeakRRP > INTERESTING_PEAK_RRP {
		// If the new peak is below INTERESTING_PEAK_RRP but the previous peak was above, publish a
		// retraction saying the peak was cancelled.
		toot = fmt.Sprintf(PEAK_CANCELLED_TOOT_FORMAT, gb.regionString, gb.lastTootedPeakRRP/1000, gb.lastTootedPeakTime.Format("15:04"))
	} else if gb.peakRRP > INTERESTING_PEAK_RRP {
		if gb.peakRRP > gb.lastTootedPeakRRP {
			toot = fmt.Sprintf(PEAK_TOOT_FORMAT, gb.regionString, gb.peakRRP/1000, gb.peakTime.Format("15:04"))
		} else {
			toot = fmt.Sprintf(PEAK_DOWNGRADE_TOOT_FORMAT, gb.regionString, gb.lastTootedPeakRRP/1000, gb.peakRRP/1000, gb.peakTime.Format("15:04"))
		}
	} else {
		return
	}

	slog.Info("Toot!", "toot", toot)

	gb.lastTootedPeakRRP = gb.peakRRP
	gb.lastTootedPeakTime = gb.peakTime
	gb.lastToot = toot

	// Toot it
	if !gb.cfg.TestMode {
		if err := gb.sendToot(toot); err != nil {
			slog.Error("Failed to send toot:", err)
		}
	}
}

func (gb *GridBot) processInterval(i Interval) {
	// Ignore data that isn't a forecast
	if i.PeriodType != "FORECAST" {
		return
	}
	// Ignore data more than 8 hours into the future.
	if i.SettlementDate.Time.After(time.Now().Add(8 * time.Hour)) {
		return
	}

	if i.RRP > gb.peakRRP {
		gb.peakRRP = i.RRP
		gb.peakTime = i.SettlementDate.Time
	}
}
