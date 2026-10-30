package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// startSSHD runs a throwaway unprivileged sshd on a loopback port, with its own
// host key and a freshly generated client key, and returns a FilesService and
// the name of a saved connection that reaches it. Skips where sshd or
// ssh-keygen is unavailable.
func startSSHD(t *testing.T) (*FilesService, string) {
	t.Helper()
	sshd, err := exec.LookPath("sshd")
	if err != nil {
		if _, e := os.Stat("/usr/sbin/sshd"); e != nil {
			t.Skip("sshd not installed")
		}
		sshd = "/usr/sbin/sshd"
	}
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not installed")
	}
	dir := t.TempDir()
	keygen := func(name string) string {
		path := filepath.Join(dir, name)
		if out, err := exec.Command("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", path).CombinedOutput(); err != nil {
			t.Fatalf("ssh-keygen: %v: %s", err, out)
		}
		return path
	}
	hostKey, clientKey := keygen("host"), keygen("client")
	pub, _ := os.ReadFile(clientKey + ".pub")
	authKeys := filepath.Join(dir, "authorized_keys")
	if err := os.WriteFile(authKeys, pub, 0o600); err != nil {
		t.Fatal(err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	cfg := filepath.Join(dir, "sshd_config")
	conf := fmt.Sprintf("Port %d\nListenAddress 127.0.0.1\nHostKey %s\nAuthorizedKeysFile %s\nPidFile %s\nUsePAM no\nPasswordAuthentication no\nStrictModes no\nSubsystem sftp internal-sftp\n",
		port, hostKey, authKeys, filepath.Join(dir, "sshd.pid"))
	if err := os.WriteFile(cfg, []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}
	srv := exec.Command(sshd, "-D", "-e", "-f", cfg)
	var log strings.Builder
	srv.Stderr = &log
	if err := srv.Start(); err != nil {
		t.Skipf("cannot start sshd: %v", err)
	}
	t.Cleanup(func() { _ = srv.Process.Kill(); _ = srv.Wait() })
	up := false
	for i := 0; i < 50 && !up; i++ {
		if c, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond); err == nil {
			c.Close()
			up = true
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
	if !up {
		t.Skipf("sshd did not start: %s", log.String())
	}

	t.Setenv("RELAY_SSH_HOME", t.TempDir())
	u, _ := user.Current()
	if err := (ProfilesService{}).Save("", Profile{Name: "box", Host: "127.0.0.1", Port: port, User: u.Username, Auth: "key", Key: clientKey}); err != nil {
		t.Fatal(err)
	}
	return &FilesService{extraOpts: []string{"-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-o", "LogLevel=ERROR"}}, "box"
}

func TestFilesAgainstARealSSHD(t *testing.T) {
	f, box := startSSHD(t)
	work := t.TempDir() // the "remote" is this machine, so it doubles as the remote directory

	home, err := f.List(box, "")
	if err != nil || home.Path == "" {
		t.Fatalf("List home = %+v, %v", home, err)
	}

	local := filepath.Join(t.TempDir(), "hello.txt")
	if err := os.WriteFile(local, []byte("relay file transfer\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	remote := filepath.Join(work, "hello.txt")
	if err := f.Upload(box, local, remote); err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if err := f.Mkdir(box, filepath.Join(work, "sub dir")); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	l, err := f.List(box, work)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	names := map[string]FileEntry{}
	for _, e := range l.Entries {
		names[e.Name] = e
	}
	if e := names["hello.txt"]; e.Dir || e.Size != int64(len("relay file transfer\n")) {
		t.Errorf("hello.txt = %+v", e)
	}
	if e := names["sub dir"]; !e.Dir {
		t.Errorf("'sub dir' should be a directory with its space intact: %+v", e)
	}

	back := filepath.Join(t.TempDir(), "copy.txt")
	if err := f.Download(box, remote, back); err != nil {
		t.Fatalf("Download: %v", err)
	}
	if got, _ := os.ReadFile(back); string(got) != "relay file transfer\n" {
		t.Errorf("downloaded %q", got)
	}

	if err := f.Rename(box, remote, filepath.Join(work, "renamed.txt")); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if _, err := os.Stat(filepath.Join(work, "renamed.txt")); err != nil {
		t.Errorf("rename did not happen: %v", err)
	}
	if err := f.Remove(box, filepath.Join(work, "renamed.txt"), false); err != nil {
		t.Fatalf("Remove file: %v", err)
	}
	if err := f.Remove(box, filepath.Join(work, "sub dir"), true); err != nil {
		t.Fatalf("Remove dir: %v", err)
	}

	// errors are reported, and a hostile name cannot smuggle in another command
	if _, err := f.List(box, filepath.Join(work, "missing")); err == nil {
		t.Error("listing a missing directory should fail")
	}
	victim := filepath.Join(work, "victim.txt")
	_ = os.WriteFile(victim, []byte("keep me"), 0o600)
	if err := f.Remove(box, work+"/x\nrm "+victim, false); err == nil {
		t.Error("a path with a newline must be refused")
	}
	if _, err := os.Stat(victim); err != nil {
		t.Errorf("the injected rm ran: %v", err)
	}
}

func TestFilesNeedsSignInWhenKeyIsRejected(t *testing.T) {
	f, box := startSSHD(t)
	// point the connection at a key the server does not know
	other := filepath.Join(t.TempDir(), "other")
	if out, err := exec.Command("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", other).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	p, _ := (ProfilesService{}).List()
	prof := p[0]
	prof.Key = other
	if err := (ProfilesService{}).Save(box, prof); err != nil {
		t.Fatal(err)
	}
	_, err := f.List(box, "")
	if err == nil || !strings.Contains(err.Error(), "sign in first") {
		t.Errorf("auth failure should explain what to do, got %v", err)
	}
}
