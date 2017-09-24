package hold

import (
	"os"

	"github.com/mikespook/golib/signal"
)

func Hold() {
	sig := signal.New(nil)
	sig.Bind(os.Interrupt, func() uint { return signal.BreakExit })
	sig.Wait()
}
