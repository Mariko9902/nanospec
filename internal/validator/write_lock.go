package validator

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gofrs/flock"
)

type writeLocks struct {
	handles []*flock.Flock
}

func lockWriteDirectories(directories []string) (*writeLocks, error) {
	paths := make(map[string]struct{}, len(directories))
	for _, directory := range directories {
		absolute, err := filepath.Abs(directory)
		if err != nil {
			return nil, err
		}
		canonical, err := filepath.EvalSymlinks(absolute)
		if err != nil {
			return nil, err
		}
		paths[canonical] = struct{}{}
	}
	ordered := make([]string, 0, len(paths))
	for path := range paths {
		ordered = append(ordered, path)
	}
	sort.Strings(ordered)
	cache, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	lockDirectory := filepath.Join(cache, "nanospec", "locks")
	if err := os.MkdirAll(lockDirectory, 0o700); err != nil {
		return nil, err
	}
	locks := &writeLocks{}
	for _, path := range ordered {
		handle := flock.New(filepath.Join(lockDirectory, fmt.Sprintf("%x.lock", sha256.Sum256([]byte(path)))))
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		locked, err := handle.TryLockContext(ctx, 20*time.Millisecond)
		cancel()
		if err != nil || !locked {
			_ = handle.Close()
			locks.release()
			return nil, fmt.Errorf("cannot lock %s for writing: %w", path, err)
		}
		locks.handles = append(locks.handles, handle)
	}
	return locks, nil
}

func (locks *writeLocks) release() {
	for index := len(locks.handles) - 1; index >= 0; index-- {
		_ = locks.handles[index].Close()
	}
}

func writeDirectory(file string) string {
	directory := filepath.Dir(file)
	for candidate := directory; ; candidate = filepath.Dir(candidate) {
		if isCycleDirectory(candidate) {
			return candidate
		}
		if filepath.Dir(candidate) == candidate {
			return directory
		}
	}
}

func lockViewUpdates(updates []ViewUpdate) (*writeLocks, error) {
	directories := make([]string, 0, len(updates))
	for _, update := range updates {
		directories = append(directories, writeDirectory(update.File))
	}
	return lockWriteDirectories(directories)
}
