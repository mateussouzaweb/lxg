package command

import (
	"bytes"
	"os/exec"
)

// Result struct
type Result struct {
	PID      int    `json:"pid"`
	ExitCode int    `json:"exitCode"`
	Error    string `json:"error"`
	Output   string `json:"output"`
}

// Execute command and return result
func Execute(command *exec.Cmd) (*Result, error) {

	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	result := &Result{
		ExitCode: 0,
		Error:    "",
		Output:   "",
		PID:      0,
	}

	err := command.Start()
	if err != nil {
		result.ExitCode = -1
		return result, err
	} else {
		result.PID = command.Process.Pid
	}

	err = command.Wait()
	result.Error = stderr.String()
	result.Output = stdout.String()
	exitErr, okErr := err.(*exec.ExitError)

	if err != nil && okErr {
		result.ExitCode = exitErr.ExitCode()
	} else if err != nil {
		result.ExitCode = -1
		return result, err
	}

	if result.Error == "" && result.ExitCode >= 1 {
		result.Error = result.Output
	}

	return result, nil
}
