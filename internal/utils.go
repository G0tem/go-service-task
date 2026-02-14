package internal

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

func ParseUint16(number string, defaultValue uint16) uint16 {
	result, err := strconv.ParseUint(number, 0, 16)
	if err != nil {
		return defaultValue
	}
	return uint16(result)
}

func ParseBool(value string) bool {
	if value == "" {
		return false
	}
	return strings.ToUpper(value) == "TRUE" ||
		strings.ToUpper(value) == "T" ||
		value == "1" ||
		strings.ToUpper(value) == "YES" ||
		strings.ToUpper(value) == "Y"
}

func ParseInt(number string, defaultValue int) int {
	result, err := strconv.ParseInt(number, 0, 32)
	if err != nil {
		return defaultValue
	}
	return int(result)
}

func ParseUnitDuration(durationUnit string) time.Duration {
	switch strings.ToUpper(durationUnit) {
	case "H":
		return time.Hour
	case "M":
		return time.Minute
	case "S":
		return time.Second
	}
	return 0
}

var durationPattern = regexp.MustCompile(`(\d+)\s*([HhMmSs])`)

func ParseDuration(duration string, defaultValue time.Duration) time.Duration {
	var result time.Duration = 0
	for _, g := range durationPattern.FindAllStringSubmatch(duration, -1) {
		d := time.Duration(ParseInt(g[1], 1)) * ParseUnitDuration(g[2])
		result += d
	}

	return result
}

// PaginationTotalPages возвращает количество страниц при заданных total и pageSize (на уровне БД: LIMIT/OFFSET).
// pageSize должен быть >= 1.
func PaginationTotalPages(total int64, pageSize int) int {
	if pageSize <= 0 {
		pageSize = 1
	}
	if total == 0 {
		return 0
	}
	n := int(total) / pageSize
	if int(total)%pageSize != 0 {
		n++
	}
	return n
}
