package main

import (
	"os/exec"
	"strings"
	"testing"
)

const authMarker = "\x1b]7771;auth\a"

// withoutHostChecks lets the test's throwaway server in without touching known_hosts.
func withoutHostChecks(argv []string) []string {
	return append([]string{argv[0], "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-o", "LogLevel=ERROR"}, argv[1:]...)
}

// A new login prints a private marker once ssh has authenticated, which is what tells the
// app it is safe to type into the session. A session that attaches to a login that is
// already open (connection sharing) authenticates nothing, so it gets no marker, and the
// app is told another way (masterAlive, sent as a "mux" frame).
func TestLoginSignalsAuthenticationAgainstARealSSHD(t *testing.T) {
	_, box := startSSHD(t)
	argv, env, err := argvFor(box)
	if err != nil {
		t.Fatal(err)
	}
	argv = withoutHostChecks(argv)
	t.Cleanup(func() {
		exit := append(append([]string{}, argv[:len(argv)-1]...), "-O", "exit", argv[len(argv)-1])
		_ = exec.Command(exit[0], exit[1:]...).Run()
	})

	if masterAlive(argv) {
		t.Fatal("masterAlive is true before any login")
	}
	first, err := startSession(argv, env, 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	defer first.close()
	got := readUntil(t, first, authMarker)
	if !strings.Contains(got, authMarker) {
		t.Fatalf("a new login printed no authentication marker:\n%q", got)
	}
	if !masterAlive(argv) {
		t.Fatal("masterAlive is false while a login is open")
	}

	second, err := startSession(argv, env, 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	defer second.close()
	if _, err := second.pty.Write([]byte("echo shared-$((6*7))\n")); err != nil {
		t.Fatal(err)
	}
	got = readUntil(t, second, "shared-42")
	if !strings.Contains(got, "shared-42") {
		t.Fatalf("the shared session never ran the command:\n%q", got)
	}
	if strings.Contains(got, authMarker) {
		t.Errorf("a session sharing an open login printed the marker (it should come as a mux frame):\n%q", got)
	}
}
