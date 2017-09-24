// Copyright 2013 Xing Xing <mikespook@gmail.com>.
// All rights reserved.
// Use of this source code is governed by a commercial
// license that can be found in the LICENSE file.

package autoinc

type AutoInc struct {
	start, step int
	queue       chan int
	quit        chan bool
}

func New(start, step int) (ai *AutoInc) {
	ai = &AutoInc{
		start: start,
		step:  step,
		queue: make(chan int, 4),
		quit:  make(chan bool),
	}
	go ai.process()
	return
}

func (ai *AutoInc) process() {
	defer func() {
		recover()
		close(ai.queue)
	}()
	for i := ai.start; ; i = i + ai.step {
		select {
		case ai.queue <- i:
		case <-ai.quit:
			break
		}
	}
}

func (ai *AutoInc) Id() int {
	return <-ai.queue
}

func (ai *AutoInc) Close() {
	close(ai.quit)
}
