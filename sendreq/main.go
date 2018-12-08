package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	httpstat "github.com/tcnksm/go-httpstat"
)

const logName = "sendreq"

var count int

func main() {
	var totalps int
	var first httpstat.Result

	// setup channels
	ch := make(chan httpstat.Result)
	killc := make(chan os.Signal, 1)
	signal.Notify(killc, os.Interrupt)

	// CLI flags
	endpoint := flag.String("endpoint", "", "HTTP requests endpoint")
	hostsFile := flag.String("hosts", "", "HTTP requests HOST header file")
	parallel := flag.Bool("parallel", true, "Send requests concurrently")
	iteration := flag.Int("iteration", -1, "Number of iterations over hosts file")
	timeout := flag.String("timeout", "0s", "HTTP requests timeout. Valid time units are \"ns\", \"us\" (or \"µs\"), \"ms\", \"s\", \"m\", \"h\"")
	flag.Parse()

	// Parse timeout
	timeoutDuration, err := time.ParseDuration(*timeout)
	if err != nil {
		log.Fatalf("[%s] ERROR: %s", logName, err.Error())
	}

	// Read hosts file and put them inside a string slice
	hosts, err := ReadHosts(*hostsFile)
	if err != nil {
		log.Fatalf("[%s] ERROR: %s", logName, err.Error())
	}

	now := time.Now()

	if *iteration == -1 {
		go func() {
			for range killc {
				log.Printf("[%s]: Total requests per second: %d", logName, totalps)
				log.Printf("[%s]: Total requests done: %d", logName, count)
				log.Printf("[%s]: Total time all requests took: %d ms", logName, int(time.Since(now)/time.Millisecond))
				firstReqLog(first)
				os.Exit(0)
			}
		}()
		for {
			// Send HTTP requests and set start time for tracing latency
			for _, host := range hosts {
				if !*parallel {
					result := SendReq(*endpoint, host, timeoutDuration)
					count++
					if count == 1 {
						first = result
					}
					continue
				}
				go SendConcurrentRequest(*endpoint, host, timeoutDuration, ch)
			}

			if *parallel {
				for range hosts {
					stat := <-ch
					if time.Since(now).Seconds() <= float64(1) {
						totalps = count
					}
					if count == 1 {
						first = stat
					}
				}
			}
		}
	}

	for i := 1; i <= *iteration; i++ {
		// Send HTTP requests and set start time for tracing latency
		for _, host := range hosts {
			if !*parallel {
				result := SendReq(*endpoint, host, timeoutDuration)
				count++
				if count == 1 {
					first = result
				}
				continue
			}
			go SendConcurrentRequest(*endpoint, host, timeoutDuration, ch)
		}
		if *parallel {
			for range hosts {
				stat := <-ch
				if time.Since(now).Seconds() <= float64(1) {
					totalps = count
				}
				if count == 1 {
					first = stat
				}
			}
		}
	}
	firstReqLog(first)

	log.Printf("[%s]: Total requests per second: %d", logName, totalps)
	log.Printf("[%s]: Total requests done: %d", logName, count)
	log.Printf("[%s]: Total time all requests took: %d ms", logName, int(time.Since(now)/time.Millisecond))
}

func ReadHosts(hosts string) ([]string, error) {
	var lines []string
	if hosts == "" {
		return nil, fmt.Errorf("Hosts file is empty. Please provide it with -hosts flag")
	}
	file, _ := os.Open(hosts)
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, nil
}

func SendConcurrentRequest(url, host string, timeout time.Duration, ch chan<- httpstat.Result) {
	result := SendReq(url, host, timeout)
	count++
	ch <- result
}

func SendReq(url, host string, timeout time.Duration) httpstat.Result {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Host", host)

	var result httpstat.Result
	ctx := httpstat.WithHTTPStat(req.Context(), &result)
	req = req.WithContext(ctx)

	client := http.DefaultClient
	if timeout.String() == "" {
		client.Timeout = timeout
	}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	res.Body.Close()

	return result
}

func firstReqLog(first httpstat.Result) {
	log.Printf("[%s]: <First Request> DNS lookup: %d ms", logName, int(first.DNSLookup/time.Millisecond))
	log.Printf("[%s]: <First Request> TCP connection: %d ms", logName, int(first.TCPConnection/time.Millisecond))
	log.Printf("[%s]: <First Request> TLS handshake: %d ms", logName, int(first.TLSHandshake/time.Millisecond))
	log.Printf("[%s]: <First Request> Server processing: %d ms", logName, int(first.ServerProcessing/time.Millisecond))
	log.Printf("[%s]: <First Request> Content transfer: %d ms", logName, int(first.ContentTransfer(time.Now())/time.Millisecond))
}
