package lsql

import (
	ltest "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/test"
	"io/ioutil"
	_ "modernc.org/sqlite"
	"os"
)

func NewTestingConfig(t ltest.T) (*Config, error) {
	file, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() {
		_, err := file.Stat()
		if !os.IsNotExist(err) {
			os.RemoveAll(file.Name())
		}
	})
	return &Config{
		Engine:       "sqlite",
		DatabaseName: "test",
		Address:      file.Name(),
	}, nil
}
