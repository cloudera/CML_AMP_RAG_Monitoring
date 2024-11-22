package sbswagger

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	lhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http"
	sbhttpbase "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/base"
	sbhttpserver "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/server"
	swaggerui "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/swagger/ui"
)

type Server struct {
	readinessProviders []sbhttpserver.ReadinessProvider
	livenessProviders  []sbhttpserver.LivenessProvider
	shutdownProviders  []sbhttpserver.ShutdownProvider
	handler            http.Handler
	spec               []byte
}

func New(cfg interface{}, handler http.Handler, spec []byte) *Server {
	readinessProviders := make([]sbhttpserver.ReadinessProvider, 0)
	livenessProviders := make([]sbhttpserver.LivenessProvider, 0)
	shutdownProviders := make([]sbhttpserver.ShutdownProvider, 0)

	swaggerui.AssetNames()

	if cfg != nil {
		rcfg := reflect.ValueOf(cfg)
		for i := 0; i < rcfg.NumField(); i++ {
			if provider, ok := rcfg.Field(i).Interface().(sbhttpserver.ReadinessProvider); ok {
				readinessProviders = append(readinessProviders, provider)
			}
			if provider, ok := rcfg.Field(i).Interface().(sbhttpserver.LivenessProvider); ok {
				livenessProviders = append(livenessProviders, provider)
			}
			if provider, ok := rcfg.Field(i).Interface().(sbhttpserver.ShutdownProvider); ok {
				shutdownProviders = append(shutdownProviders, provider)
			}
		}
	}

	return &Server{
		readinessProviders: readinessProviders,
		livenessProviders:  livenessProviders,
		shutdownProviders:  shutdownProviders,
		handler:            handler,
		spec:               spec,
	}
}

func (s *Server) GetHandlers() []sbhttpserver.HandleDescription {
	handlers := []sbhttpserver.HandleDescription{
		{
			Path:    "/external/swagger-ui/",
			Method:  "GET",
			Handler: s.swaggerFile,
		},
		{
			Path:    "/external/swagger-ui/*file",
			Method:  "GET",
			Handler: s.swaggerFile,
		},
	}
	if s.handler != nil {
		handlers = append(handlers, sbhttpserver.HandleDescription{
			NotFound: true,
			Handler: func(request *sbhttpbase.Request) {
				s.handler.ServeHTTP(request.Writer, request.Request)
			},
		})
	}
	return handlers
}

func (s *Server) swaggerFile(request *sbhttpbase.Request) {
	w := request.Writer
	r := request.Request
	ps := request.Params
	name := ps["file"]
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}
	if name == "swagger.json" {
		referer := r.Header.Get("Referer")
		refererUrl, err := url.Parse(referer)
		if err != nil {
			panic(err)
		}
		spec := s.spec

		newSwagger := string(spec)
		newSwagger = strings.ReplaceAll(newSwagger, `"localhost:3000"`, fmt.Sprintf(`"%s"`, refererUrl.Hostname()))
		newSwagger = strings.ReplaceAll(newSwagger, fmt.Sprintf(`"basePath": "%s"`, config.BasePath), fmt.Sprintf(`"basePath": "%s"`, config.NewBasePath))
		newSwagger = strings.ReplaceAll(newSwagger, fmt.Sprintf(`"basePath":"%s"`, config.BasePath), fmt.Sprintf(`"basePath":"%s"`, config.NewBasePath))
		newSwagger = strings.ReplaceAll(newSwagger, `"http"`, `"https"`)
		spec = []byte(newSwagger)

		w.Write(spec)
		return
	}
	if name == "" {
		name = "index.html"
	}
	data, err := swaggerui.Asset(name)
	if err != nil {
		log.Printf("asset not found: %s", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if name == "index.html" {
		data = []byte(strings.ReplaceAll(string(data), "https://petstore.swagger.io/v2/swagger.json", "./swagger.json"))
	}

	w.Header().Set("Content-Type", lhttp.InferContentType(name, data))
	w.Write(data)
}

func (s *Server) Live(ctx context.Context) error {
	for _, p := range s.livenessProviders {
		if err := p.Live(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) Ready(ctx context.Context) error {
	for _, p := range s.readinessProviders {
		if err := p.Ready(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) Shutdown() error {
	errors := make([]error, 0)
	for _, ss := range s.shutdownProviders {
		if err := ss.Shutdown(); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		errorMessage := ""
		for _, err := range errors {
			errorMessage += err.Error() + "\n"
		}
		return fmt.Errorf(errorMessage)
	}
	return nil
}
