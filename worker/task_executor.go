package worker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/oyavri/pi-bully/storage"
	"go.uber.org/zap"
)

type TaskExecutor struct {
	storage       storage.Storage
	pythonPath    string
	outputBaseURI string
	logger        *zap.Logger
}

func NewTaskExecutor(storage storage.Storage, pythonPath string, outputBaseURI string, logger *zap.Logger) *TaskExecutor {
	return &TaskExecutor{
		storage:       storage,
		pythonPath:    pythonPath,
		outputBaseURI: outputBaseURI,
		logger:        logger.With(zap.String("component", "task_executor")),
	}
}

func (e *TaskExecutor) Execute(ctx context.Context, a Assignment) error {
	workDir, err := os.MkdirTemp("", "pi-bully-task-*")
	if err != nil {
		return fmt.Errorf("create temp workdir: %w", err)
	}
	defer os.RemoveAll(workDir)

	scriptPath := filepath.Join(workDir, "task.py")
	inputPath := filepath.Join(workDir, "input")

	outputName := a.TaskID + ".mp4"
	outputPath := filepath.Join(workDir, outputName)
	uploadURI := buildOutputURI(e.outputBaseURI, outputName)

	if err := e.storage.Download(ctx, a.ExecutableURI, scriptPath); err != nil {
		return fmt.Errorf("download executable: %w", err)
	}

	taskInput := ""
	if a.InputURI != "" {
		if err := e.storage.Download(ctx, a.InputURI, inputPath); err != nil {
			return fmt.Errorf("download input: %w", err)
		}
		taskInput = inputPath
	}

	args := append([]string{scriptPath}, a.Args...)
	cmd := exec.CommandContext(ctx, e.pythonPath, args...)
	cmd.Dir = workDir
	cmd.Env = []string{
		"TASK_INPUT=" + taskInput,
		"TASK_OUTPUT=" + outputPath,
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("execute task: %w: %s", err, stderr.String())
		}
		if stdout.Len() > 0 {
			return fmt.Errorf("execute task: %w: %s", err, stdout.String())
		}
		return fmt.Errorf("execute task: %w", err)
	}

	if e.outputBaseURI != "" {
		if _, err := os.Stat(outputPath); err != nil {
			return fmt.Errorf("task finished but output file missing at %s: %w", outputPath, err)
		}

		if err := e.storage.Upload(ctx, uploadURI, outputPath); err != nil {
			return fmt.Errorf("upload output: %w", err)
		}
	}

	return nil
}

func buildOutputURI(baseURI string, fileName string) string {
	if strings.HasSuffix(baseURI, "/") {
		return baseURI + fileName
	}
	return baseURI + "/" + fileName
}
