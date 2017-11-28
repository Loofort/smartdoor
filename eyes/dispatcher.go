package eyes

import (
	"runtime"

	"github.com/Loofort/smartdoor/eyes/cv"
)

type PersonSID struct {
	Person cv.Person
	SID    int
}
type RFramePerson struct {
	RFrame
	Person cv.Person
}

func RunDispather(rectsMapc chan map[int]cv.Rect, cadreidc chan int, rframec chan RFrame, rframesMapc chan map[int]RFrame, idle chan struct{}) chan RFramePerson {
	taskc := make(chan []RFrame)
	personSIDc := make(chan PersonSID)
	personc := make(chan RFramePerson)
	go dispatcher(rectsMapc, cadreidc, rframec, rframesMapc, personSIDc, personc, taskc, idle)

	num := runtime.NumCPU()
	for i := 0; i < num; i++ {
		go recognizeWorker(taskc, personSIDc, personc)
	}

	return personc
}

func dispatcher(rectsMapc chan map[int]cv.Rect, cadreidc chan int, rframec chan RFrame, rframesMapc chan map[int]RFrame, personSIDc chan PersonSID, personc chan RFramePerson, taskc chan []RFrame, idle chan struct{}) {
	var taskc1 chan []RFrame
	var task []RFrame
	var isTask bool
	var personTask RFramePerson
	var rframes RFrames
	var rframesMap map[int]RFrame // recent frames
	var cadreid int
	var wait map[int]struct{}
	var waitCnt int
	var idle1 chan struct{}
	var personc1 chan RFramePerson
	var personMap map[int]cv.Person

	for {
		if taskc1 == nil && personc1 == nil {
			if task, isTask = rframes.Pop(rframesMap); isTask {
				tframe := task[len(task)-1]
				if person, isPerson := personMap[tframe.SID]; isPerson {
					personc1 = personc
					personTask = RFramePerson{tframe, person}
				} else {
					taskc1 = taskc
				}
			} else {
				idle1 = idle
			}
		}

		select {

		case personSID := <-personSIDc:
			personMap[personSID.SID] = personSID.Person
		case rframesMap = <-rframesMapc:
			newPersonMap := make(map[int]cv.Person, len(rframesMap))
			for sid := range rframesMap {
				if pers, ok := personMap[sid]; ok {
					newPersonMap[sid] = pers
				}
				personMap = newPersonMap
			}
		case cadreid = <-cadreidc:
			// we could synchronize here,
			// at this point we're guarantied that sync frame hasn't reached dispather yet.
			//
			// any session that has recent cadre id less than cadreid - 1 should be market as outdated.

			if len(rframesMap) == 0 {
				rectsMapc <- nil
				break
			}

			waitCnt = 0
			wait = make(map[int]struct{}, len(rframesMap))
			for sid, rframe := range rframesMap {
				if rframe.ID == cadreid-1 {
					wait[sid] = struct{}{}
				}
			}

		case rframe := <-rframec:
			sid := rframe.SID
			if _, ok := rframesMap[sid]; !ok {
				break
			}
			rframesMap[sid] = rframe
			rframes.Push(rframe)

			// check if we in sync state
			if wait == nil {
				break
			}
			if _, ok := wait[sid]; ok {
				waitCnt++
			}
			if waitCnt == len(wait) {
				wait = nil

				//push rects
				rectsMap := make(map[int]cv.Rect, len(rframesMap))
				for sid, rframe := range rframesMap {
					rect := rframe.Rect
					if rframe.ID != cadreid {
						rect = cv.Rect{}
					}
					rectsMap[sid] = rect
				}
				rectsMapc <- rectsMap
			}

		case taskc1 <- task:
			taskc1 = nil
		case personc1 <- personTask:
			personc1 = nil
		case idle1 <- struct{}{}:
		}
	}
}

func recognizeWorker(rframesc chan []RFrame, personSIDc chan PersonSID, rframePersonc chan RFramePerson) {
	/*
		for rframe := range rframec {
			shape := cv.GetShape(rframe.Cadre, rframe.Rect)
			shapec <- Face{rframe, shape}
		}
	*/

	for rframes := range rframesc {
		if len(rframes) == 0 {
			continue
		}

		cadres := make([]cv.Cadre, len(rframes))
		rects := make([]cv.Rect, len(rframes))
		for i, rframe := range rframes {
			cadres[i] = rframe.Frame.Cadre
			rects[i] = rframe.Rect
		}

		person, i := cv.RecognizeBest(cadres, rects)
		if i != -1 {
			personSIDc <- PersonSID{person, rframes[i].SID}
			rframePersonc <- RFramePerson{rframes[i], person}
		}
	}

}

// I don't expect the length will be greater then 10,
// otherwise need to replase it with heap type
type RFrames [][]RFrame

// todo:
// it has no sense to keep very old frames -
// limit the depth to 10
func (rframes RFrames) Push(rframe RFrame) {
	// check the length of queue, if it less than required rank - extend it
	ln := rframe.Rank - len(rframes) + 1
	if ln > 0 {
		rframes = append(rframes, make([][]RFrame, ln)...)
	}

	rframes[rframe.Rank] = append(rframes[rframe.Rank], rframe)
}

func (rframes RFrames) Pop(cidMap map[int]RFrame) ([]RFrame, bool) {
	for i, frms := range rframes {
		if len(frms) == 0 {
			continue
		}

		// pop frame
		start, end := getRFrame(frms, cidMap)
		if start == end {
			rframes[i] = frms[:0]
			continue
		}

		rframes[i] = frms[end:]
		return frms[start:end], true
	}
	return nil, false
}

func getRFrame(rframes []RFrame, cidMap map[int]RFrame) (int, int) {
	var start, end int

	for start < len(rframes) {
		sid := rframes[start].SID
		for end = start + 1; end < len(rframes); end++ {
			if rframes[end].SID != sid {
				break
			}
		}

		if _, ok := cidMap[sid]; ok {
			return start, end
		}

		start = end
	}

	return 0, 0
}
