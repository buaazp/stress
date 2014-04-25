stress
======

A test tool to send random http GET/POST requests to server.
Fork from [Vegeta](https://github.com/tsenart/vegeta).

Stress is a versatile HTTP load testing tool built out of need to drill
HTTP services with a constant request rate.
It can be used both as a command line utility and a library.

## Install
### Pre-compiled executables
Get them [here](https://github.com/buaazp/stress/releases).

### Source
You need go installed and `GOBIN` in your `PATH`. Once that is done, run the
command:

````
$ go get github.com/buaazp/stress
$ go install github.com/buaazp/stress
````

## Usage manual

````
$ stress -h
Usage: stress [globals] <command> [options]

attack command:
  -body="": Requests body file
  -duration=10s: Duration of the test
  -header=: Request header
  -ordering="random": Attack ordering [sequential, random]
  -output="stdout": Output file
  -rate=50: Requests per second
  -redirects=10: Number of redirects to follow
  -targets="stdin": Targets file
  -timeout=0: Requests timeout

report command:
  -input="stdin": Input files (comma separated)
  -output="stdout": Output file
  -reporter="text": Reporter [text, json, plot]

global flags:
  -cpus=8 Number of CPUs to use

examples:
  echo "GET HOST:ww2.sinaimg.cn resize-type:crop.100.100.200.200.100 http://127.0.0.1:8088/bmiddle/50caec1agw1ef9myz5zhoj21ck0yggv6.jpg" | stress attack -duration=5s -rate=100 | tee results.bin | stress report
  echo "POST http://127.0.0.1:12345/ form:filename:5f189.jpeg" | stress attack -duration=5s -rate=1 | tee results.bin | stress report
  stress attack -targets=targets.txt > results.bin
  stress report -input=results.bin -reporter=json > metrics.json
  cat results.bin | stress report -reporter=plot > plot.html
````

#### -cpus
Specifies the number of CPUs to be used internally.
It defaults to the amount of CPUs available in the system.

### attack

````
$ stress attack -h
Usage of stress attack:
  -body="": Requests body file
  -duration=10s: Duration of the test
  -header=: Request header
  -ordering="random": Attack ordering [sequential, random]
  -output="stdout": Output file
  -rate=50: Requests per second
  -redirects=10: Number of redirects to follow
  -targets="stdin": Targets file
  -timeout=0: Requests timeout
````

#### -body
Specifies the file whose content will be set as the body of every request.

#### -duration
Specifies the amount of time to issue request to the targets.
The internal concurrency structure's setup has this value as a variable.
The actual run time of the test can be longer than specified due to the
responses delay.

#### -header
Specifies a request header to be used in all targets defined.
You can specify as many as needed by repeating the flag.

#### -ordering
Specifies the ordering of target attack. The default is `random` and
it will randomly pick one of the targets per request without ever choosing
that target again.
The other option is `sequential` and it does what you would expect it to
do.

#### -output
Specifies the output file to which the binary results will be written
to. Made to be piped to the report command input. Defaults to stdout.

####  -rate
Specifies the requests per second rate to issue against
the targets. The actual request rate can vary slightly due to things like
garbage collection, but overall it should stay very close to the specified.

#### -redirects
Specifies the max number of redirects followed on each request. The
default is 10.

#### -targets
Specifies the attack targets in a line separated file, defaulting to stdin.
The format should be as follows.

````
Method [Header_key:Header_value ...] Url [[form:[filekey:]]BodyFile]
GET http://goku:9090/path/to/dragon?item=balls
GET http://user:password@goku:9090/path/to
HEAD http://goku:9090/path/to/success
POST http://127.0.0.1:4869/upload form:5f189.jpeg
POST http://127.0.0.1:12345/ form:filename:5f189.jpeg
GET http://127.0.0.1:4869/a87665d54a8c0dcaab04fa88b323eba1
GET HOST:ww2.sinaimg.cn resize-type:crop.100.100.200.200.100 http://127.0.0.1:8088/bmiddle/50caec1agw1ef9myz5zhoj21ck0yggv6.jpg
...
````

#### -timeout
Specifies the timeout for each request. The default is 0 which disables
timeouts.
### report
````
$ stress report -h
Usage of stress report:
  -input="stdin": Input files (comma separated)
  -output="stdout": Output file
  -reporter="text": Reporter [text, json, plot]
````

#### -input
Specifies the input files to generate the report of, defaulting to stdin.
These are the output of stress attack. You can specify more than one (comma
separated) and they will be merged and sorted before being used by the
reports.

#### -output
Specifies the output file to which the report will be written to.

#### -reporter
Specifies the kind of report to be generated. It defaults to text.

##### text
````
Requests      [total]                   1200
Duration      [total]                   1.998307684s
Latencies     [mean, 50, 95, 99, max]   223.340085ms, 240.12234ms, 326.913687ms, 416.537743ms, 7.788103259s
Bytes In      [total, mean]             3714690, 3095.57
Bytes Out     [total, mean]             0, 0.00
Success       [ratio]                   55.42%
Status Codes  [code:count]              0:535  200:665
Error Set:
Get http://localhost:6060: dial tcp 127.0.0.1:6060: connection refused
Get http://localhost:6060: read tcp 127.0.0.1:6060: connection reset by peer
Get http://localhost:6060: dial tcp 127.0.0.1:6060: connection reset by peer
Get http://localhost:6060: write tcp 127.0.0.1:6060: broken pipe
Get http://localhost:6060: net/http: transport closed before response was received
Get http://localhost:6060: http: can't write HTTP request on broken connection
````

##### json
````
{
  "latencies": {
    "mean": 9093653647,
    "50th": 2401223400,
    "95th": 12553709381,
    "99th": 12604629125,
    "max": 12604629125
  },
  "bytes_in": {
    "total": 782040,
    "mean": 651.7
  },
  "bytes_out": {
    "total": 0,
    "mean": 0
  },
  "duration": 1998307684,
  "requests": 1200,
  "success": 0.11666666666666667,
  "status_codes": {
    "0": 1060,
    "200": 140
  },
  "errors": [
    "Get http://localhost:6060: dial tcp 127.0.0.1:6060: operation timed out"
  ]
}
````
##### plot
Generates an HTML5 page with an interactive plot based on
[Dygraphs](http://dygraphs.com).
Click and drag to select a region to zoom into. Double click to zoom
out.
Input a different number on the bottom left corner input field
to change the moving average window size (in data points).

![Plot](https://dl.dropboxusercontent.com/u/83217940/plot.png)


## Usage (Library)
```go
package main

import (
  stress "github.com/buaazp/stress/lib"
  "time"
  "fmt"
)

func main() {
  targets, _ := stress.NewTargets([]string{"GET http://localhost:9100/"})
  rate := uint64(100) // per second
  duration := 4 * time.Second

  results := stress.Attack(targets, rate, duration)
  metrics := stress.NewMetrics(results)

  fmt.Printf("Mean latency: %s", metrics.Latencies.Mean)
}
```

#### Limitations
There will be an upper bound of the supported `rate` which varies on the
machine being used.
You could be CPU bound (unlikely), memory bound (more likely) or
have system resource limits being reached which ought to be tuned for
the process execution. The important limits for us are file descriptors
and processes. On a UNIX system you can get and set the current
soft-limit values for a user.

````
$ ulimit -n # file descriptors
2560
$ ulimit -u # processes / threads
709
````
Just pass a new number as the argument to change it.

