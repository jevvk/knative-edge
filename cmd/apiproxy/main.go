package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"

	"knative.dev/edge/pkg/apiproxy/authentication"
	activatorconfig "knative.dev/edge/pkg/apiproxy/config"
	"knative.dev/edge/pkg/apiproxy/websockets"
	"knative.dev/edge/pkg/networking"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/configmap"
	configmapinformer "knative.dev/pkg/configmap/informer"
	"knative.dev/pkg/injection/sharedmain"
	pkglogging "knative.dev/pkg/logging"
	"knative.dev/pkg/logging/logkey"
	"knative.dev/pkg/metrics"
	"knative.dev/pkg/profiling"
	"knative.dev/pkg/signals"
	"knative.dev/pkg/system"
	"knative.dev/pkg/tracing"
	tracingconfig "knative.dev/pkg/tracing/config"
)

const (
	component = "apiproxy"
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

	flag.BoolVar(&initialize, "init", false, "initialiaze apiproxy mandatory components")
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
	sigCtx := signals.NewContext()

	kubeClient := kubeclient.Get(ctx)

	// Set up our logger.
	loggingConfig, err := sharedmain.GetLoggingConfig(ctx)
	if err != nil {
		log.Fatal("Error loading/parsing logging configuration: ", err)
	}

	logger, atomicLevel := pkglogging.NewLoggerFromConfig(loggingConfig, component)
	logger = logger.With(zap.String(logkey.ControllerType, component),
		zap.String(logkey.Pod, env.PodName))
	ctx = pkglogging.WithLogger(ctx, logger)
	defer flush(logger)

	profilingHandler := profiling.NewHandler(logger, false)

	servers := map[string]*http.Server{
		// "http1": pkgnet.NewServer(":"+strconv.Itoa(networking.HTTPPort), ah),
		// "h2c":     pkgnet.NewServer(":"+strconv.Itoa(networking.BackendHTTP2Port), ah),
		"websockets": websockets.NewServer(":"+strconv.Itoa(networking.HTTPPort), nil),
		"profile":    profiling.NewServer(profilingHandler),
	}

	errCh := make(chan error, len(servers))

	oct := tracing.NewOpenCensusTracer(tracing.WithExporterFull(networking.ApiProxyServiceName, env.PodIP, logger))

	tracerUpdater := configmap.TypeFilter(&tracingconfig.Config{})(func(name string, value interface{}) {
		cfg := value.(*tracingconfig.Config)
		if err := oct.ApplyConfig(cfg); err != nil {
			logger.Errorw("Unable to apply open census tracer config", zap.Error(err))
			return
		}
	})

	configMapWatcher := configmapinformer.NewInformedWatcher(kubeClient, system.Namespace())
	configStore := activatorconfig.NewStore(logger, tracerUpdater)
	configStore.WatchConfigs(configMapWatcher)

	// Watch the logging config map and dynamically update logging levels.
	configMapWatcher.Watch(pkglogging.ConfigMapName(), pkglogging.UpdateLevelFromConfigMap(logger, atomicLevel, component))

	logger.Info("Starting the knative edge apiproxy")

	// Wait for the signal to drain.
	select {
	case <-sigCtx.Done():
		logger.Info("Received SIGTERM")
	case err := <-errCh:
		logger.Errorw("Failed to run HTTP server", zap.Error(err))
	}
}

func flush(logger *zap.SugaredLogger) {
	logger.Sync()
	os.Stdout.Sync()
	os.Stderr.Sync()
	metrics.FlushExporter()
}
