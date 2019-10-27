package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/kirill-scherba/teonet-go/services/teoapi"
)

func termui(api *teoapi.Teoapi) {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	// Text box
	p := widgets.NewParagraph()
	p.Title = "Teonet cdb"
	p.Text = "PRESS m TO QUIT DEMO"
	p.SetRect(0, 0, 50, 5)
	p.TextStyle.Fg = ui.ColorWhite
	p.BorderStyle.Fg = ui.ColorCyan
	// Update paragraph to draw
	updateParagraph := func(count int) {
		if count%2 == 0 {
			p.TextStyle.Fg = ui.ColorGreen
		} else {
			p.TextStyle.Fg = ui.ColorWhite
		}
	}

	// Table with number of requests
	table1 := widgets.NewTable()
	table1.Title = "Commands processed"
	table1.ColumnWidths = []int{5, 7, 88}
	table1.Rows = [][]string{[]string{" Cmd ", " Count ", " Description"}}
	cmds := api.Cmds()
	cmdsNumber := len(cmds)
	sprintCount := func(count uint64) string {
		return fmt.Sprintf(" %5d", count)
	}
	for i := 0; i < cmdsNumber; i++ {
		table1.Rows = append(table1.Rows, []string{
			" " + strconv.Itoa(int(cmds[i])), sprintCount(0), " " + api.Descr(cmds[i]),
		})
	}
	table1.Rows = append(table1.Rows, []string{"", fmt.Sprintf(" %5d", 0), " "})
	table1.TextStyle = ui.NewStyle(ui.ColorWhite)
	table1.BorderStyle.Fg = ui.ColorCyan
	table1.SetRect(0, 5, 102, 26)
	// Update table to draw
	updateTable := func(count int) {
		var tCount uint64
		for i := 0; i < cmdsNumber; i++ {
			count := api.Count(cmds[i])
			table1.Rows[i+1][1] = sprintCount(count)
			tCount += count
		}
		table1.Rows[cmdsNumber+1][1] = sprintCount(tCount)
	}

	barchartData := []float64{
		0, 0, 1, 1, 0, 0, 0, 1, 0, 0, 1, 1, 0, 1, 0, 1, 1, 1, 0, 0, 0, 1, 1,
		0, 0, 1, 1, 0, 0, 0, 1, 0, 0, 1, 1, 0, 1, 0, 1, 1, 1, 0, 0, 0, 1, 1, 0,
	}
	bc := widgets.NewBarChart()
	bc.Title = "Workerss"
	bc.SetRect(50, 0, 75, 5)
	bc.Labels = []string{"S0", "S1", "S2", "S3", "S4", "S5"}
	bc.BarColors[0] = ui.ColorGreen
	bc.NumStyles[0] = ui.NewStyle(ui.ColorBlack)

	draw := func(count int) {
		bc.Data = barchartData[count/2%10:]
		ui.Render(p, table1, bc)

	}

	tickerCount := 1
	draw(tickerCount)
	tickerCount++
	uiEvents := ui.PollEvents()
	ticker := time.NewTicker(time.Second).C
	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "m", "<C-c>":
				return
			}
		case <-ticker:
			updateParagraph(tickerCount)
			updateTable(tickerCount)
			draw(tickerCount)
			tickerCount++
		}
	}
}
