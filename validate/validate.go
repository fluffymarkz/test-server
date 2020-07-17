package validate

import (
	"fpt-util/test-alive-server/constant"
	"fpt-util/test-alive-server/util"
	"log"
)

// Validate - validate ip and hostname
func Validate(ips, hostnames []string) {
	// check if ip and hostname records are match
	if len(ips) != len(hostnames) {
		log.Fatalf("Mismatch record numbers in `%s` and `%s`", constant.IPSTXT, constant.HOSTNAMETXT)
	}

	// check for duplicates in IPs
	dupIPs := util.DupItems(ips)
	if len(dupIPs) > 0 {
		log.Fatalf("Duplicate records in %s:\n%v\n", constant.IPSTXT, dupIPs)
	}

	// check if valid IPv4
	for _, ip := range ips {
		if !util.IsIpv4Net(ip) {
			log.Fatalf("IP [%s] is not a valid IPv4\n", ip)
		}
	}

	// check for duplicates in IPs
	dupHNs := util.DupItems(hostnames)
	if len(dupHNs) > 0 {
		log.Fatalf("Duplicate records in %s:\n%v\n", constant.HOSTNAMETXT, dupHNs)
	}
}
