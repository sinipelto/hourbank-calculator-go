package main

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

func ParseInputFileType(str *string) (c ImportFileType, err error) {
	if str == nil {
		return 255, errors.New("ERROR: input ptr was null")
	}
	c, ok := InputFileTypeMapping[*str]
	if !ok {
		return 255, errors.New("ERROR: failed to parse given input to value")
	}
	return
}

func ParseOperationMode(str *string) (c OperationMode, err error) {
	if str == nil {
		return 255, errors.New("ERROR: input ptr was null")
	}
	c, ok := OperationModeMapping[*str]
	if !ok {
		return 255, errors.New("ERROR: failed to parse given input to value")
	}
	return
}

func ParseWeekday(str *string) (*time.Weekday, error) {
	if str == nil {
		return nil, errors.New("ERROR: input ptr was null")
	}
	v, ok := ConfigWeekdayMapping[*str]
	if !ok {
		return nil, errors.New("ERROR: failed to parse given string to value")
	}
	// v, nil
	return v, nil
}

func SliceContains[S ~[]*E, E comparable](s *S, v *E) bool {
	arr := make([]E, 0, len(*s))
	for _, e := range *s {
		arr = append(arr, *e)
	}
	return slices.Contains(arr, *v)
}

func ValueInArray[S ~[]*T, T comparable](element *T, array *S) bool {
	for _, e := range *array {
		// If pointing to same location, have to be same
		if e == element {
			return true
		}
		// Deference to compare the actual values
		if *e == *element {
			return true
		}
	}
	return false
}

func (fmap *FieldMap) ResetFields() {
	for k := range *fmap {
		(*fmap)[k] = false
	}
}

func (fmap *FieldMap) FieldsOk() bool {
	for _, v := range *fmap {
		if !v {
			return false
		}
	}
	return true
}

func ValidateMonthYearRow(str *string) (match bool) {
	if str == nil {
		return false
	}
	match = YEARMONTH_REGEX.MatchString(*str)
	return
}

func ParseMonthYearRow(str *string) (year Year, month Month, err error) {
	if !ValidateMonthYearRow(str) {
		// dont log error, not every non-matched row generates err msg
		return 0, 0, errors.New("invalid input format")
	}
	// Regex PASS
	// assert str fmt: 'YYYY-MM'
	raw := strings.Split(*str, "-")
	yearr, _ := strconv.ParseUint(raw[0], 10, 16)
	monthh, _ := strconv.ParseUint(raw[1], 10, 8)
	year = (Year)(yearr)
	month = (Month)(monthh)
	if year < 2000 || year > 9999 {
		fmt.Println("ERROR: Parsed year value was invalid, should be 2000 < YYYY < 9999")
		return 0, 0, errors.New("invalid year")
	}
	if month < 1 || month > 12 {
		fmt.Println("ERROR: Parsed month value was invalid, should be 01 < MM < 12")
		return 0, 0, errors.New("invalid month")
	}
	return
}

// Cannot define member functions on non-local member string
func StrFloatFiToUs(str *string) string {
	// todo more exact: replace only the correct pos in string
	return strings.Replace(*str, ",", ".", 1)
}

func StrRemoveParentheses(str *string) string {
	return strings.TrimPrefix(strings.TrimSuffix(*str, ")"), "(")
}

func StrRemoveFromBothEnds(str *string, trim *string) string {
	return strings.TrimPrefix(strings.TrimSuffix(*str, *trim), *trim)
}

func AsPtr[T any](v T, u ...any) *T {
	return &v
}

func StringsJoin(arr *ListString, sep *string) *string {
	tmp := make([]string, 0, len(*arr))
	for _, e := range *arr {
		tmp = append(tmp, *e)
	}
	return AsPtr(strings.Join(tmp, *sep))
}

// Assign plus signs in front of decimal string representation
// If value zero or above
func PlusSignIfNecessary(val float64) *string {
	var s *string = AsPtr("")
	if val >= 0 {
		*s = "+"
	}
	return AsPtr(fmt.Sprintf("%s%.2f", *s, val))
}
