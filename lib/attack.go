package stress

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

var (
	remain int64
)

// Attacker is an attack executor, wrapping an http.Client
type Attacker struct{ client http.Client }

// DefaultAttacker is the default Attacker used by Attack
var DefaultAttacker = NewAttacker()

// NewAttacker returns a pointer to a new Attacker
func NewAttacker() *Attacker {
	return &Attacker{http.Client{Transport: &defaultTransport}}
}

// Attack hits the passed Targets (http.Requests) at the rate specified for
// duration time and then waits for all the requests to come back.
// The results of the attack are put into a slice which is returned.
//
// Attack is a wrapper around DefaultAttacker.Attack
func AttackRate(tgts Targets, rate uint64, du time.Duration) Results {
	return DefaultAttacker.AttackRate(tgts, rate, du)
}

func AttackConcy(tgts Targets, concurrency uint64, number uint64) Results {
	return DefaultAttacker.AttackConcy(tgts, concurrency, number)
}

// Attack attacks the passed Targets (http.Requests) at the rate specified for
// duration time and then waits for all the requests to come back.
// The results of the attack are put into a slice which is returned.
func (a Attacker) AttackRate(tgts Targets, rate uint64, du time.Duration) Results {
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
					rsp_md5 := hex.EncodeToString(h.Sum(nil))
					if rsp_md5 != kv[1] {
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

	return res
}

// SetRedirects sets the max amount of redirects the attacker's http client
// will follow.
func (a *Attacker) SetRedirects(redirects int) {
	a.client.CheckRedirect = func(_ *http.Request, via []*http.Request) error {
		if len(via) > redirects {
			return fmt.Errorf("Stopped after %d redirects", redirects)
		}
		return nil
	}
}

// SetTimeout sets the client side timeout for each request the attacker makes.
func (a *Attacker) SetTimeout(timeout time.Duration) {
	tr := a.client.Transport.(*http.Transport)
	tr.ResponseHeaderTimeout = timeout
	a.client.Transport = tr
}

func (a Attacker) AttackConcy(tgts Targets, concurrency uint64, number uint64) Results {
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
	req_remain := atomic.LoadInt64(&remain)
	for req_remain > 0 {
		atomic.AddInt64(&remain, -1)
		var res Result
		tgt := tgts[int(req_remain)%len(tgts)]
		req, err := tgt.Request()
		if err != nil {
			res.Error = err.Error()
			results = append(results, res)
			req_remain = atomic.LoadInt64(&remain)
			continue
		}

		res.Timestamp = time.Now()
		r, err := a.client.Do(req)
		if err != nil {
			res.Error = err.Error()
			results = append(results, res)
			req_remain = atomic.LoadInt64(&remain)
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
			req_remain = atomic.LoadInt64(&remain)
			continue
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
						rsp_md5 := hex.EncodeToString(h.Sum(nil))
						if rsp_md5 != kv[1] {
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
		req_remain = atomic.LoadInt64(&remain)
	}
	return results
}

var defaultTransport = http.Transport{
	TLSClientConfig: &tls.Config{
		InsecureSkipVerify: true,
	},
}
