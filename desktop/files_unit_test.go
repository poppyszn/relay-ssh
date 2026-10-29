package main

import (
	"strings"
	"testing"
)

func TestParseListingReadsSftpOutput(t *testing.T) {
	out := `sftp> pwd
Remote working directory: /home/dev
sftp> ls -la
drwxr-xr-x    5 dev      dev           160 Oct 29 01:46 .
drwxr-xr-x    3 root     root           96 Oct  1  2025 ..
drwx------    2 dev      dev            64 Oct 29 01:46 .ssh
-rw-r--r--    1 dev      dev          1234 Oct 28 23:50 notes with spaces.txt
lrwxr-xr-x    1 dev      dev            11 Sep  3  2025 current
-rw-------    ? dev      staff           2 Oct 29 01:52 from-sftp-without-link-count.txt
`
	l := parseListing(out)
	if l.Path != "/home/dev" {
		t.Errorf("path = %q", l.Path)
	}
	if len(l.Entries) != 4 {
		t.Fatalf("entries = %+v, want 4 (. and .. dropped)", l.Entries)
	}
	byName := map[string]FileEntry{}
	for _, e := range l.Entries {
		byName[e.Name] = e
	}
	if e := byName[".ssh"]; !e.Dir || e.Perm != "drwx------" {
		t.Errorf(".ssh = %+v", e)
	}
	if e := byName["notes with spaces.txt"]; e.Dir || e.Size != 1234 || e.Owner != "dev" {
		t.Errorf("spaces file = %+v", e)
	}
	// the SFTP protocol carries no link count, so OpenSSH prints "?" there
	if e := byName["from-sftp-without-link-count.txt"]; e.Size != 2 || e.Owner != "dev" {
		t.Errorf("entry with ? link count = %+v", e)
	}
	if e := byName["current"]; !e.Link {
		t.Errorf("symlink = %+v", e)
	}
}

func TestQuoteEscapesAndRefusesNewlines(t *testing.T) {
	q, err := quote(`/tmp/a "b" \c`)
	if err != nil || q != `"/tmp/a \"b\" \\c"` {
		t.Errorf("quote = %q, %v", q, err)
	}
	for _, bad := range []string{"/tmp/x\nrm /etc/passwd", "a\rb", "", "x\x00y"} {
		if _, err := quote(bad); err == nil {
			t.Errorf("quote(%q) should fail: a newline would inject another sftp command", bad)
		}
	}
}

func TestFriendlyUnknownHostKey(t *testing.T) {
	// ssh prints the real reason first and a useless "Connection closed" last
	err := friendly("Host key verification failed.\nConnection closed\n", nil)
	if !strings.Contains(err.Error(), "accept its fingerprint") {
		t.Errorf("an untrusted host key should tell the user to connect in a tab once, got %v", err)
	}
}

func TestFriendlyAuthError(t *testing.T) {
	if err := friendly("user@host: Permission denied (publickey,password).", nil); !strings.Contains(err.Error(), "sign in first") {
		t.Errorf("auth failure should tell the user what to do, got %v", err)
	}
	if err := friendly("ssh: connect to host x port 22: Connection refused\n", nil); !strings.Contains(err.Error(), "Connection refused") {
		t.Errorf("other errors keep ssh's own last line, got %v", err)
	}
}
