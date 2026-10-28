package cmd

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/style"
)

func promptTunnels(in *bufio.Scanner) ([]config.Tunnel, error) {
	if val := prompt(in, "Add a port forward? (y/N)"); !strings.HasPrefix(strings.ToLower(strings.TrimSpace(val)), "y") {
		return nil, nil
	}
	var tunnels []config.Tunnel
	for {
		tun, err := promptOneTunnel(in)
		if err != nil {
			return nil, err
		}
		tunnels = append(tunnels, tun)
		if more := prompt(in, "Add another? (y/N)"); !strings.HasPrefix(strings.ToLower(strings.TrimSpace(more)), "y") {
			break
		}
	}
	return tunnels, nil
}

func promptOneTunnel(in *bufio.Scanner) (config.Tunnel, error) {
	for {
		tunnelTypes := []string{"local", "remote", "dynamic"}
		choice, err := tryFzf(tunnelTypes)
		if err != nil {
			choice, err = menuPick(in, tunnelTypes)
			if err != nil {
				return config.Tunnel{}, err
			}
		}
		tt := config.TunnelLocal
		switch choice {
		case "remote":
			tt = config.TunnelRemote
		case "dynamic":
			tt = config.TunnelDynamic
		}
		port, err := promptPort(in)
		if err != nil {
			return config.Tunnel{}, err
		}
		var target string
		if tt != config.TunnelDynamic {
			if target, err = promptRequired(in, "Target (host:port)"); err != nil {
				return config.Tunnel{}, err
			}
		}
		tun := config.Tunnel{Type: tt, Listen: strconv.Itoa(port), Target: target}
		if err := tun.Validate(); err != nil {
			fmt.Println(style.Fail("✗ " + err.Error()))
			continue
		}
		return tun, nil
	}
}

func promptEditTunnels(in *bufio.Scanner, current []config.Tunnel) ([]config.Tunnel, error) {
	if len(current) > 0 && !strings.HasPrefix(strings.ToLower(strings.TrimSpace(prompt(in, fmt.Sprintf("Change port forwards? (%d configured) (y/N)", len(current))))), "y") {
		return current, nil
	}
	return promptTunnels(in)
}
