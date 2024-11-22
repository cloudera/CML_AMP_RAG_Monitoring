package lconfig

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type TestStruct struct {
	StringVal    string        `env:"STRING_VAL"`
	DefaultValue string        `env:"NON_EXISTANT" envDefault:"Hello"`
	EnvVal       string        `env:"ENV_VAL"`
	IntVal       int           `env:"INT_VAL"`
	BoolVal      bool          `env:"BOOL_VAL"`
	F32Val       float32       `env:"FLOAT32_VAL"`
	F64Val       float64       `env:"FLOAT64_VAL"`
	F64Array     []float64     `env:"FLOAT64_ARRAY" envSeparator:" "`
	TimeDuration time.Duration `env:"TIME_DURATION" envDefault:"5s"`
}

func TestConfigDir(t *testing.T) {
	dir, err := os.MkdirTemp("", "env")
	if err != nil {
		log.Fatal(err)
	}

	err = os.Setenv("ENV_VAL", "env value here")
	if err != nil {
		log.Fatal(err)
		return
	}

	err = os.Setenv("CONFIG_DIR", dir)
	if err != nil {
		log.Fatal(err)

		return
	}

	err = writeTestFiles(dir)
	if err != nil {
		log.Fatal(err)
		return
	}

	var test TestStruct
	err = Parse(&test)
	if err != nil {
		log.Fatal(err)
		return
	}

	assert.Equal(t, "a string value", test.StringVal)
	assert.Equal(t, "Hello", test.DefaultValue)
	assert.Equal(t, "env value here", test.EnvVal)
	assert.Equal(t, 123, test.IntVal)
	assert.Equal(t, true, test.BoolVal)
	assert.True(t, math.Abs(float64(3.14-test.F32Val)) < 0.001)
	assert.True(t, math.Abs(2.2e-308-test.F64Val) < 0.001)
	assert.Equal(t, 3, len(test.F64Array))
	assert.Equal(t, time.Second*5, test.TimeDuration)

	err = os.RemoveAll(dir)
	if err != nil {
		log.Fatal(err)
		return
	}
}

func writeTestFiles(dir string) error {
	err := os.WriteFile(filepath.Join(dir, "STRING_VAL"), []byte("a string value"), 0600)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(dir, "INT_VAL"), []byte("123"), 0600)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(dir, "BOOL_VAL"), []byte("true"), 0600)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(dir, "FLOAT32_VAL"), []byte("3.14"), 0600)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(dir, "FLOAT64_VAL"), []byte("2.2E-308"), 0600)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(dir, "FLOAT64_ARRAY"), []byte("0.0 0.1 0.2"), 0600)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(dir, "TIME_DURATION"), []byte("5s"), 0600)
	if err != nil {
		return err
	}
	return err
}
