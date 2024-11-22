package lconfig

import (
	"encoding/json"
	"github.com/caarlos0/env/v6"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"os"
	"reflect"
	"strings"
)

var parseFuncs = map[reflect.Type]env.ParserFunc{
	reflect.TypeOf(*resource.NewQuantity(0, resource.DecimalSI)): env.ParserFunc(func(v string) (interface{}, error) {
		return resource.ParseQuantity(v)
	}),
	reflect.TypeOf(map[string]string{}): env.ParserFunc(func(v string) (interface{}, error) {
		ret := make(map[string]string)
		err := json.Unmarshal([]byte(v), &ret)
		return ret, err
	}),
}

func Parse(v interface{}) error {
	configDirPath := os.Getenv("CONFIG_DIR")
	opts := env.Options{}
	if configDirPath != "" {
		configDir, err := NewConfigDir(configDirPath)
		if err != nil {
			return err
		}
		opts.Environment, err = configDir.EnvironmentMap()
		if err != nil {
			return err
		}

		for _, existingEnv := range os.Environ() {
			envVar := strings.Split(existingEnv, "=")
			opts.Environment[envVar[0]] = os.Getenv(envVar[0])
		}
	}
	return errors.WithStack(env.ParseWithFuncs(v, parseFuncs, opts))
}

func MustParse(v interface{}) {
	if err := Parse(v); err != nil {
		panic(err)
	}
}

type ParseFuncs map[reflect.Type]env.ParserFunc

func (f ParseFuncs) With(t reflect.Type, fn env.ParserFunc) ParseFuncs {
	if f == nil {
		f = make(map[reflect.Type]env.ParserFunc)
	}
	f[t] = fn
	return f
}

func ParseWithFuncs(v interface{}, funcs ParseFuncs) error {
	newFuncs := make(map[reflect.Type]env.ParserFunc)
	for k, v := range funcs {
		newFuncs[k] = v
	}
	for k, v := range parseFuncs {
		newFuncs[k] = v
	}

	return errors.WithStack(env.ParseWithFuncs(v, newFuncs))
}
