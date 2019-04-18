package core

import (
	"os/exec"
	"syscall"

	shellquote "github.com/kballard/go-shellquote"
	"github.com/ppacher/envel/pkg/bindings/callback"
	lua "github.com/yuin/gopher-lua"
)

// AddExec adds the exec package to the lua table m
func AddExec(L *lua.LState, m *lua.LTable) {
	t := L.NewTable()

	L.SetMetatable(t, L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"__call": call,
	}))

	m.RawSetString("exec", t)
}

// call provides `lualib.exec.call()`
func call(L *lua.LState) int {
	cmdStr := L.CheckString(2)
	shell := L.CheckBool(3)
	lineCallback := callback.LGetOpt(4, L)
	doneCallback := callback.LGetOpt(5, L)

	var cmd []string

	if shell {
		cmd = append([]string{"bash", "-c"}, cmdStr)
	} else {
		var err error
		cmd, err = shellquote.Split(cmdStr)

		if err != nil {
			L.RaiseError("invalid command line")
			return 0
		}
	}

	c := exec.Command(cmd[0], cmd[1:]...)

	if lineCallback != nil {
		stdout, err := c.StdoutPipe()
		if err != nil {
			L.RaiseError("failed to create stdout pipe")
			return 0
		}

		stderr, err := c.StderrPipe()
		if err != nil {
			L.RaiseError("failed to create stderr pipe")
			return 0
		}

		_, outReader := NewReader(L, stdout)
		_, errReader := NewReader(L, stderr)
		outReader.WithLineCallback(lineCallback, nil)
		errReader.WithLineCallback(lineCallback, &LineCallbackOptions{
			// stderr should be passed as the second argument to lineCallback
			PrefixArgs: []lua.LValue{lua.LNil},
		})
	}

	if err := c.Start(); err != nil {
		L.RaiseError("failed to start process: %v", err)
		return 0
	}

	L.Push(lua.LNumber(c.Process.Pid))

	go func() {
		err := c.Wait()
		if doneCallback != nil {
			codeOrSignal := 0
			exitReason := "exit"

			if exitErr, ok := err.(*exec.ExitError); ok {
				// godoc: -1 f the process hasn't exited or was terminated by a signal
				// we know that it exited so it must have been a signal
				if exitErr.ExitCode() == -1 {
					codeOrSignal = int(exitErr.Sys().(syscall.WaitStatus).Signal())
					exitReason = "signal"
				} else {
					codeOrSignal = exitErr.ExitCode()
					exitReason = "exit"
				}
			} else {
				// normal exit but with exitCode == 0
				exitReason = "exit"
			}

			<-doneCallback.Do(lua.LString(exitReason), lua.LNumber(codeOrSignal))
		}
	}()

	return 1
}
