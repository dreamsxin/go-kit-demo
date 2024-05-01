package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/dreamsxin/go-kit/endpoint"
	"github.com/dreamsxin/go-kit/log"
	consulsd "github.com/dreamsxin/go-kit/sd/consul"
	transportclient "github.com/dreamsxin/go-kit/transport/http/client"
	transportserver "github.com/dreamsxin/go-kit/transport/http/server"

	"go-kit-demo/addsvc/pkg/addendpoint"
	"go-kit-demo/addsvc/pkg/addservice"
	"go-kit-demo/addsvc/pkg/addtransport"

	sd "github.com/dreamsxin/go-kit/sd/endpointer"
	lb "github.com/dreamsxin/go-kit/sd/endpointer/balancer"
	"github.com/dreamsxin/go-kit/sd/endpointer/executor"
)

func main() {
	var (
		httpAddr     = flag.String("http.addr", ":8000", "Address for HTTP (JSON) server")
		consulAddr   = flag.String("consul.addr", "", "Consul agent address")
		retryMax     = flag.Int("retry.max", 3, "per-request retries to different instances")
		retryTimeout = flag.Duration("retry.timeout", 500*time.Millisecond, "per-request timeout, including retries")
	)
	flag.Parse()

	logger, _ := log.NewDevelopment()

	// Service discovery domain. In this example we use Consul.
	var client consulsd.Client
	{
		consulConfig := api.DefaultConfig()
		if len(*consulAddr) > 0 {
			consulConfig.Address = *consulAddr
		}
		consulClient, err := api.NewClient(consulConfig)
		if err != nil {
			logger.Error("err", zap.Error(err))
			os.Exit(1)
		}
		client = consulsd.NewClient(consulClient)
	}
	ctx := context.Background()
	r := mux.NewRouter()
	// addsvc routes.
	{
		// Each method gets constructed with a factory. Factories take an
		// instance string, and return a specific endpoint. In the factory we
		// dial the instance string we get from Consul, and then leverage an
		// addsvc client package to construct a complete service. We can then
		// leverage the addsvc.Make{Sum,Concat}Endpoint constructors to convert
		// the complete service to specific endpoint.
		var (
			tags        = []string{}
			passingOnly = true
			endpoints   = addendpoint.Set{}
			instancer   = consulsd.NewInstancer(client, logger, "addsvc", passingOnly, consulsd.TagsInstancerOptions(tags))
		)
		{
			factory := addsvcFactory(addendpoint.MakeSumEndpoint, logger)
			endpointer := sd.NewEndpointer(instancer, factory, logger)
			balancer := lb.NewRoundRobin(endpointer)
			retry := executor.Retry(*retryMax, *retryTimeout, balancer)
			endpoints.SumEndpoint = retry
		}
		{
			factory := addsvcFactory(addendpoint.MakeConcatEndpoint, logger)
			endpointer := sd.NewEndpointer(instancer, factory, logger)
			balancer := lb.NewRoundRobin(endpointer)
			retry := executor.Retry(*retryMax, *retryTimeout, balancer)
			endpoints.ConcatEndpoint = retry
		}

		// Here we leverage the fact that addsvc comes with a constructor for an
		// HTTP handler, and just install it under a particular path prefix in
		// our router.

		r.PathPrefix("/addsvc").Handler(http.StripPrefix("/addsvc", addtransport.NewHTTPHandler(endpoints, logger)))
	}

	// stringsvc routes.
	{
		// addsvc had lots of nice importable Go packages we could leverage.
		// With stringsvc we are not so fortunate, it just has some endpoints
		// that we assume will exist. So we have to write that logic here. This
		// is by design, so you can see two totally different methods of
		// proxying to a remote service.

		var (
			passingOnly = true
			uppercase   endpoint.Endpoint
			count       endpoint.Endpoint
			instancer   = consulsd.NewInstancer(client, logger, "stringsvc", passingOnly)
		)
		{
			factory := stringsvcFactory(ctx, "GET", "/uppercase")
			endpointer := sd.NewEndpointer(instancer, factory, logger)
			balancer := lb.NewRoundRobin(endpointer)
			retry := executor.Retry(*retryMax, *retryTimeout, balancer)
			uppercase = retry
		}
		{
			factory := stringsvcFactory(ctx, "GET", "/count")
			endpointer := sd.NewEndpointer(instancer, factory, logger)
			balancer := lb.NewRoundRobin(endpointer)
			retry := executor.Retry(*retryMax, *retryTimeout, balancer)
			count = retry
		}

		// We can use the transport/http.Server to act as our handler, all we
		// have to do provide it with the encode and decode functions for our
		// stringsvc methods.

		r.Handle("/stringsvc/uppercase", transportserver.NewServer(uppercase, decodeUppercaseRequest, encodeJSONResponse))
		r.Handle("/stringsvc/count", transportserver.NewServer(count, decodeCountRequest, encodeJSONResponse))
	}

	// Interrupt handler.
	errc := make(chan error)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-c)
	}()

	// HTTP transport.
	go func() {
		logger.Sugar().Debugln("transport", "HTTP", "addr", *httpAddr)
		errc <- http.ListenAndServe(*httpAddr, r)
	}()

	// Run!
	logger.Sugar().Errorln("exit", <-errc)
}

func addsvcFactory(makeEndpoint func(addservice.Service) endpoint.Endpoint, logger *log.Logger) endpoint.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {
		// We could just as easily use the HTTP or Thrift client package to make
		// the connection to addsvc. We've chosen gRPC arbitrarily. Note that
		// the transport is an implementation detail: it doesn't leak out of
		// this function. Nice!

		conn, err := grpc.Dial(instance, grpc.WithInsecure())
		if err != nil {
			return nil, nil, err
		}
		service := addtransport.NewGRPCClient(conn, logger)
		endpoint := makeEndpoint(service)

		// Notice that the addsvc gRPC client converts the connection to a
		// complete addsvc, and we just throw away everything except the method
		// we're interested in. A smarter factory would mux multiple methods
		// over the same connection. But that would require more work to manage
		// the returned io.Closer, e.g. reference counting. Since this is for
		// the purposes of demonstration, we'll just keep it simple.

		return endpoint, conn, nil
	}
}

func stringsvcFactory(ctx context.Context, method, path string) endpoint.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {
		if !strings.HasPrefix(instance, "http") {
			instance = "http://" + instance
		}
		tgt, err := url.Parse(instance)
		if err != nil {
			return nil, nil, err
		}
		tgt.Path = path

		// Since stringsvc doesn't have any kind of package we can import, or
		// any formal spec, we are forced to just assert where the endpoints
		// live, and write our own code to encode and decode requests and
		// responses. Ideally, if you write the service, you will want to
		// provide stronger guarantees to your clients.

		var (
			enc transportclient.EncodeRequestFunc
			dec transportclient.DecodeResponseFunc
		)
		switch path {
		case "/uppercase":
			enc, dec = encodeJSONRequest, decodeUppercaseResponse
		case "/count":
			enc, dec = encodeJSONRequest, decodeCountResponse
		default:
			return nil, nil, fmt.Errorf("unknown stringsvc path %q", path)
		}

		return transportclient.NewClient(method, tgt, enc, dec).Endpoint(), nil, nil
	}
}

func encodeJSONRequest(_ context.Context, req *http.Request, request interface{}) (*http.Request, error) {
	// Both uppercase and count requests are encoded in the same way:
	// simple JSON serialization to the request body.
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(request); err != nil {
		return req, err
	}
	req.Body = ioutil.NopCloser(&buf)
	return req, nil
}

func encodeJSONResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

// I've just copied these functions from stringsvc3/transport.go, inlining the
// struct definitions.

func decodeUppercaseResponse(ctx context.Context, resp *http.Response) (interface{}, error) {
	var response struct {
		V   string `json:"v"`
		Err string `json:"err,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func decodeCountResponse(ctx context.Context, resp *http.Response) (interface{}, error) {
	var response struct {
		V int `json:"v"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func decodeUppercaseRequest(ctx context.Context, req *http.Request) (interface{}, error) {
	var request struct {
		S string `json:"s"`
	}
	if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func decodeCountRequest(ctx context.Context, req *http.Request) (interface{}, error) {
	var request struct {
		S string `json:"s"`
	}
	if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}
