package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func ErrorParse(line Line, colIdx *Column, colRaw *string, err error) error {
	fmt.Printf("ERROR: Row: %v: Could not parse Column: %v Value: '%s'. Double check input file.\n", line, *colIdx, *colRaw)
	return err
}

func ErrorLastEntry(line Line, err error) error {
	fmt.Printf("ERROR: Row: %v: Could not process last entry.\n", line)
	return err
}

func ParseImportFile(config *Config) (arr1 *ListWeekEntry, arr2 *ListSingleEntry, err error) {
	f, err := os.Open(*config.ImportFilePath)

	if err != nil {
		fmt.Printf("ERROR: Failed to open config (path: %s) Err: %s\n", *config.ImportFilePath, err.Error())
		return nil, nil, err
	}
	// AFTER err check
	defer f.Close()

	scanner := bufio.NewScanner(f)

	switch config.IfType {
	case CUSTOM_FILE:
		arr1, err = HandleCustomFile(scanner)
		if err != nil {
			fmt.Println("ERROR: Could not process custom import file. Err:", err.Error())
			return nil, nil, err
		}
	case CLOCKIFY_FILE:
		arr2, err = HandleClockifyDetailedExportFile(config, scanner)
		if err != nil {
			fmt.Println("ERROR: Could not process exported import file. Err:", err.Error())
			return nil, nil, err
		}
	default:
		fmt.Println("ERROR: Unknown input file type requested. Cannot proceed.")
		err = errors.New("unknown input file type requested")
		return
	} // switch IfType

	return arr1, arr2, nil
}

func HandleCustomFile(scanner *bufio.Scanner) (arr *ListWeekEntry, err error) {
	var line uint32 = 0

	arr = AsPtr(make(ListWeekEntry, 0, 1024))

	fieldMapping := *NewCustomFileMapping()

	// Assume new entry at start
	entry := &WeekEntry{}

	var year Year
	var month Month

	processLastEntry := func() {
		// If any of fields not set, consider last row failed
		if !fieldMapping.FieldsOk() {
			fmt.Printf("NOTE: Failed to parse previous entry: %+v\n", entry)
			// DONT EXIT => opportunistic, ignore nonrelevant rows and move forward
		} else {
			// If all flags ok, we can add the entry
			*arr = append(*arr, entry)
		}
	}

	// List all conditions to skip the current line parsing
	checkIfSkipLine := func(str *string) (match bool) {
		if strings.HasPrefix(*str, "---") {
			return true
		}
		return ValidateMonthYearRow(str)
	}
ToNextRow:
	for scanner.Scan() {
		line++
		if err = scanner.Err(); err != nil {
			fmt.Println("ERROR: Failed to read line from input file. Err:", err.Error())
			return
		}

		// Retrieve line
		rawstr := scanner.Text()

		// Trim line (remove whitespace from edges)
		rawstr = strings.TrimSpace(rawstr)

		// Skip "" "\n" ...
		// Counts as new entry starts
		if rawstr == "" {
			// Process the last entry
			processLastEntry()
			// On newline/whitespace only row, reset prev entry after processing in any case
			entry = &WeekEntry{}
			fieldMapping.ResetFields()
			// And skip the empty line
			continue ToNextRow
		}

		// Keep track of the current year-month
		yr, mth, err := ParseMonthYearRow(&rawstr)
		// If month, year parsed ok
		if err == nil {
			// Update stored values
			year = yr
			month = mth
		}

		// Determine if line is to be skipped
		if checkIfSkipLine(&rawstr) {
			continue ToNextRow
		}

		// Every entry, set the current
		entry.year = year
		entry.month = month

		// Loop all fields IN ASCENDING ORDER
		var field int
		for field = 0; field < len(fieldMapping); field++ {
			// Handle each field every row
			switch field {
			case 0:
				// Date Range: X.Y.-A.B. or X.-A.B.
				// if current field not filled yet
				if !fieldMapping[field] {
					match := DATERANGE_REGEX.MatchString(rawstr)
					if !match {
						fmt.Printf("ERROR: Line: %v: Could not parse row field: %v. Value: %s\n", line, field, rawstr)
					} else {
						entry.trange = &rawstr
						// Required field - only set to true if parsed ok
						fieldMapping[field] = true
						// move to next row if parsed ok
						continue ToNextRow
					}
				}
			case 1:
				// Comment AABBCC
				if !fieldMapping[field] {
					// Optional field - always set true after iteration
					fieldMapping[field] = true
					match := COMMENT_REGEX.MatchString(rawstr)
					if !match {
						fmt.Printf("WARNING: Line: %v: Could not parse row field: %v. Value: %s\n", line, field, rawstr)
					} else {
						entry.comment = &rawstr
						continue ToNextRow
					}
				}
			case 2:
				// Worked XX,YY
				if !fieldMapping[field] {
					match := DECIMAL_REGEX.MatchString(rawstr)
					if !match {
						fmt.Printf("ERROR: Line: %v: Could not parse row field: %v. Value: %s\n", line, field, rawstr)
					} else {
						var err error
						entry.worked, err = strconv.ParseFloat(StrFloatFiToUs(&rawstr), 64)
						if err != nil {
							fmt.Printf("ERROR: Line: %v: Could not parse row field: %v worked value from: %s as float64, Err: %s\n", line, field, rawstr, err.Error())
						} else {
							fieldMapping[field] = true
							continue ToNextRow
						}
					}
				}
			case 3:
				if !fieldMapping[field] {
					match := SIGNED_DECIMAL_REGEX.MatchString(rawstr)
					if !match {
						fmt.Printf("ERROR: Line: %v: Could not parse row field: %v. Value: %s\n", line, field, rawstr)
					} else {
						var err error
						entry.diff, err = strconv.ParseFloat(StrFloatFiToUs(&rawstr), 64)
						if err != nil {
							fmt.Printf("ERROR: Line: %v: Could not parse field: %v diff value from: %s as float64, Err: %s\n", line, field, rawstr, err.Error())
						} else {
							fieldMapping[field] = true
							continue ToNextRow
						}
					}
				}
			case 4:
				if !fieldMapping[field] {
					match := PAREN_SIGNED_DECIMAL_REGEX.MatchString(rawstr)
					if !match {
						fmt.Printf("ERROR: Line: %v: Could not parse row field: %v. Value: %s\n", line, field, rawstr)
					} else {
						var err error
						entry.balance, err = strconv.ParseFloat(StrFloatFiToUs(AsPtr(StrRemoveParentheses(&rawstr))), 64)
						if err != nil {
							fmt.Printf("ERROR: Line: %v: Could not parse field: %v diff value from: %s as float64, Err: %s\n", line, field, rawstr, err.Error())
						} else {
							fieldMapping[field] = true
							continue ToNextRow
						}
					}
				}
			} // switch fields
			// no default: needed, only known values looped
		} // for each field
		// After passing all values for entry, go next
	} // for each file line
	// After last line, process the last entry after EOF
	processLastEntry()
	return arr, nil
}

func HandleSimpleCustomFile() {
	// TODO
}

func HandleClockifyDetailedExportFile(config *Config, scanner *bufio.Scanner) (arr *ListSingleEntry, err error) {
	// Keep track of current line
	var line Line = 0

	// File lines
	// Cap set for estimation of how many lines would be at maximum
	rows := make(ListString, 0, 1024)

	// Because rows are in reverse order
	// First read all lines
	// Then reverse the array in reverse
ToNextRow:
	for scanner.Scan() {
		line++

		if err = scanner.Err(); err != nil {
			fmt.Println("ERROR: Failed to read line from input file. Err:", err.Error())
			return nil, err
		}

		// First line is header, skipped
		if line == 1 {
			continue ToNextRow
		}

		// Retrieve line
		// aa;bb;Cc;dd;ee;ff;...
		rows = append(rows, AsPtr(scanner.Text()))
	}

	// Ensure something to process
	if len(rows) <= 0 {
		println("WARNING: Nothing to process in import file. Double check input file correct.")
		return nil, errors.New("nothing to process")
	}

	fmt.Printf("Processed %v lines from input file.\n", line)

	// Reset rows for second reverse iteration
	line = 0

	// All Days, most probably max 365 or 2*365
	arr = AsPtr(make(ListSingleEntry, 0, 1024))

	// Define parsed column indexes
	columns := NewClockifyExportColumns()

	// Day entry, combination of entries from the same day
	// Defaults nil, has to be init
	var day *SingleEntry

	// Single localEntry
	// Keep track at global level to access in lastentry func
	var entry *SingleEntry

	// Asserts:
	// Work period cannot start middle of the week!
	// Work period can end middle of the week => calc rest week
	// Excluded days CAN have hours
	processLastEntry := func(isLast bool) (err error) {
		if day == nil {
			// First time, init day only
			// Otherwise current date == entry date even though full day not yet processed
			day = &SingleEntry{}
		} else {
			// d := day.date.Format("02.01.2006")
			// d2 := entry.date.Format("02.01.2006")
			// fmt.Printf("--- Day: %s = %v <=> Entry: %s = %v---\n", d, day.duration, d2, entry.duration)

			// Monkey check - the next row has to be always in future
			// Current day cannot be after next day
			if day.date.After(entry.date) {
				fmt.Printf("ERROR: Row: %v: Previous date (%s) cannot be before current date (%s)!\n", line, day.date, entry.date)
				return errors.New("previous day after current day")
			}
			// If last day date differs from current day date
			if isLast || !day.date.Equal(entry.date) {
				// We can add the day to the day array
				*arr = append(*arr, day)

				// isLast => day.date == entry.date
				// For the last entry
				// we need to add the pontential future missing days for the current week
				// If the last workday is eg in the middle of the week
				if isLast {
					var missingDate time.Time = entry.date.AddDate(0, 0, 1)
					_, wk := entry.date.ISOWeek()
					_, wk2 := missingDate.ISOWeek()
					for wk == wk2 {
						// Skip excluded weekdays, no use
						if  config.ExcludedWeekdays != nil && !ValueInArray(AsPtr(missingDate.Weekday()), config.ExcludedWeekdays) {
							*arr = append(*arr, &SingleEntry{date: missingDate, duration: 0})
						}
						missingDate = missingDate.AddDate(0, 0, 1)
						_, wk2 = missingDate.ISOWeek()
					}
				} else {
					// If a day was skipped, we have to mark it as zero hours done
					// If its included as a workday
					missingDate := day.date.AddDate(0, 0, 1)
					if missingDate.After(entry.date) {
						fmt.Printf("ERROR: Row %v: Missing date (%s) cannot be after previous date (%s)!\n", line, missingDate, day.date)
						return errors.New("missing day after current day")
					}
					// Add all the missing days between the prev and current day
					for entry.date.After(missingDate.Add(time.Hour)) {
						// Skip any missing days not workdays (eg weekends)
						if config.ExcludedWeekdays != nil && !ValueInArray(AsPtr(missingDate.Weekday()), config.ExcludedWeekdays) {
							*arr = append(*arr, &SingleEntry{date: missingDate, duration: 0})
						}
						missingDate = missingDate.AddDate(0, 0, 1)
					}
				}
				// After handling entry, reset current day to new
				day = &SingleEntry{}
			}
		}
		return nil
	}

	// REVERSE order: earliest to latest
	for i := len(rows) - 1; i >= 0; i-- {
		line++

		// Remove escaped quotes from row ends
		row := StrRemoveFromBothEnds(rows[i], AsPtr("\""))

		// Split row into columns
		// Trim out delimiter, escaped quotes between
		cols := strings.Split(row, *config.CsvDelimiter)

		// Monkey check - correct input file, enough columns in row
		if len(cols) < (int)(COL_CLOCKIFY_MAXCOL+1) {
			fmt.Printf("ERROR: Columns count mismatch in input file. Was: %v Should be at least: %v. "+
				"Double check correct import path in config.\n", len(cols), COL_CLOCKIFY_MAXCOL+1)
			return nil, errors.New("column count mismatch")
		}

		entry = &SingleEntry{}
		excluded := false

		// Returns on err, so init only once before loop
		err = nil
		// Handle only required columns for the row
		for _, idx := range *columns {
			// Trim whitespace around column
			// TODO ensure value is case insensitive in all cases!
			colRaw := cols[*idx]
			col := AsPtr(strings.ToLower(strings.TrimSpace(colRaw)))
			switch *idx {
			case COL_CLOCKIFY_TASK:
				// If current task is excluded, count entry as zero
				if config.ExcludedTasks != nil && ValueInArray(col, config.ExcludedTasks) {
					excluded = true
				}
			case COL_CLOCKIFY_DATE:
				entry.date, err = time.Parse(*config.DateParseLayout, *col)
			case COL_CLOCKIFY_DURATION:
				match := DECIMAL_REGEX.MatchString(*col)
				if !match {
					return nil, ErrorParse(line, idx, &colRaw, errors.New("could not parse column from row"))
				}
				entry.duration, err = strconv.ParseFloat(StrFloatFiToUs(col), 64)
			default:
				fmt.Printf("ERROR: Row: %v Column: %v: Value: '%s': Tried to parse column for which parsing is undefined.\n", line, idx, colRaw)
				return nil, errors.New("behaviour not defined for column")
			}
			if err != nil {
				return nil, ErrorParse(line, idx, &colRaw, err)
			}
		}

		// Consecutively, on further rounds
		// Process after entry set, right before day var is updated
		// Process last day before collecting current day
		err = processLastEntry(false)
		if err != nil {
			return nil, ErrorLastEntry(line, err)
		}

		// Copy values
		// ENSURE NO REFS TAKEN
		day.date = entry.date
		// Dont add balance if day excluded
		if !excluded {
			day.duration += entry.duration
		}
	}

	// Add the last missing day not caught from last iteration
	err = processLastEntry(true)
	if err != nil {
		return nil, ErrorLastEntry(line, err)
	}

	return arr, nil
}
