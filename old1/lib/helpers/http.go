package helpers

import (
	"encoding/json"
	"io"
	"net/http"
)

func ReadHttpResponseBody(response *http.Response) []byte {
	var body []byte
	body, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()

	return body
}

func ReadHttpResponseBodyAsString(response *http.Response) string {
	body := ReadHttpResponseBody(response)
	return string(body)
}

func UnmarshalJSONResponseBody(response *http.Response) interface{} {
	var result interface{}
	
	body := ReadHttpResponseBody(response)
	_ = json.Unmarshal(body, &result)

	return result
}
