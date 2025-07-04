package process

import (
	"encoding/csv"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid" // get the uuid package
	li "github.com/pienaahj/rmsloader/backend/logwrapper"
	"github.com/pienaahj/rmsloader/backend/model"
	"github.com/sirupsen/logrus"
)

var (
	CSVPath          string         = model.ProcessLogFileLocations()["CSVPath"]
	// SourcePath       string         = model.ProcessLogFileLocations()["SourcePath"]
	// DbLogs           string         = model.ProcessLogFileLocations()["DbLogs"]
	// AnalysisLogs     string         = model.ProcessLogFileLocations()["AnalysisLogs"]
	// Loc              *time.Location // to store the ZA time location
	// ZA_LOCATION_NAME string         = "Africa/Johannesburg"
)

// Read the csv files names in directory path, does not process sub folders and retruns a combined slice of RMSCDR structs
// (absolute path eg. "/Users/hendrikpienaar/github.com/data/rms_cdrs") in docker /recordings/csv or log.PathVars.CSVPath in local.
// Requires f to be a file pointer to the database log file and FExt to be the file extension to search for including the . eg ".csv"
func ProcessAllCSVFiles(path string, f *os.File, fExt string) ([]model.RMSCDR, error) {
	CallFrom = "ReadAllCSVFiles "
	// store the current directory
	// print working dir
	originalDir, _ := os.Getwd()
	// create the return slice
	var csvDetailX []model.RMSCDR
	var idCount int

	// change working dir to filePath(/data)
	err := os.Chdir(path)
	if err != nil {
		li.Logger.L.Info(CallFrom, "Cannot change dir to ", path)
	}
	// print working dir
	dir, _ := os.Getwd()
	// loop through all the files
	err = filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			li.Logger.ErrPathWalkPreventMessage(CallFrom, path, err)
			return err
		}
		filename := info.Name()
		// skip the directory
		if info.IsDir() {
			msg := fmt.Sprintf("skipping dir without errors: %s", filename)
			li.Logger.L.Info(CallFrom, msg)
			return nil
		}
		// skip non regular files
		if filename[0] == '.' {
			msg := fmt.Sprintf("skipping non regular file without errors: %s", filename)
			li.Logger.L.Info(CallFrom, msg)
			return nil
		}
		// check if regular filename .wav/.csv file
		if !strings.Contains(filename, fExt) {
			li.Logger.L.Printf(CallFrom, "Invalid filename format: %s", filename)
			logString := fmt.Sprintf("Called from: %s, Skipping recording file entry: %s\n", CallFrom, filename)
			_, err = f.WriteString(logString)
			if err != nil {
				msg := fmt.Sprintf("Cannot write to logs %s with error: ", filename)
				li.Logger.ErrWriteFilesMessage(CallFrom, msg, err)
			}
			// skip file
			li.Logger.L.Info(CallFrom, "skipping non %s file without errors: %s", fExt, filename)
			return nil
		}
		// check validity of filename
		if len(filename) > 100 {
			// skip the file and log it
			li.Logger.L.Printf(CallFrom, "Invalid filename format, filename too long: %s", filename)
			msg := fmt.Sprintf("Called from: %s, Skipping recording file entry: %s\n", CallFrom, filename)
			_, err = f.WriteString(msg)
			if err != nil {
				li.Logger.ErrWriteFilesMessage(CallFrom, msg, err)
			}
			// skip file
			li.Logger.L.Info(CallFrom, "skipping non %s file without errors: %s", fExt, filename)
			return nil
		}

		// read the contents of the file and return the cdr details
		csvDetail, err := readCSV(filename)
		if err != nil {
			li.Logger.ErrReadFilesMessage(CallFrom, filename, err)
			return err
		}
		csvDetailX = append(csvDetailX, csvDetail...)
		// increment the id count with the number of cdr details in the file
		idCount += len(csvDetail)
		return nil
	})
	if err != nil {
		li.Logger.ErrPathWalkMessage(CallFrom, dir, err)
		return []model.RMSCDR{}, nil
	}
	li.Logger.L.WithFields(logrus.Fields{
		"Call from":             CallFrom,
		"Number of files added": len(csvDetailX),
	})
	// return to the original folder
	err = ChangePath(originalDir)
	if err != nil {
		msg := fmt.Sprintf("Cannot change dir to %s", originalDir)
		li.Logger.ErrChangePathMessage(CallFrom, msg, err)
	}
	li.Logger.L.Printf("%s: Number of CDR records added: %d", CallFrom, idCount)
	return csvDetailX, nil
}


// readCSV reads the CDR data from csv format at a path and returns a slice of RMSCDR
func readCSV(filename string) ([]model.RMSCDR, error) {
	CallFrom := "readCSV "
	msg := fmt.Sprintf("Reading CSV file: %s", filename)
	li.Logger.L.Info(CallFrom, msg)
	csvFile, err := os.Open(filename)
	if err != nil {
		li.Logger.ErrOpenFilesMessage(CallFrom, filename, err)
		return nil, err
	}
	defer csvFile.Close()
	csvLines, err := csv.NewReader(csvFile).ReadAll()
	if err != nil {
		msg := fmt.Sprintf("Cannot read CSV file: %s", csvFile.Name())
		li.Logger.ErrProcessCSVMessage(CallFrom, msg, err)
		return nil, err
	}
	var cdrData []model.RMSCDR
	
	// loop through the lines and create a struct
	for _, line := range csvLines {
		// skip the header row
		// li.Logger.L.Info("Line: ", line)
		if strings.Contains(line[0], "Direction") {
			continue
		}
		// build the field values
		// make the uid
		uid := uuid.New().String()
		// parse the timestamp
		// format 10/01/2024 10:32:47 2023-08-22 08:09:30
		timestamp, err := time.Parse("2006/01/02 15:04:05", line[1])
		if err != nil {
			li.Logger.L.Info("Trying to convert time string: ", line[1])
			li.Logger.ErrConvertToDateMessage(CallFrom, "Cannot convert time to time.Time. ", err)
			return nil, err
		}
		// flagged as a boolean
		var flagged bool
		if line[2] == "No" {
			flagged = false
		} else {
			flagged = true
		}
		// convert the durations 2 min 50 sec
		duration, err := parseDurationString(line[4])
		if err != nil {
			li.Logger.ErrConvertToIntMessage(CallFrom, "Cannot convert duration to time.Duration. ", err)
			return nil, err
		}
		// convert the duration
		talkDurationSeconds, err := model.MyDuration(duration).Value()
		if err != nil {
			li.Logger.ErrConvertToIntMessage(CallFrom, "Cannot convert duration to time.Duration. ", err)
			return nil, err
		}
		// convert the size
		sizeString := strings.Split(line[5], " ")
		size, err := strconv.Atoi(sizeString[0])
		if err != nil {
			li.Logger.ErrConvertToIntMessage(CallFrom, "Cannot convert size to int. ", err)
			return nil, err
		}
		// convert the exists
		var exists bool
		if line[6] == "Yes" {
			exists = true
		} else {
			exists = false
		}
		var localCopy bool
		if line[7] == "Yes" {
			localCopy = true
		} else {
			localCopy = false
		}
		

		// build the struct
		cdrData = append(cdrData, model.RMSCDR{
			// RecordID:     id,
			UID:          uid,
			Direction:    line[0],
			Time:         timestamp,
			Timestamp:    int64(timestamp.Unix()),
			Flagged:      flagged,
			Source:       line[3],
			Destination:  line[4],
			TalkDuration: duration,
			Duration:     talkDurationSeconds.(int64),
			Size: 		  int64(size),
			Exists: 	  exists,
			LocalCopy:    localCopy,
			Authentic:    line[9],
			SipCallID:    line[10],
			FileName:    line[11],
		})
	}
	li.Logger.L.Printf("%s: RMS CDR Data records added: %#v", CallFrom, len(cdrData))
	return cdrData, nil
}

// Convert text like "1 hour 34 min 22 sec" to time.Duration
func parseDurationString(input string) (time.Duration, error) {
	re := regexp.MustCompile(`(\d+)\s*(hour|hr|h|minute|min|m|second|sec|s)`)
	matches := re.FindAllStringSubmatch(strings.ToLower(input), -1)

	var duration time.Duration
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		valStr := match[1]
		unit := match[2]

		val, err := strconv.Atoi(valStr)
		if err != nil {
			return 0, fmt.Errorf("invalid number %q: %w", valStr, err)
		}

		switch unit {
		case "hour", "hr", "h":
			duration += time.Duration(val) * time.Hour
		case "minute", "min", "m":
			duration += time.Duration(val) * time.Minute
		case "second", "sec", "s":
			duration += time.Duration(val) * time.Second
		default:
			return 0, fmt.Errorf("unknown unit %q", unit)
		}
	}
	return duration, nil
}