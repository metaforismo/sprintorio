package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Run the production entrypoint in a child with real stdin/stdout pipes.
func TestMCPProcess(t *testing.T) {
	if os.Getenv("SPRINTORIO_PROCESS_TEST") == "1" {
		os.Args = []string{os.Args[0], "mcp"}
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestMCPProcess$")
	cmd.Env = append(os.Environ(), "SPRINTORIO_PROCESS_TEST=1", "SPRINTORIO_URL=https://example.invalid", "SPRINTORIO_WORKSPACE=demo", "SPRINTORIO_TOKEN=spr_process_test_token")
	cmd.Stdin = strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"process-test","version":"1"}}}` + "\n" + `{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n" + `{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n")
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatal(stderr.String())
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatal(out.String())
	}
	for _, line := range lines {
		if !json.Valid([]byte(line)) {
			t.Fatal(line)
		}
	}
	if strings.Contains(out.String(), "spr_process_test_token") || !strings.Contains(lines[1], `"tools"`) {
		t.Fatal(out.String())
	}
}
