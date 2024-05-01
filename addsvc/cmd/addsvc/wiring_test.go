package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dreamsxin/go-kit/log"

	"go-kit-demo/addsvc/pkg/addendpoint"
	"go-kit-demo/addsvc/pkg/addservice"
	"go-kit-demo/addsvc/pkg/addtransport"
)

func TestHTTP(t *testing.T) {
	logger := log.NewNopLogger()
	svc := addservice.New(&logger)
	eps := addendpoint.New(svc, &logger)
	mux := addtransport.NewHTTPHandler(eps, &logger)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, testcase := range []struct {
		method, url, body, want string
	}{
		{"GET", srv.URL + "/concat", `{"a":"1","b":"2"}`, `{"v":"12"}`},
		{"GET", srv.URL + "/sum", `{"a":1,"b":2}`, `{"v":3}`},
	} {
		req, _ := http.NewRequest(testcase.method, testcase.url, strings.NewReader(testcase.body))
		resp, _ := http.DefaultClient.Do(req)
		body, _ := io.ReadAll(resp.Body)
		if want, have := testcase.want, strings.TrimSpace(string(body)); want != have {
			t.Errorf("%s %s %s: want %q, have %q", testcase.method, testcase.url, testcase.body, want, have)
		}
	}
}
