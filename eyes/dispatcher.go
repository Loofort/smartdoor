package eyes

type personSID struct {
	Person Person
	SID    int
}
type RFramePerson struct {
	RFrame
	Person Person
}

func runDispather() {

}

func dispatcher() {
	var taskc1 chan RFrame
	var task RFrame
	var isTask bool
	var personTask RFramePerson
	var frames RFrames
	var rframesMap map[int]RFrame // recent frames
	var cadreid int
	var wait map[int]struct{}
	var waitCnt int

	for {
		if taskc1 == nil && personc1 == nil {
			if task, isTask = rframes.Pop(rframesMap); isTask {
				if person, isPerson := personMap[task.SID]; isPerson {
					personc1 = personc
					personTask = RFramePerson{task, person}
				} else {
					taskc1 = taskc
				}
			}
		}

		select {

		case personSID := <-personSIDc:
			personMap[personSID.SID] = personSID.Person
		case rframesMap = <-rframesMapc:
			newPersonMap := make(map[int]Person, len(rframesMap))
			for sid := range rframesMap {
				if pers, ok := personMap[sid]; ok {
					newPersonMap[sid] = pers
				}
				personMap = newPersonMap
			}
		case cadreid = <-cadreidc:
			// we could synchronize here,
			// at this point we're garantied that sync frame hasn't reached dispather yet.
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
				counter++
			}
			if counter == len(wait) {
				wait = nil

				//push rects
				rectsMap := make(map[int]Rect, len(rframesMap))
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
		case rframePersonc1 <- personTask:
			personc1 = nil
		}
	}
}

func shapeWorker() {
	for rframe := range rframec {
		shape := cv.Shape(rframe.Cadre, rframe.Rect)
		shapec <- shape
	}
}

// I don't expect the length will be greater then 10,
// otherwise need to replase it with heap type
type RFrames [][]RFrame

func (rframes RFrames) Push(rframe RFrame) {
	// check the length of queue, if it less than required rank - extend it
	ln := rframe.Rank - len(rframes) + 1
	if ln > 0 {
		rframes = append(rframes, make([][]RFrame, ln)...)
	}

	rframes[rframe.Rank] = append(rframes[rframe.Rank], rframe)
}

func (rframes RFrames) Pop(cidMap map[int]RFrame) (RFrame, bool) {
	for i, frms := range rframes {
		if len(frms) == 0 {
			continue
		}

		// pop frame
		j, rframe := getRFrame(frms, cidMap)
		if j == -1 {
			rframes[i] = nil
			continue
		}

		rframes[i] = frms[j+1:]
		return rframe, true
	}
	return RFrame{}, false
}

func getRFrame(rframes []RFrame, cidMap map[int]RFrame) (int, RFrame) {
	for j, rframe = range rframes {
		if _, ok := cidMap[rframe.SID]; ok {
			return j, rframe
		}
	}

	return -1, RFrame{}
}
