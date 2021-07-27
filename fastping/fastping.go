package fastping

import (
	"fmt"
	"time"

	"github.com/go-ping/ping"
	"github.com/k0kubun/pp"
)

func TestAddress(addr string) (avg time.Duration, packetLoss float64, err error) {
	pinger, err := ping.NewPinger(addr)
	if err != nil {
		return 0, 0, err
	}
	pinger.Size = 128
	pinger.SetNetwork("ip")
	pinger.SetPrivileged(false)
	pinger.Count = 5
	pinger.Timeout = time.Second * 2
	err = pinger.Run() // Blocks until finished.
	if err != nil {
		return 1000000, 0, err
	}
	stats := pinger.Statistics() // get send/receive/duplicate/rtt stats

	if stats.PacketLoss > 0 {
		return 1000000, stats.PacketLoss, fmt.Errorf("packets lost: %f", stats.PacketLoss)
	}

	if stats.AvgRtt == time.Millisecond {
		pp.Println(stats)
	}

	return stats.AvgRtt, stats.PacketLoss, err
}
