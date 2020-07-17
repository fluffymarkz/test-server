package util

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

// DupItems - get duplicate items
func DupItems(s []string) []string {
	dupItem := []string{}

	m := make(map[string]bool)
	for _, item := range s {
		if _, ok := m[item]; ok {
			dupItem = append(dupItem)
		} else {
			m[item] = true
		}
	}

	return dupItem
}

// ReadLines - Read a whole file into the memory and store it as array of lines
func ReadLines(path string) (result []string) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		result = append(result, strings.TrimSpace(scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return result
}

// GetIndexStr - get index of string elem from string slice
func GetIndexStr(elem string, slice []string) int {
	for i, v := range slice {
		if v == elem {
			return i
		}
	}

	return -1
}

// Elapsed - prints elapsed time of whole function
func Elapsed(what string) func() {
	start := time.Now()
	return func() {
		fmt.Printf("\n%s took %v\n", what, time.Since(start))
	}
}

// IsIpv4Net - check if valid IPv4
func IsIpv4Net(host string) bool {
	return net.ParseIP(host) != nil
}
