package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Path to the config file to parse
// Has to be built-in
const CONFIG_FILE = "config.txt"

func ConfigErrorMissing(k *string) error {
	fmt.Printf("ERROR: Config: '%s' is required but was not defined in config file.\n", *k)
	return errors.New("required config parameter is not set")
}

func ConfigErrorParse(k *string, v *string, err error) error {
	fmt.Printf("ERROR: Failed to parse '%s' from config. Value: '%s' Error: %s\n", *k, *v, err.Error())
	return err
}

func ConfigErrorDuplicate(k *string, v *string) error {
	fmt.Printf("ERROR: Failed to parse '%s' from config. Value: '%s'. Duplicate value(s) detected, not allowed.\n", *k, *v)
	return errors.New("duplicate values found in config file, required unique")
}

func WarnEmpty(k *string, v *string) {
	fmt.Printf("WARNING: Failed to parse '%s' from config. Found empty value(s) in: '%s' defined in config file.\n", *k, *v)
}

func ParseValidateConfig() (config *Config, err error) {
	path, err := os.Executable()

	if err != nil {
		fmt.Println("ERROR: Could not get current executable path. Err:", err.Error())
		return
	}

	// Get current executable dir, parse, conv to abs, validate
	path = filepath.Dir(path)
	path, err = filepath.Abs(path)

	if err != nil {
		fmt.Println("ERROR: Could not get current working directory. Err:", err.Error())
		return
	}

	f, err := os.Open(filepath.Join(path, CONFIG_FILE))

	if err != nil {
		fmt.Printf("ERROR: Failed to open config (path: %s) Err: %s\n", path, err.Error())
		return
	}

	scanner := bufio.NewScanner(f)

	if err != nil {
		fmt.Printf("ERROR: Could not open or parse config file or config is invalid. "+
			"Double check exists file: %s in current directory. Err: %s\n", CONFIG_FILE, err)
		f.Close()
		return
	}

	var configMapping = EmptyConfigurationMapping()

	for scanner.Scan() {
		if err = scanner.Err(); err != nil {
			fmt.Println("ERROR: Failed to read line from config. Err:", err.Error())
			f.Close()
			return
		}

		// Trim spaces from ends
		rawstr := strings.TrimSpace(scanner.Text())

		// Skip empty, skip commented out
		if rawstr == "" || rawstr[0] == '#' {
			continue
		}

		raw := strings.Split(rawstr, "=")

		if len(raw) != 2 {
			fmt.Println("ERROR: Invalid line in config:", rawstr)
			f.Close()
			return nil, errors.New("invalid line in config file")
		}

		key := strings.ToLower(strings.TrimSpace(raw[0]))
		val := strings.ToLower(strings.TrimSpace(raw[1]))

		if configMapping[key] != nil {
			f.Close()
			return nil, errors.New("detected duplicate config key in config file")
		}

		configMapping[key] = &val
	}

	// Close the file after parsed through
	if err = f.Close(); err != nil {
		fmt.Println("ERROR: Failed to close config file after reading:", err.Error())
		return
	}

	config = &Config{}

	// Needed to decide if reading some config values
	impModeKey := AsPtr(CNF_IMPORT_MODE)
	importMode, ok := configMapping[*impModeKey]
	if !ok {
		return nil, ConfigErrorMissing(impModeKey)
	}
	config.ImportMode, err = ParseImportMode(importMode)
	if err != nil {
		return nil, ConfigErrorParse(impModeKey, importMode, err)
	}

	// Needed to parse the datetimes out of the config
	dateParseKey := AsPtr(CNF_DATE_PARSE_STR)
	dateParse, ok := configMapping[*dateParseKey]
	if !ok {
		return nil, ConfigErrorMissing(impModeKey)
	}
	if dateParse == nil {
		return nil, ConfigErrorMissing(dateParseKey)
	}
	config.DateParseLayout = dateParse

	for _, repl := range *ConfigurationIndexer {
		v := configMapping[*repl]
		switch *repl {
		// case CNF_IMPORT_MODE:
		// 	// Compulsory field
		// 	if v == nil {
		// 		return nil, ConfigErrorMissing(k)
		// 	}
		// 	config.ImportMode, err = ParseImportMode(v)
		// 	if err != nil {
		// 		return nil, ConfigErrorParse(k, v, err)
		// 	}
		case CNF_CLOCKIFY_API_BASE:
			// Needed only if using api mode
			if config.ImportMode == API_MODE {
				// Compulsory field
				if v == nil {
					return nil, ConfigErrorMissing(v)
				}
				// TODO: parse uri?
				config.ClockifyApiBase = v
			}
		case CNF_CLOCKIFY_API_KEY:
			// Needed only if using api mode
			if config.ImportMode == API_MODE {
				// Compulsory field
				if v == nil {
					return nil, ConfigErrorMissing(repl)
				}
				config.ClockifyApiKey = v
			}
		case CNF_CLOCKIFY_WS_ID:
			// Needed only if using api mode
			if config.ImportMode == API_MODE {
				// Compulsory field
				if repl == nil {
					return nil, ConfigErrorMissing(repl)
				}
				config.ClockifyWorkspaceId = v
			}
		case CNF_CLOCKIFY_START:
			// Needed only if using api mode
			if config.ImportMode == API_MODE {
				// Compulsory field
				if repl == nil {
					return nil, ConfigErrorMissing(repl)
				}
				val, err := time.Parse(*config.DateParseLayout, *repl)
				if err != nil {
					return nil, ConfigErrorParse(repl, v, err)
				}
				config.ClockifyReportStart = &val
			}
		case CNF_CLOCKIFY_END:
			// Needed only if using api mode
			if config.ImportMode == API_MODE {
				// Compulsory field
				if repl == nil {
					return nil, ConfigErrorMissing(repl)
				}
				val, err := time.Parse(*config.DateParseLayout, *repl)
				if err != nil {
					return nil, ConfigErrorParse(repl, v, err)
				}
				config.ClockifyReportStart = &val
			}
		case CNF_IMPORT_PATH_STR:
			// Compulsory field
			if repl == nil {
				return nil, ConfigErrorMissing(repl)
			}
			// Collect Imported filename for later exporting purposes
			// Assert: path = abspath or path = relpath to executable
			// Assert: path == regular file, exists
			stat, err := os.Stat(*repl)
			if err != nil {
				// Try relative path to executable path
				stat, err = os.Stat(filepath.Join(path, *repl))
				if err != nil {
					return nil, ConfigErrorParse(repl, v, err)
				}
			}
			if !stat.Mode().IsRegular() {
				println("ERROR: Import file path not pointing to a regular file. Double check import file path in config.")
				return nil, errors.New("import file path is not regular file")
			}
			config.ImportFilePath = v
			config.ImportFileName = AsPtr(stat.Name())
		case CNF_EXPORT_PATH_STR:
			// Optional field
			if repl != nil {
				stat, err := os.Stat(*repl)
				if err != nil {
					// Try relative path to executable path
					stat, err = os.Stat(filepath.Join(path, *repl))
					if err != nil {
						return nil, ConfigErrorParse(repl, v, err)
					}
				}
				if !stat.IsDir() {
					println("ERROR: Export file path not pointing to a directory. Double check export file path.")
					return nil, errors.New("export file path is not a directory")
				}
				config.ExportFileName = AsPtr(fmt.Sprintf("Report_%s.txt", time.Now().Format("2006-01-02")))
				config.ExportFilePath = AsPtr(filepath.Join(*repl, *config.ExportFileName))
			}
		case CNF_CSV_DELIM_STR:
			if repl == nil {
				return nil, ConfigErrorMissing(repl)
			}
			config.CsvDelimiter = v
		case CNF_OPERATION_MODE_STR:
			if repl == nil {
				return nil, ConfigErrorMissing(repl)
			}
			config.OperationMode, err = ParseOperationMode(repl)
			if err != nil {
				return nil, ConfigErrorParse(repl, v, err)
			}
		case CNF_FILE_TYPE_STR:
			if repl == nil {
				return nil, ConfigErrorMissing(repl)
			}
			config.IfType, err = ParseInputFileType(repl)
			if err != nil {
				return nil, ConfigErrorParse(repl, v, err)
			}
		case CNF_DAILY_HOURS_STR:
			if repl == nil {
				return nil, ConfigErrorMissing(repl)
			}
			config.DailyHours, err = strconv.ParseFloat(StrFloatFiToUs(repl), 64)
			if err != nil {
				return nil, ConfigErrorParse(repl, v, err)
			}
		case CNF_INITIAL_BALANCE_STR:
			// Optional field
			if repl != nil {
				conv, err := strconv.ParseFloat(StrFloatFiToUs(repl), 64)
				if err != nil {
					return nil, ConfigErrorParse(repl, v, err)
				}
				config.InitialBalance = &conv
			}
		case CNF_EXCLUDED_WEEKDAYS_STR:
			// Optional field
			if repl != nil {
				// Strip, remove whitespace and convert to array
				// "value one, value two, value three, ..." => "valueone,valuetwo,valuethree,..."
				val := strings.Split(strings.ReplaceAll(*repl, " ", ""), ",")
				if len(val) <= 0 {
					WarnEmpty(repl, v)
				} else {
					// Init array, max 7 weekdays
					config.ExcludedWeekdays = AsPtr(make(ListWeekday, 0, 7))
					for _, e := range val {
						conv, err := ParseWeekday(&e)
						if err != nil {
							return nil, ConfigErrorParse(repl, v, err)
						}
						if SliceContains(config.ExcludedWeekdays, conv) {
							return nil, ConfigErrorDuplicate(repl, v)
						}
						*config.ExcludedWeekdays = append(*config.ExcludedWeekdays, conv)
					}
				}
			}
		case CNF_EXCLUDED_TASKS_STR:
			// Optional field
			if repl != nil {
				// Convert to array
				// "value one,value two,value three, ..."
				val := strings.Split(*repl, ",")
				if len(val) <= 0 {
					WarnEmpty(repl, v)
				} else {
					// 0 => we dont know in advance how many there would be
					config.ExcludedTasks = AsPtr(make(ListString, 0))
					for _, e := range val {
						// "   " => ""
						// " value one " => "value one"
						vval := strings.TrimSpace(e)
						if len(vval) <= 0 {
							WarnEmpty(repl, v)
							continue
						}
						if SliceContains(config.ExcludedTasks, &vval) {
							return nil, ConfigErrorDuplicate(repl, v)
						}
						*config.ExcludedTasks = append(*config.ExcludedTasks, &vval)
					}
				}
			}
		default:
			// Improve backwards compatibility - ignore (yet) undefined keys
			fmt.Printf("WARNING: Key in config is unknown: '%s'. Double check config file. Config value ignored.\n", *repl)
			// return nil, errors.New("unsupported configuration key in config file")
		}
	}

	// If import file is exported/generated, there is no point doing any checking/validation
	if config.OperationMode == CHECK_MODE && config.IfType == CLOCKIFY_FILE {
		println("ERROR: Current selected mode and import file type are incompatible. " +
			"Please update config: select either check mode with ANY CUSTOM import, or report mode with ANY import file.")
		return nil, errors.New("incompatible mode and file")
	}

	return config, nil
}
