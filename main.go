package main

import (
	"time"
)

func main() {
	//init src

	framec := initSource()

}

func run() {

	tickDetect := time.Tick(500 * time.Millisecond)
	for {

		// when there is no more frames to process the priority is for frames

		// the highest priority is for tick faceDetector
		select {
		case <-tickDetect:
			faceDetect(frame)
			track(tracker, frame)
		default:
		}

		// read other frames
		select {
		case frame = <-framec:
			track(tracker, frame)
		}

	}
}

// process by the tracker batch frame
// process by the tracker each actual frame
// process by rest tracker frame queue

// no more trackers - detect
// time trigger - detect (ACTUAL FRAME) - erase unread queues and trackers

// frame queue size = ticker * fps
// ticker >> detection time

size := detectPeriod * fps * 2
framec := make(chan Frame, size)
for {

	select {
	case frame, ok := <-framec:
		if !ok {
			theend()
		}

		// worst case - we have not enough recourses 
		select {
		case <-detectTicker:
			// drain framec to get recent frame
			// if it's not empty - remove all trackers
			if frame, ok = drain(framec); ok {
				remove(trackers)
			}

			faceDetect(frame)

			assignFrame(trackers, frame)
			
			tracker[0].Track()
		default:
			assignFrame(trackers, frame)
		}

	default:
		// process all queues for all trackers
		
		// else run detect on last frame

	}

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