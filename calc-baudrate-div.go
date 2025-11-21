// calculate baud rate divisor given a cpu frequency and rate in bits per second
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 2 {
		usage()
	}

	freq, _ := strconv.ParseUint(flag.Arg(0), 0, 64)
	for i := 1; i < flag.NArg(); i++ {
		baudrate, _ := strconv.ParseUint(flag.Arg(i), 0, 64)
		fmt.Println(calc(freq, baudrate))
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: <freq> <baud_rate>")
	flag.PrintDefaults()
	os.Exit(2)
}

func calc(cpufreq, baudrate uint64) float64 {
	return float64(cpufreq)/float64(8*baudrate) - 1
}
