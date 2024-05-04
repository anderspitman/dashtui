package dashtui

import (
	"log"
	"os"

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

	stdoutFile, err := openLogFile("stdout.txt")
	if err != nil {
		return nil, err
	}
	os.Stdout = stdoutFile

	stderrFile, err := openLogFile("stderr.txt")
	if err != nil {
		return nil, err
	}
	os.Stderr = stderrFile
	log.Default().SetOutput(stderrFile)

	layout := tview.NewFlex().SetDirection(tview.FlexRow)

	layout.SetFullScreen(true)

	data := make(map[string][]float64)
	charts := make(map[string]*tvxwidgets.Plot)

	maxLen := 256

	queue := make(chan item)
	go func() {
		for {
			it := <-queue

			d, ok := data[it.id]
			if !ok {
				d = make([]float64, maxLen)
			}

			d = append(d, it.value)
			if len(d) > maxLen {
				d = d[len(d)-maxLen:]
			}

			data[it.id] = d

			axisWidth := calcAxisWidth(d)

			c, ok := charts[it.id]
			if !ok {
				c = tvxwidgets.NewPlot()

				c.SetTitle(it.id)
				c.SetBorder(true)
				c.SetMarker(tvxwidgets.PlotMarkerBraille)
				c.SetDrawXAxisLabel(false)

				layout.AddItem(c, 0, 1, true)
				app.Draw()

				charts[it.id] = c
			}

			_, _, width, _ := c.GetInnerRect()

			dat := make([][]float64, 1)
			dat[0] = d[len(d)-width+axisWidth:]

			c.SetData(dat)

			app.Draw()
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

func calcAxisWidth(data []float64) int {
	max := 0.0
	for _, elem := range data {
		if elem > max {
			max = elem
		}
	}

	if max >= 100.0 {
		return 6
	} else if max >= 10.0 {
		return 5
	} else {
		return 4
	}
}

//stdoutR, stdoutW, err := os.Pipe()
//if err != nil {
//        return nil, err
//}

//stdoutTeeR := io.TeeReader(stdoutR, stdoutFile)

//os.Stdout = stdoutW
//stdoutBR := bufio.NewReader(stdoutTeeR)

//stdoutText := widgets.NewParagraph()
//stdoutText.Title = "stdout"
//stdoutText.Text = ""
////stdoutText.WrapText = true
//stdoutText.SetRect(0, 0, termWidth, termHeight/2)

//stderrR, stderrW, err := os.Pipe()
//if err != nil {
//        return nil, err
//}

//stderrTeeR := io.TeeReader(stderrR, stderrFile)

//os.Stderr = stderrW
//log.Default().SetOutput(stderrW)

//stderrBR := bufio.NewReader(stderrTeeR)

//stderrText := widgets.NewParagraph()
//stderrText.Title = "stderr"
//stderrText.Text = ""
//stderrText.SetRect(0, termHeight/2, termWidth, termHeight)

//views := []ui.Drawable{ stdoutText, stderrText }

//maxLines := 1024
//stdoutLines := make([]string, maxLines)
//stderrLines := make([]string, maxLines)

//go func() {
//        for {
//                line, err := stdoutBR.ReadString('\n')
//                if err != nil {
//                        fmt.Fprintf(os.Stderr, "%v", err)
//                }

//                rect := stdoutText.GetRect()
//                width := rect.Max.X - rect.Min.X - 2
//                height := rect.Max.Y - rect.Min.Y - 2

//                if len(line) > width {
//                        line = line[:width-1] + "\n"
//                }

//                stdoutLines = append(stdoutLines, line)
//                stdoutLines = stdoutLines[1:maxLines]

//                fmt.Fprintf(os.Stderr, "%d\n", len(stdoutLines) - height)

//                visibleLines := stdoutLines[len(stdoutLines) - height:]
//                visibleStr := strings.Join(visibleLines, "")

//                stdoutText.Text = visibleStr
//                ui.Render(views...)
//        }
//}()

//go func() {
//        for {
//                line, err := stderrBR.ReadString('\n')
//                if err != nil {
//                        fmt.Fprintf(os.Stderr, "%v", err)
//                }

//                rect := stderrText.GetRect()
//                width := rect.Max.X - rect.Min.X - 2
//                height := rect.Max.Y - rect.Min.Y - 2

//                if len(line) > width {
//                        line = line[:width-1] + "\n"
//                }

//                stderrLines = append(stderrLines, line)
//                stderrLines = stderrLines[1:maxLines]

//                visibleLines := stderrLines[len(stderrLines) - height:]
//                visibleStr := strings.Join(visibleLines, "")

//                stderrText.Text = visibleStr
//                ui.Render(views...)
//        }
//}()
