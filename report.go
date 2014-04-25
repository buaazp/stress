package main

import (
	"flag"
	"log"
	"strings"

	stress "github.com/buaazp/stress/lib"
)

func reportCmd() command {
	fs := flag.NewFlagSet("stress report", flag.ExitOnError)
	opts := &reportOpts{}

	fs.StringVar(&opts.reporter, "reporter", "text", "Reporter [text, json, plot]")
	fs.StringVar(&opts.inputf, "input", "stdin", "Input files (comma separated)")
	fs.StringVar(&opts.outputf, "output", "stdout", "Output file")

	return command{fs, func(args []string) error {
		fs.Parse(args)
		return report(opts)
	}}
}

// reportOpts aggregates the report function command options
type reportOpts struct {
	reporter string
	inputf   string
	outputf  string
}

// report validates the report arguments, sets up the required resources
// and writes the report
func report(opts *reportOpts) error {
	rep, ok := reporters[opts.reporter]
	if !ok {
		log.Println("Reporter provided is not supported. Using text")
		rep = stress.ReportText
	}

	var all stress.Results
	for _, input := range strings.Split(opts.inputf, ",") {
		in, err := file(input, false)
		if err != nil {
			return err
		}

		var results stress.Results
		if err = results.Decode(in); err != nil {
			return err
		}
		in.Close()

		all = append(all, results...)
	}
	all.Sort()

	out, err := file(opts.outputf, true)
	if err != nil {
		return err
	}
	defer out.Close()

	data, err := rep(all)
	if err != nil {
		return err
	}
	_, err = out.Write(data)

	return err
}

var reporters = map[string]stress.Reporter{
	"text": stress.ReportText,
	"json": stress.ReportJSON,
	"plot": stress.ReportPlot,
}
