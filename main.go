package main

import (
	"fmt"
	"fpt-util/test-alive-server/service"
	"fpt-util/test-alive-server/util"
	"fpt-util/test-alive-server/validate"
	"log"
	"os"
	"strings"
)

func main() {
	defer util.Elapsed("Testing Server Connections")()

	// validate arguments
	if len(os.Args[1:]) != 2 {
		log.Fatalf(`Please provide IP and Hostnames paths only
			ex. [program].exe 'ip_list.txt'(IP) 'hostname_list.txt'(HOSTNAMES) respectively
			
			[1]st argument will be fed to test
			[2]nd argument will be as label`)
	}

	arg1 := os.Args[1]
	arg2 := os.Args[2]

	fmt.Printf("reading [%s] and [%s]...\n", arg1, arg2)
	// read ip.txt
	ips := util.ReadLines(strings.TrimSpace(arg1))

	// read hostname.txt
	hostnames := util.ReadLines(strings.TrimSpace(arg2))

	fmt.Print("validating records...\n\n[logs]\n")
	// validate records
	validate.Validate(ips, hostnames)

	// process
	service.TestConnection(ips, hostnames)
}
