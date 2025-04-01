package goldenfile

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
)

func LoadJSON(datafile string, v any) error {
	_, filename, _, _ := runtime.Caller(1)
	data, err := os.ReadFile(filepath.Clean(filepath.Join(path.Dir(filename), "testdata", datafile)))
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}

func SaveJSON(datafile string, v any) error {
	shouldUpdate := false
	if name, ok := os.LookupEnv("UPDATE_GOLDEN_FILES"); ok {
		if update, err := strconv.ParseBool(name); err == nil {
			shouldUpdate = update
		}
	}
	if !shouldUpdate {
		return nil
	}

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	_, filename, _, _ := runtime.Caller(1)
	return os.WriteFile(filepath.Clean(filepath.Join(path.Dir(filename), "testdata", datafile)), data, 0600)
}
