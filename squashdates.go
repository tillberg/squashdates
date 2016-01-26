package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/tillberg/ansi-log"
	"github.com/tillberg/squashdates/timeslice"
)

var Opts struct {
	Quiet bool `short:"q" long:"quiet" description:"Only show day totals"`
}

const DATE_PARSE_FORMAT = "2006-01-02T15:04:05-07:00"

const YEAR_FORMAT = "2006"
const MONTH_FORMAT = "Jan 2006"
const DATE_FORMAT = "Mon Jan 02"
const TIME_FORMAT = "15:04"

// Join together spans with gaps shorter than this
const MARGIN = 15 * time.Minute

// Assume that the work started and ended these amounts of time before & after the commit
const PAD_BEFORE = -6 * time.Minute
const PAD_AFTER = 3 * time.Minute

// var durationFormat = alog.Colorify("@(green:%.0f) @(dim:minutes /) @(green:%.1f) @(dim:hours.)")
var durationFormat = alog.Colorify("@(green:%.1f) @(dim:hours.)")

func formatDuration(duration time.Duration) string {
	return fmt.Sprintf(durationFormat, duration.Hours())
}

func main() {
	_, err := flags.ParseArgs(&Opts, os.Args)
	if err != nil {
		err2, ok := err.(*flags.Error)
		if ok && err2.Type == flags.ErrHelp {
			return
		}
		alog.Printf("Error parsing command-line options: %s\n", err)
		return
	}
	alog.SetPrefix("")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(bufio.ScanLines)
	dates := timeslice.TimeSlice{}
	for scanner.Scan() {
		line := scanner.Text()
		date, err := time.Parse(DATE_PARSE_FORMAT, line)
		if err != nil {
			alog.Printf("@(warn:Error parsing date from %q: %s)\n", line, err)
		} else {
			dates = append(dates, date)
		}
	}
	sort.Sort(dates)

	currYear := ""
	currMonth := ""
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
		alog.Printf("@(dim:Total for) @(cyan:%s)@(dim::) %s\n", currYear, formatDuration(yearTotalDuration))
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
		alog.Printf("  @(dim:Total for) @(cyan:%s)@(dim::) %s\n", currMonth, formatDuration(monthTotalDuration))
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
		if !Opts.Quiet {
			alog.Printf("      @(dim:Spans for) @(cyan:%s)@(dim::)\n", dateStr)
		}
		var totalDuration time.Duration
		for _, spans := range currDaySpans {
			start := spans[0]
			end := spans[1]
			duration := end.Sub(start)
			totalDuration += duration
			monthTotalDuration += duration
			yearTotalDuration += duration
			if !Opts.Quiet {
				alog.Printf("    @(cyan:%s) @(dim:->) @(cyan:%s)@(dim::) %s\n",
					start.Format(TIME_FORMAT), end.Format(TIME_FORMAT), formatDuration(duration))
			}
		}
		alog.Printf("    @(dim:Total for) @(cyan:%s)@(dim::) %s\n", dateStr, formatDuration(totalDuration))
		currDaySpans = currDaySpans[:0]
	}

	flushCurrSpan := func() {
		if spanStart.IsZero() {
			return
		}
		// alog.Println("span", spanStart, spanEnd)
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
		// alog.Println(date)
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
}
