package dashtui

import (
	"io"
	"log"
	"os"
	"runtime"
	"time"

	//"github.com/gdamore/tcell/v2"
	"github.com/navidys/tvxwidgets"
	"github.com/rivo/tview"
)

type Builder struct {
	disabled bool
}

type DashTUI struct {
	queue chan item
}

type item struct {
	id    string
	value float64
}

type datum struct {
	value     float64
	timestamp time.Time
}

func NewBuilder() *Builder {

	b := &Builder{}

	return b
}

func (b *Builder) Disable() *Builder {
	b.disabled = true
	return b
}

func (b *Builder) Build() (*DashTUI, error) {
	if b.disabled {
		return &DashTUI{}, nil
	}

	app := tview.NewApplication()

	//stdoutFile, err := openLogFile("stdout.txt")
	//if err != nil {
	//	return nil, err
	//}
	//os.Stdout = stdoutFile

	//stderrFile, err := openLogFile("stderr.txt")
	//if err != nil {
	//	return nil, err
	//}
	//os.Stderr = stderrFile
	//log.Default().SetOutput(stderrFile)

	makeTextView := func(label string) (*tview.TextView, *os.File, error) {
		text := tview.NewTextView().
			SetScrollable(false).
			SetChangedFunc(func() {
				app.Draw()
			})
		text.
			SetTitle(label).
			SetBorder(true).
			SetFocusFunc(func() {
				log.Println("selected", label)
			})

		pipeR, pipeW, err := os.Pipe()
		if err != nil {
			return nil, nil, err
		}
		go io.Copy(text, pipeR)

		return text, pipeW, nil
	}

	stdoutText, stdoutW, err := makeTextView("stdout")
	if err != nil {
		return nil, err
	}
	os.Stdout = stdoutW

	stderrText, stderrW, err := makeTextView("stderr")
	if err != nil {
		return nil, err
	}
	os.Stderr = stderrW
	log.Default().SetOutput(stderrW)

	layout := tview.NewFlex().SetDirection(tview.FlexRow)

	layout.SetFullScreen(true)

	layout.AddItem(stdoutText, 0, 1, true)
	layout.AddItem(stderrText, 0, 1, true)

	allData := make(map[string][]datum)
	charts := make(map[string]*tvxwidgets.Plot)

	maxLen := 1024
	timeWindow := 10 * time.Second
	// 60fps
	samplePeriod := 16667 * time.Microsecond

	ticker := time.NewTicker(100 * time.Millisecond)

	queue := make(chan item)
	go func() {
		for {
			select {
			case it := <-queue:

				now := time.Now()

				data, ok := allData[it.id]
				if !ok {
					data = make([]datum, maxLen)

					for i, _ := range data {
						data[i] = datum{
							value:     0,
							timestamp: now,
						}
					}

					allData[it.id] = data
				}

				timeSinceLastSample := now.Sub(data[len(data)-1].timestamp)
				if timeSinceLastSample >= samplePeriod {

					data = append(data, datum{
						value:     it.value,
						timestamp: now,
					})
					if len(data) > maxLen {
						data = data[len(data)-maxLen:]
					}

					allData[it.id] = data
				}
			case <-ticker.C:
				for id, data := range allData {
					c, exists := charts[id]
					if !exists {
						c = tvxwidgets.NewPlot()

						c.SetTitle(id)
						c.SetBorder(true)
						c.SetMarker(tvxwidgets.PlotMarkerBraille)
						c.SetDrawXAxisLabel(false)

						layout.AddItem(c, 0, 1, true)
						app.Draw()

						charts[id] = c
					}

					_, _, width, _ := c.GetPlotRect()

					numPoints := width

					plotData := make([][]float64, 1)
					plotData[0] = make([]float64, len(data))

					points := getPoints(data, numPoints, timeWindow)

					plotData[0] = points
					c.SetData(plotData)

					app.Draw()
				}
			}
		}
	}()

	go func() {
		if err := app.SetRoot(layout, false).EnableMouse(true).Run(); err != nil {
			panic(err)
		}

		os.Exit(1)
	}()

	d := &DashTUI{
		queue: queue,
	}

	go func() {
		m := runtime.MemStats{}
		for {
			runtime.ReadMemStats(&m)

			d.Set("memory MiB/s", float64(m.HeapAlloc/1024/1024))
			d.Set("goroutines", float64(runtime.NumGoroutine()))
			time.Sleep(100 * time.Millisecond)
		}
	}()

	return d, nil
}

func (dt *DashTUI) Close() {
	//ui.Close()
}

func (dt *DashTUI) Set(id string, value float64) {
	dt.queue <- item{
		id,
		value,
	}
}

func openLogFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
}

func getPoints(data []datum, numPoints int, timeWindow time.Duration) []float64 {
	var points []float64

	now := time.Now()
	endTime := now
	startTime := endTime.Add(-timeWindow)

	lastDatum := data[len(data)-1]

	if lastDatum.timestamp.Before(startTime) {
		points = make([]float64, len(data))
		for i := range points {
			points[i] = lastDatum.value
		}
	} else {
		points = samplePoints(data, numPoints, timeWindow, startTime)
	}

	return points
}

func samplePoints(data []datum, numPoints int, timeWindow time.Duration, startTime time.Time) []float64 {

	points := make([]float64, len(data))

	timeStep := timeWindow / time.Duration(numPoints)
	stepStart := startTime
	stepEnd := startTime.Add(timeStep)

	// Find the last value before startTime
	lastValueIdx := 0
	for i := len(data) - 1; i >= 0; i-- {
		if data[i].timestamp.Before(startTime) {
			lastValueIdx = i
			break
		}
	}

	lastValue := data[lastValueIdx].value
	for i := range points {

		found := false

		for _, d := range data[lastValueIdx:] {
			if d.timestamp.After(stepStart) && d.timestamp.Before(stepEnd) {
				points[i] = d.value
				lastValue = d.value
				found = true
				break
			}
		}

		if !found {
			points[i] = lastValue
		}

		stepStart = stepEnd
		stepEnd = stepStart.Add(timeStep)
	}

	return points
}
