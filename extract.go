package body2header

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/tidwall/gjson"
)

// Config the plugin configuration.
type Config struct {
	Mappings []Mapping `json:"mappings"`
}

// Mapping represents a body to header rule
type Mapping struct {
	Path   string `json:"path"`
	Header string `json:"header"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		Mappings: []Mapping{},
	}
}

// Extractor is a Traefik Plugin, which tries to extract json values from the request body and sets them as header
type Extractor struct {
	name     string
	next     http.Handler
	mappings []Mapping
}

// New created a new Extractor plugin.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config == nil {
		config = CreateConfig()
	}

	return &Extractor{
		name:     name,
		mappings: config.Mappings,
		next:     next,
	}, nil
}

func (e *Extractor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body bytes.Buffer
	tee := io.TeeReader(r.Body, &body)

	data, err := io.ReadAll(tee)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	// check body to be valid json
	if !gjson.ValidBytes(data) {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
	}

	// iterate over mappings and extract values
	for _, m := range e.mappings {
		result := gjson.GetBytes(data, m.Path)
		if result.Exists() {
			r.Header.Set(m.Header, result.String())
		}
	}

	r.Body = io.NopCloser(bytes.NewReader(data))
	e.next.ServeHTTP(w, r)
}
