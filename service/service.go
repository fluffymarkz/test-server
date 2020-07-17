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

var wg sync.WaitGroup
var mut sync.Mutex

// TestConnection - test connection service
func TestConnection(ips []string, hostnames []string) {
	siteIP := []string{}
	siteHostname := []string{}

	siteIP = append(siteIP, ips...)
	siteHostname = append(siteHostname, hostnames...)

	chanAlive := make(chan string, len(hostnames))
	chanDead := make(chan string, len(hostnames))
	chanPingFail := make(chan string, len(hostnames))
	chanDialTimeout := make(chan string, len(hostnames))

	for i := range siteIP {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			executeTest(chanAlive, chanDead, chanPingFail, chanDialTimeout, siteIP[i], siteHostname[i])
		}(i)
	}

	wg.Wait()
	close(chanAlive)
	close(chanDead)
	close(chanPingFail)
	close(chanDialTimeout)

	fmt.Println("\n\n========== TEST REPORT (by IP) ==========")
	aliveCnt := len(chanAlive)
	deadCnt := len(chanDead)
	noPingCnt := len(chanPingFail)
	dialTimeoutCnt := len(chanDialTimeout)

	// report alive
	fmt.Println(fmt.Sprintf("[ ALIVE (can be PINGed): %d ] in no particular order...", aliveCnt))

	i := 0
	for alive := range chanAlive {
		i++
		fmt.Println(fmt.Sprintf("%d. %s", i, alive))
	}

	// report ping failed
	fmt.Println(fmt.Sprintf("\n[ ALIVE (PING fail but active): %d ] in no particular order...", noPingCnt))

	i = 0
	for alive := range chanPingFail {
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
	fmt.Println(fmt.Sprintf("\n\n========= TOTAL (tested %d hosts) ========== \nALIVE (PING + other methods): %d\nDEAD (cannot be accessed): %d", len(ips), aliveCnt, deadCnt))
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

func executeTest(chanAlive chan string, chanDead chan string, chanPingFail chan string, chanDialTimeout chan string, ip string, hostname string) {
	// defer wg.Done()
	var result strings.Builder

	out, _ := exec.Command("ping", ip).Output() //, "-c 5", "-i 3", "-w 10"
	result.WriteString(fmt.Sprintf("[%s] (%s): ", ip, hostname))
	if !func() bool {
		// check by PING
		if !(strings.Contains(string(out), "Destination Host Unreachable") || strings.Contains(string(out), "Request timed out.")) {
			result.WriteString("[PING ✔] ")
			return true
		}
		result.WriteString("[PING ✘] ")

		// check by HTTP GET
		err := checkByHTTP(constant.APPLYHTTP, hostname)
		if err == nil {
			result.WriteString("[HTTP GET ✔] ")
			return true
		}
		result.WriteString("[HTTP GET ✘] ")

		// check by HTTPS GET
		err = checkByHTTP(constant.APPLYHTTPS, hostname)
		if err == nil {
			result.WriteString("[HTTPS GET ✔] ")
			return true
		}
		result.WriteString("[HTTPS GET ✘] ")
		return false
	}() {
		// check by DIAL TIME OUT
		err := checkDialTimeout(hostname, "80")
		if err == nil {
			result.WriteString("[Dial Timeout ✔] ")

			chanDialTimeout <- result.String()
		} else {
			result.WriteString("[Dial Timeout ✘] ")

			chanDead <- result.String()
		}
	} else {
		// records that cannot be PINGed but active
		if result.String() != fmt.Sprintf("[%s] (%s): [PING ✔] ", ip, hostname) {
			chanPingFail <- result.String()
		} else {
			chanAlive <- hostname
		}
	}

	fmt.Println(result.String())
}
