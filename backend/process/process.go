package process

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	dbs "github.com/pienaahj/rmsloader/backend/db"
	li "github.com/pienaahj/rmsloader/backend/logwrapper"
	"github.com/pienaahj/rmsloader/backend/model"
	"github.com/sirupsen/logrus"
)

// global variables
var (
	Loc              *time.Location // to store the ZA time location
	ZA_LOCATION_NAME string  = "Africa/Johannesburg"
	CallFrom         string    // Declare CallFrom variable for logging
	SubFolder        string    // the folder representing the sub folder twhere recordings is stored inside the recordings repo folder
	StartFolder      int     = 0 // the folder representing the extension from where to start the processing
)

// the paths
var (
	SourcePath      string = model.ProcessLogFileLocations()["SourcePath"]
	ExtensionsFile  string = model.ProcessLogFileLocations()["ExtensionsFile"]
	DestinationPath string = model.ProcessLogFileLocations()["DestinationPath"]
	LogPath         string = model.ProcessLogFileLocations()["LogPath"]
	AppLogsPath     string = model.ProcessLogFileLocations()["AppLogsPath"]
	DbLogs          string = model.ProcessLogFileLocations()["DbLogs"]
	OddDates        string = model.ProcessLogFileLocations()["OddDates"]
	AnalysisLogs    string = model.ProcessLogFileLocations()["AnalysisLogs"]
	TempStorage     string = model.ProcessLogFileLocations()["TempStorage"]
)

// get the location for data calculations
func GetDataLocation() *time.Location {
	Loc, err := time.LoadLocation(ZA_LOCATION_NAME)
	if err != nil {
		fmt.Println("could not load location:", err)
		li.Logger.ErrLocationCreateMessage("GetDataLocation", "could not load location", err)
		return nil
	}
	return Loc
}

// Process
func Process(ctx context.Context, path string, r *model.RMSCDR, db *sqlx.DB) error {
	CallFrom = "Call from Process "
	// Populate the LogFiles map
	// model.ProcessLogFileLocations()
	// open the database log file
	// fmt.Println("file to open or create dbLogs:", DbLogs)
	f, err := os.OpenFile(DbLogs, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("%s, could not create database log file: %v", CallFrom, err)
	}
	// the file will be closed by a close function in main
	// get the folder contents
	folders, err := ReadFolders(path)
	if err != nil {
		return fmt.Errorf("could not read the source folders: %v", err)
	}
	li.Logger.L.Println("all source folders read...")
	// loop through the folders
	var filenames []string
	var cdrs []model.RMSCDR
	batchSize := 100              // Adjust as needed
	var TotCount int64 = 0
	var batch []model.RMSCDR// Struct representing the CDR table
	for _, folder := range folders {
		// convert the folder to an int
		folderNumber, err := strconv.Atoi(folder)
		if err != nil {
			return fmt.Errorf("could not convert folder to int: %v", err)
		}
		// don't process any folder with a smaller extension number than the start folder
		if folderNumber < StartFolder {
			continue
		}
		// change the active folder to the source
		if _, err := os.Stat(SourcePath); os.IsNotExist(err) {
			return fmt.Errorf("could not find the source folder: %v", err)
		}
		err = ChangePath(SourcePath)
		if err != nil {
			return fmt.Errorf("could not change to the source folder: %v", err)
		}
		// store the active folder to SubFolder global variable
		SubFolder = folder
		// get the cdrs in that folder
		cdrs, err = ProcessAllCSVFiles(folder, f, "csv",)
		if err != nil {
			return fmt.Errorf("could not read the files: %v", err)
		}
		li.Logger.L.Printf("Processing folder: %s with number of files %d", folder, len(filenames))
		// loop through all the slice of cdrs and add them to the db
		for _, cdr := range cdrs {
			// Check if the record should be skipped
			if uid, _ := dbs.GetCDRByUID(context.Background(), db, cdr.UID);  cdr.UID == uid {
				li.Logger.L.Info(CallFrom, "record already exists in db, skipping cdr: ", cdr.UID)
				continue
			}

			batch = append(batch, cdr)

			// Process batch if it reaches the batchSize limit
			if len(batch) >= batchSize {
				count, err := dbs.InsertCDRsBatch(ctx, db, batch)
				if err != nil {
					return err
				}
				TotCount += count
				batch = batch[:0] // Reset batch
			}
		}
		// Insert any remaining records
		if len(batch) > 0 {
			count, err := dbs.InsertCDRsBatch(ctx, db, batch)
			if err != nil {
				return err
			}
			TotCount += count
		}

	}
	li.Logger.L.Printf("Total records processed: %d", TotCount)
	return nil
}

// ReadFolders reads all folders in a folder and returns it as a slice
func ReadFolders(path string) ([]string, error) {
	CallFrom = "Call from ReadFolders"
	// get the directory contents
	var folders []string
	files, err := os.ReadDir(path)
	if err != nil {
		li.Logger.L.WithFields(logrus.Fields{
			"source":  path,
			"file":    files,
			"appname": CallFrom,
		})
		li.Logger.ErrReadFilesMessage(CallFrom, path, err)
		return []string{}, fmt.Errorf("%s: Could not read folder: %v", CallFrom, err)
	}
	for _, file := range files {
		if !file.Type().IsDir() {
			continue
		}
		// add the folder names to the folders slice
		folders = append(folders, file.Name())
	}
	li.Logger.L.Printf("Number of folders found: %d", len(folders))
	li.Logger.L.Printf("Folders to return: %v", folders)
	return folders, nil
}

// ConvertToDate parses the incoming date time format outputs malformed names to odd_dates
func ConvertToDate(myDateString, filename string, path string) (time.Time, error) {
	CallFrom = "Call from ConvertToDate"
	var dateString string
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return time.Time{}, err
	}
	defer f.Close()
	// determine location
	loc := GetDataLocation()
	// check the length of the  date string
	dateLenght := len(myDateString)
	if dateLenght < 19 {
		li.Logger.L.Printf("Malformed date: %s for filename: %s", myDateString, filename)
		return time.Time{}, fmt.Errorf("cannot convert short date string: %s", myDateString)
	}
	// check date format
	layoutZA := "2006-01-02 15:04:05"
	myDate, err := time.ParseInLocation(layoutZA, dateString, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("cannot convert date string: %s", myDateString)
	}
	return myDate, nil
}

// // ChangePath return previous directory
// func ChangePath(path string) {
// 	CallFrom = "Call from ChangePath"
// 	err := os.Chdir(path) // cd to the directory
// 	if err != nil {
// 		li.Logger.L.WithFields(logrus.Fields{
// 			"testing": "Path change failed",
// 			"appname": CallFrom,
// 		})
// 		li.Logger.ErrChangePathMessage(CallFrom, path, err)
// 	}
// }

// ChangePath return path directory
func ChangePath(path string) error {
	CallFrom := "ChangePath "
	err := os.Chdir(path) // cd path
	if err != nil {
		li.Logger.L.WithFields(logrus.Fields{
			"appname": CallFrom,
			"testing": "Path change failed",
		})
		li.Logger.ErrChangePathMessage(CallFrom, path, err)
		return err
	}
	return nil
}
// IsDirEmpty checks if a directory is empty
func IsDirEmpty(path string) bool {
	// get the directory contents
	files, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	if len(files) == 0 {
		return true
	}
	return false
}
