package main

import (
	"errors"
	"fmt"
	"time"
)

// Same logic output report to stdout or file
// Always replace same export file
// Prefer updating file, avoid piling up files
func ExportClockifyReport(config *Config, global *Common, entries2 *ListSingleEntry, printfF FuncPrintf, printlnF FuncPrintln) error {
	if printfF == nil || printlnF == nil {
		fmt.Println("ERROR: Either of print functions not set for the output.")
		return errors.New("either print function nil")
	}

	// Keep track of some variables
	var balance float64 = 0
	if config.InitialBalance != nil {
		balance += *config.InitialBalance
	}
	var weekWorked float64 = 0
	var weekno uint16 = 0

	var _, lastWeek = (*entries2)[0].date.ISOWeek()

	// Debugging variables
	// Remember to comment out time range
	var lower *time.Time = nil
	var upper *time.Time = nil
	// lower = AsPtr(time.Parse(*config.DateParseLayout, "25.07.2023"))
	// upper = AsPtr(time.Parse(*config.DateParseLayout, "10.08.2023"))

	var cur *SingleEntry
	printWeek := func(wk int) (printed bool) {
		printed = false
		weekno += 1
		if (lower == nil && upper == nil) || (cur.date.After(*lower) && cur.date.Before(*upper)) {
			_, _ = printlnF()
			_, _ = printlnF()
			_, _ = printlnF("********************")
			_, _ = printfF("Week: %v (%v):\nWorked: %s\nWeek Diff: %s\nBalance: %s\n", weekno, lastWeek, *PlusSignIfNecessary(weekWorked), *PlusSignIfNecessary(weekWorked - global.weeklyHours), *PlusSignIfNecessary(balance))
			_, _ = printlnF("********************")
			_, _ = printlnF()
			_, _ = printlnF()
			printed = true
		}
		weekWorked = 0
		return
	}

	for i, e := range *entries2 {
		// Keep track of last entry for printweek
		cur = e

		// Get current week number
		_, wk := e.date.ISOWeek()

		if wk != lastWeek {
			printWeek(wk)
		} else if (lower == nil && upper == nil) || (cur.date.After(*lower) && cur.date.Before(*upper)) {
			_, _ = printlnF("--------------------")
		}

		// Usually only mon-fri has daily hours limit
		// Other days hours are counted as additional hours
		var diff float64
		if config.ExcludedWeekdays != nil && ValueInArray(AsPtr(e.date.Weekday()), config.ExcludedWeekdays) {
			// Count any excluded days work as extra time
			diff = e.duration
		} else {
			diff = e.duration - config.DailyHours
		}

		balance += diff
		weekWorked += e.duration

		if (lower == nil && upper == nil) || (cur.date.After(*lower) && cur.date.Before(*upper)) {
			_, _ = printfF("Entry Index: %v\nDate: %s\nWorked: %s\nDiff to limit: %s\nCurrent Balance: %s\n", i, e.date.Format(*config.DateParseLayout), *PlusSignIfNecessary(e.duration), *PlusSignIfNecessary(diff), *PlusSignIfNecessary(balance))
		}

		lastWeek = wk
	}

	if !printWeek(lastWeek) {
		_, _ = printlnF()
		_, _ = printlnF()
	}
	_, _ = printfF("Final Balance: %s\n", *PlusSignIfNecessary(balance))

	return nil
}

func ExportCustomFile(config *Config, global *Common, entries *ListWeekEntry) {
	// Keep track of some variables
	var balance float64 = 0
	if config.InitialBalance != nil {
		balance += *config.InitialBalance
	}
	var prevYear Year
	var prevMonth Month

	// Go through entries and check
	// Verify against each entry recorded balance matches etc
	for i, e := range *entries {
		fmt.Printf("\nMonth: %v Year: %v\nWeek: %s\nWorked: %.2f\nDiff: %.2f\nReported Balance: %.2f\n", e.month, e.year, *e.trange, e.worked, e.diff, e.balance)

		if i == 0 {
			// if entry is FIRST only (not middle, last)
			// All stuff specific to first entry
			_ = 0
		} else {
			// if entry NOT first, can be last (is middle, last)
			// All stuff specific to any NON-FIRST entry
			if e.year != prevYear || e.month != prevMonth {
				if prevMonth == 12 {
					// If previous month was 12, it should be next year 1
					if e.year != prevYear+1 {
						fmt.Printf("ERROR: Entry %v (%s): Year (%v) != Next Year (%v)\n", i, *e.trange, e.year, prevYear+1)
						return
					}
					if e.month != 1 {
						fmt.Printf("ERROR: Entry %v (%s): Month (%v) != Next Month (%v)\n", i, *e.trange, e.month, 1)
						return
					}
				} else {
					// IF month or year has changed since previous
					// but its NOT next year
					if e.year != prevYear {
						fmt.Printf("ERROR: Entry %v (%s): Year (%v) != Expected Year (%v)\n", i, *e.trange, e.year, prevYear)
						return
					}
					if e.month != prevMonth+1 {
						fmt.Printf("ERROR: Entry %v (%s): Month (%v) != Next Month (%v)\n", i, *e.trange, e.month, prevMonth+1)
						return
					}

				}
			}
		}

		if i == len(*entries)-1 {
			// if entry is LAST only (not first, middle)
			// All stuff specific to last entry
			_ = 0
		} else {
			// if entry NOT last, can be first (is first, middle)
			// All stuff specific to any NON-LAST entry
			_ = 0
		}
		// Stuff common to any entry
		expDiff := e.worked - global.weeklyHours
		if e.diff != expDiff {
			fmt.Printf("ERROR: Entry %v (%s): Diff (%s) != Worked (%s) - Limit (%.2f) == Expected Diff (%s)\n", i, *e.trange, *PlusSignIfNecessary(e.diff), *PlusSignIfNecessary(e.worked), global.weeklyHours, *PlusSignIfNecessary(expDiff))
			return
		}

		// Collect the current EXPECTED balance (troughout entries)
		balance += e.worked - global.weeklyHours
		fmt.Printf("Expected Balance: %s\n", *PlusSignIfNecessary(balance))

		if balance != e.balance {
			fmt.Printf("\nERROR: Entry %v (%s): Expected Balance (%s) != Reported Balance (%s)\n", i, *e.trange, *PlusSignIfNecessary(balance), *PlusSignIfNecessary(e.balance))
			return
		}

		prevYear = e.year
		prevMonth = e.month
	}

	fmt.Println()
	fmt.Printf("Final Balance: %s\n", *PlusSignIfNecessary(balance))
	fmt.Println()
}
