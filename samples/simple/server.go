package teleport

import (
// "testing"
)

// Demo controller
type Demo struct {
	SrvContext
}

func (d *Demo) Home(args string) (reply string, err error) {
	return "home response:" + args, nil
}

// Route a controller
// func Route(rcvr interface{}, defaultArgs ...interface{})
// Route(new(Demo),"default_token")
