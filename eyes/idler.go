package eyes

import (
	"fmt"
	"time"
)

func Idler(chans ...chan struct{}) chan struct{} {
	idlec := make(chan struct{})
	if len(chans) == 0 {
		return idlec
	}

	go func() {
		<-chans[0]
	Loop:
		for {
			for i, idle := range chans {
				if i == 0 {
					continue
				}
				select {
				case <-idle:
				default:
					<-idle
					chans = append(chans[i:], chans[:i]...)
					continue Loop
				}
			}

			select {
			case idlec <- struct{}{}:
			}
		}
	}()

	return idlec
}

func Timer(period time.Duration, idlec chan struct{}) chan struct{} {
	detectSig := make(chan struct{})
	timer := time.NewTimer(period)

	go func() {
		idle := false
		for {
			detectSigOut := detectSig
			if !idle {
				detectSigOut = nil
			}
			select {
			case <-timer.C:
				fmt.Printf("time trigger\n")
				reset(timer, period)
				idle = true
			case <-idlec:
				fmt.Printf("idle trigger\n")
				// todo: think about timeout for idle state
				reset(timer, period)
				idle = true
			case detectSigOut <- struct{}{}:
				idle = false
			}
		}
	}()

	return detectSig
}

func reset(timer *time.Timer, period time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(period)
}
