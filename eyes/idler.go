package eyes

import (
	"time"
)

func Idler(chans ...chan struct{}) chan struct{} {
	idlec := make(chan struct{})
	go func() {
	Loop:
		for {
			for i, idle := range chans {
				select {
				case <-idle:
				default:
					chans = append(chans[i:], chans[:i]...)
					continue Loop
				}
			}

			select {
			case idlec <- struct{}{}:
			default:
			}
		}
	}()

	return idlec
}

func Timer(period time.Duration, idlec chan struct{}) chan struct{} {
	detectSig := make(chan struct{})
	timer := time.NewTimer(period)

	go func() {
		for {
			select {
			case <-timer.C:
				reset(timer, period)
				detectSig <- struct{}{}
			case <-idlec:
				reset(timer, period)
				detectSig <- struct{}{}
			}
		}
	}()

	return detectSig
}

func reset(timer *time.Timer, period time.Duration) {
	if !timer.Stop() {
		<-timer.C
	}
	timer.Reset(period)
}
