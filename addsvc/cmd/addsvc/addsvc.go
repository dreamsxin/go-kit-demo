package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"

	"github.com/oklog/oklog/pkg/group"
	"google.golang.org/grpc"

	"github.com/dreamsxin/go-kit/log"
	kitgrpc "github.com/dreamsxin/go-kit/transport/grpc/server"

	addpb "go-kit-demo/addsvc/pb"
	"go-kit-demo/addsvc/pkg/addendpoint"
	"go-kit-demo/addsvc/pkg/addservice"
	"go-kit-demo/addsvc/pkg/addtransport"
)

func main() {
	// Define our flags. Your service probably won't need to bind listeners for
	// *all* supported transports, or support both Zipkin and LightStep, and so
	// on, but we do it here for demonstration purposes.
	fs := flag.NewFlagSet("addsvc", flag.ExitOnError)
	var (
		httpAddr = fs.String("http-addr", ":8081", "HTTP listen address")
		grpcAddr = fs.String("grpc-addr", ":8082", "gRPC listen address")
	)
	fs.Usage = usageFor(fs, os.Args[0]+" [flags]")
	fs.Parse(os.Args[1:])

	logger := log.NewNopLogger()

	// Build the layers of the service "onion" from the inside out. First, the
	// business logic service; then, the set of endpoints that wrap the service;
	// and finally, a series of concrete transport adapters. The adapters, like
	// the HTTP handler or the gRPC server, are the bridge between Go kit and
	// the interfaces that the transports expect. Note that we're not binding
	// them to ports or anything yet; we'll do that next.
	var (
		service     = addservice.New(&logger)
		endpoints   = addendpoint.New(service, &logger)
		httpHandler = addtransport.NewHTTPHandler(endpoints, &logger)
		grpcServer  = addtransport.NewGRPCServer(endpoints, &logger)
	)

	var g group.Group
	{
		// The HTTP listener mounts the Go kit HTTP handler we created.
		httpListener, err := net.Listen("tcp", *httpAddr)
		if err != nil {
			logger.Sugar().Errorln("transport", "HTTP", "during", "Listen", "err", err)
			os.Exit(1)
		}
		g.Add(func() error {
			logger.Sugar().Debugln("transport", "HTTP", "addr", *httpAddr)
			return http.Serve(httpListener, httpHandler)
		}, func(error) {
			httpListener.Close()
		})
	}
	{
		// The gRPC listener mounts the Go kit gRPC server we created.
		grpcListener, err := net.Listen("tcp", *grpcAddr)
		if err != nil {
			logger.Sugar().Errorln("transport", "gRPC", "during", "Listen", "err", err)
			os.Exit(1)
		}
		g.Add(func() error {
			logger.Sugar().Debugln("transport", "gRPC", "addr", *grpcAddr)

			baseServer := grpc.NewServer(grpc.UnaryInterceptor(kitgrpc.Interceptor))
			addpb.RegisterAddServer(baseServer, grpcServer)
			return baseServer.Serve(grpcListener)
		}, func(error) {
			grpcListener.Close()
		})
	}
	{
		// This function just sits and waits for ctrl-C.
		cancelInterrupt := make(chan struct{})
		g.Add(func() error {
			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
			select {
			case sig := <-c:
				return fmt.Errorf("received signal %s", sig)
			case <-cancelInterrupt:
				return nil
			}
		}, func(error) {
			close(cancelInterrupt)
		})
	}
	logger.Sugar().Errorln("exit", g.Run())
}

func usageFor(fs *flag.FlagSet, short string) func() {
	return func() {
		fmt.Fprintf(os.Stderr, "USAGE\n")
		fmt.Fprintf(os.Stderr, "  %s\n", short)
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "FLAGS\n")
		w := tabwriter.NewWriter(os.Stderr, 0, 2, 2, ' ', 0)
		fs.VisitAll(func(f *flag.Flag) {
			fmt.Fprintf(w, "\t-%s %s\t%s\n", f.Name, f.DefValue, f.Usage)
		})
		w.Flush()
		fmt.Fprintf(os.Stderr, "\n")
	}
}
