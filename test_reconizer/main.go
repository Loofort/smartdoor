package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Loofort/smartdoor/eyes/cv"
)

func main() {
	log.Fatal(run())
}

func run() error {
	//file := "../../../../smartdoor.bk/data/ideal/boss/vide.avi"
	//file := "../../../../smartdoor.bk/data/VIDEO/me/02.asf"
	file := "/home/illia/work/projects/mjpeg-server/video/1.asf"

	cv.InitDetectors(
		"../eyes/cv/cnn-models/mmod_human_face_detector.dat",
		"../eyes/cv/cnn-models/haarcascade_frontalface_default.xml",
	)
	persons, err := cv.InitPersons(
		//"../../../../smartdoor.bk/data/ideal/boss/train/",
		"../../../../smartdoor.bk/data/metrain2/",
		//"/home/illia/work/tmp/png/",
		//"../eyes/cv/cnn-models/shape_predictor_5_face_landmarks.dat",
		"../eyes/cv/cnn-models/shape_predictor_68_face_landmarks.dat",
		"../eyes/cv/cnn-models/dlib_face_recognition_resnet_model_v1.dat",
	)
	if err != nil {
		return err
	}
	_ = persons

	os.RemoveAll("./dump")
	os.MkdirAll("./dump", os.ModePerm)

	src, err := cv.NewCapture(file)
	if err != nil {
		return err
	}

	for {
		start := time.Now()
		cadre, err := cv.WaitForCadre(src)
		if err != nil {
			return err
		}
		if cadre.ID < 240 || cadre.ID > 340 {
			//	continue
		}

		rects, err := cv.Detect(cadre)
		if err != nil {
			return err
		}

		middle := time.Now()
		for i, rect := range rects {
			person, err := cv.RecognizeBest([]cv.Cadre{cadre}, []cv.Rect{rect})
			if err != nil {
				return err
			}

			if person.Name != "" {
				path := fmt.Sprintf("dump/%d_%d.png", cadre.ID, i)
				cv.Save(cadre, rect, path)
				fmt.Printf("#%v: idented %v\n", cadre.ID, person.Name)
			}
		}
		fmt.Printf("#%v: detect %v, ident %v faces %d\n", cadre.ID, middle.Sub(start), time.Since(middle), len(rects))
	}
}
