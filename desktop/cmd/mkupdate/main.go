// Command mkupdate writes the signed update manifest (update.json) for a Relay release, and
// generates the signing key pair. It runs in CI after the platform builds finish.
//
//	mkupdate -genkey -out private.key                  # prints the public key to pin in the app
//	RELAY_UPDATE_SIGNING_KEY=... mkupdate -version 1.2.0 -notes-file notes.md -out update.json \
//	    darwin/arm64=Relay-darwin-arm64.zip linux/amd64=Relay-linux-amd64.tar.gz
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/poppyszn/relay-ssh/desktop/internal/release"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "mkupdate:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("mkupdate", flag.ContinueOnError)
	genkey := fs.Bool("genkey", false, "generate a signing key pair; the private key goes to -out, the public key is printed")
	version := fs.String("version", "", "release version, MAJOR.MINOR.PATCH")
	notesFile := fs.String("notes-file", "", "file with the release notes")
	out := fs.String("out", "", "output file")
	keyEnv := fs.String("key-env", "RELAY_UPDATE_SIGNING_KEY", "environment variable holding the private key")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return errors.New("-out is required")
	}
	if *genkey {
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return err
		}
		if err := os.WriteFile(*out, []byte(base64.StdEncoding.EncodeToString(priv)+"\n"), 0o600); err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, base64.StdEncoding.EncodeToString(pub))
		return err
	}
	key, err := release.ParseKey(os.Getenv(*keyEnv))
	if err != nil {
		return fmt.Errorf("$%s: %w", *keyEnv, err)
	}
	var arts []release.Artifact
	for _, spec := range fs.Args() {
		target, path, ok := strings.Cut(spec, "=")
		platform, arch, ok2 := strings.Cut(target, "/")
		if !ok || !ok2 {
			return fmt.Errorf("artifact %q is not platform/arch=path", spec)
		}
		arts = append(arts, release.Artifact{Platform: platform, Arch: arch, Path: path})
	}
	notes := ""
	if *notesFile != "" {
		b, err := os.ReadFile(*notesFile)
		if err != nil {
			return err
		}
		notes = strings.TrimSpace(string(b))
	}
	data, err := release.Build(*version, notes, time.Now(), arts, key)
	if err != nil {
		return err
	}
	return os.WriteFile(*out, append(data, '\n'), 0o644)
}
