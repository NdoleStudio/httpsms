package discord

import (
	"bytes"
	"errors"
	"net/http"
	"strconv"
)

// Response captures the http response
type Response struct {
	HTTPResponse *http.Response
	Body         *[]byte
}

// Error ensures that the response can be decoded into a string inc ase it's an error response
func (r *Response) Error() error {
	switch r.HTTPResponse.StatusCode {
	case 200, 201, 202, 204, 205:
		return nil
	default:
		return errors.New(r.errorMessage())
	}
}

func (r *Response) errorMessage() string {
	var buf bytes.Buffer
	buf.WriteString(strconv.Itoa(r.HTTPResponse.StatusCode))
	buf.WriteString(": ")
	buf.WriteString(http.StatusText(r.HTTPResponse.StatusCode))
	buf.WriteString(", Body: ")
	buf.Write(*r.Body)

	return buf.String()
}
