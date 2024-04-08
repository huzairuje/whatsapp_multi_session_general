package crontab

// original crontab code is from
// https://github.com/mileusna/crontab/blob/master/crontab.go
// https://github.com/mileusna/crontab
// usage:
// ctab := crontab.New() // create cron table
//
// // AddJob and test the errors
// err := ctab.AddJob("0 12 1 * *", myFunc) // on 1st day of month
// if err != nil {
//   log.Println(err)
//   return
// }
//
// // MustAddJob is like AddJob but panics on wrong syntax or problems with func/args
// // This approach is similar to regexp.Compile and regexp.MustCompile from go's standard library,  used for easier initialization on startup
// ctab.MustAddJob("* * * * *", myFunc) // every minute
// ctab.MustAddJob("0 12 * * *", myFunc3) // noon launch
//
// // fn with args
// ctab.MustAddJob("0 0 * * 1,2", myFunc2, "Monday and Tuesday midnight", 123)
// ctab.MustAddJob("*/5 * * * *", myFunc2, "every five min", 0)
//
// // all your other app code as usual, or put sleep timer for demo
// // time.Sleep(10 * time.Minute)
// }
//
// func myFunc() {
//  	fmt.Println("Hello, world")
// }
//
// func myFunc3() {
//		fmt.Println("Noon!")
// }
//
// func myFunc2(s string, n int) {
//		fmt.Println("We have params here, string", s, "and number", n)
// }

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Crontab struct representing cron table
type Crontab struct {
	ticker *time.Ticker
	jobs   []*job
	sync.RWMutex
}

// job in cron table
type job struct {
	min       map[int]struct{}
	hour      map[int]struct{}
	day       map[int]struct{}
	month     map[int]struct{}
	dayOfWeek map[int]struct{}

	fn   interface{}
	args []interface{}
	sync.RWMutex
}

// tick is individual tick that occurs each minute
type tick struct {
	min       int
	hour      int
	day       int
	month     int
	dayOfWeek int
}

// New initializes and returns new cron table
func New() *Crontab {
	return run(time.Minute)
}

// new creates new crontab, arg provided for testing purpose
func run(t time.Duration) *Crontab {
	c := &Crontab{
		ticker: time.NewTicker(t),
		jobs:   []*job{},
	}

	go func() {
		for t := range c.ticker.C {
			c.runScheduled(t)
		}
	}()

	return c
}

// AddJob to cron table
//
// Returns error if:
//
// * Cron syntax can't be parsed or out of bounds
//
// * fn is not function
//
// * Provided args don't match the number and/or the type of fn args
func (c *Crontab) AddJob(schedule string, fn interface{}, args ...interface{}) error {
	j, err := parseSchedule(schedule)
	c.Lock()
	defer c.Unlock()
	if err != nil {
		return err
	}

	if fn == nil || reflect.ValueOf(fn).Kind() != reflect.Func {
		return fmt.Errorf("cron job must be func()")
	}

	fnType := reflect.TypeOf(fn)
	if len(args) != fnType.NumIn() {
		return fmt.Errorf("number of func() params and number of provided params doesn't match")
	}

	for i := 0; i < fnType.NumIn(); i++ {
		a := args[i]
		t1 := fnType.In(i)
		t2 := reflect.TypeOf(a)

		if t1 != t2 {
			if t1.Kind() != reflect.Interface {
				return fmt.Errorf("param with index %d shold be `%s` not `%s`", i, t1, t2)
			}
			if !t2.Implements(t1) {
				return fmt.Errorf("param with index %d of type `%s` doesn't implement interface `%s`", i, t2, t1)
			}
		}
	}

	// all checked, add job to cron tab
	j.fn = fn
	j.args = args
	c.jobs = append(c.jobs, j)
	return nil
}

// MustAddJob is like AddJob but panics if there is a problem with job
//
// It simplifies initialization, since we usually add jobs at the beginning, so you won't have to check for errors (it will panic when program starts).
// It is a similar approach as go's std lib package `regexp` and `regexp.Compile()` `regexp.MustCompile()`
// MustAddJob will panic if:
//
// * Cron syntax can't be parsed or out of bounds
//
// * fn is not function
//
// * Provided args don't match the number and/or the type of fn args
func (c *Crontab) MustAddJob(schedule string, fn interface{}, args ...interface{}) {
	if err := c.AddJob(schedule, fn, args...); err != nil {
		panic(err)
	}
}

// Shutdown the cron table schedule
//
// Once stopped, it can't be restarted.
// This function is pre-shutdown helper for your app, there is no Start/Stop functionality with crontab package.
func (c *Crontab) Shutdown() {
	c.ticker.Stop()
}

// Clear all jobs from cron table
func (c *Crontab) Clear() {
	c.Lock()
	c.jobs = []*job{}
	c.Unlock()
}

// RunAll jobs in cron table, scheduled or not
func (c *Crontab) RunAll() {
	c.RLock()
	defer c.RUnlock()
	for _, j := range c.jobs {
		go j.run()
	}
}

// RunScheduled jobs
func (c *Crontab) runScheduled(t time.Time) {
	tick := getTick(t)
	c.RLock()
	defer c.RUnlock()

	for _, j := range c.jobs {
		if j.tick(tick) {
			go j.run()
		}
	}
}

// run the job using reflection
// Recover from panic although all functions and params are checked by AddJob, but you never know.
func (j *job) run() {
	j.RLock()
	defer func() {
		if r := recover(); r != nil {
			log.Println("Crontab error", r)
		}
	}()
	v := reflect.ValueOf(j.fn)
	rargs := make([]reflect.Value, len(j.args))
	for i, a := range j.args {
		rargs[i] = reflect.ValueOf(a)
	}
	j.RUnlock()
	v.Call(rargs)
}

// tick decides should the job be launched at the tick
func (j *job) tick(t tick) bool {
	j.RLock()
	defer j.RUnlock()
	if _, ok := j.min[t.min]; !ok {
		return false
	}

	if _, ok := j.hour[t.hour]; !ok {
		return false
	}

	// cumulative day and dayOfWeek, as it should be
	_, day := j.day[t.day]
	_, dayOfWeek := j.dayOfWeek[t.dayOfWeek]
	if !day && !dayOfWeek {
		return false
	}

	if _, ok := j.month[t.month]; !ok {
		return false
	}

	return true
}

// regexps for parsing schedule string
var (
	matchSpaces = regexp.MustCompile(`\s+`)
	matchN      = regexp.MustCompile(`(.*)/(\d+)`)
	matchRange  = regexp.MustCompile(`^(\d+)-(\d+)$`)
)

// parseSchedule string and creates job struct with filled times to launch, or error if syntax is wrong
func parseSchedule(s string) (*job, error) {
	var err error
	j := &job{}
	j.Lock()
	defer j.Unlock()
	s = matchSpaces.ReplaceAllLiteralString(s, " ")
	parts := strings.Split(s, " ")
	if len(parts) != 5 {
		return j, errors.New("schedule string must have five components like * * * * *")
	}

	j.min, err = parsePart(parts[0], 0, 59)
	if err != nil {
		return j, err
	}

	j.hour, err = parsePart(parts[1], 0, 23)
	if err != nil {
		return j, err
	}

	j.day, err = parsePart(parts[2], 1, 31)
	if err != nil {
		return j, err
	}

	j.month, err = parsePart(parts[3], 1, 12)
	if err != nil {
		return j, err
	}

	j.dayOfWeek, err = parsePart(parts[4], 0, 6)
	if err != nil {
		return j, err
	}

	//  day/dayOfWeek combination
	switch {
	case len(j.day) < 31 && len(j.dayOfWeek) == 7: // day set, but not dayOfWeek, clear dayOfWeek
		j.dayOfWeek = make(map[int]struct{})
	case len(j.dayOfWeek) < 7 && len(j.day) == 31: // dayOfWeek set, but not day, clear day
		j.day = make(map[int]struct{})
	default:
		// both day and dayOfWeek are * or both are set, use combined
		// i.e. don't do anything here
	}

	return j, nil
}

// parsePart parse individual schedule part from schedule string
func parsePart(s string, min, max int) (map[int]struct{}, error) {

	r := make(map[int]struct{})

	// wildcard pattern
	if s == "*" {
		for i := min; i <= max; i++ {
			r[i] = struct{}{}
		}
		return r, nil
	}

	// */2 1-59/5 pattern
	if matches := matchN.FindStringSubmatch(s); matches != nil {
		localMin := min
		localMax := max
		if matches[1] != "" && matches[1] != "*" {
			if rng := matchRange.FindStringSubmatch(matches[1]); rng != nil {
				localMin, _ = strconv.Atoi(rng[1])
				localMax, _ = strconv.Atoi(rng[2])
				if localMin < min || localMax > max {
					return nil, fmt.Errorf("out of range for %s in %s. %s must be in range %d-%d", rng[1], s, rng[1], min, max)
				}
			} else {
				return nil, fmt.Errorf("unable to parse %s part in %s", matches[1], s)
			}
		}
		n, _ := strconv.Atoi(matches[2])
		for i := localMin; i <= localMax; i += n {
			r[i] = struct{}{}
		}
		return r, nil
	}

	// 1,2,4  or 1,2,10-15,20,30-45 pattern
	parts := strings.Split(s, ",")
	for _, x := range parts {
		if rng := matchRange.FindStringSubmatch(x); rng != nil {
			localMin, _ := strconv.Atoi(rng[1])
			localMax, _ := strconv.Atoi(rng[2])
			if localMin < min || localMax > max {
				return nil, fmt.Errorf("out of range for %s in %s. %s must be in range %d-%d", x, s, x, min, max)
			}
			for i := localMin; i <= localMax; i++ {
				r[i] = struct{}{}
			}
		} else if i, err := strconv.Atoi(x); err == nil {
			if i < min || i > max {
				return nil, fmt.Errorf("out of range for %d in %s. %d must be in range %d-%d", i, s, i, min, max)
			}
			r[i] = struct{}{}
		} else {
			return nil, fmt.Errorf("unable to parse %s part in %s", x, s)
		}
	}

	if len(r) == 0 {
		return nil, fmt.Errorf("unable to parse %s", s)
	}

	return r, nil
}

// getTick returns the tick struct from time
func getTick(t time.Time) tick {
	return tick{
		min:       t.Minute(),
		hour:      t.Hour(),
		day:       t.Day(),
		month:     int(t.Month()),
		dayOfWeek: int(t.Weekday()),
	}
}
