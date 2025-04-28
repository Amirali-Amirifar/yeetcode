package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type Submission struct {
	Code        string   `json:"code"`
	Inputs      []string `json:"inputs"`
	Outputs     []string `json:"outputs"`
	TimeLimit   int      `json:"time_limit"`   // s
	MemoryLimit int      `json:"memory_limit"` // mb
}

type TestCaseResult struct {
	Status    string `json:"status"`
	RuntimeMS int    `json:"runtime_ms,omitempty"`
	Output    string `json:"output,omitempty"`
	Error     string `json:"error,omitempty"`
}

type Result struct {
	Cases []TestCaseResult `json:"cases"`
}

const judgeScript = `#!/bin/bash
ulimit -v $((MEMORY_LIMIT*1024*1024))
go build -o prog main.go 2> compile_err.txt
if [ $? -ne 0 ]; then
  echo "JUDGE_STATUS:COMPILE_ERROR:JUDGE_TIME_MS:0" >> output.txt
  cat compile_err.txt >> output.txt
  cat output.txt
  exit
fi
START=$(date +%s%3N)
timeout --signal=KILL $((TIME_LIMIT_S))s ./prog < input.txt > program_output.txt 2>&1
STATUS=$?
END=$(date +%s%3N)
ELAPSED=$((END - START))
cat program_output.txt > output.txt
echo "JUDGE_STATUS:$STATUS:JUDGE_TIME_MS:$ELAPSED" >> output.txt
cat output.txt
`

func normalizeOutput(s string) string {
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		lines[i] = strings.TrimRightFunc(ln, unicode.IsSpace)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func runSingleTestCase(code, input string, timeLimit, memoryLimit int) (compileError bool, res TestCaseResult) {
	tmpDir, err := ioutil.TempDir("", "submission")
	if err != nil {
		res.Status = "InternalError"
		res.Error = "could not create temp dir"
		return false, res
	}
	defer os.RemoveAll(tmpDir)

	mainFile := filepath.Join(tmpDir, "main.go")
	if err := ioutil.WriteFile(mainFile, []byte(code), 0644); err != nil {
		res.Status = "InternalError"
		res.Error = "could not write source file"
		return false, res
	}
	inputFile := filepath.Join(tmpDir, "input.txt")
	if err := ioutil.WriteFile(inputFile, []byte(input), 0644); err != nil {
		res.Status = "InternalError"
		res.Error = "could not write input file"
		return false, res
	}
	judgePath := filepath.Join(tmpDir, "judge.sh")
	if err := ioutil.WriteFile(judgePath, []byte(judgeScript), 0755); err != nil {
		res.Status = "InternalError"
		res.Error = "could not write judge script"
		return false, res
	}

	// Fix permissions so "nobody" user inside the container can access files and directory
	_ = os.Chmod(tmpDir, 0755)
	_ = os.Chmod(mainFile, 0644)
	_ = os.Chmod(inputFile, 0644)
	_ = os.Chmod(judgePath, 0755)

	env := []string{
		fmt.Sprintf("TIME_LIMIT_S=%d", timeLimit),
		fmt.Sprintf("MEMORY_LIMIT=%d", memoryLimit),
		"HOME=/workspace",
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		res.Status = "InternalError"
		res.Error = "could not create docker client"
		return false, res
	}
	defer cli.Close()

	containerConfig := &container.Config{
		Image: "golang:1.20",
		Cmd: []string{"bash", "-c", `
cp /app/* /workspace/ && cd /workspace && chmod +x judge.sh && ./judge.sh
`},
		Env:          env,
		WorkingDir:   "/workspace",
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
		User:         "nobody:nogroup",
	}

	hostConfig := &container.HostConfig{
		Binds: []string{fmt.Sprintf("%s:/app:ro", tmpDir)},
		Tmpfs: map[string]string{
			"/workspace": "exec,mode=1777",
		},
		Resources: container.Resources{
			Memory:   int64(memoryLimit) * 1024 * 1024,
			NanoCPUs: 1000000000, // 1 CPU
		},
		NetworkMode: "none",
	}

	ctx := context.Background()
	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		res.Status = "InternalError"
		res.Error = "create container error: " + err.Error()
		return false, res
	}
	containerID := resp.ID
	defer func() {
		cli.ContainerRemove(context.Background(), containerID, types.ContainerRemoveOptions{Force: true, RemoveVolumes: true})
	}()

	if err := cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		res.Status = "InternalError"
		res.Error = "start container error: " + err.Error()
		return false, res
	}

	statusCh, errCh := cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case <-statusCh:
	case err := <-errCh:
		res.Status = "InternalError"
		res.Error = "wait error: " + err.Error()
		return false, res
	}

	logReader, err := cli.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		res.Status = "InternalError"
		res.Error = "read logs error: " + err.Error()
		return false, res
	}
	defer logReader.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	_, err = stdcopy.StdCopy(&stdoutBuf, &stderrBuf, logReader)
	if err != nil {
		res.Status = "InternalError"
		res.Error = "could not stdcopy logs: " + err.Error()
		return false, res
	}

	outStr := stdoutBuf.String() + stderrBuf.String()
	lines := strings.Split(outStr, "\n")

	var (
		userOutputLines []string
		runtimeMS       int
		statusMsg       = "OK"
		errorMsg        string
	)
	oomDetected := false
	for _, ln := range lines {
		if strings.Contains(ln, "fatal error: failed to reserve page summary memory") {
			statusMsg = "Memory Limit Exceeded"
			errorMsg = "Go program ran out of memory"
			oomDetected = true
		}
		if strings.HasPrefix(ln, "JUDGE_STATUS:COMPILE_ERROR") {
			statusMsg = "Compile Error"
			errorMsg = strings.Join(lines, "\n")
			return true, TestCaseResult{
				Status:    statusMsg,
				RuntimeMS: 0,
				Output:    "",
				Error:     errorMsg,
			}
		}
		if strings.HasPrefix(ln, "JUDGE_STATUS:") {
			fields := strings.Split(ln, ":")
			if len(fields) == 4 {
				statusCode := fields[1]
				if statusCode == "124" || statusCode == "137" {
					statusMsg = "Time Limit Exceeded"
					errorMsg = "execution timed out"
				} else if statusCode != "0" && !oomDetected {
					statusMsg = "Runtime Error"
					errorMsg = fmt.Sprintf("exit code: %s", statusCode)
				}
				fmt.Sscanf(fields[3], "%d", &runtimeMS)
			}
			break
		}
		if ln != "" && !strings.Contains(ln, "fatal error: failed to reserve page summary memory") {
			userOutputLines = append(userOutputLines, ln)
		}
	}
	finalOutput := strings.Join(userOutputLines, "\n")

	res = TestCaseResult{
		Status:    statusMsg,
		RuntimeMS: runtimeMS,
		Output:    strings.TrimSpace(finalOutput),
		Error:     errorMsg,
	}

	return false, res
}

func runCodeHandler(w http.ResponseWriter, r *http.Request) {
	var sub Submission
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	numCases := len(sub.Inputs)
	if numCases == 0 || len(sub.Outputs) != numCases {
		http.Error(w, "inputs and outputs must be non-empty and have the same length", http.StatusBadRequest)
		return
	}

	compileError, compileErrResult := runSingleTestCase(sub.Code, sub.Inputs[0], sub.TimeLimit, sub.MemoryLimit)
	if compileError {
		res := Result{Cases: make([]TestCaseResult, numCases)}
		for i := 0; i < numCases; i++ {
			res.Cases[i] = compileErrResult
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
		return
	}

	res := Result{Cases: make([]TestCaseResult, numCases)}
	for i := 0; i < numCases; i++ {
		_, tcResult := runSingleTestCase(sub.Code, sub.Inputs[i], sub.TimeLimit, sub.MemoryLimit)
		if tcResult.Status == "OK" {
			if normalizeOutput(tcResult.Output) == normalizeOutput(sub.Outputs[i]) {
				tcResult.Status = "OK"
			} else {
				tcResult.Status = "Wrong Answer"
				tcResult.Error = ""
			}
		}
		res.Cases[i] = tcResult
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func main() {
	http.HandleFunc("/run", runCodeHandler)
	port := ":8084"
	log.Println("Listening on", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
