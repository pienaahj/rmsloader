package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	li "github.com/pienaahj/rmsloader/backend/logwrapper"
	"github.com/pienaahj/rmsloader/backend/model"
)


var CallFrom string // Declare CallFrom variable for logging

var schema = `
CREATE TABLE IF NOT EXISTS rsmcdr (
	id BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
	uid VARCHAR(20) NOT NULL,
	direction VARCHAR(10),
	time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	timestamp INTEGER,    
	flagged TINYINT(1),  
	source VARCHAR(100),
	destination VARCHAR(100),
	duration INTEGER,    
	size INTEGER,    
	exists TINYINT(1), 
	local_copy TINYINT(1),     
	authentic VARCHAR(20),
	sip_call_id VARCHAR(150),
	file_name VARCHAR(100),
	INDEX (timestamp, uid)
);`

// CheckDB checks if the database is up and running
func CheckDB(ctx context.Context, db *sqlx.DB) error {
	CallFrom = "CheckDB in db "
	msg := fmt.Sprintf("CheckDB invoked with %v", db)
	li.Logger.L.Info(CallFrom, msg)
	// check if db is still contactable
	status := "up"
	err := db.Ping()
	if err != nil {
		li.Logger.L.Printf("Ping failed without context: %v", err)
		return fmt.Errorf("db.PingContext: %w", err)
	}
	li.Logger.L.Info("Mysql status after normal ping: ", status)
	if err := db.PingContext(ctx); err != nil {
		status = "down"
		msg := fmt.Sprintf("PingContext error: %v , db: %s ", err, status)
		li.Logger.L.Info(CallFrom, msg, err)
		return fmt.Errorf("db.PingContext: %w", err)
	}

	li.Logger.L.Info("Mysql status: ", status)
	li.Logger.Sync()
	return nil
}

// check if the table is existing
func CheckTableExistsWithShow(ctx context.Context, db *sqlx.DB, tableName string) (bool, error) {
	var table string
	// Safely format the query string
	query := fmt.Sprintf("SHOW TABLES LIKE '%s'", tableName)
	err := db.GetContext(ctx, &table, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil // Table doesn't exist
		}
		newErr := fmt.Errorf("db.GetContext error in CheckTable: %w", err)
		return false, newErr // An error occurred
	}
	return table == tableName, nil
}
// AddCDR adds new recording metadata to the database
func AddCDR(ctx context.Context, db *sqlx.DB, r *model.RMSCDR) (int64, error) {
	CallFrom = "AddCDR in db "
	// get a connection to the database
	// check if db is still contactable
	status := "up"
	if err := db.PingContext(ctx); err != nil {
		status = "down"
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		li.Logger.ErrMySQLConnectionMessage(CallFrom, err)
		return 0, err
	}
	defer conn.Close() // Return the connection to the pool.
	// check if the table is existing
	tableOK, err := CheckTableExistsWithShow(ctx, db, "rsmcdr")
	if err != nil {
		li.Logger.ErrMySQLFilesMessage(CallFrom, err)
		return 0, err
	}
	if !tableOK {
		li.Logger.L.Info(CallFrom, "Table does not exist, create it")
		// create the schema if it doesn't exist
		result := db.MustExecContext(ctx, schema)
		if result != nil {
			msg := fmt.Sprintf("db : %s schema created successfully, result %v ", status, result)
			li.Logger.L.Info(CallFrom, msg)
		}
	}

	// add the new cdr record to the database
	tx := db.MustBeginTx(ctx, &sql.TxOptions{Isolation: 6, ReadOnly: false})
	// spew.Dump(r)
	//durationInSeconds := int64(record.Duration.Seconds())
	res, err := tx.NamedExecContext(ctx, `INSERT INTO rsmcdr (uid, direction, time, timestamp, flagged, source, destination, duration, size, exists, local_copy, authentic, sip_call_id, file_name)
		 VALUES (:uid, :direction, :time, :timestamp, :flagged, :source, :duration, :destination, :talk_duration, :size, :exists, :local_copy, :authentic, :sip_call_id, :file_name)`, r)
	if err != nil {
		li.Logger.L.Printf("problematic cdr: %#v", r)
		li.Logger.ErrMySQLWriteMessage(CallFrom, err)
		return 0, err
	}
	// ra, _ := res.RowsAffected()
	idInserted, _ := res.LastInsertId()
	err = tx.Commit()
	if err != nil {
		li.Logger.ErrDbCommitMessage(CallFrom, err)
		return 0, err
	}
	return idInserted, nil
}

// insertCDRsBatch executes batch inserts for better performance
func InsertCDRsBatch(ctx context.Context, db *sqlx.DB, batch []model.RMSCDR) (int64, error) {
	query := `INSERT INTO rsmcdr (uid, direction, time, timestamp, flagged, source, destination, duration, size, exists, local_copy, authentic, sip_call_id, file_name)
		 VALUES (:uid, :direction, :time, :timestamp, :flagged, :source, :duration, :destination, :talk_duration, :size, :exists, :local_copy, :authentic, :sip_call_id, :file_name)`
	res, err := db.NamedExecContext(ctx, query, batch)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

// GetDatabaseName gets the name of a db
func GetDatabaseName(db *sqlx.DB, dbType string) (string, error) {
	var query string
	switch dbType {
	case "postgres":
		query = "SELECT current_database()"
	case "mysql":
		query = "SELECT DATABASE()"
	case "sqlserver":
		query = "SELECT DB_NAME()"
	default:
		return "", fmt.Errorf("unsupported database type: %s", dbType)
	}

	var dbName string
	err := db.QueryRow(query).Scan(&dbName)
	if err != nil {
		return "", err
	}

	return dbName, nil
}

// CountCDR returns the number of records in the database
func CountCDR(ctx context.Context, db *sqlx.DB) (int64, error) {
	CallFrom = "CountCDR "
	var count int64
	err := db.GetContext(ctx, &count, "SELECT COUNT(*) FROM rmscdr")
	if err != nil {
		newMsg := fmt.Sprintf("error counting CDR records: %v", err)
		newErr := errors.New(newMsg)
		li.Logger.ErrCDRRetrievalMessage(CallFrom, newErr)
		return 0, err
	}
	return count, nil
}

// get a cdr by uid
func GetCDRByUID(ctx context.Context, db *sqlx.DB, uid string) (string, error) {
	CallFrom = "GetCDRById "
	var cdr *model.RMSCDR
	err := db.GetContext(ctx, &cdr, "SELECT * FROM cdr WHERE uid = ?", uid)
	if err != nil {
		li.Logger.ErrMySQLRetrieveMessage(CallFrom, fmt.Sprint(uid), err)
		return "", err
	}
	return cdr.UID, nil
}