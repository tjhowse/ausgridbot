package main

import (
	"image/color"
	"io"
	"strconv"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

func GetPlot(xAxisLabels []time.Time, data []float64, w io.Writer) error {
	// create a new line instance
	p := plot.New()

	p.Title.Text = "Energy price forecast"
	p.X.Label.Text = "Time"
	p.Y.Label.Text = "Price ($/MWh)"
	grid := plotter.NewGrid()
	p.Add(grid)

	// Add the price series as a line with no point markers

	items := make(plotter.XYs, len(xAxisLabels))
	for i := range xAxisLabels {
		// add the data to the plot
		_, tzOffset := xAxisLabels[i].Zone()
		items[i].X = float64(xAxisLabels[i].Unix() + int64(tzOffset))
		items[i].Y = data[i]
	}

	if line, err := plotter.NewLine(items); err != nil {
		return err
	} else {
		line.Color = color.RGBA{B: 255, A: 255}
		line.Width = vg.Points(1)
		p.Add(line)
	}

	// if err := plotutil.AddLinePoints(p, items); err != nil {
	// 	return err
	// }
	// p.NominalX(xAxisLabels...)
	p.X.Tick.Marker = plot.TimeTicks{Format: "15:04", Ticker: myTicker{TickCount: len(xAxisLabels)}}
	// p.X.Tick.Marker = plot.TimeTicks{Format: "15:04"}

	// Remove the circles gylphs from the plot

	if wt, err := p.WriterTo(7*vg.Inch, 3*vg.Inch, "png"); err != nil {
		return err
	} else {
		wt.WriteTo(w)
	}

	return nil
}

type myTicker struct {
	TickCount int
}

func (t myTicker) Ticks(min, max float64) []plot.Tick {
	ticks := make([]plot.Tick, t.TickCount)
	for i := range ticks {
		ticks[i].Value = min + (max-min)*float64(i)/float64(len(ticks)-1)
		ticks[i].Label = strconv.FormatFloat(ticks[i].Value, 'f', -1, 64)
	}

	return ticks
}

// func (myTicker) Ticks2(min, max float64) []plot.Tick {
// 	if max <= min {
// 		panic("illegal range")
// 	}

// 	const suggestedTicks = 3

// 	labels, step, q, mag := talbotLinHanrahan(min, max, suggestedTicks, withinData, nil, nil, nil)
// 	majorDelta := step * math.Pow10(mag)
// 	if q == 0 {
// 		// Simple fall back was chosen, so
// 		// majorDelta is the label distance.
// 		majorDelta = labels[1] - labels[0]
// 	}

// 	// Choose a reasonable, but ad
// 	// hoc formatting for labels.
// 	fc := byte('f')
// 	var off int
// 	if mag < -1 || 6 < mag {
// 		off = 1
// 		fc = 'g'
// 	}
// 	if math.Trunc(q) != q {
// 		off += 2
// 	}
// 	prec := minInt(6, maxInt(off, -mag))
// 	ticks := make([]Tick, len(labels))
// 	for i, v := range labels {
// 		ticks[i] = Tick{Value: v, Label: strconv.FormatFloat(v, fc, prec, 64)}
// 	}

// 	var minorDelta float64
// 	// See talbotLinHanrahan for the values used here.
// 	switch step {
// 	case 1, 2.5:
// 		minorDelta = majorDelta / 5
// 	case 2, 3, 4, 5:
// 		minorDelta = majorDelta / step
// 	default:
// 		if majorDelta/2 < dlamchP {
// 			return ticks
// 		}
// 		minorDelta = majorDelta / 2
// 	}

// 	// Find the first minor tick not greater
// 	// than the lowest data value.
// 	var i float64
// 	for labels[0]+(i-1)*minorDelta > min {
// 		i--
// 	}
// 	// Add ticks at minorDelta intervals when
// 	// they are not within minorDelta/2 of a
// 	// labelled tick.
// 	for {
// 		val := labels[0] + i*minorDelta
// 		if val > max {
// 			break
// 		}
// 		found := false
// 		for _, t := range ticks {
// 			if math.Abs(t.Value-val) < minorDelta/2 {
// 				found = true
// 			}
// 		}
// 		if !found {
// 			ticks = append(ticks, Tick{Value: val})
// 		}
// 		i++
// 	}

// 	return ticks
// }
