package logwrapper

import (
	"errors"
	"io/fs"
	"log"
	"os"

	"sync"

	"github.com/sirupsen/logrus"
)

// Event stores messages to log later, from our standard interface
type Event struct {
	id      int
	message string
}

// StandardLogger enforces specific log message formats
type StandardLogger struct {
	L    *logrus.Logger
	File *os.File
	Mu   sync.Mutex
}

// Logger is the central logger for all packages used
var Logger *StandardLogger

// NewLogger initializes the standard logger (constructor with cutomization to json output writes to file)
func NewLogger(logFile *os.File) *StandardLogger {
	var (
		baseLogger = logrus.New()
	)

	standardLogger := &StandardLogger{
		L:    baseLogger,
		File: logFile,
	}
	standardLogger.L.Formatter = &logrus.JSONFormatter{} // setup writing format
	standardLogger.L.SetOutput(logFile)                  // Ensure logs go to the file

	return standardLogger
}

// Add a closer function
func CloseLogger() {
	if Logger != nil && Logger.File != nil {
		Logger.L.Info("Closing logger")
		Logger.File.Close()
	}
}

// protect the file from concurrent writes
func (lm *StandardLogger) Sync() error {
	lm.Mu.Lock()
	defer lm.Mu.Unlock()

	if lm.File != nil {
		return lm.File.Sync()
	}
	return nil
}

// SyncHook
type SyncHook struct{}

func (h *SyncHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
func (h *SyncHook) Fire(entry *logrus.Entry) error {
	if Logger != nil && Logger.File != nil {
		return Logger.File.Sync()
	}
	return nil
}

// Add a syncLog manager to the logger
func SyncLogFile() error {
	if Logger != nil && Logger.File != nil {
		return Logger.File.Sync()
	}
	return nil
}

// initialize the logger
func CreateLogger() {

	Logger.L.WithFields(logrus.Fields{
		"Logger Startup": "Logger started",
	}).Info()
	// Force an initial sync
	Logger.Sync()
}

func init() {
	// get the log file path
	dir, err := os.Getwd()
	if err != nil {
		log.Println("Error getting the current working directory")
	}
	logPath := dir + "/logs/appLogs.txt"
	log.Println("Log path is ", logPath)
	logDir := "logs"
	// Create the log directory if it doesn't exist
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		log.Println("Log directory does not exist, creating it")
		os.Mkdir(logDir, 0755)
	}

	// open the app log file
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsPermission(err) {
			log.Printf("Permission denied opening log file with path %s\n", logPath)
		} else {
			log.Println("Failed to log to file, using default stderr", err)
			if errors.Is(err, fs.ErrNotExist) {
				log.Println("log file, model.PathVars.AppLogsPath, does not exist ", logPath)
			}
		}

		log.Println("Failed to log to file, using default stderr")
		file = os.Stdout // Fallback to stdout if file opening fails
	}
	// initialize the logger
	Logger = NewLogger(file)

	// Declare the app logger and add the sync hook
	Logger.L.AddHook(&SyncHook{})

	Logger.L.WithFields(logrus.Fields{
		"Logger Startup": "Logger started",
		"log file":       logPath,
	}).Info()
	// Force an initial sync
	Logger.Sync()
}

// Declare variables to store log messages as new Events
var (
	// errors
	errEncodeMessage            = Event{1, "%s: Could not encode %v to data: %v"}
	errMongoWriteMessage        = Event{2, "%s: Could not write to mongodb - %v"}
	errMongoFilesMessage        = Event{3, "%s: Could not store all files in mongodb abandoning process with error: %v"}
	errRetrieveMongoSubMessage  = Event{4, "%s: Could not find records for subscriberID: %s with error: %v"}
	errRetrieveMongoMessage     = Event{5, "%s: Could not find record in mongodb : %s with error: %v"}
	errConvertToDateMessage     = Event{6, "%s: Cannot convert date to string %s : %v"}
	errConvertToIntMessage      = Event{7, "%s: Cannot convert string to interger %s : %v"}
	errConvertToFloatMessage    = Event{8, "%s: Cannot convert string  to float %s : %v"}
	errMongoConnectionMessage   = Event{9, "%s: An error occured connecting to mongodb: %q"}
	errReadFilesMessage         = Event{10, "%s: Could not read file: %s with error: %v"}
	errWriteFilesMessage        = Event{11, "%s: Could not write file: %s with error: %v"}
	errMoveFilesMessage         = Event{12, "%s: Could not move file: %s with error: %v"}
	errOpenFilesMessage         = Event{13, "%s: Could not open file: %s with error: %v"}
	errCreateFilesMessage       = Event{14, "%s: Could not open file: %s with error: %v"}
	errExistFilesMessage        = Event{15, "%s: File does not exist: %s with error: %v"}
	errChangePathMessage        = Event{16, "%s: Cannot change dir to %s with error: %v"}
	errPathWalkMessage          = Event{17, "%s: Error walking the path %q with error:%v"}
	errPathWalkPreventMessage   = Event{18, "%s: Prevent panic by handling failure accessing a path %q with error: %v"}
	errRowParsingMessage        = Event{19, "%s: Parsing error processing row %q with error: %v"}
	errTimeParsingMessage       = Event{20, "%s: Parsing time error %q with error: %v"}
	errCreateDirMessage         = Event{21, "%s: Cannot create dir: %s with error: %v"}
	errPostgresWriteMessage     = Event{22, "%s: Could not write to postgres - %v"}
	errRetrievePostgresMessage  = Event{23, "%s: Could not find record in postgres: %s with error: %v"}
	errMySQLWriteMessage        = Event{24, "%s: Could not write to mysql - %v"}
	errMySQLRetrieveMessage     = Event{25, "%s: Could not find record in mysql: %s with error: %v"}
	errMySQLFilesMessage        = Event{26, "%s: Could not store all files in mysql abandoning process with error: %v"}
	errMySQLConnectionMessage   = Event{27, "%s: An error occured connecting to mysql: %q"}
	errProcessCSVMessage        = Event{28, "%s: Could not process csv file: %s with error: %v"}
	errDbCommitMessage          = Event{29, "%s: Could not commit db record - %v"}
	errFailREGEXMessage         = Event{30, "%s: Failed REGEX: %s with error: %v"}
	errMapBuildMessage          = Event{31, "%s: Could not build map - %v"}
	errMapLookupMessage         = Event{32, "%s: Could not lookup map - %v"}
	errCertificateMessage       = Event{33, "%s: Could not load certificates - %v"}
	errGRPCListenerMessage      = Event{34, "%s: Could not create gRPC listener - %v"}
	errGRPCServerMessage        = Event{35, "%s: Could not create gRPC server - %v"}
	errReadCallByteMessage      = Event{36, "%s: Could not read recording bytes - %v"}
	errConnectionMessage        = Event{37, "%s: An error occured connecting to mongodb: %q"}
	errDeleteFilesMessage       = Event{38, "%s: Could not delete file or folder: %s with error: %v"}
	errStreamFilesMessage       = Event{39, "%s: Could not stream file: %s with error: %v"}
	errStreamCreateMessage      = Event{40, "%s: Could not create stream: %s with error: %v"}
	errLocationCreateMessage    = Event{41, "%s: Could not create location: %s with error: %v"}
	errCreateLogPathsMessage    = Event{42, "%s: Could not create log path: %s with error: %v"}
	errLoadingEnvMessage        = Event{43, "%s: Could not load environment: %s with error: %v"}
	errCreateRecordingMessage   = Event{44, "%s: Could not build recording: %s with error: %v"}
	errCrtTmpStorageDirMessage  = Event{45, "%s: Could not create temp storage dir: %s with error: %v"}
	errCallListRetrievalMessage = Event{46, "%s: A PBX error occured requesting call recording list: %v with message %s"}
	errTokenMessage             = Event{47, "%s: Could not get Token from PBX - %v"}
	errCDRRetrievalMessage      = Event{48, "%s: Could not get CDR - %v"}
	errInitDataSourcesMessage   = Event{49, "%s: Could not initialize data source - %v"}
	errAuthorizationMessage     = Event{50, "%s: No authorization - %v"}
	errMalformedMessage         = Event{51, "%s: Malformed object: %v with message %s"}
	errMarshallingMessage       = Event{52, "%s: Marshalling error: %v with message %s"}
	errResponseMessage          = Event{53, "%s: Response error: %v with message %s"}
	errDecodeMessage            = Event{54, "%s: Could not decode %v from data: %v"}
	errRequetMessage            = Event{55, "%s: Request error: %v with message %s"}

	// information
	infoDirUsedMessage           = Event{60, "%s: Working dir used to read files: %s"}
	infoDirBypassedMessage       = Event{61, "%s: Skipping dir without errors: %+v \n"}
	infoFindMessage              = Event{62, "%s: Could not find intro text match in scanned line: %s"}
	infoSaveFilesMessage         = Event{63, "%s: Saving files to: %s"}
	infoLoadArchFilesMessage     = Event{64, "%s: Processed %d files from %s stored in mongodb and archived in %s"}
	infoGetReportMessage         = Event{65, "%s: Processed %d files from %s stored in postgres"}
	infoGetReportMessageMySQL    = Event{66, "%s: Processed %d files from %s stored in mysql"}
	infoLogFileCreateMessage     = Event{63, "%s: Log file created: %s"}
	infoCleanLocalStorageMessage = Event{64, "%s: Cleaned %d files from local storage at: %s"}
)

//*****************************************error messages*************************************************

// ErrEncodeMessage is a standard error message
func (l *StandardLogger) ErrEncodeMessage(argumentCall string, argumentValue interface{}, argumentError error) {
	l.L.Errorf(errEncodeMessage.message, argumentCall, argumentValue, argumentError)
}

// ErrMongoWriteMessage is a standard error message
func (l *StandardLogger) ErrMongoWriteMessage(argumentCall string, argumentError error) {
	l.L.Errorf(errMongoWriteMessage.message, argumentCall, argumentError)
}

// ErrConvertToIntMessage is a standard error message
func (l *StandardLogger) ErrConvertToIntMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errConvertToIntMessage.message, argumentCall, argumentName, argumentError)
}

// ErrConvertToFloatMessage is a standard error message
func (l *StandardLogger) ErrConvertToFloatMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errConvertToFloatMessage.message, argumentCall, argumentName, argumentError)
}

// ErrConvertToDateMessage is a standard error message
func (l *StandardLogger) ErrConvertToDateMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errConvertToDateMessage.message, argumentCall, argumentName, argumentError)
}

// errMongoFilesMessageis a standard error message
func (l *StandardLogger) ErrMongoFilesMessage(argumentCall string, argumentError error) {
	l.L.Errorf(errMongoFilesMessage.message, argumentCall, argumentError)
}

// ErrRetrieveMongoSubMessage is a standard error message
func (l *StandardLogger) ErrRetrieveMongoSubMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errRetrieveMongoSubMessage.message, argumentCall, argumentName, argumentError)
}

// ErrRetrieveMongoMessage is a standard error message
func (l *StandardLogger) ErrRetrieveMongoMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errRetrieveMongoMessage.message, argumentCall, argumentName, argumentError)
}

// ErrMongoConnectionMessage is a standard error message
func (l *StandardLogger) ErrMongoConnectionMessage(argumentCall string, argumentError error) {
	l.L.Errorf(errMongoConnectionMessage.message, argumentCall, argumentError)
}

// ErrReadFilesMessage is a standard error message
func (l *StandardLogger) ErrReadFilesMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errReadFilesMessage.message, argumentCall, argumentName, argumentError)
}

// ErrWriteFilesMessage is a standard error message
func (l *StandardLogger) ErrWriteFilesMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errWriteFilesMessage.message, argumentCall, argumentName, argumentError)
}

// ErrMoveFilesMessage is a standard error message
func (l *StandardLogger) ErrMoveFilesMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errMoveFilesMessage.message, argumentCall, argumentName, argumentError)
}

// ErrOpenFilesMessage is a standard error message
func (l *StandardLogger) ErrOpenFilesMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errOpenFilesMessage.message, argumentCall, argumentName, argumentError)
}

// ErrCreateFilesMessage is a standard error message
func (l *StandardLogger) ErrCreateFilesMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errCreateFilesMessage.message, argumentCall, argumentName, argumentError)
}

// ErrExistFilesMessage is a standard error message
func (l *StandardLogger) ErrExistFilesMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errExistFilesMessage.message, argumentCall, argumentName, argumentError)
}

// ErrChangePathMessage is a standard error message
func (l *StandardLogger) ErrChangePathMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errChangePathMessage.message, argumentCall, argumentName, argumentError)
}

// ErrPathWalkMessage is a standard error message
func (l *StandardLogger) ErrPathWalkMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errPathWalkMessage.message, argumentCall, argumentName, argumentError)
}

// ErrPathWalkPreventMessage is a standard error message
func (l *StandardLogger) ErrPathWalkPreventMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errPathWalkPreventMessage.message, argumentCall, argumentName, argumentError)
}

// ErrRowParsingPreventMessage is a standard error message
func (l *StandardLogger) ErrRowParsingMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errRowParsingMessage.message, argumentCall, argumentName, argumentError)
}

// ErrTimeParsingMessage is a standard error message
func (l *StandardLogger) ErrTimeParsingMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errTimeParsingMessage.message, argumentCall, argumentName, argumentError)
}

// ErrCreateDirMessage is a standard error message
func (l *StandardLogger) ErrCreateDirMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errCreateDirMessage.message, argumentCall, argumentName, argumentError)
}

// ErrPostgressWriteMessage is a standard error message
func (l *StandardLogger) ErrPostgresWriteMessage(argumentCall string, argumentError error) {
	l.L.Errorf(errPostgresWriteMessage.message, argumentCall, argumentError)
}

// ErrRetrievePostgressSubMessage is a standard error message
func (l *StandardLogger) ErrRetrievePostgresMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errRetrievePostgresMessage.message, argumentCall, argumentName, argumentError)
}

// ErrMySQLWriteMessage is a standard error message
func (l *StandardLogger) ErrMySQLWriteMessage(argumentCall string, argumentError error) {
	l.L.Errorf(errMySQLWriteMessage.message, argumentCall, argumentError)
}

// ErrMySQLRetrieveMessage is a standard error message
func (l *StandardLogger) ErrMySQLRetrieveMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errMySQLRetrieveMessage.message, argumentCall, argumentName, argumentError)
}

// ErrMySQLFilesMessageis a standard error message
func (l *StandardLogger) ErrMySQLFilesMessage(argumentCall string, argumentError error) {
	l.L.Errorf(errMySQLFilesMessage.message, argumentCall, argumentError)
}

// ErrMySQLConnectionMessage is a standard error message
func (l *StandardLogger) ErrMySQLConnectionMessage(argumentCall string, argumentError error) {
	l.L.Errorf(errMySQLConnectionMessage.message, argumentCall, argumentError)
}

// ErrProcessCSVMessage is a standard error message
func (l *StandardLogger) ErrProcessCSVMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errProcessCSVMessage.message, argumentCall, argumentName, argumentError)
}

// ErrDbCommitMessage is a standard error message
func (l *StandardLogger) ErrDbCommitMessage(argumentCall string, argumentError error) {
	l.L.Errorf(errDbCommitMessage.message, argumentCall, argumentError)
}

// ErrFailREGEXMessage is a standard error message
func (l *StandardLogger) ErrFailREGEXMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errFailREGEXMessage.message, argumentCall, argumentName, argumentError)
}

// ErrMapBuildMessage  is a standard error message
func (l *StandardLogger) ErrMapBuildMessage(argumentCall string, argumentError error) {
	l.L.Errorf(errMapBuildMessage.message, argumentCall, argumentError)
}

// ErrMapLookupMessage  is a standard error message
func (l *StandardLogger) ErrMapLookupMessage(argumentCall string, argumentError error) {
	l.L.Errorf(errMapLookupMessage.message, argumentCall, argumentError)
}

// ErrCertificateMessage  is a standard error message
func (l *StandardLogger) ErrCertificateMessage(argumentCall string, argumentError error) {
	l.L.Errorf(errCertificateMessage.message, argumentCall, argumentError)
}

// ErrGRPCListenerMessage  is a standard error message
func (l *StandardLogger) ErrGRPCListenerMessage(argumentCall string, argumentError error) {
	l.L.Errorf(errGRPCListenerMessage.message, argumentCall, argumentError)
}

// ErrGRPCServerMessage  is a standard error message
func (l *StandardLogger) ErrGRPCServerMessage(argumentCall string, argumentError error) {
	l.L.Errorf(errGRPCServerMessage.message, argumentCall, argumentError)
}

// ErrReadCallByteMessage  is a standard error message
func (l *StandardLogger) ErrReadCallByteMessage(argumentCall string, argumentError error) {
	l.L.Errorf(errReadCallByteMessage.message, argumentCall, argumentError)
}

// ErrConnectionMessage  is a standard error message
func (l *StandardLogger) ErrConnectionMessage(argumentCall string, argumentError error) {
	l.L.Errorf(errConnectionMessage.message, argumentCall, argumentError)
}

// ErrDeleteFilesMessage is a standard error message
func (l *StandardLogger) ErrDeleteFilesMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errDeleteFilesMessage.message, argumentCall, argumentName, argumentError)
}

// ErrStreamFilesMessage is a standard error message
func (l *StandardLogger) ErrStreamFilesMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errStreamFilesMessage.message, argumentCall, argumentName, argumentError)
}

// ErrStreamCreateMessage is a standard error message
func (l *StandardLogger) ErrStreamCreateMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errStreamCreateMessage.message, argumentCall, argumentName, argumentError)
}

// ErrLocationCreateMessage is a standard error message
func (l *StandardLogger) ErrLocationCreateMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errLocationCreateMessage.message, argumentCall, argumentName, argumentError)
}

// ErrCreateLogPathsMessage is a standard error message
func (l *StandardLogger) ErrCreateLogPathsMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errCreateLogPathsMessage.message, argumentCall, argumentName, argumentError)
}

// ErrLoadingEnvMessage is a standard error message
func (l *StandardLogger) ErrLoadingEnvMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errLoadingEnvMessage.message, argumentCall, argumentName, argumentError)
}

// ErrCreateRecordingMessage is a standard error message
func (l *StandardLogger) ErrCreateRecordingMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errCreateRecordingMessage.message, argumentCall, argumentName, argumentError)
}

// ErrCreateDirMessage is a standard error message
func (l *StandardLogger) ErrCrtTmpStorageDirMessage(argumentCall string, argumentName string, argumentError error) {
	l.L.Errorf(errCrtTmpStorageDirMessage.message, argumentCall, argumentName, argumentError)
}

// ErrCallListRetrievalMessage is a standard error message
func (l *StandardLogger) ErrCallListRetrievalMessage(argumentCall string, argumentName error, argumentError string) {
	l.L.Errorf(errCallListRetrievalMessage.message, argumentCall, argumentName, argumentError)
}

// ErrTokenMessage is a standard error message
func (l *StandardLogger) ErrTokenMessage(argumentCall string, argumentError error) {
	l.L.Errorf(errTokenMessage.message, argumentCall, argumentError)
}

// ErrCDRRetrievalMessage is a standard error message
func (l *StandardLogger) ErrCDRRetrievalMessage(argumentCall string, argumentError error) {
	l.L.Errorf(errCDRRetrievalMessage.message, argumentCall, argumentError)
}

// ErrInitDataSourcesMessage is a standard error message
func (l *StandardLogger) ErrInitDataSourcesMessage(argumentCall string, argumentError error) {
	l.L.Errorf(errInitDataSourcesMessage.message, argumentCall, argumentError)
}

// ErrAuthorizationMessage is a standard error message
func (l *StandardLogger) ErrAuthorizationMessage(argumentCall string, argumentError error) {
	l.L.Errorf(errAuthorizationMessage.message, argumentCall, argumentError)
}

// ErrMalformedMessage is a standard error message
func (l *StandardLogger) ErrMalformedMessage(argumentCall string, argumentName error, argumentError string) {
	l.L.Errorf(errMalformedMessage.message, argumentCall, argumentName, argumentError)
}

// ErrMarshallingMessage is a standard error message
func (l *StandardLogger) ErrMarshallingMessage(argumentCall string, argumentName error, argumentError string) {
	l.L.Errorf(errMarshallingMessage.message, argumentCall, argumentName, argumentError)
}

// ErrResponseMessage  is a standard error message
func (l *StandardLogger) ErrResponseMessage(argumentCall string, argumentName error, argumentError string) {
	l.L.Errorf(errResponseMessage.message, argumentCall, argumentName, argumentError)
}

// ErrDecodeMessage  is a standard error message
func (l *StandardLogger) ErrDecodeMessage(argumentCall string, argumentName error, argumentError string) {
	l.L.Errorf(errDecodeMessage.message, argumentCall, argumentName, argumentError)
}

// ErrRequetMessage  is a standard error message
func (l *StandardLogger) ErrRequetMessage(argumentCall string, argumentName error, argumentError string) {
	l.L.Errorf(errRequetMessage.message, argumentCall, argumentName, argumentError)
}

//*****************************************info messages*************************************************

// InfoDirUsedMessage is a standard information message
func (l *StandardLogger) InfoDirUsedMessage(argumentCall string, argumentName string) {
	l.L.Infof(infoDirUsedMessage.message, argumentCall, argumentName)
}

// InfoDirBypassedMessage is a standard information message
func (l *StandardLogger) InfoDirBypassedMessage(argumentCall string, argumentName string) {
	l.L.Infof(infoDirBypassedMessage.message, argumentCall, argumentName)
}

// InfoFindMessage	 is a standard information message
func (l *StandardLogger) InfoFindMessage(argumentCall string, argumentName string) {
	l.L.Infof(infoFindMessage.message, argumentCall, argumentName)
}

// InfoSaveFilesMessage is a standard information message
func (l *StandardLogger) InfoSaveFilesMessage(argumentCall string, argumentName string) {
	l.L.Infof(infoSaveFilesMessage.message, argumentCall, argumentName)
}

// InfoLogFileCreateMessage is a standard information message
func (l *StandardLogger) InfoLogFileCreateMessage(argumentCall string, argumentName string) {
	l.L.Infof(infoLogFileCreateMessage.message, argumentCall, argumentName)
}

// InfoLoadArchFilesMessage is a standard information message
func (l *StandardLogger) InfoLoadArchFilesMessage(argumentCall string, argumentCount int, argumentName1 string, argumentName2 string) {
	l.L.Infof(infoLoadArchFilesMessage.message, argumentCall, argumentCount, argumentName1, argumentName2)
}

// InfoLoadArchFilesMessage is a standard information message
func (l *StandardLogger) InfoGetReportMessage(argumentCall string, argumentCount int, argumentName1 string) {
	l.L.Infof(infoGetReportMessage.message, argumentCall, argumentCount, argumentName1)
}

// InfoGetReportMessageMySQL is a standard information message
func (l *StandardLogger) InfoGetReportMessageMySQL(argumentCall string, argumentName1 string) {
	l.L.Infof(infoGetReportMessageMySQL.message, argumentCall, argumentName1)
}

// InfoCleanLocalStorageMessage is a standard information message
func (l *StandardLogger) InfoCleanLocalStorageMessage(argumentCall string, argumentCount int, argumentName1 string) {
	l.L.Infof(infoCleanLocalStorageMessage.message, argumentCall, argumentCount, argumentName1)
}

//*****************************************warning messages**********************************************
