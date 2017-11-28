package eyes

import (
	"runtime"

	"github.com/Loofort/smartdoor/eyes/cv"
)

type Face struct {
	RFrame
	Shape cv.Shape
}

func (fc Face) Better(face Face) bool {
	// todo
	return true
}

func RunRecognizer(shapec chan Face, personSIDc chan PersonSID, rframePersonc chan RFramePerson, idle chan struct{}) {
	shapesc := make(chan []Face)
	go recognizer(shapec, shapesc, idle)

	num := runtime.NumCPU()
	for i := 0; i < num; i++ {
		go recogznizeWorker(shapesc, personSIDc, rframePersonc)
	}
}

func recognizer(shapec chan Face, shapesc chan []Face, idle chan struct{}) {
	var task []Face
	var isTask bool
	var shapesc1 chan []Face
	var shapes Shapes
	var idle1 chan struct{}

	for {
		if shapesc1 == nil {
			if task, isTask = shapes.Pop(); isTask {
				shapesc1 = shapesc
			} else {
				idle1 = idle
			}
		}

		select {
		case shape := <-shapec:
			shapes.Push(shape)
		case shapesc1 <- task:
			shapesc1 = nil
		case idle1 <- struct{}{}:
		}
	}
}

func recogznizeWorker(shapesc chan []Face, personSIDc chan PersonSID, rframePersonc chan RFramePerson) {
	for faces := range shapesc {
		var face Face
		for _, fc := range faces {
			if fc.Better(face) {
				face = fc
			}
		}

		person, ok := cv.Recognize(face.Cadre, face.Rect)
		if ok {
			personSIDc <- PersonSID{person, face.SID}
			rframePersonc <- RFramePerson{face.RFrame, person}
		}
	}
}

type Shapes []chan Face

func (ts Shapes) Push(shape Face) {
	ln := shape.Rank - len(ts) + 1
	for i := 0; i < ln; i++ {
		shapeBuf := make(chan Face, 10)
		ts = append(ts, shapeBuf)
	}

	shapeBuf := ts[shape.Rank]
	if len(shapeBuf) == 10 {
		<-shapeBuf
	}
	shapeBuf <- shape
}

func (ts Shapes) Pop() ([]Face, bool) {
	for _, shapeBuf := range ts {
		ln := len(shapeBuf)
		if ln == 0 {
			continue
		}

		shapes := make([]Face, 0, ln)
		sid := 0
		for shape := range shapeBuf {
			if shape.SID != sid {
				sid = shape.SID
				shapes = shapes[:0]
			}
			shapes = append(shapes, shape)
		}
		return shapes, true
	}
	return nil, false
}
