package go_bard

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	config ClientConfig

	requestBuilder requestBuilder
}

func NewClient(authToken, bardApi string) *Client {
	config := DefaultConfig(authToken, bardApi)
	return NewClientWithConfig(config)
}

func NewClientWithConfig(config ClientConfig) *Client {
	return &Client{
		config:         config,
		requestBuilder: newRequestBuilder(),
	}
}

func (c *Client) sendRequest(req *http.Request, v any) error {
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req.Header.Set("Authorization", c.config.authToken)

	// Check whether Content-Type is already set, Upload Files API requires
	// Content-Type == multipart/form-data
	contentType := req.Header.Get("Content-Type")
	if contentType == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusBadRequest {
		return c.handleErrorResp(res)
	}

	return decodeResponse(res.Body, v)
}

func decodeResponse(body io.Reader, v any) error {
	if v == nil {
		return nil
	}

	if result, ok := v.(*string); ok {
		return decodeString(body, result)
	}
	return json.NewDecoder(body).Decode(v)
}

func decodeString(body io.Reader, output *string) error {
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	*output = string(b)
	return nil
}

func (c *Client) fullURL(suffix string) string {
	return fmt.Sprintf("%s%s", c.config.BaseURL, suffix)
}

func (c *Client) handleErrorResp(resp *http.Response) error {
	var errRes ErrorResponse
	err := json.NewDecoder(resp.Body).Decode(&errRes)

	if err != nil || errRes.Error == nil {
		reqErr := RequestError{
			HTTPStatusCode: resp.StatusCode,
			Err:            err,
		}
		return fmt.Errorf("error, %w", &reqErr)
	}
	errRes.Error.HTTPStatusCode = resp.StatusCode
	return fmt.Errorf("error, status code: %d, message: %w", resp.StatusCode, errRes.Error)
}
