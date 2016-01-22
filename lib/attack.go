package stress

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

var (
	remain int64
)

// Attacker is an attack executor which wraps an http.Client
type Attacker struct{ client http.Client }

var (
	// DefaultRedirects represents the number of times the DefaultAttacker
	// follows redirects
	DefaultRedirects = 10
	// DefaultTimeout represents the amount of time the DefaultAttacker waits
	// for a request before it times out
	DefaultTimeout = 30 * time.Second
	// DefaultLocalAddr is the local IP address the DefaultAttacker uses in its
	// requests
	DefaultLocalAddr = net.IPAddr{IP: net.IPv4zero}
)

// DefaultAttacker is the default Attacker used by Attack
var DefaultAttacker = NewAttacker(DefaultRedirects, DefaultTimeout, DefaultLocalAddr)

// NewAttacker returns a pointer to a new Attacker
//
// redirects is the max amount of redirects the attacker will follow.
// Use DefaultRedirects for a sensible default.
//
// timeout is the client side timeout for each request.
// Use DefaultTimeout for a sensible default.
//
// laddr is the local IP address used for each request.
// Use DefaultLocalAddr for a sensible default.
func NewAttacker(redirects int, timeout time.Duration, laddr net.IPAddr) *Attacker {
	return &Attacker{http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   timeout,
				KeepAlive: 30 * time.Second,
				LocalAddr: &net.TCPAddr{IP: laddr.IP, Zone: laddr.Zone},
			}).Dial,
			ResponseHeaderTimeout: timeout,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			TLSHandshakeTimeout: 10 * time.Second,
		},
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) > redirects {
				return fmt.Errorf("stopped after %d redirects", redirects)
			}
			return nil
		},
	}}
}

// AttackRate hits the passed Targets (http.Requests) at the rate specified for
// duration time and then waits for all the requests to come back.
// The results of the attackrate are put into a slice which is returned.
//
// AttackRate is a wrapper around DefaultAttacker.Attack
func AttackRate(tgts Targets, rate uint64, du time.Duration) Results {
	return DefaultAttacker.AttackRate(tgts, rate, du)
}

// AttackRate attacks the passed Targets (http.Requests) at the rate specified for
// duration time and then waits for all the requests to come back.
// The results of the attack are put into a slice which is returned.
func (a *Attacker) AttackRate(tgts Targets, rate uint64, du time.Duration) Results {
	hits := int(rate * uint64(du.Seconds()))
	resc := make(chan Result)
	throttle := time.NewTicker(time.Duration(1e9 / rate))
	defer throttle.Stop()

	for i := 0; i < hits; i++ {
		<-throttle.C
		go func(tgt Target) { resc <- a.hit(tgt) }(tgts[i%len(tgts)])
	}
	results := make(Results, 0, hits)
	for len(results) < cap(results) {
		results = append(results, <-resc)
	}

	return results.Sort()
}

func (a *Attacker) hit(tgt Target) (res Result) {
	req, err := tgt.Request()
	if err != nil {
		res.Error = err.Error()
		return res
	}

	res.Timestamp = time.Now()
	r, err := a.client.Do(req)
	if err != nil {
		res.Error = err.Error()
		return res
	}

	res.BytesOut = uint64(req.ContentLength)
	res.Code = uint16(r.StatusCode)
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		if res.Code >= 300 || res.Code < 200 {
			res.Error = fmt.Sprintf("%s %s: %s", tgt.Method, tgt.URL, r.Status)
		}
		return res
	}

	res.Latency = time.Since(res.Timestamp)
	res.BytesIn = uint64(len(body))
	if res.Code >= 300 || res.Code < 200 {
		res.Error = fmt.Sprintf("%s %s: %s", tgt.Method, tgt.URL, r.Status)
	} else {
		if strings.Contains(tgt.File, "md5") {
			//fmt.Printf("checking [%s]\n", tgt.File)
			kv := strings.Split(tgt.File, ":")
			/*
				for k, v := range kv {
					fmt.Printf("[%d]: %s\n", k, v)
				}
			*/
			if len(kv) == 2 {
				if kv[1] != "" && len(kv[1]) == 32 {
					//fmt.Println("checking [%s]\n", kv[1])
					h := md5.New()
					h.Write(body)
					rspMd5 := hex.EncodeToString(h.Sum(nil))
					if rspMd5 != kv[1] {
						//fmt.Println("md5 not match!")
						res.Code = 250
						res.Error = fmt.Sprintf("%s %s: MD5 not matced", tgt.Method, tgt.URL)
					}
				}
			}
		}
	}
	r.Body.Close()
	if res.Code >= 250 || res.Code < 200 {
		log.Printf("%s\n", res.Error)
	}

	return res
}

// AttackConcy shoots the passed Targets (http.Requests) at the concurrency level
// specified for times and then waits for all the requests to come back.
// The results of the AttackConcy are put into a slice which is returned.
//
// AttackConcy is a wrapper around DefaultAttacker.Attack
func AttackConcy(tgts Targets, concurrency uint64, number uint64) Results {
	return DefaultAttacker.AttackConcy(tgts, concurrency, number)
}

// AttackConcy attacks the passed Targets (http.Requests) at the concurrency level
// specified for times and then waits for all the requests to come back.
// The results of the AttackConcy are put into a slice which is returned.
func (a *Attacker) AttackConcy(tgts Targets, concurrency uint64, number uint64) Results {
	retsc := make(chan Results)
	atomic.StoreInt64(&remain, int64(number))

	if concurrency > number {
		concurrency = number
	}

	var i uint64
	for i = 0; i < concurrency; i++ {
		go func(tgts Targets) { retsc <- a.shoot(tgts) }(tgts)
	}
	results := make(Results, 0, number)
	for i = 0; i < concurrency; i++ {
		results = append(results, <-retsc...)
	}
	return results.Sort()
}

func (a *Attacker) shoot(tgts Targets) Results {
	results := make(Results, 0, 1)
	reqRemain := atomic.LoadInt64(&remain)
	for reqRemain > 0 {
		atomic.AddInt64(&remain, -1)
		var res Result
		tgt := tgts[int(reqRemain)%len(tgts)]
		req, err := tgt.Request()
		if err != nil {
			res.Error = err.Error()
			results = append(results, res)
			reqRemain = atomic.LoadInt64(&remain)
			continue
		}

		res.Timestamp = time.Now()
		r, err := a.client.Do(req)
		if err != nil {
			res.Error = err.Error()
			results = append(results, res)
			reqRemain = atomic.LoadInt64(&remain)
			continue
		}

		res.BytesOut = uint64(req.ContentLength)
		res.Code = uint16(r.StatusCode)
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			if res.Code >= 300 || res.Code < 200 {
				res.Error = fmt.Sprintf("%s %s: %s", tgt.Method, tgt.URL, r.Status)
			}
			results = append(results, res)
			reqRemain = atomic.LoadInt64(&remain)
			continue
		}

		res.Latency = time.Since(res.Timestamp)
		res.BytesIn = uint64(len(body))
		r.Body.Close()
		if res.Code >= 300 || res.Code < 200 {
			res.Error = fmt.Sprintf("%s %s: %s", tgt.Method, tgt.URL, r.Status)
		} else {
			if strings.Contains(tgt.File, "md5") {
				//fmt.Printf("checking [%s]\n", tgt.File)
				kv := strings.Split(tgt.File, ":")
				/*
					for k, v := range kv {
						fmt.Printf("[%d]: %s\n", k, v)
					}
				*/
				if len(kv) == 2 {
					if kv[1] != "" && len(kv[1]) == 32 {
						//fmt.Println("checking [%s]\n", kv[1])
						h := md5.New()
						h.Write(body)
						rspMd5 := hex.EncodeToString(h.Sum(nil))
						if rspMd5 != kv[1] {
							//fmt.Println("md5 not match!")
							res.Code = 250
							res.Error = fmt.Sprintf("%s %s: MD5 not matced", tgt.Method, tgt.URL)
						}
					}
				}
			}
		}
		if res.Code >= 250 || res.Code < 200 {
			log.Printf("%s\n", res.Error)
		}

		results = append(results, res)
		reqRemain = atomic.LoadInt64(&remain)
	}
	return results
}

var defaultTransport = http.Transport{
	TLSClientConfig: &tls.Config{
		InsecureSkipVerify: true,
	},
}
