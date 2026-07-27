package services

import (
	"context"
	"fmt"
	"time"

	"github.com/dop251/goja"
)

type CodeRunResult struct {
	Logs          []string `json:"logs"`
	Data          any      `json:"data"`
	Error         string   `json:"error,omitempty"`
	ExecutionTime string   `json:"executionTime,omitempty"`
}

// RunJSScript executes user JS in an isolated goja VM (pure Go, no Node dependency).
// Supports console.log / console.error. User code that requires Node modules (axios, moment)
// will fail unless rewritten to pure JS — same sandbox spirit as the Next.js worker.
func RunJSScript(code string, timeoutMs int) CodeRunResult {
	if timeoutMs <= 0 {
		timeoutMs = 10000
	}
	started := time.Now().UTC().Format(time.RFC3339)

	result := CodeRunResult{
		Logs:          []string{},
		Data:          map[string]any{},
		ExecutionTime: started,
	}

	vm := goja.New()
	logs := &result.Logs

	console := vm.NewObject()
	_ = console.Set("log", func(call goja.FunctionCall) goja.Value {
		msg := formatArgs(call.Arguments)
		*logs = append(*logs, msg)
		return goja.Undefined()
	})
	_ = console.Set("error", func(call goja.FunctionCall) goja.Value {
		msg := formatArgs(call.Arguments)
		*logs = append(*logs, msg)
		return goja.Undefined()
	})
	_ = vm.Set("console", console)

	// Timeout via interrupt
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()
	go func() {
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			vm.Interrupt("Execution timed out")
		}
	}()

	// Wrap to allow return values similar to the original async IIFE.
	wrapped := fmt.Sprintf(`(function(){ %s })()`, code)
	val, err := vm.RunString(wrapped)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	if val != nil && !goja.IsUndefined(val) && !goja.IsNull(val) {
		exported := val.Export()
		if exported != nil {
			result.Data = exported
		}
	}
	return result
}

func formatArgs(args []goja.Value) string {
	parts := make([]string, 0, len(args))
	for _, a := range args {
		if a == nil {
			parts = append(parts, "undefined")
			continue
		}
		parts = append(parts, fmt.Sprint(a.Export()))
	}
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " "
		}
		out += p
	}
	return out
}
