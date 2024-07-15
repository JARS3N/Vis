package main

import (
    "database/sql"
    "flag"
    "fmt"
    "log"
    "os"
    "os/exec"
    "path/filepath"
    "regexp"
    "time"

    _ "github.com/mattn/go-sqlite3"
)

const (
    visWrangler = "viswrangler.exe"
    dbPath      = `G:\Spotting\Logging\CSVs\machine-vision.sqlite`
)

var directories = []string{
    `G:\Spotting\Logging\XFe24`,
    `G:\Spotting\Logging\XFe96`,
    `G:\Spotting\Logging\XFp`,
}

func main() {
    silentFlag := flag.Bool("silent", true, "Run in silent mode")
    verboseFlag := flag.Bool("verbose", false, "Run in verbose mode")
    flag.Parse()

    if *silentFlag && !*verboseFlag {
        log.SetOutput(os.Stdout) // Change this to os.Stderr if you want silent logs to go to stderr
        log.SetFlags(0)
    }

    log.Println("Starting program...")

    // Get the executable path
    exePath, err := os.Executable()
    if err != nil {
        log.Fatalf("Failed to get executable path: %v", err)
    }

    // Get the directory of the executable
    exeDir := filepath.Dir(exePath)

    // Check if viswrangler is in the PATH
    visWranglerPath, err := exec.LookPath(visWrangler)
    if err != nil {
        log.Println("viswrangler not found in PATH, checking local directory...")

        // Check if viswrangler is in the same directory as the executable
        localPath := filepath.Join(exeDir, visWrangler)
        if _, err := os.Stat(localPath); err == nil {
            visWranglerPath = localPath
        } else {
            log.Println("viswrangler not found, exiting.")
            return
        }
    }

    // Check if the SQLite file exists
    if _, err := os.Stat(dbPath); os.IsNotExist(err) {
        log.Println("SQLite database file does not exist, exiting.")
        return
    }

    // Try to open the SQLite database
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        log.Fatalf("Failed to open SQLite database: %v", err)
    }
    defer db.Close()

    // List and print all tables in the SQLite database
    tables, err := listTables(db)
    if err != nil {
        log.Fatalf("Failed to list tables in SQLite database: %v", err)
    }
    if len(tables) == 0 {
        log.Println("No tables found in the SQLite database, exiting.")
        return
    }

    //log.Println("Fetching existing Lots from the database...")
    existingLots, err := getExistingLots(db)
    if err != nil {
        log.Fatalf("Failed to get existing Lots from SQLite database: %v", err)
    }

    //log.Printf("Found %d existing Lots in the database.\n", len(existingLots))

    // Get the current directories
    currentDirs := getCurrentDirs()

    //log.Printf("Found %d current directories.\n", len(currentDirs))

    // Debugging: Print the directories
    //if *verboseFlag {
    //    log.Println("Existing Lots in the database:")
    //    for lot := range existingLots {
    //        log.Println(lot)
    //    }
    //    log.Println("Current directories:")
    //    for _, dir := range currentDirs {
    //        log.Println(dir)
    //    }
    //}

    // Filter directories to process
    dirsToProcess := filterDirectories(existingLots, currentDirs)

    //log.Printf("Found %d directories to process.\n", len(dirsToProcess))

    if len(dirsToProcess) == 0 {
        log.Println("No new directories found, exiting.")
        return
    }

    //if *verboseFlag {
    //    log.Println("Directories to process:")
    //    for _, dir := range dirsToProcess {
    //        log.Println(dir)
    //    }
    //}

    // Process each directory
    for _, dir := range dirsToProcess {
        //log.Printf("Processing directory: %s\n", dir)
        runVisWrangler(visWranglerPath, dir)
        updateDatabase(db, dir)
    }

    // Print processed lots if verbose flag is set
    if *verboseFlag {
        log.Println("Processed Lots:")
        for _, dir := range dirsToProcess {
            log.Printf("%s\n", filepath.Base(dir))
        }
    }
}

func listTables(db *sql.DB) ([]string, error) {
    rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table'")
    if err != nil {
        return nil, fmt.Errorf("query error: %w", err)
    }
    defer rows.Close()

    var tables []string
    for rows.Next() {
        var tableName string
        if err := rows.Scan(&tableName); err != nil {
            return nil, fmt.Errorf("scan error: %w", err)
        }
        tables = append(tables, tableName)
    }

    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("rows iteration error: %w", err)
    }

    return tables, nil
}

func getExistingLots(db *sql.DB) (map[string]bool, error) {
    rows, err := db.Query("SELECT Lot FROM `machine-vision`")
    if err != nil {
        return nil, fmt.Errorf("query error: %w", err)
    }
    defer rows.Close()

    existingLots := make(map[string]bool)
    for rows.Next() {
        var lot string
        if err := rows.Scan(&lot); err != nil {
            return nil, fmt.Errorf("scan error: %w", err)
        }
        existingLots[lot] = true
    }

    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("rows iteration error: %w", err)
    }

    return existingLots, nil
}

func getCurrentDirs() []string {
    var currentDirs []string
    validDirPattern := regexp.MustCompile(`^[A-Z]{1}[0-9]{5}$`)

    for _, dir := range directories {
        files, err := os.ReadDir(dir)
        if err != nil {
            log.Printf("Error reading directory %s: %v", dir, err)
            continue
        }
        for _, file := range files {
            if file.IsDir() {
                baseName := file.Name()
                if validDirPattern.MatchString(baseName) {
                    currentDirs = append(currentDirs, filepath.Join(dir, baseName))
                }
            }
        }
    }
    return currentDirs
}

func filterDirectories(existingLots map[string]bool, currentDirs []string) []string {
    var dirsToProcess []string
    for _, dir := range currentDirs {
        lot := filepath.Base(dir)
        if !existingLots[lot] {
            dirsToProcess = append(dirsToProcess, dir)
        }
    }
    return dirsToProcess
}

func runVisWrangler(visWranglerPath, dir string) {
    args := []string{dir, "-d", "-silent"}
    cmd := exec.Command(visWranglerPath, args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    //log.Printf("Running viswrangler on directory: %s\n", dir)
    err := cmd.Run()
    if err != nil {
        log.Printf("Failed to run viswrangler on directory %s: %v", dir, err)
    } else {
        //log.Printf("Successfully ran viswrangler on directory: %s\n", dir)
    }
}

func updateDatabase(db *sql.DB, dir string) {
    lot := filepath.Base(dir)
    resultCSV := fmt.Sprintf("%s_MV.csv", lot)
    resultDate := time.Now().Format("2006-01-02")

    //log.Printf("Updating database for directory: %s\n", dir)
    _, err := db.Exec("INSERT INTO `machine-vision` (dir, Lot, result_csv, result_date) VALUES (?, ?, ?, ?)",
        dir, lot, resultCSV, resultDate)
    if err != nil {
        log.Printf("Failed to update SQLite database for directory %s: %v", dir, err)
    } else {
        //log.Printf("Database updated for directory: %s\n", dir)
    }
}
