// +build !windows
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
	"context"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func graceSignal() {
	// subscribe to SIGINT signals
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR2)
	sig := <-ch
	signal.Stop(ch)
	switch sig {
	case syscall.SIGINT, syscall.SIGTERM:
		Shutdown()
	case syscall.SIGUSR2:
		Reboot()
	}
}

// Reboot all the frame process gracefully.
// NOTE: Windows system are not supported!
func Reboot(timeout ...time.Duration) {
	defer os.Exit(0)
	log.Infof("rebooting process...")

	var (
		ppid     = os.Getppid()
		graceful = true
	)
	contextExec(timeout, "reboot", func(ctxTimeout context.Context) <-chan struct{} {
		endCh := make(chan struct{})
		go func() {
			defer close(endCh)

			var reboot = true
			if err := firstSweep(); err != nil {
				log.Errorf("[reboot-firstSweep] %s", err.Error())
				graceful = false
			}

			// Starts a new process passing it the active listeners. It
			// doesn't fork, but starts a new process using the same environment and
			// arguments as when it was originally started. This allows for a newly
			// deployed binary to be started.
			_, err := startProcess()
			if err != nil {
				log.Errorf("[reboot-startNewProcess] %s", err.Error())
				reboot = false
			}

			// shut down
			graceful = shutdown(ctxTimeout, "reboot") && graceful
			if !reboot {
				if graceful {
					log.Errorf("process reboot failed, but shut down gracefully!")
				} else {
					log.Errorf("process reboot failed, and did not shut down gracefully!")
				}
				log.Flush()
				os.Exit(-1)
			}
		}()

		return endCh
	})

	// Close the parent if we inherited and it wasn't init that started us.
	if ppid != 1 {
		if err := syscall.Kill(ppid, syscall.SIGTERM); err != nil {
			log.Errorf("[reboot-killOldProcess] %s", err.Error())
			graceful = false
		}
	}

	if graceful {
		log.Infof("process is rebooted gracefully.")
	} else {
		log.Infof("process is rebooted, but not gracefully.")
	}
	log.Flush()
}

var (
	allInheritedProcFiles          = []*os.File{os.Stdin, os.Stdout, os.Stderr}
	defaultInheritedProcFilesCount = len(allInheritedProcFiles)
)

// AddInherited adds the files and envs to be inherited by the new process.
// NOTE:
//  Only for reboot!
//  Windows system are not supported!
func AddInherited(procFiles []*os.File, envs []*Env) {
	locker.Lock()
	defer locker.Unlock()
	for _, f := range procFiles {
		var had bool
		for _, ff := range allInheritedProcFiles {
			if ff == f {
				had = true
				break
			}
		}
		if !had {
			allInheritedProcFiles = append(allInheritedProcFiles, f)
		}
	}
	for _, env := range envs {
		customEnvs[env.K] = env.V
	}
}

// In order to keep the working directory the same as when we started we record
// it at startup.
var originalWD, _ = os.Getwd()

// startProcess starts a new process passing it the active listeners. It
// doesn't fork, but starts a new process using the same environment and
// arguments as when it was originally started. This allows for a newly
// deployed binary to be started. It returns the pid of the newly started
// process when successful.
func startProcess() (int, error) {
	for i, f := range allInheritedProcFiles {
		if i >= defaultInheritedProcFilesCount {
			defer f.Close()
		}
	}

	// Use the original binary location. This works with symlinks such that if
	// the file it points to has been changed we will use the updated symlink.
	argv0, err := exec.LookPath(os.Args[0])
	if err != nil {
		return 0, err
	}

	// Pass on the environment and replace the old count key with the new one.
	var envs []string
	for _, env := range os.Environ() {
		k := strings.Split(env, "=")[0]
		if _, ok := customEnvs[k]; !ok {
			envs = append(envs, env)
		}
	}
	for k, v := range customEnvs {
		envs = append(envs, k+"="+v)
	}

	process, err := os.StartProcess(argv0, os.Args, &os.ProcAttr{
		Dir:   originalWD,
		Env:   envs,
		Files: allInheritedProcFiles,
	})
	if err != nil {
		return 0, err
	}
	return process.Pid, nil
}
