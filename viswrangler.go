package main

import (
    "encoding/csv"
    "encoding/xml"
    "flag"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "regexp"
    "sort"
    "strings"
    "sync"
    "time"
)

const (
    defaultDir   = `G:\Spotting\Logging\CSVs`
    currentVersion = "2.1"
    isoDate       = "2024-07-13"
)

type Config struct {
    SearchDir  string
    OutputDir  string
    SilentFlag bool
}

type InspectionDetailsItem struct {
    Name    string `xml:"Name"`
    Details string `xml:"Details"`
}

type List struct {
    Items []InspectionDetailsItem `xml:"InspectionDetailsItem"`
}

type Root struct {
    List List `xml:"List"`
}

func ParseFlags() Config {
    helpFlag := flag.Bool("help", false, "Show usage information")
    silentFlag := flag.Bool("silent", false, "Suppress output")
    flag.Parse()

    args := flag.Args()
    if len(args) < 1 || *helpFlag {
        fmt.Printf("Vision Wrangler | version %s | %s\n", currentVersion, isoDate)
        fmt.Println("Usage: viswrangler [origin_directory] [output_option] [-silent]")
        fmt.Println("Options:")
        fmt.Println("  -o       Output directory same as origin directory")
        fmt.Println("  -c       Output directory as current directory")
        fmt.Println("  -d       Output directory as default directory (G:\\Spotting\\Logging\\CSV)")
        fmt.Println("  <path>   Specify a specific output directory")
        fmt.Println("  -silent  Suppress output")
        fmt.Println("  -help    Show usage information.")
        fmt.Println("\nNote: Paths must be enclosed in double quotes.")
        os.Exit(0)
    }

    var searchDir, outputDir string

    workingDir, err := os.Getwd()
    if err != nil {
        log.Fatalf("Failed to get working directory: %v", err)
    }

    searchDir = args[0]

    if len(args) > 1 {
        secondArg := args[1]
        switch secondArg {
        case "-o":
            outputDir = searchDir
        case "-c":
            outputDir = workingDir
        case "-d":
            outputDir = defaultDir
        default:
            outputDir = secondArg
        }
    } else {
        outputDir = searchDir
    }

    err = os.MkdirAll(outputDir, os.ModePerm)
    if err != nil {
        log.Fatalf("Failed to create output directory: %v", err)
    }

    return Config{
        SearchDir:  searchDir,
        OutputDir:  outputDir,
        SilentFlag: *silentFlag,
    }
}

func GetDetailsXMLFiles(root string) ([]string, error) {
    var files []string

    err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if !info.IsDir() && filepath.Base(path) == "details.xml" {
            files = append(files, path)
        }
        return nil
    })

    return files, err
}

func ExtractDetailsFromXML(fileName string) (string, string, error) {
    byteValue, err := os.ReadFile(fileName)
    if err != nil {
        return "", "", err
    }

    var root Root
    if err := xml.Unmarshal(byteValue, &root); err != nil {
        return "", "", err
    }

    var barcode, results string
    for _, item := range root.List.Items {
        if item.Name == "Bar Code" {
            barcode = item.Details
        } else if item.Name == "Results" {
            results = item.Details
        }
    }

    return barcode, results, nil
}

func ProcessRow(rowText string) map[string]string {
    dataframe := make(map[string]string)

    wellRe := regexp.MustCompile(`<center><b>([A-Za-z]+\d+)</b></center>`)
    wellMatch := wellRe.FindStringSubmatch(rowText)
    if len(wellMatch) > 1 {
        well := wellMatch[1]
        letterPart := well[:1]
        numberPart := well[1:]
        if len(numberPart) == 1 {
            numberPart = "0" + numberPart
        }
        dataframe["Well"] = letterPart + numberPart
    }

    opticalSectionRe := regexp.MustCompile(`Optical Window(.*?)Spot 1`)
    opticalSectionMatch := opticalSectionRe.FindStringSubmatch(rowText)
    if len(opticalSectionMatch) > 1 {
        extractKeyValuePairs(opticalSectionMatch[1], "Optical_", dataframe)
    }

    spotSectionRe := regexp.MustCompile(`Spot 1(.*?)Ports`)
    spotSectionMatch := spotSectionRe.FindStringSubmatch(rowText)
    if len(spotSectionMatch) > 1 {
        extractKeyValuePairs(spotSectionMatch[1], "Spot_", dataframe)
    }

    portsSectionRe := regexp.MustCompile(`Ports(.*)`)
    portsSectionMatch := portsSectionRe.FindStringSubmatch(rowText)
    if len(portsSectionMatch) > 1 {
        extractPortKeyValuePairs(portsSectionMatch[1], dataframe)
    }

    return dataframe
}

func extractKeyValuePairs(text, prefix string, dataframe map[string]string) {
    cleanText := regexp.MustCompile(`<[^>]*>`).ReplaceAllString(text, "")
    re := regexp.MustCompile(`([^:]+):\s*([\d\.\-]+)`)
    matches := re.FindAllStringSubmatch(cleanText, -1)

    replaceRe := regexp.MustCompile(`[\s-]`)
    for _, match := range matches {
        if len(match) > 2 {
            key := strings.TrimSpace(match[1])
            key = replaceRe.ReplaceAllString(key, "_")
            value := strings.TrimSpace(match[2])
            dataframe[prefix+key] = value
        }
    }
}

func extractPortKeyValuePairs(text string, dataframe map[string]string) {
    cleanText := regexp.MustCompile(`<[^>]*>`).ReplaceAllString(text, "")
    re := regexp.MustCompile(`Port (\d+), diameter:\s*([\d\.\-]+)`)
    matches := re.FindAllStringSubmatch(cleanText, -1)

    for _, match := range matches {
        if len(match) > 2 {
            key := fmt.Sprintf("Port_%s_diameter", match[1])
            value := strings.TrimSpace(match[2])
            dataframe[key] = value
        }
    }
}

func ProcessRows(results string) []map[string]string {
    rows := strings.Split(results, "<td>")
    var processedRows []map[string]string
    for _, row := range rows {
        if strings.TrimSpace(row) != "" {
            processedRow := ProcessRow(row)
            processedRows = append(processedRows, processedRow)
        }
    }
    return processedRows
}

func WriteCSV(filePath string, data []map[string]string) error {
    file, err := os.Create(filePath)
    if err != nil {
        return err
    }
    defer file.Close()

    writer := csv.NewWriter(file)
    defer writer.Flush()

    if len(data) == 0 {
        return nil
    }

    var headers []string
    orderedColumns := []string{"Well"}
    for _, prefix := range []string{"Optical_", "Spot_", "Port_"} {
        for k := range data[0] {
            if strings.HasPrefix(k, prefix) {
                orderedColumns = append(orderedColumns, k)
            }
        }
    }
    headers = append(headers, orderedColumns...)
    for k := range data[0] {
        if k != "Well" && !strings.HasPrefix(k, "Optical_") && !strings.HasPrefix(k, "Spot_") && !strings.HasPrefix(k, "Port_") {
            headers = append(headers, k)
        }
    }

    writer.Write(headers)

    for _, row := range data {
        var record []string
        for _, header := range headers {
            record = append(record, row[header])
        }
        writer.Write(record)
    }

    return nil
}

func BindRows(tables [][]map[string]string) []map[string]string {
    var allRows []map[string]string
    for _, table := range tables {
        allRows = append(allRows, table...)
    }

    sort.SliceStable(allRows, func(i, j int) bool {
        wellI := allRows[i]["Well"]
        wellJ := allRows[j]["Well"]
        return wellI < wellJ
    })

    return allRows
}

func ExtractBarcodeDetails(text string) map[string]string {
    barcodeData := make(map[string]string)

    if len(text) >= 11 {
        barcodeData["type"] = string(text[0])
        barcodeData["sn"] = strings.TrimLeft(text[1:6], "0")
        barcodeData["Lot"] = string(text[0]) + text[6:11]
    }

    return barcodeData
}

func main() {
    config := ParseFlags()

    if !config.SilentFlag {
        fmt.Printf("Vision Wrangler | version %s | %s\n", currentVersion, isoDate)
    }

    start := time.Now()

    detailsFiles, err := GetDetailsXMLFiles(config.SearchDir)
    if err != nil {
        log.Fatalf("Failed to get details.xml files: %v", err)
    }

    if !config.SilentFlag {
        fmt.Printf("Processing %d files...\n", len(detailsFiles))
    }

    var allResults [][]map[string]string
    var allBarcodeDetails []map[string]string
    var wg sync.WaitGroup
    var mu sync.Mutex

    for _, file := range detailsFiles {
        wg.Add(1)
        go func(file string) {
            defer wg.Done()
            barcode, results, err := ExtractDetailsFromXML(file)
            if err != nil {
                log.Printf("Error extracting details from %s: %v", file, err)
                return
            }

            processedRows := ProcessRows(results)
            barcodeDetails := ExtractBarcodeDetails(barcode)

            mu.Lock()
            allResults = append(allResults, processedRows)
            for i := 0; i < len(processedRows); i++ {
                allBarcodeDetails = append(allBarcodeDetails, barcodeDetails)
            }
            mu.Unlock()
        }(file)
    }
    wg.Wait()

    combinedResults := BindRows(allResults)

    for i := range combinedResults {
        for k, v := range allBarcodeDetails[i] {
            combinedResults[i][k] = v
        }
    }

    if len(combinedResults) > 0 {
        lot := combinedResults[0]["Lot"]
        outputFilePath := filepath.Join(config.OutputDir, fmt.Sprintf("%s_MV.csv", lot))
        if err := WriteCSV(outputFilePath, combinedResults); err != nil {
            log.Fatalf("Error writing combined CSV file: %v", err)
        }

        if !config.SilentFlag {
            fmt.Printf("Combined CSV file created successfully at %s\n", outputFilePath)
            fmt.Printf("Processed %d files in %v\n", len(detailsFiles), time.Since(start))
        }
    }
}
