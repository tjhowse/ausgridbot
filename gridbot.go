package main

import (
	"fmt"
	"log/slog"
	"math"
	"time"
)

const INTERESTING_PEAK_RRP = 300
const PEAK_TOOT_FORMAT = "A new %s wholesale electricity price peak of $%.2f/kWh is predicted at %s: https://aemo.com.au/aemo/apps/visualisations/elec-nem-priceanddemand.html"
const PEAK_DOWNGRADE_TOOT_FORMAT = "The %s predicted wholesale electricity price peak of $%.2f/kWh been downgraded to a peak of $%.2f/kWh at %s has: https://aemo.com.au/aemo/apps/visualisations/elec-nem-priceanddemand.html"
const PEAK_CANCELLED_TOOT_FORMAT = "The %s wholesale electricity price peak of $%.2f/kWh at %s has been averted. Thanks AEMO! https://aemo.com.au/aemo/apps/visualisations/elec-nem-priceanddemand.html"

type GridBot struct {
	m                  *Mastodon
	input              chan Interval
	cfg                config
	regionString       string
	lastTootedPeakRRP  float64
	lastTootedPeakTime time.Time
	lastToot           string

	peakRRP  float64
	peakTime time.Time
}

func FloatEquals(a, b float64) bool {
	return math.Abs(a-b) < 0.00000001
}

func NewGridBot(cfg config, regionString string) *GridBot {
	gb := &GridBot{}
	gb.cfg = cfg
	gb.regionString = regionString
	gb.resetIntervalChannel()
	return gb
}

func (gb *GridBot) GetIntervalChannel() chan Interval {
	return gb.input
}

func (gb *GridBot) resetIntervalChannel() {
	gb.input = make(chan Interval)
	gb.peakRRP = -20000
}

func (gb *GridBot) Mainloop() {
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
	var err error

	// We've already tooted about this peak.
	if FloatEquals(gb.lastTootedPeakRRP, gb.peakRRP) && gb.lastTootedPeakTime.Equal(gb.peakTime) {
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
		fmt.Println("Not tooting, too boring")
		return
	}

	slog.Info("Toot!", "toot", toot)

	gb.lastTootedPeakRRP = gb.peakRRP
	gb.lastTootedPeakTime = gb.peakTime
	gb.lastToot = toot

	// Toot it
	if !gb.cfg.TestMode {
		if gb.m == nil {
			gb.m, err = NewMastodon(gb.cfg.MastodonURL, gb.cfg.MastodonClientID, gb.cfg.MastodonClientSecret)
			if err != nil {
				slog.Error("Failed to connect to mastodon: " + err.Error())
				return
			}
		}

		err = gb.m.PostStatus(toot)
		if err != nil {
			slog.Error("Failed to toot: " + err.Error())
			return
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
