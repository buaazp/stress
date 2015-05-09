package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	stress "github.com/buaazp/stress/lib"
)

func attackCmd() command {
	fs := flag.NewFlagSet("stress attack", flag.ExitOnError)
	opts := &attackOpts{
		headers: headers{http.Header{}},
		laddr:   localAddr{&stress.DefaultLocalAddr},
	}

	fs.StringVar(&opts.targetsf, "targets", "stdin", "Targets file")
	fs.StringVar(&opts.outputf, "output", "result.json", "Output file")
	fs.StringVar(&opts.bodyf, "body", "", "Requests body file")
	fs.StringVar(&opts.ordering, "ordering", "random", "Attack ordering [sequential, random]")
	fs.DurationVar(&opts.duration, "duration", 10*time.Second, "Duration of the test")
	fs.DurationVar(&opts.timeout, "timeout", 0, "Requests timeout")
	fs.Uint64Var(&opts.rate, "rate", 0, "Requests per second")
	fs.Uint64Var(&opts.concurrency, "c", 0, "Concurrency level")
	fs.Uint64Var(&opts.number, "n", 1000, "Requests number")
	fs.IntVar(&opts.redirects, "redirects", 10, "Number of redirects to follow")
	fs.Var(&opts.headers, "header", "Request header")
	fs.Var(&opts.laddr, "laddr", "Local IP address")

	return command{fs, func(args []string) error {
		fs.Parse(args)
		return attack(opts)
	}}
}

// attackOpts aggregates the attack function command options
type attackOpts struct {
	targetsf    string
	outputf     string
	bodyf       string
	ordering    string
	timeout     time.Duration
	rate        uint64
	duration    time.Duration
	concurrency uint64
	number      uint64
	redirects   int
	headers     headers
	laddr       localAddr
}

// attack validates the attack arguments, sets up the
// required resources, launches the attack and writes the results
func attack(opts *attackOpts) error {
	if opts.rate == 0 && opts.concurrency == 0 {
		return fmt.Errorf(errRatePrefix + "or " + errConcurrencyPrefix + "can't be zero")
	} else if opts.rate != 0 && opts.concurrency != 0 {
		return fmt.Errorf(errRatePrefix + "is conflict with " + errConcurrencyPrefix)
	}

	if opts.rate != 0 && opts.duration == 0 {
		return fmt.Errorf(errDurationPrefix + "can't be zero")
	}

	if opts.concurrency != 0 && opts.number == 0 {
		return fmt.Errorf(errNumberPrefix + "can't be zero")
	}

	in, err := file(opts.targetsf, false)
	if err != nil {
		return fmt.Errorf(errTargetsFilePrefix+"(%s): %s", opts.targetsf, err)
	}
	defer in.Close()

	var body []byte
	if opts.bodyf != "" {
		bodyr, err := file(opts.bodyf, false)
		if err != nil {
			return fmt.Errorf(errBodyFilePrefix+"(%s): %s", opts.bodyf, err)
		}
		defer bodyr.Close()

		if body, err = ioutil.ReadAll(bodyr); err != nil {
			return fmt.Errorf(errBodyFilePrefix+"(%s): %s", opts.bodyf, err)
		}
	}

	targets, err := stress.NewTargetsFrom(in, body, opts.headers.Header)
	if err != nil {
		return fmt.Errorf(errTargetsFilePrefix+"(%s): %s", opts.targetsf, err)
	}
	if len(targets) == 0 {
		return fmt.Errorf(errTargetsFilePrefix+" : is empty")
	}

	switch opts.ordering {
	case "random":
		targets.Shuffle(time.Now().UnixNano())
	case "sequential":
		break
	default:
		return fmt.Errorf(errOrderingPrefix+"`%s` is invalid", opts.ordering)
	}

	out, err := file(opts.outputf, true)
	if err != nil {
		return fmt.Errorf(errOutputFilePrefix+"(%s): %s", opts.outputf, err)
	}
	defer out.Close()

	attacker := stress.NewAttacker(opts.redirects, opts.timeout, *opts.laddr.IPAddr)

	var results stress.Results
	if opts.rate != 0 {
		log.Printf(
			"Stress is attacking %d targets in %s order and %d rate for %s...\n",
			len(targets),
			opts.ordering,
			opts.rate,
			opts.duration,
		)
		results = attacker.AttackRate(targets, opts.rate, opts.duration)
	} else if opts.concurrency != 0 {
		concurrency := opts.concurrency
		if opts.concurrency > opts.number {
			concurrency = opts.number
		}
		log.Printf(
			"Stress is attacking %d targets in %s order and %d concurrency level for %d times...\n",
			len(targets),
			opts.ordering,
			concurrency,
			opts.number,
		)
		results = attacker.AttackConcy(targets, concurrency, opts.number)
	}

	log.Printf("Done! Writing results to '%s'...", opts.outputf)
	err = results.Encode(out)
	if err != nil {
		return err
	}

	data, err := stress.ReportText(results)
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(data)

	return err
}

const (
	errRatePrefix        = "Rate: "
	errDurationPrefix    = "Duration: "
	errConcurrencyPrefix = "Concurrency Level: "
	errNumberPrefix      = "Number: "
	errOutputFilePrefix  = "Output file: "
	errTargetsFilePrefix = "Targets file: "
	errBodyFilePrefix    = "Body file: "
	errOrderingPrefix    = "Ordering: "
	errReportingPrefix   = "Reporting: "
)

// headers is the http.Header used in each target request
// it is defined here to implement the flag.Value interface
// in order to support multiple identical flags for request header
// specification
type headers struct{ http.Header }

func (h headers) String() string {
	buf := &bytes.Buffer{}
	if err := h.Write(buf); err != nil {
		return ""
	}
	return buf.String()
}

func (h headers) Set(value string) error {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("Header '%s' has a wrong format", value)
	}
	key, val := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	if key == "" || val == "" {
		return fmt.Errorf("Header '%s' has a wrong format", value)
	}
	h.Add(key, val)
	return nil
}

// localAddr implements the Flag interface for parsing net.IPAddr
type localAddr struct{ *net.IPAddr }

func (ip *localAddr) Set(value string) (err error) {
	ip.IPAddr, err = net.ResolveIPAddr("ip", value)
	return
}
