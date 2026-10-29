package core

import (
	"strings"

	"github.com/poppyszn/relay-ssh/internal/config"
)

// FolderWithin reports whether path is folder or lies beneath it.
func FolderWithin(path, folder string) bool {
	return path == folder || strings.HasPrefix(path, folder+"/")
}

// KeepFolder makes sure a folder that a change has just emptied still exists:
// folders are managed by the user, so moving or deleting the last connection
// must not make the folder vanish.
func KeepFolder(s *config.Store, folder string) {
	if folder == "" {
		return
	}
	for _, p := range s.Profiles {
		if FolderWithin(p.Folder, folder) {
			return
		}
	}
	for _, f := range s.Folders {
		if FolderWithin(f, folder) {
			return
		}
	}
	s.Folders = append(s.Folders, folder)
}
