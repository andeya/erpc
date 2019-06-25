// +build windows
//
// Copyright 2016 HenryLee. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package graceful

import (
	"os"
	"os/signal"
	"time"
)

func graceSignal() {
	// subscribe to SIGINT signals
	signal.Notify(ch, os.Interrupt, os.Kill)
	<-ch // wait for SIGINT
	signal.Stop(ch)
	Shutdown()
}

// Reboot all the frame process gracefully.
// NOTE: Windows system are not supported!
func Reboot(timeout ...time.Duration) {
	defer os.Exit(0)
	log.Infof("windows system doesn't support reboot! call Shutdown() is recommended.")
	log.Flush()
}

// AddInherited adds the files and envs to be inherited by the new process.
// NOTE:
//  Only for reboot!
//  Windows system are not supported!
func AddInherited(procFiles []*os.File, envs []*Env) {}
