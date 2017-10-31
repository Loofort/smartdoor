package cv

type Cadre struct {
	ID int
}
type Rect struct{}

func (r Rect) Size() int {
	return 0
}

func Detect(cadre Cadre) Rect {
	return Rect{}
}

type Tracker struct {
	Rect Rect
}

// correlation_tracker need img and rect
func CreateTracker(cadre Cadre, rect Rect) *Tracker {
	return &Tracker{}
}

func UpdateTracker(tracker *Tracker, cadre Cadre) Rect {
	return Rect{}
}

type Person struct{}

func Recognize(cadre Cadre, rect Rect) Person {}
