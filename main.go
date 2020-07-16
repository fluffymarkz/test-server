package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var wg sync.WaitGroup
var mut sync.Mutex

const (
	// APPLYHTTP - apply http:// in GET
	APPLYHTTP = false

	// APPLYHTTPS - apply https:// in GET
	APPLYHTTPS = true

	// IPSTXT - ip list text file
	IPSTXT = "ip.txt"

	// HOSTNAMETXT - hostname list text file
	HOSTNAMETXT = "hostname.txt"

	// TESTIP - test one by ip
	TESTIP = true

	// TESTHN - test one by hostname
	TESTHN = false
)

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
	defer wg.Done()
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
		err := checkByHTTP(APPLYHTTP, hostname)
		if err == nil {
			result.WriteString("[HTTP GET ✔] ")
			return true
		}
		result.WriteString("[HTTP GET ✘] ")

		// check by HTTPS GET
		err = checkByHTTP(APPLYHTTPS, hostname)
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
		}

		chanDead <- result.String()
	} else {
		// records that cannot be PINGed but active
		if result.String() != fmt.Sprintf("[%s] (%s): [PING ✔] ", ip, hostname) {
			chanPingFail <- result.String()
		}

		chanAlive <- hostname
	}

	fmt.Println(result.String())
}

func testConnection(ips []string, hostnames []string) {
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
		go executeTest(chanAlive, chanDead, chanPingFail, chanDialTimeout, siteIP[i], siteHostname[i])
	}

	wg.Wait()
	close(chanAlive)
	close(chanDead)
	close(chanPingFail)

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

// Read a whole file into the memory and store it as array of lines
func readLines(path string) (lines []string, err error) {
	var (
		file   *os.File
		part   []byte
		prefix bool
	)
	if file, err = os.Open(path); err != nil {
		return
	}
	reader := bufio.NewReader(file)
	buffer := bytes.NewBuffer(make([]byte, 1024))
	for {
		if part, prefix, err = reader.ReadLine(); err != nil {
			break
		}
		buffer.Write(part)
		if !prefix {
			lines = append(lines, buffer.String())
			buffer.Reset()
		}
	}
	if err == io.EOF {
		err = nil
	}
	return
}

func elapsed(what string) func() {
	start := time.Now()
	return func() {
		fmt.Printf("\n%s took %v\n", what, time.Since(start))
	}
}

func getIndexStr(elem string, slice []string) int {
	for i, v := range slice {
		if v == elem {
			return i
		}
	}

	return -1
}

func dupItems(s []string) []string {
	dupItem := []string{}

	m := make(map[string]bool)
	for _, item := range s {
		if _, ok := m[item]; ok {
			dupItem = append(dupItem, item)
		} else {
			m[item] = true
		}
	}

	return dupItem
}

func validate(ips, hostnames []string) {
	if len(ips) != len(hostnames) {
		log.Fatalf("Mismatch record numbers in `%s` and `%s`", IPSTXT, HOSTNAMETXT)
	}

	// check for duplicates in IPs
	dupIPs := dupItems(ips)
	if len(dupIPs) > 0 {
		log.Fatalf("Duplicate records in %s:\n%v\n", IPSTXT, dupIPs)
	}

	// check for duplicates in IPs
	dupHNs := dupItems(hostnames)
	if len(dupHNs) > 0 {
		log.Fatalf("Duplicate records in %s:\n%v\n", HOSTNAMETXT, dupHNs)
	}
}

func main() {
	defer elapsed("TEST Servers")()

	// read ip.txt
	ips, err := readLines("ip.txt")
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	// read hostname.txt
	hostnames, err := readLines("hostname.txt")
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	// validate records
	validate(ips[1:], hostnames[1:])

	// process
	testConnection(ips[1:], hostnames[1:])
}
