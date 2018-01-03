package eyes

import (
	"log"
	"sort"

	"github.com/Loofort/smartdoor/eyes/cv"
)

type TrackerRect struct {
	Tracker cv.Tracker
	Rect    cv.Rect
}
type CadreFunc func() (cv.Cadre, chan []TrackerRect)

func Detector(cadrec chan cv.Cadre, detectSig chan struct{}, cadreFuncc chan CadreFunc) {
	for cadre := range cadrec {
		span := cadre.SpawnSpan("sending_new_cadre")

		select {
		case <-detectSig:
			cadreFuncc <- detectedCadre(cadre)
		default:
			c := cadre
			cadreFuncc <- func() (cv.Cadre, chan []TrackerRect) { return c, nil }
		}
		span.Finish()
	}
}

func detectedCadre(cadre cv.Cadre) CadreFunc {
	trackersc := make(chan []TrackerRect)
	go func() {
		span := cadre.SpawnSpan("detect_face_rects")
		defer span.Finish()

		rects, err := cv.Detect(cadre)
		if err != nil {
			// todo: log error
			log.Printf("something worng with c detecor: %v", err)
			trackersc <- nil
			return
		}

		trackers, err := newTrackers(cadre, rects)
		if err != nil {
			// todo: log error
			log.Printf("something worng with c tracker: %v", err)
			trackersc <- nil
			return
		}

		trackersc <- trackers
	}()

	cadreFunc := func() (cv.Cadre, chan []TrackerRect) { return cadre, trackersc }
	return cadreFunc
}

func newTrackers(cadre cv.Cadre, rects []cv.Rect) ([]TrackerRect, error) {
	trackers := make([]TrackerRect, 0, len(rects))

	// todo: possible improvement:
	// create trackers in parallel (if needed)
	for _, rect := range rects {
		tracker, err := cv.CreateTracker(cadre, rect)
		if err != nil {
			return nil, err
		}
		trackers = append(trackers, TrackerRect{tracker, rect})
	}

	//sort trackers
	sort.Sort(BySquare(trackers))
	return trackers, nil
}

type BySquare []TrackerRect

func (a BySquare) Len() int           { return len(a) }
func (a BySquare) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BySquare) Less(i, j int) bool { return a[i].Rect.Size() > a[j].Rect.Size() }
