package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"os"

	li "github.com/pienaahj/rmsloader/backend/logwrapper"

	"github.com/sirupsen/logrus"
)

// the log files
var LogFiles map[string]string

// make the log file map to close them later
var LogFileLiterals map[string]*os.File
var PathVars struct {
	SourcePath      string `json:"sourcePath"`
	ExtensionsFile  string `json:"extensionsFile"`
	DestinationPath string `json:"destinationPath"`
	LogPath         string `json:"logPath"`
	AppLogsPath     string `json:"appLogsPath"`
	DbLogs          string `json:"dbLogs"`
	OddDates        string `json:"oddDates"`
	AnalysisLogs    string `json:"analysisLogs"`
	TempStorage     string `json:"tempStorage"`
}

func ProcessLogFileLocations() map[string]string {
	CallFrom := "ProcessLogFileLocations "
	// make the logFiles map
	LogFileLiterals = make(map[string]*os.File)
	// get the paths loaded from the json file
	paths, err := os.Open("./pathConfig.json")
	if err != nil {
		li.Logger.L.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to load pathConfig.json")
		os.Exit(1)
	}
	defer paths.Close()
	err = json.NewDecoder(paths).Decode(&PathVars)
	if err != nil {
		li.Logger.L.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to decode pathConfig.json variables")
		os.Exit(1)
	}

	// make the logFiles map
	logFiles := make(map[string]string)
	logFiles["SourcePath"] = PathVars.SourcePath
	logFiles["ExtensionsFile"] = PathVars.ExtensionsFile
	logFiles["DestinationPath"] = PathVars.DestinationPath
	logFiles["LogPath"] = PathVars.LogPath
	logFiles["AppLogsPath"] = PathVars.AppLogsPath
	logFiles["DbLogs"] = PathVars.DbLogs
	logFiles["OddDates"] = PathVars.OddDates
	logFiles["AnalysisLogs"] = PathVars.AnalysisLogs
	logFiles["TempStorage"] = PathVars.TempStorage

	// log the logmap creation
	li.Logger.L.Info("Log files map created")
	// check the current working folder
	// create the log folders if they don't exist
	dir, err := os.Getwd()
	if err != nil {
		li.Logger.L.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to get current working directory")
		return nil
	}
	li.Logger.L.Info("Current working directory:", dir)
	// create the folders
	err = os.Mkdir(PathVars.TempStorage, fs.ModePerm)
	if err != nil && !errors.Is(err, os.ErrExist) {
		li.Logger.L.WithFields(logrus.Fields{
			"called from": CallFrom,
			"folder":      PathVars.TempStorage,
			"err":         err,
		}).Error("Failed to create destination folder")
		return nil
	}
	// change to the log folder
	err = os.Chdir(PathVars.LogPath)
	if err != nil {
		li.Logger.L.WithFields(logrus.Fields{
			"called from": CallFrom,
			"folder":      PathVars.LogPath,
			"err":         err,
		}).Error("Failed to change to log folder")
			return nil
	}
	// create the logs folder
	err = os.Mkdir(PathVars.DestinationPath, fs.ModePerm)
	if err != nil && !errors.Is(err, os.ErrExist) {
		li.Logger.L.WithFields(logrus.Fields{
			"called from": CallFrom,
			"folder":      PathVars.DestinationPath,
			"err":         err,
		}).Error("Failed to create destination folder")
		return nil
	}

	// create the log files
	// create the files
	// move to the logs folder
	err = ChangePath(dir + PathVars.LogPath)
	CallFrom = "ProcessLogFileLocations"
	if err != nil {
		li.Logger.L.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to change directory to log folder")
		os.Exit(1)
	}
	// create the log files
	err = createLogFiles(strings.TrimPrefix(PathVars.DbLogs, "/logs/"))
	CallFrom = "ProcessLogFileLocations"
	if err != nil {
		li.Logger.ErrCreateFilesMessage(CallFrom, PathVars.DbLogs, err)
	}
	err = createLogFiles(strings.TrimPrefix(PathVars.AnalysisLogs, "/logs/"))
	CallFrom = "ProcessLogFileLocations"
	if err != nil {
		li.Logger.ErrCreateFilesMessage(CallFrom, PathVars.AnalysisLogs, err)
	}
	err = createLogFiles(strings.TrimPrefix(PathVars.OddDates, "/logs/"))
	CallFrom = "ProcessLogFileLocations"
	if err != nil {
		li.Logger.ErrCreateFilesMessage(CallFrom, PathVars.OddDates, err)
	}
	return logFiles
}

// create all the log files if they don't exist
func createLogFiles(filename string) error {
	CallFrom := "createLogFiles "
	var file *os.File
	// Create the log file if it doesn't exist
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		li.Logger.L.Printf("%s: %s log file does not exist, creating it", CallFrom, filename)
		file, err = os.Create(filename)
		if err != nil {
			if os.IsPermission(err) {
				li.Logger.ErrWriteFilesMessage(CallFrom, "Permission denied creating log file", err)
				return err
			} else {
				msg := fmt.Sprintf("Failed to create log file, %s", filename)
				li.Logger.ErrWriteFilesMessage(CallFrom, msg, err)
			}
			return err
			}
		}
	
	// li.Logger.InfoLogFileCreateMessage(CallFrom, filename)
	// defer file.Close() moved to Close()
	// if created remember the file
	LogFileLiterals[filename] = file
	return nil
}
// Close file resources
func Close() {
	// close the log files
	CloseFile(LogFileLiterals["SourcePath"])
	CloseFile(LogFileLiterals["SourcePathCDR"])
	CloseFile(LogFileLiterals["ExtensionsFile"])
	CloseFile(LogFileLiterals["DestinationPath"])
}

// close the logFile
func CloseFile(file *os.File) {
	err := file.Close()
	if err != nil {
		li.Logger.L.Printf("Error closing file: %s with error:%v", file.Name(), err)
	}
}
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