package body2header

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const TestHeader = "SAMPLE_HEADER"

type Input interface {
	AsReader() io.Reader
}

func TestExtractor_ServeHTTP(t *testing.T) {
	tests := []struct {
		name    string
		input   Input
		mapping Mapping
		want    string
		error   bool
	}{
		{
			name:    "empty",
			input:   String(""),
			mapping: Mapping{},
			want:    "",
			error:   true,
		},
		{
			name:    "non json",
			input:   String("I AM NOT A JSON"),
			mapping: Mapping{},
			want:    "",
			error:   true,
		},
		{
			name:    "top level",
			input:   Map(map[string]any{"a": "b"}),
			mapping: Mapping{Path: "a", Header: TestHeader},
			want:    "b",
			error:   false,
		},
		{
			name:    "no value found",
			input:   Map(map[string]any{"a": "b"}),
			mapping: Mapping{Path: "a.b", Header: TestHeader},
			want:    "",
			error:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vh := validationHandler{
				t:      t,
				header: tt.mapping.Header,
				want:   tt.want,
				error:  tt.error,
			}

			e, err := New(nil, vh, newConfig(tt.mapping), tt.name)
			if err != nil {
				t.Errorf("Failed initializing Extractor: %s", err)
				t.FailNow()
			}

			recorder := httptest.NewRecorder()
			e.ServeHTTP(recorder, httptest.NewRequest("GET", "/", tt.input.AsReader()))

			// Validate response code
			if recorder.Code != http.StatusOK && !tt.error {
				t.Errorf("expected status code 200 but got %d", recorder.Code)
				t.FailNow()
			}
		})
	}

}

type String string

func (s String) AsReader() io.Reader {
	return io.NopCloser(strings.NewReader(string(s)))
}

type Map map[string]any

func (m Map) AsReader() io.Reader {
	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return io.NopCloser(bytes.NewReader(b))
}

func newConfig(m Mapping) *Config {
	c := CreateConfig()
	if len(m.Header) > 0 {
		c.Mappings = append(c.Mappings, m)
	}

	return c
}

type validationHandler struct {
	t      *testing.T
	header string
	want   string
	error  bool
}

func (vh validationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if vh.header == "" {
		return
	}

	val := r.Header.Get(vh.header)
	if val != vh.want && !vh.error {
		vh.t.Errorf("expected %s but got %s", vh.want, val)
		vh.t.FailNow()
	}
}
