package cv

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

func TestCapture(t *testing.T) {
	getCadre(t)
	runtime.GC()

	time.Sleep(1 * time.Millisecond)
}
func getCadre(t *testing.T) {
	src, err := NewCapture("/home/illia/work/go/src/smartdoor.bk/data/VIDEO/me/01.asf")
	require.NoError(t, err)

	_, err = WaitForCadre(src)
	require.NoError(t, err)
}

func TestInitPersons(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(dir)

	files, err := InitPersons(
		"../../../../../smartdoor.bk/data/ideal/boss/train/",
		"cnn-models/shape_predictor_5_face_landmarks.dat",
	)
	require.NoError(t, err)

	t.Logf("Files %v", spew.Sdump(files))

}
