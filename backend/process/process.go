package process

import (
	"context"
	"fmt"
	"os"
	"strings"
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

// Process all csv files in path
func Process(ctx context.Context, path string, db *sqlx.DB) error {
	CallFrom = "Process "
	var cdrs []model.RMSCDR
	batchSize := 100              // Adjust as needed
	var TotCount int64 = 0
	var batch []model.RMSCDR// Struct representing the CDR table
	// var err error
	cdrs, err := ProcessAllCSVFiles(path, model.LogFileLiterals[strings.TrimPrefix(model.PathVars.AnalysisLogs, "/logs/")], ".csv")
	if err != nil {
		li.Logger.L.WithFields(logrus.Fields{
			"CallFrom": CallFrom,
			"err": err,
		}).Error("error while reading the csv files")
		return err
	}
	// check that files were processed
	if len(cdrs) == 0 {
		li.Logger.L.Info(CallFrom, "no files to process")
		return fmt.Errorf("no files to process")
	}
	// loop through all the slice of cdrs returned and add them to the db
	for _, cdr := range cdrs {
		// Check if the record should be skipped
		if uid, errDb := dbs.GetCDRByUID(context.Background(), db, cdr.UID);  cdr.UID == uid {
			li.Logger.L.Info(CallFrom, "record already exists in db, skipping cdr: ", cdr.UID)
			continue
		} else if errDb != nil && errDb.Error() == "table does not exist"{
			li.Logger.L.WithFields(logrus.Fields{
				"CallFrom": CallFrom,
				"err": err,
			}).Error("db empty")
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
	li.Logger.L.Printf("Total records processed: %d", TotCount)
	return nil
}

// ReadFolders reads all folders in a folder and returns it as a slice
func ReadFolders(path string) ([]string, error) {
	CallFrom = "ReadFolders "
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
