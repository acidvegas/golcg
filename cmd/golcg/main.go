package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/acidvegas/golcg"
)

const Version = "1.0.0"

func main() {
	cidr := flag.String("cidr", "", "Target IP range in CIDR format")
	shardNum := flag.Int("shard-num", 1, "Shard number (1-based)")
	totalShards := flag.Int("total-shards", 1, "Total number of shards")
	seed := flag.Int("seed", 0, "Random seed for LCG")
	stateStr := flag.String("state", "", "Resume from specific LCG state")
	version := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *version {
		fmt.Printf("golcg version %s\n", Version)
		os.Exit(0)
	}

	if *cidr == "" {
		fmt.Println("Error: CIDR is required")
		flag.Usage()
		os.Exit(1)
	}

	var state *uint32
	if *stateStr != "" {
		stateVal, err := strconv.ParseUint(*stateStr, 10, 32)
		if err != nil {
			fmt.Printf("Error parsing state: %v\n", err)
			os.Exit(1)
		}
		stateUint32 := uint32(stateVal)
		state = &stateUint32
	}

	stream, err := golcg.IPStream(*cidr, *shardNum, *totalShards, *seed, state)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	for ip := range stream {
		fmt.Println(ip)
	}
}
