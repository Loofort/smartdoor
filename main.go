package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"runtime"
	"time"

	"github.com/Loofort/smartdoor/eyes"
	"github.com/Loofort/smartdoor/eyes/cv"

	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

//"../../../smartdoor.bk/data/VIDEO/me/me.avi"
var infile = flag.String("in", "../../../smartdoor.bk/data/ideal/boss/vide.avi", "file path to input video")

func main() {
	flag.Parse()
	//closer := openTracer()
	//defer closer.Close()

	persons, err := cv.InitPersons(
		"../../../smartdoor.bk/data/ideal/boss/train/",
		"eyes/cv/cnn-models/shape_predictor_5_face_landmarks.dat",
	)
	if err != nil {
		log.Fatal(err)
	}
	_ = persons

	idleDispather := make(chan struct{})
	idleTracker := make(chan struct{})
	idlec := eyes.Idler(idleDispather, idleTracker)
	detectSig := eyes.Timer(500*time.Millisecond, idlec)

	cadrec := make(chan cv.Cadre, 24) // the buffer should be calculated based on fps, acceptable delay is about 1 sec.
	cadreFuncc := make(chan eyes.CadreFunc)
	go eyes.Detector(cadrec, detectSig, cadreFuncc)

	err = ProduceCadre(cadrec, *infile)
	if err != nil {
		log.Fatal(err)
	}

	framec := make(chan eyes.Frame)
	trackersMapc := make(chan map[int]cv.Tracker)
	rectsMapc := make(chan map[int]cv.Rect)
	cadreidc := make(chan int)
	rframesMapc := make(chan map[int]eyes.RFrame)

	rframec := eyes.RunTracker(framec, trackersMapc, idleTracker)
	go eyes.Synchronizer(cadreFuncc, framec, trackersMapc, rectsMapc, cadreidc, rframesMapc, rframec)

	personc := eyes.RunDispather(rectsMapc, cadreidc, rframec, rframesMapc, idleDispather)

	// read persons
	for person := range personc {
		span := person.Cadre.SpawnSpan("person")
		fmt.Printf("get person %v \n", person)
		span.Finish()
	}
}

func ProduceCadre(cadrec chan cv.Cadre, file string) error {
	src, err := cv.NewCapture(file)
	if err != nil {
		return err
	}

	go func() {
		cnt := 0
		for {
			cadre, err := cv.WaitForCadre(src)
			if err != nil {
				runtime.GC()

				log.Printf("err: %v; shoutdown in 1 sec\n", err)
				time.Sleep(20 * time.Minute)
				for _, closer := range cv.Closers {
					closer.Close()
				}

				log.Fatal(err)
			}
			cnt++
			if cnt > 20 {
				//time.Sleep(40 * time.Minute)
				time.Sleep(10 * time.Millisecond)
			} else {
				time.Sleep(10 * time.Millisecond)
			}

			select {
			case cadrec <- cadre:
			default:
			}

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
		"CV",
		//jaegercfg.Logger(jLogger),
		//jaegercfg.Metrics(jMetricsFactory),
		//jaegercfg.Observer(rpcmetrics.NewObserver(jMetricsFactory, rpcmetrics.DefaultNameNormalizer)),
	)
	if err != nil {
		log.Fatalf("Could not initialize jaeger tracer: %s", err.Error())
		return nil
	}

	tracer, _, err := cfg.New("eye")
	if err != nil {
		log.Fatalf("Could not initialize jaeger tracer: %s", err.Error())
		return nil
	}
	trc = tracer

	return closer
}

var trc opentracing.Tracer

func secondTracer() opentracing.Tracer {
	return trc
}
