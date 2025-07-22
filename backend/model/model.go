package model

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

// Dates represents the requested start and end dates
type Dates struct {
	// date format "20160418095643"
	BeginDate string `db:"begin_date" json:"begin_date"`
	EndDate   string `db:"end_date" json:"end_date"`
}

// DatesResponse represents the response sent for a dates request
type DatesResponse struct {
	ID int64 `db:"id" json:"id"`
	UID string `db:"uid" json:"uid"`
	Direction string `db:"direction" json:"direction"`
	RecordingDate int64 `db:"recording_date" json:"recording_date"` // unix timestamp
	Flagged bool `db:"flagged" json:"flagged"`
	Caller string `db:"caller" json:"caller"`
	Callee string `db:"callee" json:"callee"`
	Duration int64 `db:"duration" json:"duration"`
	Size int64 `db:"size" json:"size"`
	ExistsINDB bool `db:"exists_in_db" json:"exists_in_db"`
	LocalCopy bool `db:"local_copy" json:"local_copy"`
	Authentic string `db:"authentic" json:"authentic"`
	RawFilename string `db:"raw_filename" json:"raw_filename"`
	SipCallID string `db:"sip_call_id" json:"sip_call_id"`
}

// Recording represents a wav file recording in the database.
type RMSCDR struct {
	// The recording id
	ID int64 `db:"id" json:"id"`
	// uid
	UID string `db:"uid" json:"uid"`
	// The Direction
	Direction string `db:"direction" json:"direction"`
	// The recording date
	Time time.Time `db:"time" json:"time"`
	// timestamp
	UnixTimestamp int64 `db:"unix_timestamp" json:"unix_timestamp"`
	// flagged
	Flagged bool `db:"flagged" json:"flagged"`
	// The Caller
	Source string `db:"source" json:"source"`
	// The callee
	Destination string `db:"destination" json:"destination"`
	// derived duration in seconds
	TalkDuration  time.Duration `db:"-"` // don't map this to DB directly
	// duration
	Duration      int64         `db:"duration"` // this is duration in seconds
	// The size
	Size float64 `db:"size" json:"size"`
	// Exists on server
	ExistsINDB bool `db:"exists_in_db" json:"exists_in_db"`
	// Local copy
	LocalCopy bool `db:"local_copy" json:"local_copy"`
	// Is Authentic
	Authentic string `db:"authentic" json:"authentic"`
	// The raw string representation of filename
	FileName string `db:"file_name" json:"file_name"`
	// sip call ID
	SipCallID string `db:"sip_call_id" json:"sip_call_id"`
}

// cater for the duration conversion and database storage
type MyDuration time.Duration

// convert the duration value to int64 value for database field compatibility
func (d MyDuration) Value() (driver.Value, error) {
	return int64(time.Duration(d) / time.Second), nil
}

// scan the duration value from the database
func (d *MyDuration) Scan(value interface{}) error {
	switch v := value.(type) {
	case int64:
		*d = MyDuration(time.Duration(v) * time.Second)
	case []byte:
		i, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return err
		}
		*d = MyDuration(time.Duration(i) * time.Second)
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return err
		}
		*d = MyDuration(time.Duration(i) * time.Second)
	default:
		return fmt.Errorf("cannot scan type %T into MyDuration", value)
	}
	return nil
}
// Filename represents a wav filename
type Filename string

// Filename represents a wav filenames
type Filenames []Filename

// Log represents a logger
var Log = logrus.New()

// Extensions represents a PABX extentions in the database.
// type Extension struct {
// 	ID      int64     `db:"id" json:"id"`
// 	Number  string    `db:"number" json:"number"`
// 	Name    string    `db:"name" json:"name"`
// 	Version string    `db:"version" json:"version"`
// 	Date    time.Time `db:"date" json:"date"`
// }

// // ByName map of extensions by name
// type ByName map[string]string

// // ByNumber map of extensions by number
// type ByNumber map[string]string

// // Extensions represents a slice of Extention
// type Extensions []Extension

// // Recordings represents a slice of Recording
// type Recordings []Recording

// Params represents a query set of parameters
// type Params struct {
// 	Caller         string `db:"caller" json:"caller"`
// 	CallerCategory string `db:"caller_cat" json:"caller_cat"`
// 	Callee         string `db:"callee" json:"callee"`
// 	CalleeCategory string `db:"callee_cat" json:"calleCat"`
// 	Extension      string `db:"extension" json:"extension"`
// 	// date format "20160418095643"
// 	BeginDate string `db:"begin_date" json:"begin_date"`
// 	EndDate   string `db:"end_date" json:"end_date"`
// }

// Param represents a query set of parameters for a filename request
// type Param struct {
// 	Date      string `db:"date" json:"date"`
// 	RawString string `db:"raw_string" json:"raw_string"`
// }

// QuerySingle represents a sql query with one date
// type QuerySingle struct {
// 	QueryText string `db:"query_text" json:"query_text"`
// 	// date format "20160418095643"
// 	Date      string `db:"date" json:"date"`
// 	RawString string `db:"raw_string" json:"raw_string"`
// }

// Query represents a sql query
// type Query struct {
// 	QueryText string `db:"query_text" json:"query_text"`
// 	// date format "20160418095643"
// 	BeginDate      string `db:"begin_date" json:"begin_date"`
// 	EndDate        string `db:"end_date" json:"end_date"`
// 	Caller         string `db:"caller" json:"caller"`
// 	CallerCategory string `db:"caller_category" json:"caller_category"`
// 	Callee         string `db:"callee" json:"callee"`
// 	CalleeCategory string `db:"callee_category" json:"callee_category"`
// }
