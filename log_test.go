package tp

import (
	"testing"
)

func TestLog(t *testing.T) {
	Printf("test: %s", "Printf()")
	Criticalf("test: %s", "Criticalf()")
	Errorf("test: %s", "Errorf()")
	Warnf("test: %s", "Warnf()")
	Noticef("test: %s", "Noticef()")
	Infof("test: %s", "Infof()")
	Debugf("test: %s", "Debugf()")
	Tracef("test: %s", "Tracef()")
}
