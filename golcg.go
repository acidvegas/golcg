package main

import (
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type LCG struct {
	m       uint32
	a       uint32
	c       uint32
	current uint32
}

func NewLCG(seed int, m uint32) *LCG {
	return &LCG{
		m:       m,
		a:       1664525,
		c:       1013904223,
		current: uint32(seed),
	}
}

func (l *LCG) Next() uint32 {
	l.current = (l.a*l.current + l.c) % l.m
	return l.current
}

type IPRange struct {
	start uint32
	total uint32
}

func NewIPRange(cidr string) (*IPRange, error) {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	start := ipToUint32(network.IP)
	ones, bits := network.Mask.Size()
	hostBits := uint(bits - ones)
	broadcast := start | (1<<hostBits - 1)
	total := broadcast - start + 1

	return &IPRange{
		start: start,
		total: uint32(total),
	}, nil
}

func (r *IPRange) GetIPAtIndex(index uint32) (string, error) {
	if index >= r.total {
		return "", errors.New("IP index out of range")
	}

	ip := uint32ToIP(r.start + index)
	return ip.String(), nil
}

func ipToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func uint32ToIP(n uint32) net.IP {
	ip := make(net.IP, 4)
	ip[0] = byte(n >> 24)
	ip[1] = byte(n >> 16)
	ip[2] = byte(n >> 8)
	ip[3] = byte(n)
	return ip
}

func SaveState(seed int, cidr string, shard int, total int, lcgCurrent uint32) error {
	fileName := fmt.Sprintf("pylcg_%d_%s_%d_%d.state", seed, strings.Replace(cidr, "/", "_", -1), shard, total)
	stateFile := filepath.Join(os.TempDir(), fileName)

	return os.WriteFile(stateFile, []byte(fmt.Sprintf("%d", lcgCurrent)), 0644)
}

func IPStream(cidr string, shardNum, totalShards, seed int, state *uint32) (<-chan string, error) {
	ipRange, err := NewIPRange(cidr)
	if err != nil {
		return nil, err
	}

	shardIndex := shardNum - 1

	if seed == 0 {
		rand.Seed(time.Now().UnixNano())
		seed = rand.Intn(1<<32 - 1)
	}

	lcg := NewLCG(seed+shardIndex, 1<<32-1)
	if state != nil {
		lcg.current = *state
	}

	shardSize := ipRange.total / uint32(totalShards)

	if uint32(shardIndex) < (ipRange.total % uint32(totalShards)) {
		shardSize++
	}

	out := make(chan string)
	go func() {
		defer close(out)
		remaining := shardSize

		for remaining > 0 {
			index := lcg.Next() % ipRange.total
			if totalShards == 1 || index%uint32(totalShards) == uint32(shardIndex) {
				ip, err := ipRange.GetIPAtIndex(index)
				if err != nil {
					continue
				}
				out <- ip
				remaining--

				if remaining%1000 == 0 {
					SaveState(seed, cidr, shardNum, totalShards, lcg.current)
				}
			}
		}
	}()

	return out, nil
}

func main() {
	cidr := flag.String("cidr", "", "Target IP range in CIDR format")
	shardNum := flag.Int("shard-num", 1, "Shard number (1-based)")
	totalShards := flag.Int("total-shards", 1, "Total number of shards")
	seed := flag.Int("seed", 0, "Random seed for LCG")
	stateStr := flag.String("state", "", "Resume from specific LCG state")
	flag.Parse()

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

	stream, err := IPStream(*cidr, *shardNum, *totalShards, *seed, state)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	for ip := range stream {
		fmt.Println(ip)
	}
}
