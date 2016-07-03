package main

import (
	"fmt"
	"os"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/tillberg/ansi-log"
	"github.com/tillberg/squashdates/squashdates"
	"github.com/tillberg/squashdates/timeslice"
)

var Opts struct {
	Quiet bool   `short:"q" long:"quiet" description:"Only show day totals"`
	Mech  bool   `long:"mech" description:"Output # seconds followed by last time seen"`
	Since string `long:"since" description:"Only include dates after this"`
}

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

	dates := squashdates.ReadDates(os.Stdin)

	if Opts.Since != "" {
		since, err := squashdates.ParseDate(Opts.Since)
		alog.BailIf(err)
		_dates := timeslice.TimeSlice{}
		for _, date := range dates {
			if date.After(since) || date == since {
				_dates = append(_dates, date)
			}
		}
		dates = _dates
	}

	totalDuration, mostRecent := squashdates.Squash(dates, Opts.Quiet)

	if Opts.Mech {
		alog.SetPrefix("")
		alog.SetOutput(os.Stdout)
		alog.Printf("%d\n", totalDuration.Nanoseconds()/1e9)
		if len(dates) > 0 {
			alog.Printf("%s\n", mostRecent.UTC().Format("2006-01-02T15:04:05Z"))
		}
	}
}
