package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
)

type httpResponseWriter interface {
	http.ResponseWriter
}

func jsonEncoder(w io.Writer) *json.Encoder {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder
}
