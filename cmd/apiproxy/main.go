package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"

	"github.com/kelseyhightower/envconfig"
	"github.com/klyve/go-healthz"

	"knative.dev/edge/pkg/apiproxy"
	"knative.dev/edge/pkg/apiproxy/authentication"
	"knative.dev/edge/pkg/networking"
)

type config struct {
	PodName                  string `split_words:"true" required:"true"`
	PodIP                    string `split_words:"true" required:"true"`
	EdgeAuthenticationSecret string `split_words:"true" required:"true"`
}

func main() {
	// Set up a context that we can cancel to tell informers and other subprocesses to stop.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var initialize bool

	flag.BoolVar(&initialize, "init", false, "initialize apiproxy mandatory components")
	flag.Usage = func() {
		fmt.Printf("Usage: %s [-init]", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	var env config
	if err := envconfig.Process("", &env); err != nil {
		log.Fatal("Failed to process env: ", err)
	}

	if initialize {
		setup(ctx, env)
	} else {
		serve(ctx, env)
	}

	log.Printf("Exiting.")
}

func setup(ctx context.Context, env config) {
	authentication.Initialize(ctx, env.EdgeAuthenticationSecret)
}

func serve(ctx context.Context, env config) {
	ctx, cancel := context.WithCancel(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	apiproxy, apiproxyHealth := apiproxy.New(ctx, ":"+strconv.Itoa(networking.HTTPPort))

	healthzInstance := healthz.Instance{
		Detailed: true,
		Providers: []healthz.Provider{
			{
				Handle: apiproxyHealth,
				Name:   "apiproxy",
			},
		},
	}

	healthz := healthz.Server{
		ListenAddr: ":" + strconv.Itoa(networking.HealthzPort),
		Instance:   &healthzInstance,
	}

	go healthz.Start()

	log.Printf("Starting apiproxy.")
	go apiproxy.ListenAndServe()

	<-quit
	cancel()

	log.Printf("Stopped apiproxy.")
}
