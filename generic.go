package csfloat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type GenericResponse struct {
	// Ratelimits will have zero values if the request fails completly.
	Ratelimits Ratelimits `json:"-"`
	// Error will only be set if an error happened after successfully reaching
	// the server. However, there might still be other errors, for example when
	// decoding the server response.
	Error *Error `json:"-"`
}

func (response *GenericResponse) setRatelimits(ratelimits *Ratelimits) {
	response.Ratelimits = *ratelimits
}
func (response *GenericResponse) setError(err *Error) {
	response.Error = err
}
func (response *GenericResponse) responseBody() any {
	// By default, we don't carry any data here.
	return nil
}

type Response interface {
	setError(*Error)
	setRatelimits(*Ratelimits)
	// responseBody must return any pointer value that we'll JSON-decode into.
	responseBody() any
}

func handleRequest[T Response](
	client *http.Client,
	method string,
	endpoint string,
	apiKey string,
	payload any,
	form url.Values,
	result T,
) (T, error) {
	var body io.Reader
	var buffer bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&buffer).Encode(payload); err != nil {
			return result, fmt.Errorf("error encoding payload: %w", err)
		}
		body = &buffer
	}

	request, err := http.NewRequest(
		method,
		endpoint,
		body)

	request.URL.RawQuery = form.Encode()

	request.Header.Set("Authorization", apiKey)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Content-Length", strconv.Itoa(buffer.Len()))
	}

	if err != nil {
		return result, fmt.Errorf("error creating request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return result, fmt.Errorf("error sending request: %w", err)
	}

	ratelimits, err := ratelimitsFrom(response)
	if err != nil {
		return result, fmt.Errorf("error getting ratelimits: %w", err)
	}
	result.setRatelimits(&ratelimits)

	if response.StatusCode != http.StatusOK {
		csfloatError, err := errorFrom(response)
		if err != nil {
			return result, fmt.Errorf("invalid status code, couldn't read error message: %d",
				response.StatusCode)
		}
		result.setError(&csfloatError)

		return result, fmt.Errorf("invalid status code: %d; %v", response.StatusCode, csfloatError)
	}

	if target := result.responseBody(); target != nil {
		if err := json.NewDecoder(response.Body).Decode(result.responseBody()); err != nil {
			return result, fmt.Errorf("error decoding response: %w", err)
		}
	}

	return result, nil
}

func concatInts[Number int | uint](n ...Number) string {
	var b strings.Builder
	for i, val := range n {
		if i != 0 {
			b.WriteRune(',')
		}
		b.WriteString(fmt.Sprintf("%d", val))
	}
	return b.String()
}
