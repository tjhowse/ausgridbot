package main

import (
	"fmt"
	"log/slog"
	"math"
	"time"
)

const INTERESTING_PEAK_RRP = 300
const PEAK_TOOT_FORMAT = "A new %s wholesale electricity price peak of $%.2f/kWh is predicted at %s: https://aemo.com.au/aemo/apps/visualisations/elec-nem-priceanddemand.html"
const PEAK_CANCELLED_TOOT_FORMAT = "The %s wholesale electricity price peak of $%.2f/kWh at %s has been averted. Thanks AEMO! https://aemo.com.au/aemo/apps/visualisations/elec-nem-priceanddemand.html"

type GridBot struct {
	m                  *Mastodon
	input              chan Interval
	cfg                config
	regionString       string
	intervalBuffer     []Interval
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
	gb.resetIntervalBuffer()
	gb.peakRRP = -1000
	return gb
}

func (gb *GridBot) GetIntervalChannel() chan Interval {
	return gb.input
}

func (gb *GridBot) resetIntervalBuffer() {
	gb.intervalBuffer = make([]Interval, 0, 1000)
	gb.input = make(chan Interval)
}

func (gb *GridBot) Mainloop() {
	for {

		for i := range gb.input {
			gb.ProcessInterval(i)
		}
		gb.ConsiderPostingToot()
		gb.resetIntervalBuffer()
	}
}

func (gb *GridBot) ConsiderPostingToot() {
	var err error

	// We've already tooted about this peak.
	if FloatEquals(gb.lastTootedPeakRRP, gb.peakRRP) && gb.lastTootedPeakTime.Equal(gb.peakTime) {
		return
	}

	var toot string

	// If the new peak is below INTERESTING_PEAK_RRP but the previous peak was above, publish a
	// retraction saying the peak was cancelled.
	if gb.peakRRP < INTERESTING_PEAK_RRP && gb.lastTootedPeakRRP > INTERESTING_PEAK_RRP {
		toot = fmt.Sprintf(PEAK_CANCELLED_TOOT_FORMAT, gb.regionString, gb.lastTootedPeakRRP/1000, gb.lastTootedPeakTime.Format("15:04"))
	} else if gb.peakRRP > INTERESTING_PEAK_RRP {
		toot = fmt.Sprintf(PEAK_TOOT_FORMAT, gb.regionString, gb.peakRRP/1000, gb.peakTime.Format("15:04"))
	} else {
		fmt.Println("Not tooting, too boring")
		return
	}

	// Toot it
	if gb.cfg.TestMode {
		slog.Info("Would toot:", "toot", toot)
	} else {
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
	gb.lastToot = toot
	gb.lastTootedPeakRRP = gb.peakRRP
	gb.lastTootedPeakTime = gb.peakTime
}

func (gb *GridBot) ProcessInterval(i Interval) {
	// Ignore data that isn't a forecast
	if i.PeriodType != "FORECAST" {
		return
	}
	// Ignore data more than 8 hours into the future.
	if i.SettlementDate.Time.After(time.Now().Add(8 * time.Hour)) {
		return
	}

	// If this is the first interval we've received after a reset,
	// reset the peak values.
	if len(gb.intervalBuffer) == 0 {
		gb.peakRRP = i.RRP
		gb.peakTime = i.SettlementDate.Time
	} else if i.RRP > gb.peakRRP {
		gb.peakRRP = i.RRP
		gb.peakTime = i.SettlementDate.Time
	}

	fmt.Println("Adding to intervalbuffer")

	gb.intervalBuffer = append(gb.intervalBuffer, i)
}
