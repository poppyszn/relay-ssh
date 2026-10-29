package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSpecValidation(t *testing.T) {
	cases := []struct {
		kind, listen, target, flag, arg string
		ok                              bool
	}{
		{"local", "8080", "localhost:80", "-L", "8080:localhost:80", true},
		{"local", "127.0.0.1:8080", "db.internal:5432", "-L", "127.0.0.1:8080:db.internal:5432", true},
		{"remote", "9000", "127.0.0.1:3000", "-R", "9000:127.0.0.1:3000", true},
		{"dynamic", "1080", "", "-D", "1080", true},
		{"local", "99999", "h:80", "", "", false},
		{"local", "8080", "no-port", "", "", false},
		{"local", "8080", "h:80 -o ProxyCommand=evil", "", "", false}, // option smuggling
		{"local", "8080;rm", "h:80", "", "", false},
		{"bogus", "8080", "h:80", "", "", false},
	}
	for _, c := range cases {
		flag, arg, err := spec(c.kind, c.listen, c.target)
		if (err == nil) != c.ok || (c.ok && (flag != c.flag || arg != c.arg)) {
			t.Errorf("spec(%q,%q,%q) = %q %q, %v; want ok=%v %q %q", c.kind, c.listen, c.target, flag, arg, err, c.ok, c.flag, c.arg)
		}
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

// reachesSSH reports whether connecting to the port yields an SSH banner, i.e.
// the forward really carries traffic to the throwaway sshd.
func reachesSSH(port string) bool {
	c, err := net.DialTimeout("tcp", "127.0.0.1:"+port, time.Second)
	if err != nil {
		return false
	}
	defer c.Close()
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 4)
	n, _ := c.Read(buf)
	return string(buf[:n]) == "SSH-"
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for i := 0; i < 40; i++ {
		if cond() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Errorf("timed out waiting for: %s", what)
}

func TestForwardsAgainstARealSSHD(t *testing.T) {
	files, box := startSSHD(t)
	f := &ForwardsService{extraOpts: files.extraOpts}
	sshdPort := ""
	if p, _ := (ProfilesService{}).List(); len(p) == 1 {
		sshdPort = strconv.Itoa(p[0].Port)
	}

	// not connected yet: a state, not an error
	st, err := f.Status(box)
	if err != nil || st.Connected || st.Message == "" {
		t.Fatalf("Status before connecting = %+v, %v", st, err)
	}
	if err := f.Add(box, "local", strconv.Itoa(freePort(t)), "127.0.0.1:"+sshdPort); err == nil || !strings.Contains(err.Error(), "not open") {
		t.Errorf("adding without a live connection should say so, got %v", err)
	}

	// the terminal tab would be the master; start one the same way
	startMaster(t, files, box)
	t.Cleanup(func() { _, _ = f.ssh(box, "-O", "exit") })
	st, _ = f.Status(box)
	if !st.Connected {
		t.Fatalf("master should be live: %+v", st)
	}

	// local forward carries real traffic, then stops
	lp := strconv.Itoa(freePort(t))
	if err := f.Add(box, "local", lp, "127.0.0.1:"+sshdPort); err != nil {
		t.Fatalf("Add local: %v", err)
	}
	eventually(t, "local forward to reach sshd", func() bool { return reachesSSH(lp) })
	st, _ = f.Status(box)
	if len(st.Forwards) != 1 || !st.Forwards[0].Listening || st.Forwards[0].Source != "runtime" {
		t.Errorf("Status after add = %+v", st)
	}
	if err := f.Cancel(box, "local", lp, "127.0.0.1:"+sshdPort); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	eventually(t, "local forward to stop", func() bool { return !reachesSSH(lp) })
	if st, _ = f.Status(box); len(st.Forwards) != 0 {
		t.Errorf("cancelled runtime forward should disappear: %+v", st)
	}

	// SOCKS proxy
	dp := strconv.Itoa(freePort(t))
	if err := f.Add(box, "dynamic", dp, ""); err != nil {
		t.Fatalf("Add dynamic: %v", err)
	}
	eventually(t, "SOCKS port to listen", func() bool {
		c, err := net.DialTimeout("tcp", "127.0.0.1:"+dp, time.Second)
		if err == nil {
			c.Close()
		}
		return err == nil
	})
	_ = f.Cancel(box, "dynamic", dp, "")

	// remote forward: the server listens and sends traffic back to this machine's sshd
	rp := strconv.Itoa(freePort(t))
	if err := f.Add(box, "remote", rp, "127.0.0.1:"+sshdPort); err != nil {
		t.Fatalf("Add remote: %v", err)
	}
	eventually(t, "remote forward to reach sshd", func() bool { return reachesSSH(rp) })
	st, _ = f.Status(box)
	if len(st.Forwards) != 1 || st.Forwards[0].Type != "remote" || !st.Forwards[0].Listening {
		t.Errorf("remote status = %+v", st)
	}
	_ = f.Cancel(box, "remote", rp, "127.0.0.1:"+sshdPort)
	eventually(t, "remote forward to stop", func() bool { return !reachesSSH(rp) })

	// a port that is taken, and bad input, are refused with clear messages
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	if err := f.Add(box, "local", strconv.Itoa(busy.Addr().(*net.TCPAddr).Port), "127.0.0.1:"+sshdPort); err == nil {
		t.Error("a port already in use should be refused")
	}
	if err := f.Add(box, "local", "nope", "h:1"); err == nil {
		t.Error("an invalid port should be refused")
	}
}

// startMaster opens a background multiplex master like the terminal tab would.
func startMaster(t *testing.T, files *FilesService, box string) {
	t.Helper()
	p, _ := (ProfilesService{}).List()
	if len(p) == 0 {
		t.Fatal("no profile")
	}
	conf := fmt.Sprintf("%s/.ssh/sssh.d/config", mustEnv(t, "SSSH_HOME"))
	args := []string{"-F", conf}
	args = append(args, muxOptions(true)...)
	args = append(args, files.extraOpts...)
	// The forked master would keep a captured stdout/stderr pipe open forever, so
	// its diagnostics go to a log file instead of a pipe.
	logFile := filepath.Join(t.TempDir(), "master.log")
	args = append(args, "-o", "BatchMode=yes", "-E", logFile, "-N", "-f", box)
	if err := exec.Command("ssh", args...).Run(); err != nil {
		out, _ := os.ReadFile(logFile)
		t.Fatalf("starting master: %v: %s", err, out)
	}
}

func mustEnv(t *testing.T, k string) string {
	t.Helper()
	v := osGetenv(k)
	if v == "" {
		t.Fatalf("%s not set", k)
	}
	return v
}

var osGetenv = os.Getenv
