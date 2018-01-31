package cv

// #cgo CXXFLAGS: -std=c++11
// #cgo LDFLAGS: -I/usr/local/include/opencv -I/usr/local/include -L/usr/local/lib -lopencv_dnn -lopencv_ml -lopencv_objdetect -lopencv_shape -lopencv_stitching -lopencv_superres -lopencv_videostab -lopencv_calib3d -lopencv_features2d -lopencv_highgui -lopencv_videoio -lopencv_imgcodecs -lopencv_video -lopencv_photo -lopencv_imgproc -lopencv_flann -lopencv_viz -lopencv_core
// #cgo LDFLAGS: -I/usr/local/include -I/usr/include/libpng12 -L/usr/local/lib -ldlib -lpng12 -L/usr/lib/x86_64-linux-gnu/ -lgif
// #include "cv.h"
// #include <stdlib.h>
import "C"
import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"
	"unsafe"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	jaeger "github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

type cpointer struct {
	p       unsafe.Pointer
	destroy func(unsafe.Pointer)
}

func (cp *cpointer) free() {
	cp.destroy(cp.p)
	cp.p = nil
}

func newCPointer(p unsafe.Pointer, destroy func(unsafe.Pointer)) *cpointer {
	cp := &cpointer{p, destroy}
	runtime.SetFinalizer(cp, (*cpointer).free)
	return cp
}

func toGo(res C.struct_Result) (unsafe.Pointer, error) {
	if res.Err != nil {
		str := C.GoString(res.Err)
		C.free(unsafe.Pointer(res.Err))

		err := errors.New(str)
		return nil, err
	}
	return res.Res, nil
}

func toGoArr(res C.struct_ResultArr) (unsafe.Pointer, int, error) {
	if res.Err != nil {
		str := C.GoString(res.Err)
		C.free(unsafe.Pointer(res.Err))

		err := errors.New(str)
		return nil, 0, err
	}

	if res.Cnt == 0 {
		return nil, 0, nil
	}

	return res.Res, int(res.Cnt), nil
}

var tracers = &sync.Map{}
var Closers []io.Closer
var cfg = jaegercfg.Configuration{
	Sampler: &jaegercfg.SamplerConfig{
		Type:  jaeger.SamplerTypeConst,
		Param: 1,
	},
	Reporter: &jaegercfg.ReporterConfig{
		LocalAgentHostPort: "127.0.0.1:6831",
	},
}

func getTracer(name string) opentracing.Tracer {
	value, ok := tracers.Load(name)
	if ok {
		return value.(opentracing.Tracer)
	}

	tracer, closer, err := cfg.New(name)
	if err != nil {
		// todo: check err
	}
	Closers = append(Closers, closer)

	tracers.Store(name, tracer)

	return tracer
}

type Span struct {
	opentracing.Span
}

func (s *Span) SpawnSpan(name string) opentracing.Span {
	tracer := getTracer(name)

	var opts []opentracing.StartSpanOption
	if s.Span != nil {
		//opt := opentracing.FollowsFrom(s.Span.Context())
		opts = append(opts, opentracing.ChildOf(s.Span.Context()))
	}

	span := tracer.StartSpan(name, opts...)
	ext.SamplingPriority.Set(span, 1)
	s.Span = span
	return span
}

func (s *Span) CreateSpan(name string) *Span {
	tracer := getTracer(name)
	opt := opentracing.FollowsFrom(s.Span.Context())
	span := tracer.StartSpan(name, opt)
	ext.SamplingPriority.Set(span, 1)
	return &Span{span}
}

/******************* CADRE ***************************/

type Source struct {
	capture *cpointer
}
type Cadre struct {
	*Span
	cadre *cpointer
	ID    int
}

var id int

func NewCapture(path string) (Source, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	ptr, err := toGo(C.NewCapture(cpath))
	if err != nil {
		err := fmt.Errorf("can't open video %s: %s", path, err)
		return Source{}, err
	}

	src := Source{newCPointer(ptr, destroyCapture)}
	return src, nil
}
func destroyCapture(p unsafe.Pointer) {
	C.DestroyCapture(p)
}

func WaitForCadre(src Source) (Cadre, error) {
	ptr, err := toGo(C.NewCadre(src.capture.p))
	if err != nil {
		return Cadre{}, err
	}

	id++
	s := &Span{}
	span := s.SpawnSpan("cadre")
	destroy := func(p unsafe.Pointer) {
		span.Finish()
		destroyCadre(p)
	}
	cadre := Cadre{s, newCPointer(ptr, destroy), id}
	return cadre, nil
}
func destroyCadre(p unsafe.Pointer) {
	C.DestroyCadre(p)
}

/***************** Detect Rect ************************/
type Rect struct {
	//	*Span
	rect   *cpointer
	Left   int
	Top    int
	Right  int
	Bottom int
}

func (r Rect) Size() int {
	return (r.Right - r.Left) * (r.Bottom - r.Top)
}
func (r Rect) Nil() bool {
	return r.Right == 0 && r.Left == 0 && r.Bottom == 0 && r.Top == 0
}

func (r Rect) Overlap(target Rect) bool {
	left := max(r.Left, target.Left)
	top := max(r.Top, target.Top)
	right := min(r.Right, target.Right)
	bottom := min(r.Bottom, target.Bottom)
	rect := Rect{
		Left:   left,
		Top:    top,
		Right:  right,
		Bottom: bottom,
	}

	rs := rect.Size()
	return rs > r.Size()/2 && rs > target.Size()/2
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func Detect(cadre Cadre) ([]Rect, error) {

	ptr, cnt, err := toGoArr(C.HAARDetect(cadre.cadre.p))
	//ptr, cnt, err := toGoArr(C.Detect(cadre.cadre.p))
	if err != nil {
		err = fmt.Errorf("can't detect faces: %v", err)
		return nil, err
	}
	if cnt == 0 {
		return nil, nil
	}
	defer C.free(ptr)

	slice := (*[1 << 30]C.struct_Rectangle)(ptr)[:cnt:cnt]

	rects := make([]Rect, len(slice))
	for i, crect := range slice {
		//crect := (*C.struct_Rectangle)(v)
		//span := cadre.CreateSpan("rect")
		destroy := func(p unsafe.Pointer) {
			//span.Finish()
			destroyRect(p)
		}
		rects[i] = Rect{
			//Span:   span,
			rect:   newCPointer(unsafe.Pointer(crect.Rect), destroy),
			Left:   int(crect.Left),
			Top:    int(crect.Top),
			Right:  int(crect.Right),
			Bottom: int(crect.Bottom),
		}
	}

	return rects, nil
}

func destroyRect(p unsafe.Pointer) {
	C.DestroyRect(p)
}

func InitDetectors(modelPath, haarPath string) {
	cmodelPath := C.CString(modelPath)
	defer C.free(unsafe.Pointer(cmodelPath))

	chaarPath := C.CString(haarPath)
	defer C.free(unsafe.Pointer(chaarPath))

	C.InitDetectors(cmodelPath, chaarPath)
}

/***************** Tracker ************************/

type Tracker struct {
	tracker *cpointer
}

// correlation_tracker need img and rect
func CreateTracker(cadre Cadre, rect Rect) (Tracker, error) {
	span := cadre.CreateSpan("create_tracker")
	defer span.Finish()

	ptr, err := toGo(C.NewTracker(cadre.cadre.p, rect.rect.p))
	if err != nil {
		err = fmt.Errorf("can't create tracker: %v", err)
		return Tracker{}, err
	}

	destroy := func(p unsafe.Pointer) {
		destroyTracker(p)
	}
	trk := Tracker{newCPointer(ptr, destroy)}
	return trk, nil
}

func destroyTracker(p unsafe.Pointer) {
	C.DestroyTracker(p)
}

func UpdateTracker(tracker Tracker, cadre Cadre) (Rect, error) {
	span := cadre.CreateSpan("update_tracker")
	defer span.Finish()

	ptr, err := toGo(C.UpdateTracker(tracker.tracker.p, cadre.cadre.p))
	if err != nil {
		err = fmt.Errorf("can't update tracker: %v", err)
		return Rect{}, err
	}
	defer C.free(ptr)

	destroy := func(p unsafe.Pointer) {
		destroyRect(p)
	}
	crect := (*C.struct_Rectangle)(ptr)
	rect := Rect{
		//Span:   span,
		rect:   newCPointer(crect.Rect, destroy),
		Left:   int(crect.Left),
		Top:    int(crect.Top),
		Right:  int(crect.Right),
		Bottom: int(crect.Bottom),
	}

	return rect, nil
}

/**************** Person ******************************/

type Person struct {
	Name string
}

func RecognizeBest(cadres []Cadre, rects []Rect) (Person, error) {

	ln := len(cadres)
	ccadres := make([]unsafe.Pointer, ln)
	for i, cadre := range cadres {
		ccadres[i] = cadre.cadre.p
	}

	crects := make([]unsafe.Pointer, ln)
	for i, rect := range rects {
		crects[i] = rect.rect.p
	}

	//fmt.Printf("DUMP %v\n", spew.Sdump(ccadres, crects))
	//fmt.Printf("DEB %#v %#v\n", &ccadres[0], &crects[0])
	ptr, err := toGo(C.Recognize(&ccadres[0], &crects[0], C.int(ln)))
	// shouldn't free ptr;
	str := C.GoString((*C.char)(ptr))

	return Person{str}, err
}

func InitPersons(folder, modelPath, netPath string) ([]string, error) {
	cfolder := C.CString(folder)
	defer C.free(unsafe.Pointer(cfolder))

	cmodelPath := C.CString(modelPath)
	defer C.free(unsafe.Pointer(cmodelPath))

	cnetPath := C.CString(netPath)
	defer C.free(unsafe.Pointer(cnetPath))

	ptr, cnt, err := toGoArr(C.InitPersons(cfolder, cmodelPath, cnetPath))
	if err != nil {
		err = fmt.Errorf("can't init persons: %v", err)
		return nil, err
	}
	if cnt == 0 {
		return nil, nil
	}
	defer C.free(ptr)

	slice := (*[1 << 30](*C.char))(ptr)[:cnt:cnt] //  C type - *char

	files := make([]string, len(slice)) // new Go slice with proper Go types - string
	for i, cpath := range slice {
		files[i] = C.GoString(cpath)
	}

	return files, nil
}

/*************************** helpers for debug ***********************/

func Save(cadre Cadre, rect Rect, path string) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	C.SaveRFrame(cadre.cadre.p, rect.rect.p, cpath)
}

func Show(cadre Cadre, rect Rect) {
	C.ShowRFrame(cadre.cadre.p, rect.rect.p)
}
