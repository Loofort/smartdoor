package eyes

func recognizer() {
	var task []Shape
	var isTask bool

	for {
		if shapesc1 = nil {
			if task, isTask = shapes.Pop(); isTask {
				shapesc1 = shapesc
			}
		}

		select {
		case shape := <-shapec:
			shapes.Push(shape)
		case shapesc1 <- task:
			shapesc1 = nil
		}
		
	}
}


func recogznizeWorker() {
	for shapes := range shapesc {
		var shape Shape
		for shp := range shapes {
			shape = chooseBest(shp, shape)
		}

		person, ok := cv.Recognize(shape)
		if ok {
			personSIDc <- personSID{ person, SID}
			rframePersonc <- RFramePersonc{shape.RFrame, person}
		}
	}
}


type Shapes []chan shape
func (ts Shapes) Push(shape cv.Shape) {
	ln := shape.Rank - len(ts) + 1
	for i := 0; i < ln ; i++ {
		shapeBuf := make(chan cv.Shape, 10)
		ts = append(ts, shapeBuf)
	}

	shapeBuf = ts[shape.Rank]
	if len(shapeBuf) == 10 {
		<-shapeBuf
	}
	shapeBuf <- shape
}

func (ts Shapes) Pop() ([]cv.Shape, bool){
	for i, shapeBuf := range ts {
		ln := len(shapeBuf) 
		if ln == 0 {
			continue
		}

		shapes := make([]cv.Shape, 0 , ln)
		sid := 0
		for shape := range shapeBuf {
			if shape.sid != sid {
				sid = shape.sid 
				shapes = shapes[:0]
			}
			shapes = append(shapes, shape)
		}
		return shapes, true
	}
	return nil, false
}