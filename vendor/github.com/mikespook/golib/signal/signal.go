package signal

import (
	"os"
	S "os/signal"
	"syscall"

	"github.com/mikespook/golib/idgen"
)

const (
	Break = iota + 1
	BreakExit
	Continue
	ContinueExit
)

// Returns Break, BreakExit, Continue and ContinueExit
// for different purposes
type Callback func() uint

type handler struct {
	Signal   os.Signal
	Id       interface{}
	callback Callback
	sh       *handlers
}

func (h *handler) Unbind() bool {
	return h.sh.Unbind(h.Signal, h.Id)
}

type handlers struct {
	schan chan os.Signal
	cb    map[os.Signal][]*handler
	id    idgen.IdGen
}

func New(id idgen.IdGen) (sh *handlers) {
	if id == nil {
		id = idgen.NewObjectId()
	}
	sh = &handlers{
		schan: make(chan os.Signal),
		cb:    make(map[os.Signal][]*handler),
		id:    id,
	}
	return
}

func (sh *handlers) Bind(s os.Signal, cb Callback) (h *handler) {
	if s == syscall.SIGKILL {
		panic("Can not handle `SIGKILL`.")
	}
	S.Notify(sh.schan, s)
	h = &handler{s, sh.id.Id(), cb, sh}
	sh.cb[s] = append(sh.cb[s], h)
	return
}

func (sh *handlers) Unbind(s os.Signal, id interface{}) bool {
	for k, v := range sh.cb[s] {
		if v.Id == id {
			sh.cb[s] = append(sh.cb[s][:k], sh.cb[s][k+1:]...)
			return true
		}
	}
	return false
}

func (sh *handlers) Wait() os.Signal {
	for s := range sh.schan {
		if cbs, ok := sh.cb[s]; ok && cbs != nil {
			var exit bool
		Loop:
			for _, v := range cbs {
				switch v.callback() {
				case Break:
					break Loop
				case Continue:
					continue Loop
				case BreakExit:
					exit = true
					break Loop
				case ContinueExit:
					exit = true
					continue Loop
				}
			}
			if exit {
				return s
			}
		}
	}
	return nil
}

func (sh *handlers) Close() {
	S.Stop(sh.schan)
	close(sh.schan)
}

var (
	Default = New(nil)
)

func Bind(s os.Signal, cb Callback) *handler {
	return Default.Bind(s, cb)
}

func Unbind(s os.Signal, id interface{}) bool {
	return Default.Unbind(s, id)
}

func Wait() os.Signal {
	return Default.Wait()
}

func Send(pid int, signal os.Signal) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := proc.Signal(signal); err != nil {
		return err
	}
	return nil
}

func Close() {
	Default.Close()
}
