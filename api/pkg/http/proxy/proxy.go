package proxy

import (
	"bufio"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"

	sbhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http"
)

type Proxy struct {
	HttpClient http.Client
	Remote     string
	Configure  func(new, original *http.Request)
}

func (p *Proxy) Handle(w http.ResponseWriter, r *http.Request, params map[string]string) {

	defer func() {
		// Exhaust the input so that it can be re-used
		_, _ = io.Copy(ioutil.Discard, r.Body)
		_ = r.Body.Close()
	}()

	// Create a request with the right header
	request, err := http.NewRequest(r.Method, p.Remote+r.URL.Path, r.Body)
	if err != nil {
		sbhttp.ReturnError(w, http.StatusInternalServerError, "failed to create internal request", err)
		return
	}
	if p.Configure != nil {
		p.Configure(request, r)
	}

	// Make the request
	resp, err := p.HttpClient.Do(request)
	if resp != nil {
		defer func() {
			// Exhaust the response so that it can be re-used
			_, _ = io.Copy(ioutil.Discard, resp.Body)
			_ = resp.Body.Close()
		}()
	}

	if err != nil {
		sbhttp.ReturnError(w, http.StatusInternalServerError, "failed to perform request", err)
		return
	}

	if resp.StatusCode != 200 {
		responseBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			sbhttp.ReturnError(w, http.StatusInternalServerError, "failed to read response body", err)
			return
		}
		sbhttp.ReturnError(w, http.StatusBadGateway, string(responseBody), nil)
		return
	}
	_, err = io.Copy(w, bufio.NewReaderSize(resp.Body, 1024*32))
	if err != nil {
		log.Printf("failed to copy response body: %s", err)
	}
}
