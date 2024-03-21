package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
)

type (
	stats     map[string]map[string]int // map[ip][method]count
	hourStats map[string]map[string]int // map[hour][method]count
)

func parseIPMethodStats(filepath string) (stats, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	s := make(stats)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 10 {
			log.Printf("unexpected number of fields: %d", len(fields))
			continue
		}
		ip, method := fields[0], fields[5][1:]
		if _, ok := s[ip]; !ok {
			s[ip] = make(map[string]int)
		}
		s[ip][method]++
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan file: %w", err)
	}
	return s, nil
}

func parseHourMethodStats(filepath string) (hourStats, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	s := make(hourStats)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 10 {
			log.Printf("unexpected number of fields: %d", len(fields))
			continue
		}
		hour, method := fields[4][:2], fields[5][1:]
		if _, ok := s[hour]; !ok {
			s[hour] = make(map[string]int)
		}
		s[hour][method]++
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan file: %w", err)
	}
	return s, nil
}

func main() {
	s, err := parseIPMethodStats("/home/ricardo/git/GoLang/file/proxy_server_log_apache_format.log")
	if err != nil {
		fmt.Println(err)
		return
	}
	for ip, methods := range s {
		for method, count := range methods {
			fmt.Printf("%s %s %d\n", ip, method, count)
		}
	}

	//
	hourStats, err := parseHourMethodStats("/home/ricardo/git/GoLang/file/proxy_server_log_apache_format.log")
	if err != nil {
		fmt.Println(err)
		return
	}
	// sort hourStats by hour
	hours := make([]string, 0, len(hourStats))
	for k := range hourStats {
		hours = append(hours, k)
	}
	sort.Strings(hours)

	for _, hour := range hours {
		for method, count := range hourStats[hour] {
			fmt.Printf("%s:00 %s %d\n", hour, method, count)
		}
	}

	fmt.Println("main done")
}
