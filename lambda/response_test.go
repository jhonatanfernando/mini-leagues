package main

import "testing"

func TestJSONResponse(t *testing.T) {
	assertResponse(t, jsonResponse(202, map[string]string{"message": "ok"}), nil, 202, `{"message":"ok"}`)
	assertResponse(t, jsonResponse(200, make(chan int)), nil, 500, `{"message":"Unable to encode response"}`)
}
