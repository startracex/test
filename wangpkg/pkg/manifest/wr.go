package manifest

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
)

func WriteRequest(req *http.Request, w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s %s %s\r\n",
		req.Method,
		req.URL.RequestURI(),
		req.Proto,
	); err != nil {
		return err
	}

	for k, vv := range req.Header {
		k = textproto.CanonicalMIMEHeaderKey(k)
		for _, v := range vv {
			if _, err := fmt.Fprintf(w, "%s: %s\r\n", k, v); err != nil {
				return err
			}
		}
	}

	if _, err := w.Write([]byte("\r\n")); err != nil {
		return err
	}

	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return err
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		if _, err := w.Write(bodyBytes); err != nil {
			return err
		}
	}

	return nil
}
