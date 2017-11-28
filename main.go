package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/Loofort/smartdoor/eyes"
	"github.com/Loofort/smartdoor/eyes/cv"
	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"sourcegraph.com/sourcegraph/appdash"
	appdashtracer "sourcegraph.com/sourcegraph/appdash/opentracing"
	"sourcegraph.com/sourcegraph/appdash/traceapp"
)

func main() {
	closer := openTracer()
	defer closer.Close()

	idleDispather := make(chan struct{})
	idleTracker := make(chan struct{})
	idleRecognizer := make(chan struct{})
	idlec := eyes.Idler(idleDispather, idleTracker, idleRecognizer)
	detectSig := eyes.Timer(500*time.Millisecond, idlec)

	cadrec := make(chan cv.Cadre, 24) // the buffer should be calculated based on fps, acceptable delay is about 1 sec.
	cadreFuncc := make(chan eyes.CadreFunc)
	go eyes.Detector(cadrec, detectSig, cadreFuncc)

	err := ProduceCadre(cadrec)
	if err != nil {
		log.Fatal(err)
	}

	framec := make(chan eyes.Frame)
	trackersMapc := make(chan map[int]cv.Tracker)
	rectsMapc := make(chan map[int]cv.Rect)
	cadreidc := make(chan int)
	rframesMapc := make(chan map[int]eyes.RFrame)
	go eyes.Synchronizer(cadreFuncc, framec, trackersMapc, rectsMapc, cadreidc, rframesMapc)

	rframec := eyes.RunTracker(framec, trackersMapc, idleTracker)
	personc := eyes.RunDispather(rectsMapc, cadreidc, rframec, rframesMapc, idleDispather)

	//eyes.RunRecognizer(shapec, personSIDc, personc, idleRecognizer)

	// read persons
	for person := range personc {
		fmt.Printf("get person %v \n", person)
	}

}

func idleMock() chan struct{} {
	ticker := time.NewTicker(100 * time.Millisecond)
	idlec := make(chan struct{})
	go func() {
		for range ticker.C {
			idlec <- struct{}{}
		}
	}()

	return idlec
}

func ProduceCadre(cadrec chan cv.Cadre) error {
	src, err := cv.NewCapture("/home/illia/work/go/src/smartdoor.bk/data/VIDEO/me/01.asf")
	if err != nil {
		return err
	}

	go func() {
		cnt := 0
		for {
			cadre, err := cv.WaitForCadre(src)
			if err != nil {
				log.Printf("err: %v; shoutdown in 10 minutes\n", err)
				//time.Sleep(10 * time.Minute)
				log.Fatal(err)
			}
			cnt++
			fmt.Printf("#%d ", cnt)
			if cnt > 20 {
				//time.Sleep(40 * time.Minute)
				time.Sleep(10 * time.Millisecond)
			} else {
				time.Sleep(10 * time.Millisecond)
			}

			span := cadre.NewSpan("wait_for_detector")

			select {
			case cadrec <- cadre:
			default:
			}

			span.Finish()
		}
	}()
	return nil
}

func openTracer() io.Closer {
	// Sample configuration for testing. Use constant sampling to sample every trace
	// and enable LogSpan to log every span via configured Logger.
	cfg := jaegercfg.Configuration{
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LocalAgentHostPort: "127.0.0.1:6831",
		},
	}

	// Example logger and metrics factory. Use github.com/uber/jaeger-client-go/log
	// and github.com/uber/jaeger-lib/metrics respectively to bind to real logging and metrics
	// frameworks.
	//jLogger := jaegerlog.StdLogger
	//jMetricsFactory := metrics.NullFactory

	// Initialize tracer with a logger and a metrics factory
	closer, err := cfg.InitGlobalTracer(
		"serviceName",
		//jaegercfg.Logger(jLogger),
		//jaegercfg.Metrics(jMetricsFactory),
		//jaegercfg.Observer(rpcmetrics.NewObserver(jMetricsFactory, rpcmetrics.DefaultNameNormalizer)),
	)
	if err != nil {
		log.Fatalf("Could not initialize jaeger tracer: %s", err.Error())
		return nil
	}

	return closer
}

func openTracerAppDash() {
	memStore := appdash.NewMemoryStore()
	store := &appdash.RecentStore{
		MinEvictAge: 20 * time.Minute,
		DeleteStore: memStore,
	}

	// Start the Appdash web UI on port 8700.
	url, err := url.Parse("http://localhost:8700")
	if err != nil {
		log.Fatal(err)
	}
	tapp, err := traceapp.New(nil, url)
	if err != nil {
		log.Fatal(err)
	}
	tapp.Store = store
	tapp.Queryer = memStore
	log.Println("Appdash web UI running on HTTP :8700")
	go func() {
		log.Fatal(http.ListenAndServe(":8700", tapp))
	}()

	// We will use a local collector (as we are running the Appdash web UI
	// embedded within our app).
	//
	// A collector is responsible for collecting the information about traces
	// (i.e. spans and annotations) and placing them into a store. In this app
	// we use a local collector (we could also use a remote collector, sending
	// the information to a remote Appdash collection server).
	collector := appdash.NewLocalCollector(store)

	// Here we use the local collector to create a new opentracing.Tracer
	tracer := appdashtracer.NewTracer(collector)
	opentracing.InitGlobalTracer(tracer)

}
