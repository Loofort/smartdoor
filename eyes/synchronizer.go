package eyes

import (
	"fmt"

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

func Synchronizer(CadreFuncc chan CadreFunc, framec chan Frame, trackersMapc chan map[int]cv.Tracker, rectsMapc chan map[int]cv.Rect, cadreidc chan int, rframesMapc chan map[int]RFrame, rframec chan RFrame) {
	var sessions []Session

	for cf := range CadreFuncc {
		//fmt.Printf("#")
		cadre, trackers := callCadreFunc(cf, cadreidc, len(sessions))
		//fmt.Printf("%d ", cadre.ID)

		// transmit cadre to each session
		for i, session := range sessions {
			cdr := cadre
			cdr.Span = cadre.CreateSpan("push_cadre_to_tracker")

			framec <- Frame{cdr, i, session.ID}
			cdr.Finish()
		}

		if trackers == nil {
			continue
		}

		span := cadre.SpawnSpan("synchronize_new_trackers")

		fmt.Printf("\n[%d] trackers %d, ", cadre.ID, len(trackers))
		rects := getSessionRects(sessions, rectsMapc)
		fmt.Printf("rects %d, ", len(rects))
		sessions = synchronize(sessions, rects, trackers)
		fmt.Printf("sessions %d\n", len(sessions))

		// create rectMap and push to to dispather
		rframesMap := make(map[int]RFrame, len(sessions))
		for _, session := range sessions {
			rframesMap[session.ID] = RFrame{}
		}
		rframesMapc <- rframesMap

		// create rframe from each new tracker and push to dispatcher
		// if new tracker is one of the old trucker , no big deal, we'll get two similar frames for recognition
		for i, session := range sessions {
			trackerRect, ok := findTrackerRect(session.Tracker, trackers)
			if !ok {
				continue
			}

			rframe := RFrame{
				Frame{
					Cadre: cadre,
					Rank:  i,
					SID:   session.ID,
				},
				trackerRect.Rect,
			}

			rframec <- rframe
		}

		// create trackersMap and push to tracker
		trackersMap := make(map[int]cv.Tracker, len(sessions))
		for _, session := range sessions {
			trackersMap[session.ID] = session.Tracker
		}
		trackersMapc <- trackersMap

		span.Finish()
	}
}

func findTrackerRect(tracker cv.Tracker, trackers []TrackerRect) (TrackerRect, bool) {
	for _, trackerRect := range trackers {
		if trackerRect.Tracker == tracker {
			return trackerRect, true
		}
	}
	return TrackerRect{}, false
}

func callCadreFunc(cadreFunc CadreFunc, cadreidc chan int, sesslen int) (cv.Cadre, []TrackerRect) {
	cadre, trackersc := cadreFunc()
	if trackersc == nil {
		return cadre, nil
	}

	span := cadre.SpawnSpan("wait_for_new_trackers")
	defer span.Finish()

	if sesslen != 0 {
		cadreidc <- cadre.ID
	}

	trackers := <-trackersc
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
	fmt.Printf("old(%d) new(%d) ", len(sessions), len(newSessions))
	for i, session := range sessions {
		// - do nothing if it was matched with new tracker
		if _, ok := found[i]; ok {
			continue
		}

		// - or give one more chance (up to 3 times) for session that wasn't match
		fmt.Printf("(%t && %d) ", !rects[i].Nil(), session.Try)
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
	rects := make([]cv.Rect, 0, len(sessions))
	if len(sessions) == 0 {
		return rects
	}

	rectsMap := <-rectsMapc
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
