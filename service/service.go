package service

import (
	"fmt"
	"fpt-util/test-alive-server/constant"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	hitNONE        = 0
	hitPING        = 1
	hitHTTPGet     = 2
	hitHTTPSGet    = 3
	hitDialTimeout = 4
)

// TestConnection - test connection service
func TestConnection(ips []string, hostnames []string) {
	var wg sync.WaitGroup
	siteIP := []string{}
	siteHostname := []string{}

	siteIP = append(siteIP, ips...)
	siteHostname = append(siteHostname, hostnames...)

	chanAlive := make(chan string, len(hostnames))
	chanDead := make(chan string, len(hostnames))
	chanDialTimeout := make(chan string, len(hostnames))

	// execute test concurrently
	for i := range siteIP {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			execute(chanAlive, chanDead, chanDialTimeout, siteIP[i], siteHostname[i])
		}(i)
	}

	// wait and close channels
	wg.Wait()
	close(chanAlive)
	close(chanDead)
	close(chanDialTimeout)

	aliveCnt := len(chanAlive)
	deadCnt := len(chanDead)
	dialTimeoutCnt := len(chanDialTimeout)

	// report alive
	fmt.Println("\n\n========== TEST REPORT (by IP) ==========")
	fmt.Println(fmt.Sprintf("[ ALIVE (can be PINGed): %d ] in no particular order...", aliveCnt))

	i := 0
	for alive := range chanAlive {
		i++
		fmt.Println(fmt.Sprintf("%d. %s", i, alive))
	}

	// report dead
	fmt.Println(fmt.Sprintf("\n[ DEAD (server is up but cannot connect): %d ] in no particular order ...", dialTimeoutCnt))

	i = 0
	for dead := range chanDialTimeout {
		i++
		fmt.Println(fmt.Sprintf("%d. %s", i, dead))
	}

	// report dead
	fmt.Println(fmt.Sprintf("\n[ DEAD (total dead): %d ] in no particular order ...", deadCnt))

	i = 0
	for dead := range chanDead {
		i++
		fmt.Println(fmt.Sprintf("%d. %s", i, dead))
	}

	// report dead
	fmt.Println(fmt.Sprintf("\n\n========= TOTAL (tested %d hosts) ========== \nALIVE (PING + other methods): %d\nDEAD (cannot be accessed): %d", len(ips), aliveCnt, deadCnt+dialTimeoutCnt))
}

func checkByHTTP(isSecured bool, hostname string) (err error) {
	var prefix string

	if isSecured {
		prefix = "http://"
	} else {
		prefix = "https://"
	}

	_, err = http.Get(prefix + hostname)
	return err
}

func checkDialTimeout(hostname string, port string) (err error) {
	timeout := 1 * time.Second
	_, err = net.DialTimeout("tcp", hostname+":"+port, timeout)
	return err
}

func execute(chanAlive, chanDead, chanDialTimeout chan<- string, ip string, hostname string) {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("[%s] (%s): ", ip, hostname))

	// run concurrent test
	hitCode := concurrentTest(ip, hostname)

	switch hitCode {
	case hitNONE:
		result.WriteString("[PING ✘] [HTTP GET ✘] [HTTPS GET ✘] [Dial Timeout ✘]")
		chanDead <- result.String()
	case hitPING:
		result.WriteString("[PING ✔] ")
		chanAlive <- result.String()
	case hitHTTPGet:
		result.WriteString("[HTTP GET ✔] ")
		chanAlive <- result.String()
	case hitHTTPSGet:
		result.WriteString("[HTTPS GET ✔] ")
		chanAlive <- result.String()
	case hitDialTimeout:
		result.WriteString("[Dial Timeout ✔] ")
		chanDialTimeout <- result.String()
	}

	fmt.Println(result.String())
}

func concurrentTest(ip string, hostname string) int {
	result := hitNONE

	// pipe channels
	pingPipe := make(chan bool)
	httpPipe := make(chan bool)
	httpsPipe := make(chan bool)

	// check by PING
	go func(pingPipe chan<- bool) {
		defer close(pingPipe)

		out, _ := exec.Command("ping", ip).Output() //, "-c 5", "-i 3", "-w 10"
		if !(strings.Contains(string(out), "Destination Host Unreachable") || strings.Contains(string(out), "Request timed out.")) {
			pingPipe <- true
		}
		pingPipe <- false
	}(pingPipe)

	// check by HTTP GET
	go func(httpPipe chan<- bool) {
		defer close(httpPipe)

		err := checkByHTTP(constant.APPLYHTTP, hostname)
		if err == nil {
			httpPipe <- true
		}
		httpPipe <- false
	}(httpPipe)

	// check by HTTPS GET
	go func(httpsPipe chan<- bool) {
		defer close(httpsPipe)

		err := checkByHTTP(constant.APPLYHTTPS, hostname)
		if err == nil {
			httpsPipe <- true
		}
		httpsPipe <- false
	}(httpsPipe)

DoneFlagTrue:
	for pingPipe != nil || httpPipe != nil || httpsPipe != nil {
		select {
		case p, ok := <-pingPipe:
			if !ok {
				pingPipe = nil
				continue
			}
			if p {
				result = hitPING
				break DoneFlagTrue
			}
		case h, ok := <-httpPipe:
			if !ok {
				httpPipe = nil
				continue
			}
			if h {
				result = hitHTTPGet
				break DoneFlagTrue
			}
		case hs, ok := <-httpsPipe:
			if !ok {
				httpsPipe = nil
				continue
			}
			if hs {
				result = hitHTTPSGet
				break DoneFlagTrue
			}
		}
	}

	// check by DIAL TIME OUT
	if result == hitNONE {
		err := checkDialTimeout(hostname, "80")
		if err == nil {
			return hitDialTimeout
		}
	}

	return result
}
