package lconfig

import (
	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"io"
	"os"
	"path/filepath"
	"time"
)

type DynamicConfigUnmarshal func(rawVal interface{}) error
type DynamicConfigCallback func(unmarshal DynamicConfigUnmarshal) error

func LoadDynamicConfig(filename string, filesystem afero.Fs, callback DynamicConfigCallback) error {

	err := callback(func(rawVal interface{}) error {
		return LoadStaticYamlConfig(filename, filesystem, rawVal)
	})
	if err != nil {
		log.Printf("failed to perform first load %s", err)
		return err
	}

	go func() {
		for range time.NewTicker(time.Second).C {
			err := callback(func(rawVal interface{}) error {
				return LoadStaticYamlConfig(filename, filesystem, rawVal)
			})
			if err != nil {
				log.Printf("failed to refresh file %s", err)
			}
		}
	}()

	log.Printf("loaded config file")

	return nil
}

func LoadStaticYamlConfig(filename string, filesystem afero.Fs, target interface{}) error {
	if os.Getenv("TELEPRESENCE_ROOT") != "" {
		filename = filepath.Join(os.Getenv("TELEPRESENCE_ROOT"), filename)
	}

	file, err := filesystem.OpenFile(filename, os.O_RDONLY, 0)
	if err != nil {
		return err
	}

	content, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(content, target)
}
