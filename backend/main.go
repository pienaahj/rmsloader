package main

import (
	"context"
	"crypto/tls"
	"os/signal"
	"runtime/debug"
	"syscall"

	"fmt"
	"os"
	"time"

	"github.com/go-sql-driver/mysql"
	// _ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	li "github.com/pienaahj/rmsloader/backend/logwrapper"
	"github.com/pienaahj/rmsloader/backend/model"
	"github.com/pienaahj/rmsloader/backend/process"

	"github.com/sirupsen/logrus"
)

// "user:password@/dbname"

const (
	maxRetries      = 5
	retryInterval   = 600 * time.Microsecond
	connectionRetry = 500 * time.Millisecond
)

func main() {
	// Block until an interrupt signal is received
	fmt.Println("Main started") // <<< this should appear regardless
	// create the context
	ctx , cancel := context.WithCancel(context.Background())
	defer cancel()
	// Start signal listener in a separate goroutine
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
		<-sigs
		fmt.Println("rmsloader shutting down gracefully")
		cancel()
	}()

	// check the directories
	fileSystem, _ := os.Getwd()
	// Start the app logger
	li.CreateLogger()
	li.Logger.L.Println("rmsloader is running...")
	li.Logger.L.Println("Current working directory: ", fileSystem)
	// Populate the LogFiles map
	model.ProcessLogFileLocations()

	var (
		err error
		db  *sqlx.DB
	)

	defer func() {
		err := recover()
		if err != nil {
			// Capture stack trace for debugging
			stackTrace := string(debug.Stack())
			switch err := err.(type) {
			case *logrus.Entry:
				entry := err
				li.Logger.L.WithFields(logrus.Fields{
					"recovery": true,
					"err_":     entry.Data["zut!"],
					"err_size": entry.Data["size"],
					"stack":    stackTrace,
				}).Error("Experienced a recovery event")
			case *mysql.MySQLError:
				entry := err
				li.Logger.L.WithFields(logrus.Fields{
					"recovery": true,
					"err_":     entry,
					"stack":    stackTrace,
				}).Error("Experienced fatal a recovery event")
			default:
				// Handle unexpected panic types (string, error, etc.)
				li.Logger.L.WithFields(logrus.Fields{
					"recovery": true,
					"err_":     err,
					"stack":    stackTrace,
				}).Error("Experienced unexpected panic")

			}

			// Ensure logs are flushed before program exits
			li.Logger.Sync()
		}
	}()

	
	// Get the username and password
	// config := LoadEnvironment(false, "cdr")
	// load the tls config
	// Before establishing the database connection
	mysql.RegisterTLSConfig("custom", &tls.Config{
		// Customize TLS configuration as needed
		// For example, to skip server certificate verification:
		InsecureSkipVerify: true,
	})
	// initialize the db connection
	li.Logger.L.Println("Connecting to database...")
	// load the mysql config
	configcdr := LoadEnvironment(true, "cdr")
	li.Logger.L.Printf("Connecting to mysql databases cdr with config: %#v", configcdr)
	// load the tls config
	// Before establishing the database connection
	mysql.RegisterTLSConfig("customcdr", &tls.Config{
		// Customize TLS configuration as needed
		// For example, to skip server certificate verification:
		InsecureSkipVerify: true,
	})
	// Retry connecting to the MySQL server
	for i := 0; i < maxRetries; i++ {
		db, err = sqlx.Connect("mysql", configcdr.FormatDSN())
		if err != nil {
			time.Sleep(connectionRetry)
			continue
		}

		// Ping the database to check if it's alive
		err = db.PingContext(ctx)
		if err == nil {
			li.Logger.L.Println("Connected to MySQL successfully")
			break
		} else {
			li.Logger.L.Printf("Error pinging MySQL: %v", err)
			time.Sleep(retryInterval)
		}
	}
	if db == nil {
		li.Logger.ErrMySQLConnectionMessage("config failed after muliple tries", err)
		li.Logger.L.Println("Could not connect to db after mulitple tries, error ", err.Error())
		os.Exit(1)
	}
	defer db.Close()

	li.Logger.L.Println("Database connection established")
	li.Logger.L.Println("Parsing new recordings...")

	li.Logger.L.Info("Starting process")
	li.Logger.L.Info("Proccessing db")
	// make a new db object

	li.Logger.L.Printf("Main: Proccessing csv files at %s", model.PathVars.CSVPath)
	err = process.Process(ctx, model.PathVars.CSVPath, db)
	if err != nil {
		li.Logger.L.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error processing wav files, terminating...")
		GracefulShutdown(db)
		return
	}
	li.Logger.L.Println("Database populated successfully")

	li.Logger.L.Println("****************************************************************")
	li.Logger.L.Println()
	li.Logger.L.Println("RMSLOADER COMPLETED SUCCESSFULLY")
	li.Logger.L.Println("****************************************************************")
	li.Logger.L.Println()
	GracefulShutdown(db)

}

func GracefulShutdown(db *sqlx.DB) {
	fmt.Println("Shutting down...")
	// sync the logger
	li.Logger.Sync()
	// close the logger
	li.CloseLogger()
	// close the db connection
	if db != nil {
		db.Close()
	}
	Close()  // Close the log resources
	time.Sleep(5 * time.Second) // Give time for logs to appear
	os.Exit(0)
}

// load the environmental variables env true if app is running in docker
func LoadEnvironment(env bool, name string) mysql.Config {
	// DB_PASSWORD := os.Getenv("DB_PASSWORD")
	if name == "cdr" {
		// DB_NAME := os.Getenv("DB_NAME_GO_CDR")
		// DB_USER := os.Getenv("DB_USER_GO")
		// DB_ADDR := os.Getenv("DB_ADDR_GO_CDR")
		// DB_PASSWORD := os.Getenv("DB_PASSWORD_GO")
		cfg := mysql.Config{
			User:                 "new_gouser",
			Passwd:               "NOTAPASSWORD",
			Net:                  "tcp",
			Addr:                 "192.168.128.10:3306",
			DBName:               "Rmsdb",
			AllowNativePasswords: true,
			ParseTime:            true,
			CheckConnLiveness:    true,
			Collation:            "utf8mb4_0900_ai_ci", // Or any other appropriate collation
		}
		// Enable TLS (optional)
		cfg.TLSConfig = "customcdr"
		return cfg
	}
	
	return mysql.Config{}
}
// Close resources
func Close() {
	model.Close()
	li.SyncLogFile()
	li.Logger.L.Info("Closing the application")
	li.Logger.L.Info("Application closed")
	li.CloseLogger()
}

