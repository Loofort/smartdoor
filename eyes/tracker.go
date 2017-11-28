package eyes

import (
	"log"
	"runtime"
	"sync/atomic"

	"github.com/Loofort/smartdoor/eyes/cv"
)

type TrackSync struct {
	Rank int
	SID  int
}

type TrackTask struct {
	Frame
	Tracker cv.Tracker
}

type RFrame struct {
	Frame
	Rect cv.Rect
}

func RunTracker(framec chan Frame, trackersMapc chan map[int]cv.Tracker, idle chan struct{}) chan RFrame {
	taskc := make(chan TrackTask)
	trackSyncc := make(chan TrackSync)
	go trackTasks(framec, taskc, trackersMapc, trackSyncc, idle)

	rframec := make(chan RFrame)
	num := runtime.NumCPU()
	for i := 0; i < num; i++ {
		go trackWorker(taskc, rframec, trackSyncc)
	}

	return rframec
}

func trackTasks(framec chan Frame, taskc chan TrackTask, trackersMapc chan map[int]cv.Tracker, trackSyncc chan TrackSync, idle chan struct{}) {
	var taskc1 chan TrackTask
	var task TrackTask
	var trackers map[int]cv.Tracker
	var frames Frames
	var idle1 chan struct{}

	for {
		if taskc1 == nil {
			frame, ok := frames.Pop()
			if ok {
				taskc1 = taskc
				tracker := trackers[frame.SID]
				task = TrackTask{frame, tracker}
			} else {
				idle1 = idle
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
		case idle1 <- struct{}{}:
		}
	}
}

func trackWorker(taskc chan TrackTask, rframec chan RFrame, trackSyncc chan TrackSync) {
	for task := range taskc {
		rect, err := cv.UpdateTracker(task.Tracker, task.Cadre)
		if err != nil {
			// todo: log err
			log.Printf("can't update tracker: %v", err)
			panic(err)
		}

		trackSyncc <- TrackSync{
			Rank: task.Rank,
			SID:  task.SID,
		}

		// even if error we have to send RFrame
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

func (t Frames) Unlock(rank int) {
	atomic.StoreInt32(t.locks[rank], 0)
}
