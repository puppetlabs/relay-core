package workflow

import (
	"net"
	"sort"
)

func copyIP(ip net.IP) net.IP {
	nip := make(net.IP, len(ip))
	copy(nip, ip)
	return nip
}

func ipNetExcludeSorted(nw *net.IPNet, ips []net.IP) []*net.IPNet {
	if len(ips) == 0 {
		return []*net.IPNet{nw}
	}

	ip := ips[0]
	if !nw.Contains(ip) {
		return ipNetExcludeSorted(nw, ips[1:])
	}

	ones, bits := nw.Mask.Size()
	if ones == bits {
		// This network *is* the IP.
		return nil
	}

	// The IP we're working with might be a 16-byte slice, in which case we need
	// to offset our ones/bits by the appropriate amount.
	if cap := len(ip) - len(nw.Mask); cap > 0 {
		ones, bits = ones+(cap*8), bits+(cap*8)
	}

	// Create a new reference range based on the most restrictive IP being
	// excluded.
	ref := &net.IPNet{
		IP:   copyIP(ip),
		Mask: net.CIDRMask(bits, bits),
	}

	var nws []*net.IPNet

	for i := bits - 1; i >= ones; i-- {
		idx, bit := i/8, byte(1<<(7-uint8(i&7)))

		// Flip the bit to create an incluced range (the current reference will
		// always start as the excluded range or a range already accounted for).
		ref.IP[idx] ^= bit

		if ref.IP[idx]&bit != 0 {
			// If the new bit is high, that means we're above the original
			// allocation, so test/split the next IP out.
			nws = append(nws, ipNetExcludeSorted(ref, ips[1:])...)
		} else {
			// Otherwise we simply add the result to our existing list.
			nws = append(nws, ref)
		}

		// Create a copy of the network and zero out the current bit for the
		// next range.
		ref = &net.IPNet{
			IP:   copyIP(ref.IP),
			Mask: net.CIDRMask(i, bits),
		}
		ref.IP[idx] &= ^bit
	}

	return nws
}

func ipNetExclude(nw *net.IPNet, ips []net.IP) []*net.IPNet {
	// Sort IPs so we're always cutting from the tail.
	sort.SliceStable(ips, func(i, j int) bool { return string(ips[i]) < string(ips[j]) })

	return ipNetExcludeSorted(nw, ips)
}
