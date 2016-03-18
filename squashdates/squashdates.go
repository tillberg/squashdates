package squashdates

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/tillberg/ansi-log"
	"github.com/tillberg/squashdates/timeslice"
)

const DATE_PARSE_FORMAT_TZ = "2006-01-02T15:04:05-07:00"
const DATE_PARSE_FORMAT_UTC = "2006-01-02T15:04:05Z"

const YEAR_FORMAT = "2006"
const MONTH_FORMAT = "Jan 2006"
const DATE_FORMAT = "Mon Jan 02"
const TIME_FORMAT = "15:04"

// Join together spans with gaps shorter than this
const MARGIN = 15 * time.Minute

// Assume that the work started and ended these amounts of time before & after the commit
const PAD_BEFORE = -5 * time.Minute
const PAD_AFTER = 4 * time.Minute

// var durationFormat = alog.Colorify("@(green:%.0f) @(dim:minutes /) @(green:%.1f) @(dim:hours.)")
var durationFormat = alog.Colorify("@(green:%.1f) @(dim:hours.)")

func formatDuration(duration time.Duration) string {
	return fmt.Sprintf(durationFormat, duration.Hours())
}

func ReadDates(reader io.Reader) timeslice.TimeSlice {
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)
	dates := timeslice.TimeSlice{}
	for scanner.Scan() {
		line := scanner.Text()
		format := DATE_PARSE_FORMAT_TZ
		if len(line) >= len(DATE_PARSE_FORMAT_UTC) && line[len(DATE_PARSE_FORMAT_UTC)-1] == 'Z' {
			format = DATE_PARSE_FORMAT_UTC
		}
		date, err := time.Parse(format, line[:len(format)])
		if err != nil {
			alog.Printf("@(warn:Error parsing date from %q: %s)\n", line, err)
		} else {
			dates = append(dates, date)
		}
	}
	return dates
}

func Squash(dates timeslice.TimeSlice, quiet bool) (totalDuration time.Duration, mostRecent time.Time) {
	lg := alog.New(os.Stderr, "", 0)
	// if Opts.Mech {
	//  alog.SetOutput(io.Discard)
	// }
	sort.Sort(dates)

	currYear := ""
	currMonth := ""
	var overallDuration time.Duration
	var yearTotalDuration time.Duration
	var monthTotalDuration time.Duration
	currDay := ""
	currDaySpans := [][2]time.Time{}
	spanStart := time.Time{}
	spanEnd := time.Time{}

	flushCurrYear := func() {
		if yearTotalDuration == 0 {
			return
		}
		lg.Printf("@(dim:Total for) @(cyan:%s)@(dim::) %s\n", currYear, formatDuration(yearTotalDuration))
		yearTotalDuration = 0
	}

	flushCurrMonth := func() {
		if monthTotalDuration == 0 {
			return
		}
		if len(currDaySpans) > 0 {
			year := currDaySpans[0][0].Format(YEAR_FORMAT)
			if currYear != "" && year != currYear {
				flushCurrYear()
			}
			currYear = year
		}
		lg.Printf("  @(dim:Total for) @(cyan:%s)@(dim::) %s\n", currMonth, formatDuration(monthTotalDuration))
		monthTotalDuration = 0
	}

	flushCurrDay := func() {
		if len(currDaySpans) == 0 {
			return
		}
		month := currDaySpans[0][0].Format(MONTH_FORMAT)
		if currMonth != "" && month != currMonth {
			flushCurrMonth()
		}
		currMonth = month
		dateStr := currDaySpans[0][0].Format(DATE_FORMAT)
		if !quiet {
			lg.Printf("      @(dim:Spans for) @(cyan:%s)@(dim::)\n", dateStr)
		}
		var totalDuration time.Duration
		for _, spans := range currDaySpans {
			start := spans[0]
			end := spans[1]
			duration := end.Sub(start)
			totalDuration += duration
			monthTotalDuration += duration
			yearTotalDuration += duration
			overallDuration += duration
			if !quiet {
				lg.Printf("    @(cyan:%s) @(dim:->) @(cyan:%s)@(dim::) %s\n",
					start.Format(TIME_FORMAT), end.Format(TIME_FORMAT), formatDuration(duration))
			}
		}
		lg.Printf("    @(dim:Total for) @(cyan:%s)@(dim::) %s\n", dateStr, formatDuration(totalDuration))
		currDaySpans = currDaySpans[:0]
	}

	flushCurrSpan := func() {
		if spanStart.IsZero() {
			return
		}
		// lg.Println("span", spanStart, spanEnd)
		day := spanStart.Format(DATE_FORMAT)
		if currDay == "" {
			currDay = day
		}
		if currDay != day {
			flushCurrDay()
			currDay = day
		}
		currDaySpans = append(currDaySpans, [2]time.Time{spanStart, spanEnd})
	}
	for _, date := range dates {
		// lg.Println(date)
		start := date.Add(PAD_BEFORE)
		end := date.Add(PAD_AFTER)
		if spanStart.IsZero() {
			spanStart = start
			spanEnd = end
		} else if start.After(spanEnd.Add(MARGIN)) {
			flushCurrSpan()
			spanStart = start
			spanEnd = end
		} else {
			spanEnd = end
		}
	}
	flushCurrSpan()
	flushCurrDay()
	flushCurrMonth()
	flushCurrYear()
	if len(dates) == 0 {
		return overallDuration, time.Time{}
	}
	return overallDuration, dates[len(dates)-1]
}
