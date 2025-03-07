package traefik_json_body2header

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
)

// Config the plugin configuration.
type Config struct {
	Mappings []Mapping `json:"mappings"`
}

// Mapping represents a body to header rule
type Mapping struct {
	Match    string `json:"match"`
	Property string `json:"property"`
	Header   string `json:"header"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		Mappings: []Mapping{},
	}
}

// internalMapping is the compiled config Mapping
type internalMapping struct {
	requestMatcher *regexp.Regexp
	property       string
	header         string
}

// newInternalMapping compiles the values from the provided mapping for later usage
func newInternalMapping(m Mapping) (*internalMapping, error) {
	if m.Property == "" {
		return nil, errors.New("property must not be empty")
	}
	if m.Header == "" {
		return nil, errors.New("header must not be empty")
	}

	// no match provided -> match all
	if m.Match == "" {
		m.Match = ".*"
	}
	requestMatcher, err := regexp.Compile(m.Match)
	if err != nil {
		return nil, err
	}

	return &internalMapping{
		requestMatcher: requestMatcher,
		property:       m.Property,
		header:         m.Header,
	}, nil
}

// Extractor is a Traefik Plugin, which tries to extract top level json values from the request body and sets them as header
type Extractor struct {
	name     string
	next     http.Handler
	mappings []*internalMapping
}

// New created a new Extractor plugin.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config == nil {
		config = CreateConfig()
	}

	mappings := make([]*internalMapping, 0, len(config.Mappings))
	for _, m := range config.Mappings {
		mapping, err := newInternalMapping(m)
		if err != nil {
			return nil, err
		}
		mappings = append(mappings, mapping)
	}

	return &Extractor{
		name:     name,
		mappings: mappings,
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

	// only act on non-empty bodies
	if len(data) > 0 {
		// try parsing json body
		var jbody map[string]any
		err = json.Unmarshal(data, &jbody)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		// iterate over mappings and extract values
		for _, m := range e.mappings {
			if !m.requestMatcher.MatchString(r.URL.String()) {
				// URL does not match -> continue
				continue
			}
			result, ok := jbody[m.property]
			if !ok {
				// no value found -> do not set
				continue
			}

			var value string
			switch v := result.(type) {
			case string:
				// do not marshal string values -> superfluous double quotes
				value = v
			default:
				val, err := json.Marshal(result)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				value = string(val)
			}

			r.Header.Set(m.header, value)
		}
	}

	r.Body = io.NopCloser(bytes.NewReader(data))
	e.next.ServeHTTP(w, r)
}
