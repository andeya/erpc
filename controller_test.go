package teleport

import (
	"testing"
	"text/html"
)

// Demo controller
type Demo Controller

func (d *Demo) Home(token string) (string, error) {
	return "home response:" + token, nil
}

// Route a controller
// func Route(rcvr interface{}, defaultArgs ...interface{})
// Route(new(Demo),"default_token")
