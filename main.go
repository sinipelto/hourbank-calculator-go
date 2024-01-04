package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
)

// Path to the config file to parse
// Has to be built-in
const CONFIG = "config.txt"

func ConsoleBlock() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Program Complete.")
	fmt.Println("Hit Enter to Rerun.. 'e' to export to file or 'q' to quit")
	r, _, _ := reader.ReadLine()
	rs := string(r)
	if len(rs) > 0 {
		if rs[0] == 'q' {
			return
		}
		if rs[0] == 'e' {
			oper(true)
			return
		}
	}
	oper(false)
}

func main() {
	oper(false)
}

func oper(export bool) {
	// Console holder
	// Deferred, will be called when this func returns
	defer ConsoleBlock()

	cpath, perr := os.Getwd()

	if perr != nil {
		fmt.Println("ERROR: Could not get current directory. Err:", perr.Error())
		return
	}

	config, err := ParseValidateConfig(filepath.Join(cpath, CONFIG))

	if err != nil {
		fmt.Printf("ERROR: Could not open or parse config file or config is invalid. "+
			"Double check exists file: %s in current directory. Err: %s\n", CONFIG, err)
		return
	}

	fmt.Println("Config read OK.")
	fmt.Println()

	entries, entries2, err := ParseImportFile(config)

	if err != nil {
		fmt.Println("Failed to parse input file. Err:", err)
		return
	}

	fmt.Println("Import file parsed OK. Note any errors above.")
	fmt.Println()

	// Share some readonly variables
	global := &Common{
		weeklyHours: config.DailyHours * float64((func() uint8 {
			var days uint8
			for _, v := range ConfigWeekdayMapping {
				if config.ExcludedWeekdays != nil && !ValueInArray(v, config.ExcludedWeekdays) {
					days += 1
				}
			}
			return days
		})()),
	}

	if config.ExcludedWeekdays != nil {
		fmt.Println("Ignored Weekdays:")
		for _, e := range *config.ExcludedWeekdays {
			fmt.Println(e.String())
		}
	}
	fmt.Println("Daily Work Hours:", config.DailyHours)
	fmt.Println("Weekly Work Hours:", global.weeklyHours)
	fmt.Printf("Running In '%s' Mode", *OperationModeRevMapping[config.Mode])
	fmt.Println()

	switch config.Mode {
	case CHECK_MODE:
		switch config.IfType {
		case CUSTOM_FILE:
			ExportCustomFile(config, global, entries)
		default:
			fmt.Println("ERROR: Handling not defined for given input file type.")
			return
		}
	case REPORT_MODE:
		switch config.IfType {
		case CLOCKIFY_FILE:
			if len(*entries2) <= 0 {
				fmt.Println("ERROR: No entries to report from input file. Double check input file correct.")
				return
			}

			fmt.Println()
			fmt.Println("Listing collected days:")
			fmt.Println()

			// First write to stdout
			_ = ExportClockifyReport(config, global, entries2, fmt.Printf, fmt.Println)

			// Then write into export file
			// outfile, err := nil
			// _ = ExportClockifyReport(config, global, entries2, nil, nil)
			// println("Report exported into report file:", outFile)
		default:
			fmt.Println("ERROR: Handling not defined for given input file type.")
			return
		}
	default:
		fmt.Println("ERROR: Unknown operation mode set. Cannot proceed.")
		return
	}
}
