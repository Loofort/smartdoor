package cv

// #cgo CXXFLAGS: -std=c++11
// #cgo LDFLAGS: -I/usr/local/include/opencv -I/usr/local/include -L/usr/local/lib -lopencv_dnn -lopencv_ml -lopencv_objdetect -lopencv_shape -lopencv_stitching -lopencv_superres -lopencv_videostab -lopencv_calib3d -lopencv_features2d -lopencv_highgui -lopencv_videoio -lopencv_imgcodecs -lopencv_video -lopencv_photo -lopencv_imgproc -lopencv_flann -lopencv_viz -lopencv_core
// #cgo LDFLAGS: -I/usr/local/include -I/usr/include/libpng12 -L/usr/local/lib -ldlib -lpng12
// #include "cv.h"
// #include <stdlib.h>
import "C"
import (
	"errors"
	"fmt"
	"log"
	"runtime"
	"unsafe"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
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

/******************* CADRE ***************************/

type Source struct {
	capture *cpointer
}
type Cadre struct {
	cadre *cpointer
	ID    int
	Span  opentracing.Span
}

func (c *Cadre) NewSpan(name string) opentracing.Span {
	//c.Span = opentracing.GlobalTracer().StartSpan(name, opentracing.FollowsFrom(c.Span.Context()))
	span := opentracing.GlobalTracer().StartSpan(name, opentracing.ChildOf(c.Span.Context()))
	return span
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
	span := opentracing.StartSpan("create_cadre")
	ext.SamplingPriority.Set(span, 1)
	destroy := func(p unsafe.Pointer) {
		span.Finish()
		destroyCadre(p)
	}
	cadre := Cadre{newCPointer(ptr, destroy), id, span}
	return cadre, nil
}
func destroyCadre(p unsafe.Pointer) {
	C.DestroyCadre(p)
}

/***************** Detect Rect ************************/
type Rect struct {
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

	ptr, cnt, err := toGoArr(C.Detect(cadre.cadre.p))
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
		log.Printf("struct_Rectangle %#v \n", crect)

		rects[i] = Rect{
			rect:   newCPointer(unsafe.Pointer(crect.Rect), destroyRect),
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

/***************** Tracker ************************/

type Tracker struct {
	tracker *cpointer
}

// correlation_tracker need img and rect
func CreateTracker(cadre Cadre, rect Rect) (Tracker, error) {
	ptr, err := toGo(C.NewTracker(cadre.cadre.p, rect.rect.p))
	if err != nil {
		err = fmt.Errorf("can't create tracker: %v", err)
		return Tracker{}, err
	}

	trk := Tracker{newCPointer(ptr, destroyTracker)}
	return trk, nil
}

func destroyTracker(p unsafe.Pointer) {
	C.DestroyTracker(p)
}

func UpdateTracker(tracker Tracker, cadre Cadre) (Rect, error) {
	ptr, err := toGo(C.UpdateTracker(tracker.tracker.p, cadre.cadre.p))
	if err != nil {
		err = fmt.Errorf("can't update tracker: %v", err)
		return Rect{}, err
	}
	defer C.free(ptr)

	crect := (*C.struct_Rectangle)(ptr)
	rect := Rect{
		rect:   newCPointer(crect.Rect, destroyRect),
		Left:   int(crect.Left),
		Top:    int(crect.Top),
		Right:  int(crect.Right),
		Bottom: int(crect.Bottom),
	}

	return rect, nil
}

/**************** Shape of Face******************************/

type Shape struct {
}

func GetShape(cadre Cadre, rect Rect) Shape {

	return Shape{}
}

func ChooseBest(shp Shape, shape Shape) Shape {
	return Shape{}
}

type Person struct{}

func Recognize(cadre Cadre, rect Rect) (Person, bool) {
	return Person{}, true
}

func RecognizeBest(cadres []Cadre, rects []Rect) (Person, int) {
	return Person{}, -1
}
