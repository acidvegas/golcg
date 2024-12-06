package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/acidvegas/golcg"
)

const Version = "1.0.0"

func parseShardArg(shard string) (int, int, error) {
	if shard == "" {
		return 1, 1, nil
	}

	parts := strings.Split(shard, "/")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid shard format. Expected INDEX/TOTAL, got %s", shard)
	}

	index, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid shard index: %v", err)
	}

	total, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid shard total: %v", err)
	}

	if index < 1 || index > total {
		return 0, 0, fmt.Errorf("shard index must be between 1 and total")
	}

	if total < 1 {
		return 0, 0, fmt.Errorf("total shards must be greater than 0")
	}

	return index, total, nil
}

func main() {
	cidr := flag.String("cidr", "", "Target IP range in CIDR format")
	shard := flag.String("shard", "", "Shard specification in INDEX/TOTAL format (e.g., 1/4)")
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

	shardNum, totalShards, err := parseShardArg(*shard)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
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

	stream, err := golcg.IPStream(*cidr, shardNum, totalShards, *seed, state)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	for ip := range stream {
		fmt.Println(ip)
	}
}
