package sbswagger

import lconfig "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/config"

type ConfigType struct {
	BasePath    string `env:"SWAGGER_BASE_PATH" envDefault:"/external"`
	NewBasePath string `env:"NEW_SWAGGER_BASE_PATH"`
	Title       string `env:"SWAGGER_TITLE"`
	Version     string `env:"SWAGGER_VERSION"`
}

var config ConfigType

func init() {
	lconfig.MustParse(&config)
}
