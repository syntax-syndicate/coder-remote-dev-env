package provisionersdk
import (
	"fmt"
	"errors"
	"context"
	"path/filepath"
	"time"
	"github.com/spf13/afero"
	"cdr.dev/slog"
)
// CleanStaleSessions browses the work directory searching for stale session
// directories. Coder provisioner is supposed to remove them once after finishing the provisioning,
// but there is a risk of keeping them in case of a failure.
func CleanStaleSessions(ctx context.Context, workDirectory string, fs afero.Fs, now time.Time, logger slog.Logger) error {
	entries, err := afero.ReadDir(fs, workDirectory)
	if err != nil {
		return fmt.Errorf("can't read %q directory", workDirectory)
	}
	for _, fi := range entries {
		dirName := fi.Name()
		if fi.IsDir() && isValidSessionDir(dirName) {
			sessionDirPath := filepath.Join(workDirectory, dirName)
			modTime := fi.ModTime() // fallback to modTime if modTime is not available (afero)
			if modTime.Add(staleSessionRetention).After(now) {
				continue
			}
			logger.Info(ctx, "remove stale session directory", slog.F("session_path", sessionDirPath))
			err = fs.RemoveAll(sessionDirPath)
			if err != nil {
				return fmt.Errorf("can't remove %q directory: %w", sessionDirPath, err)
			}
		}
	}
	return nil
}
func isValidSessionDir(dirName string) bool {
	match, err := filepath.Match(sessionDirPrefix+"*", dirName)
	return err == nil && match
}
