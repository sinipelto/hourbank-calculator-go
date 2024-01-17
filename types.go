package main

import (
	"regexp"
	"time"
)

// DO NOT REDEFINE FLOAT64!
// COMPILE ERRORS FROM STRCNV METHODS

// Order from most primitive to most advanced

type ImportFileType uint8
type OperationMode uint8
type ImportMode uint8
type Line uint32  // 0-2^32 lines?
type Column uint8 // 0-255 columns?
type Year uint16  // 1-9999
type Month uint8  // 1-12

type ListWeekday []*time.Weekday
type ListString []*string

type ListColumn []*Column
type ListWeekEntry []*WeekEntry
type ListSingleEntry []*SingleEntry

type FieldMap map[int]bool // int required for indexing
type StringPtrMap map[string]*string
type WeekdayMap map[string]*time.Weekday

type ImportFileTypeMap map[string]ImportFileType
type OperationModeMap map[string]OperationMode
type ImportModeMap map[string]ImportMode

type WeekdayRevMap map[time.Weekday]*string
type OperationModeRevMap map[OperationMode]*string
type ImportModeRevMap map[ImportMode]*string

type FuncPrintf func(format string, a ...any) (n int, err error)
type FuncPrintln func(a ...any) (n int, err error)

type Config struct {
	IfType              ImportFileType
	OperationMode       OperationMode
	ImportMode          ImportMode
	ImportFilePath      *string
	ImportFileName      *string
	ExportFilePath      *string
	ExportFileName      *string
	CsvDelimiter        *string
	DateParseLayout     *string
	ExcludedWeekdays    *ListWeekday
	ExcludedTasks       *ListString
	InitialBalance      *float64
	DailyHours          float64
	ClockifyApiBase     *string
	ClockifyReportUrl   *string
	ClockifyApiKey      *string
	ClockifyWorkspaceId *string
	ClockifyReportStart *time.Time
	ClockifyReportEnd   *time.Time
}

type Common struct {
	weeklyHours float64
}

type WeekEntry struct {
	trange  *string
	comment *string
	year    Year
	month   Month
	worked  float64
	diff    float64
	balance float64
}

type SingleEntry struct {
	date     time.Time
	duration float64
}

var (
	DECIMAL_REGEX              = regexp.MustCompile(`[0-9]|[1-9][0-9](\,|\.)[0-9]|[1-9][0-9][0-9]?`)
	SIGNED_DECIMAL_REGEX       = regexp.MustCompile(`[+\-]` + DECIMAL_REGEX.String())
	PAREN_SIGNED_DECIMAL_REGEX = regexp.MustCompile(`\(` + SIGNED_DECIMAL_REGEX.String() + `\)`)
	DATERANGE_REGEX            = regexp.MustCompile(`([1-9]|[1-2][0-9]|3[0-1])\.([1-9]|1[012]\.)?\-([1-9]|[1-2][0-9]|3[0-1])\.([1-9]|1[012])\.`)
	COMMENT_REGEX              = regexp.MustCompile(`[A-Z]`)
	YEARMONTH_REGEX            = regexp.MustCompile(`[1-9][0-9]{3}\-(0?[1-9]|1[012])`)
	// DATE_REGEX                 = regexp.MustCompile(`(0?[1-9]|[1-2][0-9]|3[0-1])\.(0?[1-9]|1[012])\.[1-9][0-9]{3}`)
)

const (
	CNF_IMPORT_MODE           string = "import_mode"
	CNF_IMPORT_PATH_STR       string = "import_path"
	CNF_EXPORT_PATH_STR       string = "export_dir"
	CNF_FILE_TYPE_STR         string = "file_type"
	CNF_DAILY_HOURS_STR       string = "required_daily_hours"
	CNF_OPERATION_MODE_STR    string = "operation_mode"
	CNF_CSV_DELIM_STR         string = "csv_delimiter"
	CNF_DATE_PARSE_STR        string = "date_layout"
	CNF_EXCLUDED_WEEKDAYS_STR string = "excluded_weekdays"
	CNF_INITIAL_BALANCE_STR   string = "initial_balance"
	CNF_EXCLUDED_TASKS_STR    string = "excluded_clockify_tasks"
	CNF_CLOCKIFY_API_BASE     string = "clockify_api_base_url"
	CNF_CLOCKIFY_REPORT_URL   string = "clockify_api_report_url"
	CNF_CLOCKIFY_API_KEY      string = "clockify_api_key"
	CNF_CLOCKIFY_WS_ID        string = "clockify_workspace_id"
	CNF_CLOCKIFY_START        string = "clockify_report_start"
	CNF_CLOCKIFY_END          string = "clockify_report_end"
)

func EmptyConfigurationMapping() StringPtrMap {
	return StringPtrMap{
		CNF_IMPORT_MODE:           nil,
		CNF_IMPORT_PATH_STR:       nil,
		CNF_EXPORT_PATH_STR:       nil,
		CNF_FILE_TYPE_STR:         nil,
		CNF_DAILY_HOURS_STR:       nil,
		CNF_OPERATION_MODE_STR:    nil,
		CNF_CSV_DELIM_STR:         nil,
		CNF_DATE_PARSE_STR:        nil,
		CNF_EXCLUDED_WEEKDAYS_STR: nil,
		CNF_INITIAL_BALANCE_STR:   nil,
		CNF_EXCLUDED_TASKS_STR:    nil,
		CNF_CLOCKIFY_API_BASE:     nil,
		CNF_CLOCKIFY_REPORT_URL:   nil,
		CNF_CLOCKIFY_API_KEY:      nil,
		CNF_CLOCKIFY_WS_ID:        nil,
		CNF_CLOCKIFY_START:        nil,
		CNF_CLOCKIFY_END:          nil,
	}
}

// Define column constants
const (
	COL_CLOCKIFY_TASK     Column = 3
	COL_CLOCKIFY_DATE     Column = 9
	COL_CLOCKIFY_DURATION Column = 14
)

// Highest value of columns here
// To determine min columns needed from input file
const COL_CLOCKIFY_MAXCOL = max(
	COL_CLOCKIFY_TASK,
	COL_CLOCKIFY_DATE,
	COL_CLOCKIFY_DURATION,
)

func NewClockifyExportColumns() *ListColumn {
	return &ListColumn{
		AsPtr(COL_CLOCKIFY_TASK),
		AsPtr(COL_CLOCKIFY_DATE),
		AsPtr(COL_CLOCKIFY_DURATION),
	}
}

// Input data source
// Determines way of handling input
// Custom: handmade balance.txt
// Export: e.g. from Clockify App excel/csv export
const (
	CUSTOM_FILE ImportFileType = iota
	CUSTOM_SHORT_FILE
	CLOCKIFY_FILE
)

// Possible config values for inputfiletype
// ENSURE CORRECT ORDERING FROM ABOVE
// CONSTANT READONLY
var InputFileTypeMapping = ImportFileTypeMap{
	"custom":          CUSTOM_FILE,
	"customshort":     CUSTOM_SHORT_FILE,
	"clockify_export": CLOCKIFY_FILE,
}

const (
	CHECK_MODE OperationMode = iota
	REPORT_MODE
)

// Possible config values for operationmode
// ENSURE CORRECT ORDERING FROM ABOVE
// CONSTANT READONLY
var OperationModeMapping = OperationModeMap{
	"check":  CHECK_MODE,
	"report": REPORT_MODE,
}

// CONSTANT READONLY
var OperationModeRevMapping = OperationModeRevMap{
	CHECK_MODE:  AsPtr("check"),
	REPORT_MODE: AsPtr("report"),
}

const (
	API_MODE ImportMode = iota
	FILE_MODE
)

// CONSTANT READONLY
var ImportModeRevMapping = ImportModeRevMap{
	API_MODE:  AsPtr("api"),
	FILE_MODE: AsPtr("file"),
}

var ImportModeMapping = ImportModeMap{
	"api":  API_MODE,
	"file": FILE_MODE,
}

// CONSTANT READONLY
var ConfigWeekdayMapping = WeekdayMap{
	"mon": AsPtr(time.Monday),
	"tue": AsPtr(time.Tuesday),
	"wed": AsPtr(time.Wednesday),
	"thu": AsPtr(time.Thursday),
	"fri": AsPtr(time.Friday),
	"sat": AsPtr(time.Saturday),
	"sun": AsPtr(time.Sunday),
}

func NewCustomFileMapping() *FieldMap {
	return &FieldMap{
		0: false, // dates
		1: false, // comment
		2: false, // worked
		3: false, // diff to weekly limit
		4: false, // calced current balance
	}
}
