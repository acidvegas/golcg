package golcg

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LCG struct {
	M       uint32
	A       uint32
	C       uint32
	Current uint32
}

func NewLCG(seed int, m uint32) *LCG {
	return &LCG{
		M:       1<<32 - 1,
		A:       1664525,
		C:       1013904223,
		Current: uint32(seed),
	}
}

func (l *LCG) Next() uint32 {
	l.Current = (l.A*l.Current + l.C) % l.M
	return l.Current
}

type IPRange struct {
	Start uint32
	Total uint32
}

func NewIPRange(cidr string) (*IPRange, error) {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	start := ipToUint32(network.IP)
	ones, bits := network.Mask.Size()
	hostBits := uint(bits - ones)

	var total uint32
	if hostBits == 32 {
		total = 0
	} else {
		total = 1 << hostBits
	}

	return &IPRange{
		Start: start,
		Total: total,
	}, nil
}

func (r *IPRange) GetIPAtIndex(index uint32) (string, error) {
	if r.Total > 0 && index >= r.Total {
		return "", errors.New("IP index out of range")
	}

	ip := uint32ToIP(r.Start + index)
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
	fileName := fmt.Sprintf("golcg_%d_%s_%d_%d.state", seed, strings.Replace(cidr, "/", "_", -1), shard, total)
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
		lcg.Current = *state
	}

	var shardSize uint32
	if ipRange.Total == 0 {
		shardSize = (1<<32 - 1) / uint32(totalShards)
		if uint32(shardIndex) < uint32(totalShards-1) {
			shardSize++
		}
	} else {
		shardSize = ipRange.Total / uint32(totalShards)
		if uint32(shardIndex) < ipRange.Total%uint32(totalShards) {
			shardSize++
		}
	}

	out := make(chan string, 1000)
	go func() {
		defer close(out)
		remaining := shardSize

		for remaining > 0 {
			next := lcg.Next()
			var index uint32
			if ipRange.Total > 0 {
				index = next % ipRange.Total
			} else {
				index = next
			}

			if totalShards == 1 || index%uint32(totalShards) == uint32(shardIndex) {
				ip, err := ipRange.GetIPAtIndex(index)
				if err != nil {
					continue
				}
				out <- ip
				remaining--

				if remaining%1000 == 0 {
					SaveState(seed, cidr, shardNum, totalShards, lcg.Current)
				}
			}
		}
	}()

	return out, nil
}
