package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

const csvDirectory = "G:\\Spotting\\Logging\\CSVs"
const dbFileName = "machine-vision.sqlite"

func main() {
	dbFilePath := filepath.Join(csvDirectory, dbFileName)
	// Open the SQLite database
	db, err := sql.Open("sqlite3", dbFilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Query the database to get the Lot values
	rows, err := db.Query("SELECT Lot FROM `machine-vision`")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	// Create a slice to hold the Lot values
	var lots []string
	for rows.Next() {
		var lot string
		if err := rows.Scan(&lot); err != nil {
			log.Fatal(err)
		}
		lots = append(lots, lot)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	// Create a new Fyne application
	myApp := app.New()
	myWindow := myApp.NewWindow("VisViewer")

	// Create a label to display the selected result_csv
	resultCSVLabel := widget.NewLabel("Selected CSV: ")

	// Create an entry widget for the searchable drop-down
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Type to search...")

	// Create a list widget to display filtered Lot values
	filteredLots := lots
	selectedLotIndex := -1
	list := widget.NewList(
		func() int {
			return len(filteredLots)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(filteredLots[i])
		},
	)

	// Filter the list based on the entry's text
	entry.OnChanged = func(s string) {
		filteredLots = []string{}
		for _, lot := range lots {
			if len(s) == 0 || (len(lot) >= len(s) && lot[:len(s)] == s) {
				filteredLots = append(filteredLots, lot)
			}
		}
		list.Refresh()
	}

	// Create a dropdown for the CSV headers
	headerDropdown := widget.NewSelect([]string{}, func(value string) {
		fmt.Println("Selected header:", value)
	})

	// Create a plot widget placeholder
	plotImage := canvas.NewImageFromFile("")
	plotImage.FillMode = canvas.ImageFillContain
	plotImage.SetMinSize(fyne.NewSize(600, 400))

	// Handle list item selection
	list.OnSelected = func(id widget.ListItemID) {
		selectedLotIndex = id
		selectedLot := filteredLots[id]
		// Query the database to get the result_csv value for the selected Lot
		var resultCSV string
		err := db.QueryRow("SELECT result_csv FROM `machine-vision` WHERE Lot = ?", selectedLot).Scan(&resultCSV)
		if err != nil {
			log.Fatal(err)
		}
		// Update the label with the selected result_csv
		resultCSVLabel.SetText(fmt.Sprintf("Selected CSV: %s", resultCSV))

		// Load the CSV file and update the header dropdown
		csvFilePath := filepath.Join(csvDirectory, resultCSV)
		headers := getCSVHeaders(csvFilePath)
		headerDropdown.Options = headers
		headerDropdown.Refresh()

		// Print the first two rows of the CSV file for debugging
		printFirstTwoRows(csvFilePath)
	}

	// Handle header selection to update the plot
	headerDropdown.OnChanged = func(header string) {
		if selectedLotIndex < 0 || selectedLotIndex >= len(filteredLots) {
			return
		}
		// Query the database to get the result_csv value for the selected Lot
		var resultCSV string
		err := db.QueryRow("SELECT result_csv FROM `machine-vision` WHERE Lot = ?", filteredLots[selectedLotIndex]).Scan(&resultCSV)
		if err != nil {
			log.Fatal(err)
		}
		// Load the CSV file and update the plot
		csvFilePath := filepath.Join(csvDirectory, resultCSV)
		selectedLot := filteredLots[selectedLotIndex]
		plotPath := plotAverages(csvFilePath, header, selectedLot)
		img, err := os.Open(plotPath)
		if err != nil {
			log.Fatal(err)
		}
		defer img.Close()
		imgSrc, _, err := image.Decode(img)
		if err != nil {
			log.Fatal(err)
		}
		plotImage.Image = imgSrc
		plotImage.Refresh()
	}

	// Set up the window content
	split := container.NewHSplit(
		container.NewVBox(
			widget.NewLabel("Select Lot:"),
			entry,
			list,
			resultCSVLabel,
			widget.NewLabel("Select CSV Header:"),
			headerDropdown,
		),
		plotImage,
	)
	split.SetOffset(0.3)

	myWindow.SetContent(split)

	// Show and run the application
	myWindow.Resize(fyne.NewSize(800, 600))
	myWindow.ShowAndRun()
}

func getCSVHeaders(filePath string) []string {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		log.Fatal(err)
	}

	// Exclude "Lot", "sn", and "Well" from the headers
	var filteredHeaders []string
	for _, header := range headers {
		if header != "Lot" && header != "sn" && header != "Well" {
			filteredHeaders = append(filteredHeaders, header)
		}
	}

	return filteredHeaders
}

func printFirstTwoRows(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	rows, err := reader.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	for i, row := range rows {
		if i >= 2 {
			break
		}
		fmt.Println(row)
	}
}

func plotAverages(filePath, header, lot string) string {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	rows, err := reader.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	// Find the indices of the "sn" and selected header columns
	headers := rows[0]
	snIndex := -1
	headerIndex := -1
	for i, h := range headers {
		if h == "sn" {
			snIndex = i
		} else if h == header {
			headerIndex = i
		}
	}
	if snIndex == -1 || headerIndex == -1 {
		log.Fatalf("Failed to find indices for sn or %s", header)
	}

	// Aggregate data by sn
	data := map[string][]float64{}
	for _, row := range rows[1:] {
		sn := row[snIndex]
		value, err := strconv.ParseFloat(row[headerIndex], 64)
		if err != nil {
			continue
		}
		data[sn] = append(data[sn], value)
	}

	// Calculate averages
	type Point struct {
		sn  float64
		avg float64
	}
	points := []Point{}
	for snStr, values := range data {
		sn, err := strconv.ParseFloat(snStr, 64)
		if err != nil {
			continue
		}
		sum := 0.0
		for _, value := range values {
			sum += value
		}
		avg := sum / float64(len(values))
		points = append(points, Point{sn, avg})
	}

	// Sort points by sn
	sort.Slice(points, func(i, j int) bool {
		return points[i].sn < points[j].sn
	})

	// Create the plot
	p := plot.New()
	p.Title.Text = fmt.Sprintf("Plot of Average %s for Lot %s", header, lot)
	p.X.Label.Text = "sn"
	p.Y.Label.Text = header

	// Create a scatter plot with default glyphs
	pts := make(plotter.XYs, len(points))
	for i, point := range points {
		pts[i].X = point.sn
		pts[i].Y = point.avg
	}
	scatter, err := plotter.NewScatter(pts)
	if err != nil {
		log.Fatal(err)
	}
	scatter.GlyphStyle.Radius = vg.Points(3)
	scatter.GlyphStyle.Color = color.RGBA{0, 0, 0, 102} // Semi-transparent black with alpha 0.4

	p.Add(scatter)

	// Save the plot to a PNG file with the Lot appended to the file name
	imgPath := filepath.Join(csvDirectory, fmt.Sprintf("plot_%s_Average_%s.png", lot, header))
	if err := p.Save(6*vg.Inch, 4*vg.Inch, imgPath); err != nil {
		log.Fatal(err)
	}

	return imgPath
}
