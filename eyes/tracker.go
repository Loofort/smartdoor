package eyes

import (
	"runtime"
	"sync/atomic"
)

type TrackSync {
	Rank int
	SID int
}

type TrackTask struct {
	Frame
	Tracker cv.Tracker
}

type RFrame struct {
	Frame
	Rect  cv.Rect
}

func runTracker() (chan map[int]Tracker, chan Frame, chan RFrame) {
	framec := make(chan Frame)
	trackersMapc := make(chan map[int]Tracker)
	taskc := make(chan TrackTask)
	trackSyncc := make(chan TrackSync)
	go trackTasks(framec, taskc, trackersMapc, trackSyncc)

	rframec := make(chan RFrame)
	num := runtime.NumCPU()
	for i := 0; i < num; i++ {
		go trackWorker(taskoutc, unlock)
		trackWorker(taskc, rframec, trackSyncc )
	}

	return trackersMapc, framec, rframec
}

func trackTasks(framec chan Frame, taskc chan TrackTask, trackersMapc chan map[int]Tracker, trackSyncc chan TrackSync) {
	var taskc1 chan TrackTask
	var trackers map[int]Tracker
	var frames Frames

	for {
		if taskc1 == nil {
			frame, ok := frames.Pop()
			if ok {
				taskc1 = taskc
				tracker = trackers[frame.SID]
				task = TrackTask{frame, tracker}
			}
		}

		select {
		case trackers = <-trackersMapc:
			frames = Frames{}
			taskc1 = nil
		case t := <-trackSyncc:
			if _, ok := trackers[t.SID]; !ok {
				break
			}
			frames.Unlock(t.Rank)
		case frame := <-framec:
			frames.Push(frame)
		case taskc1 <- task:
			taskc1 = nil
		}
	}
}

func trackWorker(taskc chan TrackTask, rframec chan RFrame, trackSyncc chan TrackSync ) {
	for task := range taskc {
		rect := cv.UpdateTracker(task.Tracker, task.Cadre)
		trackSyncc <- trackSync {
			Rank : task.Rank,
			SID : task.SID,
		}

		rframec <- RFrame{task.Frame, rect}
	}
}

// Tasks represents priority queue
// I don't expect the length will be greater then 10,
// otherwise need to replase it with heap type
type Frames struct {
	frames [][]Frame
	locks  []*int32
}

func (t Frames) Push(frame Frame) {
	// check the length of queue, if it less than required rank - extend it
	ln := frame.Rank - len(t.frames) + 1
	if ln > 0 {
		t.frames = append(t.frames, make([][]Frame, ln)...)
		t.locks = append(t.locks, make([]*int32, ln)...)
	}

	t.frames[frame.Rank] = append(t.frames[frame.Rank], frame)
}

func (t Frames) Pop() (Frame, bool) {
	for i, frms := range t.frames {
		if len(frms) == 0 {
			continue
		}

		// check lock for current session tracker
		if l := atomic.LoadInt32(t.locks[i]); l == 1 {
			continue
		}

		// pop frame
		frame, frms := frms[0], frms[1:]
		t.frames[i] = frms
		atomic.StoreInt32(t.locks[i], 1)
		return frame, true

	}
	return Frame{}, false
}

func (t Frames) Unlock(sid, rank int) {
	atomic.StoreInt32(t.locks[rank], 0)
}
