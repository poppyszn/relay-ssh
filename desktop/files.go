package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/core"
	"github.com/poppyszn/relay-ssh/internal/paths"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// FilesService browses and transfers files on a saved connection by running
// the system `sftp` (OpenSSH stays the client, decision D3). It attaches to the
// terminal's already-authenticated connection through OpenSSH multiplexing, so
// passwords, passphrases and 2FA are never needed here and never stored.
type FilesService struct {
	// extraOpts are added to every ssh/sftp command; tests use them to skip
	// host-key prompts against a throwaway server.
	extraOpts []string
}

// FileEntry is one row of a remote directory listing.
type FileEntry struct {
	Name     string `json:"name"`
	Dir      bool   `json:"dir"`
	Link     bool   `json:"link"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
	Perm     string `json:"perm"`
	Owner    string `json:"owner"`
}

// Listing is a directory and its entries.
type Listing struct {
	Path    string      `json:"path"`
	Entries []FileEntry `json:"entries"`
}

// controlDir is where multiplexing sockets live. It is kept short because unix
// socket paths are limited (about 104 bytes on macOS) and %C adds 40.
func controlDir() string {
	base := "/tmp"
	if runtime.GOOS == "windows" {
		return "" // Windows OpenSSH has no multiplexing
	}
	return filepath.Join(base, fmt.Sprintf("relay-ssh-%d", os.Getuid()))
}

// muxOptions are the ssh options that share one connection per host. The
// terminal passes ControlMaster=auto (so the first session becomes the master);
// file operations only attach (ControlMaster=no).
func muxOptions(master bool) []string {
	dir := controlDir()
	if dir == "" {
		return nil
	}
	_ = os.MkdirAll(dir, 0o700)
	mode := "no"
	opts := []string{"-o", "ControlPath=" + filepath.Join(dir, "%C")}
	if master {
		mode = "auto"
		opts = append(opts, "-o", "ControlPersist=60s")
	}
	return append([]string{"-o", "ControlMaster=" + mode}, opts...)
}

// quote makes a path safe for an sftp batch line. A newline would end the line
// and let a crafted file name run another sftp command, so control characters
// are refused outright.
func quote(p string) (string, error) {
	if p == "" {
		return "", errors.New("empty path")
	}
	for _, r := range p {
		if r < 0x20 || r == 0x7f {
			return "", fmt.Errorf("path contains a control character: %q", p)
		}
	}
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(p) + `"`, nil
}

var lsLine = regexp.MustCompile(`^([\-dlcbps])([rwxsStTl\-]{9})\S*\s+(?:\d+|\?)\s+(\S+)\s+(\S+)\s+(\d+)\s+([A-Z][a-z]{2}\s+\d+\s+(?:\d{4}|\d{1,2}:\d{2}))\s+(.+)$`)

// parseListing reads the output of `pwd` followed by `ls -la`.
func parseListing(out string) Listing {
	var l Listing
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if rest, ok := strings.CutPrefix(line, "Remote working directory: "); ok {
			l.Path = strings.TrimSpace(rest)
			continue
		}
		m := lsLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := m[7]
		if name == "." || name == ".." {
			continue
		}
		size, _ := strconv.ParseInt(m[5], 10, 64)
		l.Entries = append(l.Entries, FileEntry{
			Name: name, Dir: m[1] == "d", Link: m[1] == "l", Size: size,
			Modified: m[6], Perm: m[1] + m[2], Owner: m[3],
		})
	}
	return l
}

// friendly turns ssh's stderr into something the user can act on.
func friendly(stderr string, err error) error {
	s := strings.TrimSpace(stderr)
	switch {
	case strings.Contains(s, "Permission denied"), strings.Contains(s, "Too many authentication failures"):
		return errors.New("This host needs you to sign in first. Open it in a tab and log in, then try again: file transfers reuse that connection.")
	case strings.Contains(s, "Host key verification failed"):
		return errors.New("This host's key is not trusted yet. Open it in a tab once and accept its fingerprint, then try again.")
	case strings.Contains(s, "REMOTE HOST IDENTIFICATION HAS CHANGED"):
		return errors.New("The host's key has changed since you last connected. Relay will not continue; check with the host's owner before connecting.")
	case s != "":
		lines := strings.Split(s, "\n")
		return errors.New(strings.TrimSpace(lines[len(lines)-1]))
	}
	return err
}

// run executes an sftp batch against a saved connection.
func (f *FilesService) run(target, script string, timeout time.Duration) (string, error) {
	p, err := paths.Default()
	if err != nil {
		return "", err
	}
	store, err := config.Load(p.StoreFile)
	if err != nil {
		return "", err
	}
	prof, ok := core.ResolveProfile(store, target)
	if !ok {
		return "", fmt.Errorf("connection %q not found", target)
	}
	if err := core.Regenerate(p, store); err != nil {
		return "", err
	}
	args := []string{"-F", p.GenConfig, "-o", "BatchMode=yes"}
	args = append(args, muxOptions(false)...)
	args = append(args, f.extraOpts...)
	args = append(args, "-b", "-", prof.Name)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sftp", args...)
	cmd.Stdin = strings.NewReader(script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", errors.New("timed out talking to the host")
		}
		return stdout.String(), friendly(stderr.String(), err)
	}
	return stdout.String(), nil
}

// List returns the entries of dir on the connection ("" means the login directory).
func (f *FilesService) List(target, dir string) (Listing, error) {
	script := ""
	if dir != "" {
		q, err := quote(dir)
		if err != nil {
			return Listing{}, err
		}
		script += "cd " + q + "\n"
	}
	script += "pwd\nls -la\n"
	out, err := f.run(target, script, 30*time.Second)
	if err != nil {
		return Listing{}, err
	}
	l := parseListing(out)
	if l.Path == "" {
		return l, errors.New("the host did not report a working directory")
	}
	if l.Entries == nil {
		l.Entries = []FileEntry{}
	}
	return l, nil
}

func (f *FilesService) simple(target string, timeout time.Duration, verb string, args ...string) error {
	line := verb
	for _, a := range args {
		q, err := quote(a)
		if err != nil {
			return err
		}
		line += " " + q
	}
	_, err := f.run(target, line+"\n", timeout)
	return err
}

// Download copies a remote file to a local path.
func (f *FilesService) Download(target, remote, local string) error {
	if err := grants.require(local); err != nil {
		return err
	}
	return f.simple(target, 30*time.Minute, "get -p", remote, local)
}

// Upload copies a local file to a remote path.
func (f *FilesService) Upload(target, local, remote string) error {
	if err := grants.require(local); err != nil {
		return err
	}
	return f.simple(target, 30*time.Minute, "put -p", local, remote)
}

func (f *FilesService) Mkdir(target, dir string) error {
	return f.simple(target, 30*time.Second, "mkdir", dir)
}
func (f *FilesService) Remove(target, path string, isDir bool) error {
	if isDir {
		return f.simple(target, 30*time.Second, "rmdir", path)
	}
	return f.simple(target, 30*time.Second, "rm", path)
}
func (f *FilesService) Rename(target, from, to string) error {
	return f.simple(target, 30*time.Second, "rename", from, to)
}

// PickDownloadPath opens the native save dialog with the file's name. Empty if cancelled.
func (*FilesService) PickDownloadPath(name string) (string, error) {
	return grantPicked(application.Get().Dialog.SaveFile().SetFilename(name).CanCreateDirectories(true).PromptForSingleSelection())
}

// PickUploadPath opens the native open dialog. Empty if cancelled.
func (*FilesService) PickUploadPath() (string, error) {
	return grantPicked(application.Get().Dialog.OpenFile().SetTitle("Choose a file to upload").CanChooseFiles(true).PromptForSingleSelection())
}
