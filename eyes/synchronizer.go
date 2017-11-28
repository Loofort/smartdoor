package eyes

import (
	"github.com/Loofort/smartdoor/eyes/cv"
)

type Frame struct {
	cv.Cadre
	Rank int
	SID  int
}

type Session struct {
	ID      int
	Tracker cv.Tracker
	Try     int
}

var sid = 0

func newSession(tracker cv.Tracker) Session {
	sid++
	return Session{sid, tracker, 0}
}

func Synchronizer(CadreFuncc chan CadreFunc, framec chan Frame, trackersMapc chan map[int]cv.Tracker, rectsMapc chan map[int]cv.Rect, cadreidc chan int, rframesMapc chan map[int]RFrame) {
	var sessions []Session

	for cf := range CadreFuncc {
		cadre, trackers := callCadreFunc(cf, cadreidc)
		span := cadre.NewSpan("synchronizer")

		// transmit cadre to each session
		for i, session := range sessions {
			framec <- Frame{cadre, i, session.ID}
		}
		span.Finish()

		if len(trackers) == 0 {
			continue
		}

		span = cadre.NewSpan("synchronizer_do")

		rects := getSessionRects(sessions, rectsMapc)
		sessions = synchronize(sessions, rects, trackers)

		// create rectMap and push to to dispather
		rframesMap := make(map[int]RFrame, len(sessions))
		for _, session := range sessions {
			rframesMap[session.ID] = RFrame{}
		}
		rframesMapc <- rframesMap

		// create trackersMap and push to tracker
		trackersMap := make(map[int]cv.Tracker, len(sessions))
		for _, session := range sessions {
			trackersMap[session.ID] = session.Tracker
		}
		trackersMapc <- trackersMap

		// todo: release cadres` C resources
		// clear track and recognize task queues
		// wait for all worker complete
		// call C destructors

		span.Finish()
	}
}

func callCadreFunc(cadreFunc CadreFunc, cadreidc chan int) (cv.Cadre, []TrackerRect) {
	cadre, trackersc := cadreFunc()
	if trackersc == nil {
		return cadre, nil
	}

	trackers := <-trackersc
	if len(trackers) == 0 {
		return cadre, nil
	}

	cadreidc <- cadre.ID
	return cadre, trackers
}

// set new tracker for existing sessions
// create new session
// left without changes chance session
// remove old session
func synchronize(sessions []Session, rects []cv.Rect, trackers []TrackerRect) []Session {
	// for each new tracker:
	// - found existing session
	// - or create new one
	newSessions := make([]Session, 0, len(trackers))
	found := map[int]struct{}{}
	for _, tracker := range trackers {

		var session Session
		ndx, ok := searchRect(rects, tracker.Rect)
		if !ok {
			session = newSession(tracker.Tracker)
		} else {
			found[ndx] = struct{}{}
			session = sessions[ndx]

			// provision existing session
			session.Try = 0
			session.Tracker = tracker.Tracker
		}
		newSessions = append(newSessions, session)
	}

	// for each exiting session:
	for i, session := range sessions {
		// - do nothing if it was matched with new tracker
		if _, ok := found[i]; ok {
			continue
		}

		// - or give one more chance (up to 3 times) for session that wasn't match
		if !rects[i].Nil() && session.Try < 3 {
			session.Try++
			newSessions = append(newSessions, session)
			continue
		}
	}

	return newSessions
}

// obtain rects from all sessions,
// the rects array has sessions` order
func getSessionRects(sessions []Session, rectsMapc chan map[int]cv.Rect) []cv.Rect {
	rectsMap := <-rectsMapc
	rects := make([]cv.Rect, 0, len(sessions))
	for _, session := range sessions {
		rect := rectsMap[session.ID]
		rects = append(rects, rect)
	}

	return rects
}

func searchRect(rects []cv.Rect, target cv.Rect) (int, bool) {
	for i, rect := range rects {
		if !rect.Nil() && rect.Overlap(target) {
			return i, true
		}
	}
	return 0, false
}
