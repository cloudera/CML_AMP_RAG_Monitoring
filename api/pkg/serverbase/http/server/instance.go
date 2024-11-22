package sbhttpserver

import (
	"github.com/dimfeld/httptreemux"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/app"
	cbhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/clientbase/http"
	lconfig "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/config"
	"net/http"
	"strconv"
	"time"
)

type Config struct {
	Port              int           `env:"SERVER_HTTP_PORT" envDefault:"3000"`
	ReadTimeout       time.Duration `env:"SERVER_HTTP_READ_TIMEOUT"  envDefault:"60s"`
	ReadHeaderTimeout time.Duration `env:"SERVER_HTTP_READ_HEADER_TIMEOUT" envDefault:"15s"`
	WriteTimeout      time.Duration `env:"SERVER_HTTP_WRITE_TIMEOUT" envDefault:"60s"`
	IdleTimeout       time.Duration `env:"SERVER_HTTP_IDLE_TIMEOUT" envDefault:"60s"` // Close idle connections after 60s
	MaxHeaderBytes    int           `env:"SERVER_HTTP_MAX_HEADER_BYTES"`
	MetricsProxy      string        `env:"SERVER_HTTP_METRICS_PROXY"`
	KeepEnvoyPath     bool          `env:"SERVER_HTTP_KEEP_ENVOY_PATH"`
}

func NewConfigFromEnv() (*Config, error) {
	var cfg Config
	err := lconfig.Parse(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

type Instance struct {
	app    *app.Instance
	client *cbhttp.Instance
	router *httptreemux.TreeMux
	server *http.Server
	config *Config
}

func NewInstance(cfg *Config, client *cbhttp.Instance, app *app.Instance) (*Instance, error) {
	router := httptreemux.New()
	router.RedirectTrailingSlash = false

	localServer := &http.Server{
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			envoyPreRouter(writer, request, cfg, router)
		}),
		Addr:              ":" + strconv.Itoa(cfg.Port),
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}

	return &Instance{
		app:    app,
		config: cfg,
		client: client,
		router: router,
		server: localServer,
	}, nil
}

func (instance *Instance) Register(server Server) error {
	instance.app.AddCloseFunc(func() error {
		err := server.Shutdown()
		return err
	})

	instance.registerStatusHandlers(server)

	if err := instance.registerHandlers(server); err != nil {
		return err
	}

	return nil
}
