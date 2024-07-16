// Version 2.2

package main

import (
    "encoding/csv"
    "encoding/xml"
    "flag"
    "fmt"
    "io"
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
    currentVersion = "2.2"
    isoDate       = "2024-07-16"
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
    // Define command-line flags
    helpFlag := flag.Bool("help", false, "Show usage information")
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
    silentFlag := false

    for _, arg := range args {
        if arg == "-silent" {
            silentFlag = true
        }
    }

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

    if _, err := os.Stat(searchDir); os.IsNotExist(err) {
        log.Fatalf("Search directory '%s' does not exist.", searchDir)
    }

    err = os.MkdirAll(outputDir, os.ModePerm)
    if err != nil {
        log.Fatalf("Failed to create output directory: %v", err)
    }

    return Config{
        SearchDir:  searchDir,
        OutputDir:  outputDir,
        SilentFlag: silentFlag,
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
    file, err := os.Open(fileName)
    if err != nil {
        return "", "", err
    }
    defer file.Close()

    decoder := xml.NewDecoder(file)
    var barcode, results string
    for {
        token, err := decoder.Token()
        if err == io.EOF {
            break
        } else if err != nil {
            return "", "", err
        }

        switch se := token.(type) {
        case xml.StartElement:
            if se.Name.Local == "InspectionDetailsItem" {
                var item InspectionDetailsItem
                if err := decoder.DecodeElement(&item, &se); err != nil {
                    return "", "", err
                }
                if item.Name == "Bar Code" {
                    barcode = item.Details
                } else if item.Name == "Results" {
                    results = item.Details
                }
            }
        }
    }

    return barcode, results, nil
}

func ExtractTDsFromResults(results string) []string {
    var tds []string
    tdRe := regexp.MustCompile(`<td[^>]*>(.*?)</td>`)
    tdMatches := tdRe.FindAllStringSubmatch(results, -1)
    for _, match := range tdMatches {
        if len(match) > 1 {
            tds = append(tds, match[1])
        }
    }
    return tds
}

func ProcessRow(rowText string) map[string]string {
    dataframe := make(map[string]string)

    // Extract Well
    wellRe := regexp.MustCompile(`<center><b>([A-Za-z]+\d+)</b></center>`)
    wellMatch := wellRe.FindStringSubmatch(rowText)
    if len(wellMatch) > 1 {
        dataframe["Well"] = ZeroPadWell(wellMatch[1])
    }

    // Extract key-value pairs between "Optical Window" and "Spot 1"
    opticalSectionRe := regexp.MustCompile(`Optical Window(.*?)Spot 1`)
    opticalSectionMatch := opticalSectionRe.FindStringSubmatch(rowText)
    if len(opticalSectionMatch) > 1 {
        extractKeyValuePairs(opticalSectionMatch[1], "Optical_", dataframe)
    }

    // Extract key-value pairs between "Spot 1" and "Ports"
    spotSectionRe := regexp.MustCompile(`Spot 1(.*?)Ports`)
    spotSectionMatch := spotSectionRe.FindStringSubmatch(rowText)
    if len(spotSectionMatch) > 1 {
        extractKeyValuePairs(spotSectionMatch[1], "Spot_", dataframe)
    }

    // Extract key-value pairs from "Ports" to the end
    portsSectionRe := regexp.MustCompile(`Ports(.*)`)
    portsSectionMatch := portsSectionRe.FindStringSubmatch(rowText)
    if len(portsSectionMatch) > 1 {
        extractPortKeyValuePairs(portsSectionMatch[1], dataframe)
    }

    return dataframe
}

func ZeroPadWell(well string) string {
    wellRe := regexp.MustCompile(`([A-Za-z]+)(\d+)`)
    wellMatch := wellRe.FindStringSubmatch(well)
    if len(wellMatch) > 2 {
        letter := wellMatch[1]
        number := wellMatch[2]
        if len(number) == 1 {
            number = "0" + number
        }
        return letter + number
    }
    return well
}

func extractKeyValuePairs(text, prefix string, dataframe map[string]string) {
    // Remove HTML tags from the text
    cleanText := regexp.MustCompile(`<[^>]*>`).ReplaceAllString(text, "")

    // Extract key-value pairs
    re := regexp.MustCompile(`([^:]+):\s*([\d\.\-]+)`)
    matches := re.FindAllStringSubmatch(cleanText, -1)

    for _, match := range matches {
        if len(match) > 2 {
            key := strings.TrimSpace(match[1])
            key = strings.ReplaceAll(key, " ", "_")
            key = strings.ReplaceAll(key, "-", "_")
            value := strings.TrimSpace(match[2])
            dataframe[prefix+key] = value
        }
    }
}

func extractPortKeyValuePairs(text string, dataframe map[string]string) {
    // Remove HTML tags from the text
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

func bind_rows(tables []map[string]string) []map[string]string {
    combinedTable := make([]map[string]string, len(tables))

    allKeys := make(map[string]struct{})
    for _, table := range tables {
        for key := range table {
            allKeys[key] = struct{}{}
        }
    }

    var keys []string
    for key := range allKeys {
        keys = append(keys, key)
    }
    sort.Strings(keys)

    for i, table := range tables {
        combinedRow := make(map[string]string)
        for _, key := range keys {
            if value, ok := table[key]; ok {
                combinedRow[key] = value
            } else {
                combinedRow[key] = ""
            }
        }
        combinedTable[i] = combinedRow
    }

    return combinedTable
}

func joinTables(barcodeTable map[string]string, combinedTable []map[string]string) []map[string]string {
    finalTable := make([]map[string]string, len(combinedTable))

    for i, row := range combinedTable {
        combinedRow := make(map[string]string)
        for k, v := range barcodeTable {
            combinedRow[k] = v
        }
        for k, v := range row {
            combinedRow[k] = v
        }
        finalTable[i] = combinedRow
    }

    return finalTable
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

    // Write headers
    var headers []string
    orderedHeaders := []string{}
    opticalHeaders := []string{}
    spotHeaders := []string{}
    portHeaders := []string{}
    wellHeader := "Well"

    for k := range data[0] {
        if strings.HasPrefix(k, "Optical_") {
            opticalHeaders = append(opticalHeaders, k)
        } else if strings.HasPrefix(k, "Spot_") {
            spotHeaders = append(spotHeaders, k)
        } else if strings.HasPrefix(k, "Port_") {
            portHeaders = append(portHeaders, k)
        } else if k != "Well" {
            headers = append(headers, k)
        }
    }

    sort.Strings(headers)
    sort.Strings(opticalHeaders)
    sort.Strings(spotHeaders)
    sort.Strings(portHeaders)

    orderedHeaders = append(orderedHeaders, headers...)
    orderedHeaders = append(orderedHeaders, opticalHeaders...)
    orderedHeaders = append(orderedHeaders, spotHeaders...)
    orderedHeaders = append(orderedHeaders, portHeaders...)
    orderedHeaders = append(orderedHeaders, wellHeader)

    writer.Write(orderedHeaders)

    // Write data
    for _, row := range data {
        var record []string
        for _, header := range orderedHeaders {
            record = append(record, row[header])
        }
        writer.Write(record)
    }

    return nil
}

func ExtractBarcodeDetails(text string) map[string]string {
    barcodeData := make(map[string]string)

    if len(text) >= 11 {
        barcodeData["Type"] = string(text[0])
        barcodeData["SN"] = strings.TrimLeft(text[1:6], "0")
        barcodeData["Lot"] = string(text[0]) + text[6:11]
    }

    return barcodeData
}

func main() {
    start := time.Now() // Start timing

    config := ParseFlags()

    // Get all details.xml files in the search directory
    detailsFiles, err := GetDetailsXMLFiles(config.SearchDir)
    if err != nil {
        log.Fatalf("Failed to get details.xml files: %v", err)
    }

    if !config.SilentFlag {
        fmt.Printf("Processing %d files...\n", len(detailsFiles))
    }

    allCombinedTables := make([]map[string]string, 0)
    resultChan := make(chan []map[string]string)
    var wg sync.WaitGroup

    processFile := func(file string) {
        defer wg.Done()

        barcode, results, err := ExtractDetailsFromXML(file)
        if err != nil {
            log.Printf("Error extracting details from %s: %v", file, err)
            return
        }

        // Process the barcode into a table
        barcodeTable := ExtractBarcodeDetails(barcode)

        // Extract each <td> section from the results
        resultsTD := ExtractTDsFromResults(results)

        // Process each <td> section into individual rows
        tables := make([]map[string]string, len(resultsTD))
        for i, td := range resultsTD {
            tables[i] = ProcessRow(td)
        }

        // Combine all rows into one table
        combinedTable := bind_rows(tables)

        // Sort the combined table by the Well column
        sort.Slice(combinedTable, func(i, j int) bool {
            return combinedTable[i]["Well"] < combinedTable[j]["Well"]
        })

        // Join barcode table with the results table
        finalTable := joinTables(barcodeTable, combinedTable)

        // Send the final table to the result channel
        resultChan <- finalTable
    }

    for _, file := range detailsFiles {
        wg.Add(1)
        go processFile(file)
    }

    // Close the result channel when all files are processed
    go func() {
        wg.Wait()
        close(resultChan)
    }()

    // Collect all results from the result channel
    for result := range resultChan {
        allCombinedTables = append(allCombinedTables, result...)
    }

    // If there are any combined tables, save them to CSV
    if len(allCombinedTables) > 0 {
        // Get the unique Lot values
        lotValues := make(map[string]struct{})
        for _, row := range allCombinedTables {
            lotValues[row["Lot"]] = struct{}{}
        }

        // Save each Lot's data to a separate CSV file
        for lot := range lotValues {
            lotData := []map[string]string{}
            for _, row := range allCombinedTables {
                if row["Lot"] == lot {
                    lotData = append(lotData, row)
                }
            }

            outputFilePath := filepath.Join(config.OutputDir, fmt.Sprintf("%s_MV.csv", lot))
            err = WriteCSV(outputFilePath, lotData)
            if err != nil {
                log.Fatalf("Error writing combined CSV file: %v", err)
            }

            if !config.SilentFlag {
                fmt.Printf("Combined CSV file created successfully at %s\n", outputFilePath)
            }
        }
    }

    end := time.Now() // End timing
    duration := end.Sub(start)

    if !config.SilentFlag {
        fmt.Printf("Processed %d files in %v\n", len(detailsFiles), duration)
    }
}
