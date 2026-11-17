package main

import (
	"net"
	"sort"
	"strings"
)

type LocalIP struct {
	IP    net.IP
	Net   *net.IPNet
	Iface string
}

func isPrivateV4(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	switch {
	case ip4[0] == 10:
		return true
	case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
		return true
	case ip4[0] == 192 && ip4[1] == 168:
		return true
	case ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127: // CGNAT, some home setups
		return true
	}
	return false
}

// virtualIfaceHints are interface name fragments of adapters that are usually
// not the home network (VMs, WSL, VPNs). They are listed last.
var virtualIfaceHints = []string{"virtual", "vmware", "vbox", "vethernet", "wsl", "hyper-v", "docker", "br-", "veth", "tailscale", "zerotier", "tun", "tap", "utun", "loopback"}

func isVirtualName(name string) bool {
	n := strings.ToLower(name)
	for _, h := range virtualIfaceHints {
		if strings.Contains(n, h) {
			return true
		}
	}
	return false
}

// LocalIPv4s lists private IPv4 addresses of active interfaces, best candidates first.
func LocalIPv4s() []LocalIP {
	out := collectIPv4s(true)
	if len(out) == 0 {
		// Unusual networks with public addressing on the LAN.
		out = collectIPv4s(false)
	}
	primary := outboundIP()
	sort.SliceStable(out, func(i, j int) bool {
		pi := primary != nil && out[i].IP.Equal(primary)
		pj := primary != nil && out[j].IP.Equal(primary)
		if pi != pj {
			return pi
		}
		vi, vj := isVirtualName(out[i].Iface), isVirtualName(out[j].Iface)
		if vi != vj {
			return !vi
		}
		return false
	})
	return out
}

func collectIPv4s(privateOnly bool) []LocalIP {
	var out []LocalIP
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipn, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipn.IP.To4()
			if ip4 == nil || ip4.IsLinkLocalUnicast() || ip4.IsLoopback() {
				continue
			}
			if privateOnly && !isPrivateV4(ip4) {
				continue
			}
			out = append(out, LocalIP{IP: ip4, Net: &net.IPNet{IP: ip4.Mask(ipn.Mask), Mask: ipn.Mask}, Iface: ifc.Name})
		}
	}
	return out
}

// outboundIP returns the local address used for the default route.
// No packets are sent: UDP "connect" only selects a route.
func outboundIP() net.IP {
	c, err := net.Dial("udp4", "1.1.1.1:80")
	if err != nil {
		return nil
	}
	defer c.Close()
	if ua, ok := c.LocalAddr().(*net.UDPAddr); ok {
		return ua.IP.To4()
	}
	return nil
}

// isLocalAddress reports whether ip belongs to this machine.
func isLocalAddress(ip net.IP) bool {
	if ip.IsLoopback() {
		return true
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}
	for _, a := range addrs {
		if ipn, ok := a.(*net.IPNet); ok && ipn.IP.Equal(ip) {
			return true
		}
	}
	return false
}
