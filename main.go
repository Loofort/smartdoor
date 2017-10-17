package main

import (
	"time"
)

func main() {
	//init src

	framec := initSource()

}

// process by the tracker batch frame
// process by the tracker each actual frame
// process by rest tracker frame queue

// no more trackers - detect
// time trigger - detect (ACTUAL FRAME) - erase unread queues and trackers

// frame queue size = ticker * fps
// ticker >> detection time

func run() {
	period := 500 * time.Millisecond
	timer := time.NewTimer(period)
	
	size := detectPeriod * fps * 2
	framec := make(chan Frame, size)
	var trash *[]Frame
	for {

		select {
		case frame, ok := <-framec:
			if !ok {
				return
			}

			// worst case - we have not enough recourses 
			select {
			case <-timer.C:
				// drain framec to get recent frame
				// if it's not empty - remove all trackers
				if frame, ok = drain(framec); ok {
					remove(trackers)
				}

				sessions = syncup(sessions, frame, trash)
				
			default:
				putin(sessions, frame, trash)
			}

		default:
			
			// process all queues for all trackers
			sess, ok := iterate(sessions)
			if ok {
				track(sess)
			} else {
				// else run detect on last frame
				frame, ok := <-framec:
				if !ok {
					return
				}
							
				sessions = syncup(sessions, frame, trash)
			}
		}
	}
}

func syncup (frame, sessions, trash *[]Frame) sessions {
	if !timer.Stop() {
		<-timer.C
	}
	timer.Reset(period)

	impl.Release(trash)

	areas := impl.Detect(frame)
	sessions = syncTrackers(areas, sessions, frame)
	putin(sessions, frame, trash)

}

func syncTrackers(areas, sessions, frame) []session {
	sort.Sort(areas)
	
	result := make([]Session, 0, len(areas))
	for _, area := range areas {
		var match bool
		for i, sess := range session {
			if overlap(area, sess) {
				result = append(result, sess)
				session = append(session[:i], session[i+1:]...)
				match = true
				break
			}
		}

		if !match {
			sess = newSession(area)
			result = append(result, sess)
		}
	}
	return result
}

func putin(sessions, frame, trash *[]Frame) {
	*trash = append(*trash, frame)
	for _, sess := sessions {
		sess.Frames = append(sess.Frames, frame)
	}
}


func iterate(sessions []Session) (Session, bool) {
	for _, sess := range sessions {
		if len(sess.Frames) > 0 {
			return sess, true
		}
	}
	return nil, false
}


func release(frames []Frame) {

}

func drain (framec chan Frame) bool {
	var ok bool
	for {
		select {
		case frame, ok = <-framec:
		default:
			return frame, ok
		}
	}
}

/*********************************** Business logic *************************/
type impl struct{}
func (impl) Release(frames []Frame) {
	// free C memory
	for _, frame := range frames {
		frame.Clear()
	}
}

func (impl) Source(file string) interface{} {
	
}

func (impl) Detect(frame) []interface{} {

}