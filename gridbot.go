package main

import (
	"fmt"
	"log/slog"
	"time"
)

const INTERESTING_PEAK_RRP = 250

type GridBot struct {
	m                  *Mastodon
	input              chan Interval
	cfg                config
	regionString       string
	lastInterval       Interval
	lastTootedPeakRRP  float64
	lastTootedPeakTime time.Time
	urgentMessage      chan bool

	peakRRP  float64
	peakTime time.Time
}

func NewGridBot(cfg config, regionString string) *GridBot {
	gb := &GridBot{}
	gb.input = make(chan Interval)
	gb.cfg = cfg
	gb.regionString = regionString
	gb.urgentMessage = make(chan bool)
	return gb
}

func (gb *GridBot) GetChannel() chan Interval {
	return gb.input
}

func (gb *GridBot) Mainloop() {
	for {
		select {
		case i := <-gb.input:
			// Do something with the interval
			gb.lastInterval = i
			gb.ProcessInterval(i)
		case <-gb.urgentMessage:
			gb.ConsiderPostingToot()
		case <-time.After(time.Duration(gb.cfg.MastodonTootInterval) * time.Second):
			// Post to mastodon
			gb.ConsiderPostingToot()
		}
	}
}

func (gb *GridBot) ConsiderPostingToot() {
	var err error
	if gb.m == nil {
		gb.m, err = NewMastodon(gb.cfg.MastodonURL, gb.cfg.MastodonClientID, gb.cfg.MastodonClientSecret)
		if err != nil {
			slog.Error("Failed to connect to mastodon: " + err.Error())
			return
		}
	}
	if gb.lastTootedPeakRRP == gb.peakRRP && gb.lastTootedPeakTime == gb.peakTime {
		return
	}
	toot := fmt.Sprintf("A new %s wholesale electricity price peak of $%.2f/kWh is predicted at %s: https://aemo.com.au/aemo/apps/visualisations/elec-nem-priceanddemand.html", gb.regionString, gb.peakRRP/1000, gb.peakTime.Format("15:04"))

	// Toot it
	err = gb.m.PostStatus(toot)
	if err != nil {
		slog.Error("Failed to toot: " + err.Error())
		return
	}
	// slog.Info("Tootin'", "peakRRP", gb.peakRRP, "peakTime", gb.peakTime.String())
	gb.lastTootedPeakRRP = gb.peakRRP
	gb.lastTootedPeakTime = gb.peakTime
}

func (gb *GridBot) ProcessInterval(i Interval) {
	// Ignore data more than 8 hours into the future.
	if !i.SettlementDate.Time.Before(time.Now().Add(8 * time.Hour)) {
		return
	}
	// Ignore boring prices.
	if i.RRP < gb.peakRRP {
		return
	}
	// If the new value is bigger than the current peak, or the same but at a different time,
	// log a new peak.
	if i.RRP > gb.peakRRP || (i.RRP == gb.peakRRP && i.SettlementDate.Time != gb.peakTime) {
		gb.peakRRP = i.RRP
		gb.peakTime = i.SettlementDate.Time
	}
}
