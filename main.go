package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
)

func main() {
	commands := map[string]command{"attack": attackCmd(), "report": reportCmd()}

	flag.Usage = func() {
		fmt.Println("Usage: stress [globals] <command> [options]")
		for name, cmd := range commands {
			fmt.Printf("\n%s command:\n", name)
			cmd.fs.PrintDefaults()
		}
		fmt.Printf("\nglobal flags:\n  -cpus=%d Number of CPUs to use\n", runtime.NumCPU())
		fmt.Println(examples)
	}

	cpus := flag.Int("cpus", runtime.NumCPU(), "Number of CPUs to use")
	flag.Parse()

	runtime.GOMAXPROCS(*cpus)

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	if cmd, ok := commands[args[0]]; !ok {
		log.Fatalf("Unknown command: %s", args[0])
	} else if err := cmd.fn(args[1:]); err != nil {
		log.Fatal(err)
	}
}

const examples = `
examples:
  echo "GET HOST:ww2.sinaimg.cn resize-type:crop.100.100.200.200.100 http://127.0.0.1:8088/bmiddle/50caec1agw1ef9myz5zhoj21ck0yggv6.jpg" | stress attack -duration=5s -rate=100 | tee results.bin | stress report
  echo "POST http://127.0.0.1:12345/ form:filename:5f189.jpeg" | stress attack -duration=5s -rate=1 | tee results.bin | stress report
  stress attack -targets=targets.txt > results.bin
  stress report -input=results.bin -reporter=json > metrics.json
  cat results.bin | stress report -reporter=plot > plot.html
`

type command struct {
	fs *flag.FlagSet
	fn func(args []string) error
}
