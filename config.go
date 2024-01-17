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

	importMode, ok := configMapping[CNF_IMPORT_MODE]
	if !ok {
		return nil, ConfigErrorMissing(AsPtr(CNF_IMPORT_MODE))
	}
	config.ImportMode, err = ParseImportMode(importMode)
	if err != nil {
		return nil, ConfigErrorParse(AsPtr(CNF_IMPORT_MODE), configMapping[CNF_IMPORT_MODE], err)
	}

	for k, v := range configMapping {
		switch k {
		case CNF_IMPORT_MODE:
			// Compulsory field
			if v == nil {
				return nil, ConfigErrorMissing(&k)
			}
			config.ImportMode, err = ParseImportMode(v)
			if err != nil {
				return nil, ConfigErrorParse(&k, v, err)
			}
		case CNF_IMPORT_PATH_STR:
			// Compulsory field
			if v == nil {
				return nil, ConfigErrorMissing(&k)
			}
			// Collect Imported filename for later exporting purposes
			// Assert: path = abspath or path = relpath to executable
			// Assert: path == regular file, exists
			stat, err := os.Stat(*v)
			if err != nil {
				// Try relative path to executable path
				stat, err = os.Stat(filepath.Join(path, *v))
				if err != nil {
					return nil, ConfigErrorParse(&k, v, err)
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
			if v != nil {
				stat, err := os.Stat(*v)
				if err != nil {
					// Try relative path to executable path
					stat, err = os.Stat(filepath.Join(path, *v))
					if err != nil {
						return nil, ConfigErrorParse(&k, v, err)
					}
				}
				if !stat.IsDir() {
					println("ERROR: Export file path not pointing to a directory. Double check export file path.")
					return nil, errors.New("export file path is not a directory")
				}
				config.ExportFileName = AsPtr(fmt.Sprintf("Report_%s.txt", time.Now().Format("2006-01-02")))
				config.ExportFilePath = AsPtr(filepath.Join(*v, *config.ExportFileName))
			}
		case CNF_CSV_DELIM_STR:
			if v == nil {
				return nil, ConfigErrorMissing(&k)
			}
			config.CsvDelimiter = v
		case CNF_DATE_PARSE_STR:
			if v == nil {
				return nil, ConfigErrorMissing(&k)
			}
			config.DateParseLayout = v
		case CNF_OPERATION_MODE_STR:
			if v == nil {
				return nil, ConfigErrorMissing(&k)
			}
			config.OperationMode, err = ParseOperationMode(v)
			if err != nil {
				return nil, ConfigErrorParse(&k, v, err)
			}
		case CNF_FILE_TYPE_STR:
			if v == nil {
				return nil, ConfigErrorMissing(&k)
			}
			config.IfType, err = ParseInputFileType(v)
			if err != nil {
				return nil, ConfigErrorParse(&k, v, err)
			}
		case CNF_DAILY_HOURS_STR:
			if v == nil {
				return nil, ConfigErrorMissing(&k)
			}
			config.DailyHours, err = strconv.ParseFloat(StrFloatFiToUs(v), 64)
			if err != nil {
				return nil, ConfigErrorParse(&k, v, err)
			}
		case CNF_INITIAL_BALANCE_STR:
			// Optional field
			if v != nil {
				conv, err := strconv.ParseFloat(StrFloatFiToUs(v), 64)
				if err != nil {
					return nil, ConfigErrorParse(&k, v, err)
				}
				config.InitialBalance = &conv
			}
		case CNF_EXCLUDED_WEEKDAYS_STR:
			// Optional field
			if v != nil {
				// Strip, remove whitespace and convert to array
				// "value one, value two, value three, ..." => "valueone,valuetwo,valuethree,..."
				val := strings.Split(strings.ReplaceAll(*v, " ", ""), ",")
				if len(val) <= 0 {
					WarnEmpty(&k, v)
				} else {
					// Init array, max 7 weekdays
					config.ExcludedWeekdays = AsPtr(make(ListWeekday, 0, 7))
					for _, e := range val {
						conv, err := ParseWeekday(&e)
						if err != nil {
							return nil, ConfigErrorParse(&k, v, err)
						}
						if SliceContains(config.ExcludedWeekdays, conv) {
							return nil, ConfigErrorDuplicate(&k, v)
						}
						*config.ExcludedWeekdays = append(*config.ExcludedWeekdays, conv)
					}
				}
			}
		case CNF_EXCLUDED_TASKS_STR:
			// Optional field
			if v != nil {
				// Convert to array
				// "value one,value two,value three, ..."
				val := strings.Split(*v, ",")
				if len(val) <= 0 {
					WarnEmpty(&k, v)
				} else {
					// 0 => we dont know in advance how many there would be
					config.ExcludedTasks = AsPtr(make(ListString, 0))
					for _, e := range val {
						// "   " => ""
						// " value one " => "value one"
						vval := strings.TrimSpace(e)
						if len(vval) <= 0 {
							WarnEmpty(&k, v)
							continue
						}
						if SliceContains(config.ExcludedTasks, &vval) {
							return nil, ConfigErrorDuplicate(&k, v)
						}
						*config.ExcludedTasks = append(*config.ExcludedTasks, &vval)
					}
				}
			}
		default:
			// Improve backwards compatibility - ignore (yet) undefined keys
			fmt.Printf("WARNING: Key in config is unknown: '%s'. Double check config file. Config value ignored.\n", k)
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
