package cmd

import (
	"os"
	"path/filepath"
)

func saveConfig(path string, data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".claimctl-config-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
