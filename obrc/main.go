package main

import (
	"bufio"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"runtime/pprof"
	"sort"
	"time"

	"github.com/lmittmann/tint"
)

var (
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile = flag.String("memprofile", "", "write memory profile to `file`")
)

func Logger() *slog.Logger {
	w := os.Stderr
	logger := slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.TimeOnly,
		}),
	)
	return logger
}

type StationData struct {
	Min   float32
	Max   float32
	Sum   float32
	Count int
}

var stations = map[string]StationData{}

func convertStringToFloat32(bs []byte) float32 {
	// We assume that the number always has a decimal point

	isNegative := false
	if bs[0] == '-' {
		isNegative = true
		bs = bs[1:]
	}

	var f float32
	var multiplier float32 = 10.0
	for i := 0; i < len(bs); i++ {
		digit := bs[i]
		if digit == '.' {
			continue
		}
		digit -= '0'

		f *= multiplier
		f += float32(digit)
	}

	f /= 10

	if isNegative {
		f *= -1
	}
	return f
}

func run() {
	// file, err := os.Open("obrc/data/measurements.txt")
	file, err := os.Open("/media/leal/New Volume/1brc/1000000/measurements.txt")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for scanner.Scan() {
		b := scanner.Bytes()
		if len(b) == 0 {
			continue
		}
		for i := 0; i < len(b); i++ {
			if b[i] != ';' {
				continue
			}
			station := string(b[:i])
			data := stations[station]
			value := convertStringToFloat32(b[i+1:])
			if data.Count == 0 {
				data.Min = value
				data.Max = value
			}
			if value < data.Min {
				data.Min = value
			}
			if value > data.Max {
				data.Max = value
			}
			data.Sum += value
			data.Count++
			stations[station] = data
			break
		}
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}
}

func printResult() {
	// sort stations
	keys := make([]string, 0, len(stations))
	for k := range stations {
		keys = append(keys, k)
	}
	// sort keys
	sort.Strings(keys)
	// print result
	for _, k := range keys {
		v := stations[k]
		fmt.Printf("%s: min=%.2f max=%.2f avg=%.2f\n", k, v.Min, v.Max, v.Sum/float32(v.Count))
	}
}

func main() {
	l := Logger()
	l.Info("Running main")

	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			l.Error("could not create CPU profile: ", err)
			return
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			l.Error("could not start CPU profile: ", err)
			return
		}
		defer pprof.StopCPUProfile()
	}

	run()
	printResult()

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			l.Error("could not create memory profile: ", err)
			return
		}
		defer f.Close() // error handling omitted for example
		runtime.GC()    // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			l.Error("could not write memory profile: ", err)
			return
		}
	}

	l.Info("Done")
}
