package goldenfile

import (
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
)

func LoadFile(datafile string) ([]byte, error) {
	_, filename, _, _ := runtime.Caller(1)
	return os.ReadFile(filepath.Clean(filepath.Join(path.Dir(filename), "testdata", datafile)))
}

func SaveFile(datafile string, data []byte) error {
	shouldUpdate := false
	if name, ok := os.LookupEnv("UPDATE_GOLDEN_FILES"); ok {
		if update, err := strconv.ParseBool(name); err == nil {
			shouldUpdate = update
		}
	}
	if !shouldUpdate {
		return nil
	}
	_, filename, _, _ := runtime.Caller(1)
	return os.WriteFile(filepath.Clean(filepath.Join(path.Dir(filename), "testdata", datafile)), data, 0600)
}
