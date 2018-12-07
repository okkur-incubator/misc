package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	httpstat "github.com/tcnksm/go-httpstat"
)

const logFormat = "02/Jan/2006:15:04:05 -0700"

func main() {
	ch := make(chan string)

	// CLI flags
	endpoint := flag.String("endpoint", "", "HTTP requests endpoint")
	hostsFile := flag.String("hosts", "", "HTTP requests HOST header file")
	// parallel := flag.Int("parallel", 1, "Number of parallel requests")
	// iteration := flag.Int("iteration", -1, "Number of iterations over hosts file")
	// timeout := flag.Int("timeout", 0, "HTTP requests timeout")
	flag.Parse()

	// Read hosts file and put them inside a string slice
	hosts, err := ReadHosts(*hostsFile)
	if err != nil {
		log.Fatalf("<%s> ERROR: %s", time.Now().Format(logFormat), err.Error())
	}

	for _, host := range hosts {
		go SendRequest(*endpoint, host, ch)
	}

	for range hosts {
		fmt.Println(<-ch)
	}
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

func SendRequest(url string, host string, ch chan<- string) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Host", host)

	var result httpstat.Result
	ctx := httpstat.WithHTTPStat(req.Context(), &result)
	req = req.WithContext(ctx)

	client := http.DefaultClient
	start := time.Now()
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	secs := time.Since(start).Seconds()

	body, _ := ioutil.ReadAll(res.Body)
	res.Body.Close()

	ch <- fmt.Sprintf("%.2f elapsed with response length: %d %s", secs, len(body), url)
}
