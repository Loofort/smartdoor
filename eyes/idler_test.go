package eyes

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIdle(t *testing.T) {
	in := make(chan struct{})
	out := make(chan struct{})
	idlec := Idler(in)

	go func() { out <- <-idlec }()
	in <- struct{}{}

	select {
	case <-out:
		fmt.Printf("success\n")
	default:
		assert.Fail(t, "idle is not triggered")
	}
}

func TestIdleStat(t *testing.T) {
	testcases := []struct {
		pause time.Duration
		ins   int
	}{
		{time.Microsecond, 2},
		{time.Microsecond, 10},
		{time.Microsecond, 100},
		{time.Microsecond, 1000},
	}

	for _, tc := range testcases {
		ins := []chan struct{}{}
		for i := 0; i < tc.ins; i++ {
			ins = append(ins, make(chan struct{}))
		}

		idlec := Idler(ins...)

		for i := 0; i < tc.ins; i++ {
			d := tc.pause / time.Duration(rand.Intn(100)+1)
			time.Sleep(d)
			go runIdle(tc.pause, ins[i])
		}

		select {
		case <-idlec:
		case <-time.After(time.Millisecond):
			assert.Fail(t, "idle signal timeout")
		}
	}
}

func runIdle(d time.Duration, c chan struct{}) {
	for _, ok := <-time.After(d); ok; {
		c <- struct{}{}
	}
}

func TestTimer(t *testing.T) {
	in := make(chan struct{})
	idlec := Idler(in)
	detectSig := Timer(time.Millisecond, idlec)
	select {
	case <-detectSig:
	case <-time.After(5 * time.Second):
		assert.Fail(t, "idle is not triggered")
	}
}
