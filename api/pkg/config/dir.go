package lconfig

import (
	"fmt"
	"github.com/spf13/afero"
	"io"
	"io/fs"
	"strings"
)

type ConfigDir struct {
	dirPath string
	fs      afero.Fs
}

func NewConfigDir(dirPath string) (*ConfigDir, error) {
	if dirPath == "" {
		return nil, fmt.Errorf("empty config dir path")
	}
	configDir := &ConfigDir{
		dirPath: dirPath,
		fs:      afero.NewBasePathFs(afero.NewOsFs(), dirPath),
	}

	stat, err := configDir.fs.Stat(".")
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("config dir path is not a directory")
	}
	return configDir, nil
}

func (config *ConfigDir) EnvironmentMap() (map[string]string, error) {
	envMap := make(map[string]string)

	err := afero.Walk(config.fs, ".", func(path string, fileInfo fs.FileInfo, err error) error {
		if fileInfo.IsDir() {
			return nil
		}
		name := fileInfo.Name()
		_, alreadyExists := envMap[name]
		if alreadyExists {
			return fmt.Errorf("duplicate configuration value %s", name)
		}
		file, err := config.fs.Open(path)
		if err != nil {
			return err
		}
		contents, err := io.ReadAll(file)
		if err != nil {
			return err
		}
		envMap[name] = strings.TrimSpace(string(contents))
		return nil
	})

	if err != nil {
		return nil, err
	}

	return envMap, err
}
