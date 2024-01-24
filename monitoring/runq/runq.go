package main

import (
	"fmt"
	"scheduling/util"
	"strconv"
	"strings"
)

// Global vars
var (
	argDuration int64
	argInterval int64
	argCount    int64
	argMethod   string
	flagQueue   bool
	flagLatency bool
	flagBlocked bool
	flagVerbose bool
	flagHelp    bool
)

const (
	procCPUPrefix = "cpu"
)

var (
	cpuLatency map[string]int64 = make(map[string]int64)
)

func main() {
	var (
		outer int64
		//lastLatency          int64
		newLatency int64
		//deltaLatency         int64
		averageLatency       int64
		averageLatencyOutput string
	)
	argCount = 600
	//lastLatency = 0
	argDuration = 1
	var allScore int64
	for outer = 0; outer < argCount; outer++ {
		newLatency = getCPULatency()
		//deltaLatency = newLatency - lastLatency
		averageLatency = int64(float64(newLatency) / (float64(argDuration)) * 1000) // Milliseconds
		//fmt.Printf("new, last, deltal, averageLatency: %d - %d - %d - %d\n", newLatency, lastLatency, deltaLatency, averageLatency)
		// Blank out first response
		if outer == 0 {
			averageLatencyOutput = "-"
			println(averageLatencyOutput)

		} else {
			averageLatencyOutput = fmt.Sprintf("%d", averageLatency)
			println(averageLatencyOutput)
			allScore += averageLatency
		}

	}
	println(allScore)
	//22866000
	//23948000
}

// Parse and sum CPUs in /proc/schedstats to get task latency
func getCPULatency() (sumLatency int64) {

	var (
		lines []string
	)

	lines = strings.Split(string(util.ObtainPodFile()), "\n")
	// Parsing - could be improved
	sumLatency = 0
	for _, line := range lines {
		fields := strings.Fields(line)
		//fmt.Printf("Line: %s\n", line)
		// Need to check length or else will get index error on blank line
		// 是否以cpu开头
		if len(fields) > 0 && strings.Contains(fields[0], procCPUPrefix) {

			val, err := strconv.ParseUint(fields[7], 10, 64)
			if err != nil {
				continue
			}
			_, exist := cpuLatency[fields[0]]
			if exist == true {
				sumLatency += int64(val) - cpuLatency[fields[0]]
			}
			cpuLatency[fields[0]] = int64(val)
		}
	}
	sumLatency = sumLatency / 1000000
	if len(cpuLatency) == 0 {
		return
	}
	sumLatency = sumLatency / int64(len(cpuLatency))
	return
}
