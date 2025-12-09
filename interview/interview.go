package interview

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

/*
	Có 10 người chuyền bóng qua lại, trong 10s ai không có bóng thì la lên
*/

type State struct {
	mu            sync.Mutex
	currentHolder int
}

func Do(skip bool) {
	if skip {
		return
	}
	state := &State{}
	ballChan := make(chan struct{}, 1)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int, ballChan chan struct{}, state *State, wg *sync.WaitGroup) {
			defer wg.Done()

			for {
				<-ballChan

				state.mu.Lock()
				state.currentHolder = id
				state.mu.Unlock()

				time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)

				ballChan <- struct{}{}
			}

		}(i, ballChan, state, &wg)
	}

	fmt.Println("start")
	ballChan <- struct{}{}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C

		state.mu.Lock()
		holder := state.currentHolder
		state.mu.Unlock()

		for i := 0; i < 10; i++ {
			if i == holder {
				fmt.Printf("%v hold", i)
			}
		}
	}
}
