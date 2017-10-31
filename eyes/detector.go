package eyes

import (
	"smartdoor/eyes/cv"

	"sort"
	"sync"
	"time"
)
type TrackerRect {
	Tracker cv.Tracker
	Rect cv.Rect
}
type CadreFunc func() (cv.Cadre, chan []TrackerRect)

func detector(cadrec chan cv.Cadre) {
	period := 500 * time.Millisecond
	timer := time.NewTimer(period)

	for cadre := range cadrec {
		select {
		case <-timer.C:
			reset(timer, period)
			cadreFuncc <- detectedCadre(cadre)
		case <-idlec:
			reset(timer, period)
			cadreFuncc <- detectedCadre(cadre)
		default:
			c := cadre
			cadreFuncc <- func() (Cadre, chan []TrackerRect) { return c, nil }
		}
	}
}

func detectedCadre(cadre cv.Cadre) CadreFunc {
	trackersc := make(chan []TrackerRect)
	go func() {
		rects := cv.Detect(cadre)
		trackers := newTrackers(cadre, rects)
		trackersc <- TrackerRect
	}()

	cadreFunc = func() (cv.Cadre, chan []TrackerRect) { return cadre, trackersc }
	return cadreFunc
}

func newTrackers(cadre cv.Cadre, rects []cv.Rects) []TrackerRect {
	trackers := make([]TrackerRect, 0, len(rects))

	// create trackers in parallel
	var mtx sync.Mutex
	for _, rect := range rects {
		go func() {
			tracker := cv.CreateTracker(cadre, rect)
			mtx.Lock()
			trackers := append(trackers, TrackerRect{tracker,rect} )
			mtx.Unlock()
		}()
	}

	//sort trackers
	sort.Sort(BySquare(trackers))

	return trackers
}

type BySquare []TrackerRect

func (a BySquare) Len() int           { return len(a) }
func (a BySquare) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BySquare) Less(i, j int) bool { return a[i].Rect.Size() > a[j].Rect.Size() }
