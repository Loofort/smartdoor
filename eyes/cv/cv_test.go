package cv

import (
	"runtime"
	"testing"
	"time"
)

func TestCapture(t *testing.T) {
	getCadre(t)
	runtime.GC()

	time.Sleep(1 * time.Millisecond)
}
func getCadre(t *testing.T) {
	src, err := NewCapture("/home/illia/work/go/src/smartdoor.bk/data/VIDEO/me/01.asf")
	if err != nil {
		t.Fatal()
	}

	_, err = WaitForCadre(src)
	if err != nil {
		t.Fatal()
	}

}
