package main

import (
	"context"
)

func main() {

	// buffered chan
	frameFuncc
}

func source(framec chan Frame) {
	for {
		frame := api.Source(src)

		// framec is unbeffered channel
		// if it bussy skip the frame
		select {
		case framec <- frame:
		default:
		}
	}
}

func sessionProc() {

	go Queue(ctx, cadreQuCh, trackResCh, recogQuCh)

	go Queue(ctx, recogQuCh, recogResCh, doorQuCh)

}

func Queue(ctx context.Context, in chan interface{}, out chan interface{}, cb chan interface{}) {
	var task interface{}
	queue := make([]interface{}, 0, 10)

	out1 := out
	out1 = nil
	for {
		if out1 == nil && len(queue) > 0 {
			task, queue = queue[0], queue[1:]
			out1 = out
		}

		select {
		case <-ctx.Done():
			return
		case item := <-in:
			queue := append(queue, item)
		case out1 <- task:
			out1 = nil
		}
	}
}

func trackRes() {

}

func recogRes() {

}
