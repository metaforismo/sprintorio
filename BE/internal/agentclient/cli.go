package agentclient

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const Help = `sprintorio — workspace operations for agents
Environment: SPRINTORIO_URL, SPRINTORIO_TOKEN, SPRINTORIO_WORKSPACE
  discover [resource]           operation catalogue, no credentials needed
  schema OPERATION              exact JSON argument schema, no credentials needed
  call OPERATION --input -|FILE  JSON stdin/file, default input {}
       [--detail] [--fields id,title,...] [--jsonl]
  context                       workspace, teams, labels and projects
  upload --file FILE            multipart asset upload, max 10 MiB
  export --output FILE          ZIP export, max 64 MiB, new file only
  mcp                           stdio MCP, protocol JSON only on stdout
  mcp-http [--listen 127.0.0.1:8091] [--origins https://client.example]
                                HTTP /mcp; each caller supplies Bearer token
Input: {"id":"...","team_id":"...","query":{...},"body":{...},"detail":true}
Issue lists default to page 1 / 25; inspect has_more and advance query.page.
Delivery updates require body {"version":LAST_READ,"plan":{...}}.
Exit: 0 success, 1 I/O, 2 input/config/API, 3 auth, 4 conflict,
      5 not found, 6 rate limit, 7 transport/server failure.
`

func RunCLI(ctx context.Context, args []string, in io.Reader, out, stderr io.Writer) int {
	fail := func(err error) int {
		e, ok := err.(*Error)
		if !ok {
			e = Err("IO_ERROR", "Could not complete command", 0)
		}
		_ = json.NewEncoder(stderr).Encode(map[string]any{"error": e})
		return ExitCode(err)
	}
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		_, err := io.WriteString(out, Help)
		if err != nil {
			return fail(err)
		}
		return 0
	}
	var result any
	var err error
	jsonl := false
	switch args[0] {
	case "discover":
		if len(args) > 2 {
			return fail(Err("INVALID_INPUT", "discover takes an optional resource", 0))
		}
		resource := ""
		if len(args) == 2 {
			resource = args[1]
		}
		result = map[string]any{"operations": Discover(resource)}
	case "schema":
		if len(args) != 2 {
			return fail(Err("INVALID_INPUT", "schema requires an operation", 0))
		}
		result, err = Schema(args[1])
	case "mcp":
		if len(args) != 1 {
			return fail(Err("INVALID_INPUT", "mcp takes no arguments", 0))
		}
		client, e := FromEnv()
		if e != nil {
			return fail(e)
		}
		if err := ServeMCP(ctx, in, out, client); err != nil {
			return fail(err)
		}
		return 0
	case "upload", "export":
		f := flag.NewFlagSet(args[0], flag.ContinueOnError)
		f.SetOutput(io.Discard)
		fileName := f.String("file", "", "upload file")
		outputName := f.String("output", "", "export output file")
		if f.Parse(args[1:]) != nil || f.NArg() != 0 {
			return fail(Err("INVALID_INPUT", "Invalid transfer flags", 0))
		}
		client, e := FromEnv()
		if e != nil {
			return fail(e)
		}
		if args[0] == "export" {
			if *fileName != "" {
				return fail(Err("INVALID_INPUT", "export uses --output", 0))
			}
			result, err = client.Export(ctx, *outputName)
		} else {
			if *fileName == "" || *outputName != "" {
				return fail(Err("INVALID_INPUT", "upload requires --file", 0))
			}
			file, e := os.Open(*fileName)
			if e != nil {
				return fail(Err("IO_ERROR", "Cannot open upload file", 0))
			}
			defer file.Close()
			result, err = client.Upload(ctx, filepath.Base(*fileName), file, MaxUpload)
		}
	case "mcp-http":
		f := flag.NewFlagSet("mcp-http", flag.ContinueOnError)
		f.SetOutput(io.Discard)
		listen := f.String("listen", "127.0.0.1:8091", "listen host:port")
		origins := f.String("origins", "", "exact origin allowlist")
		if f.Parse(args[1:]) != nil || f.NArg() != 0 {
			return fail(Err("INVALID_INPUT", "Invalid mcp-http flags", 0))
		}
		allow := []string{}
		if *origins != "" {
			allow = strings.Split(*origins, ",")
		}
		if e := ServeHTTP(ctx, *listen, os.Getenv("SPRINTORIO_URL"), os.Getenv("SPRINTORIO_WORKSPACE"), allow); e != nil {
			return fail(e)
		}
		return 0
	case "context":
		if len(args) != 1 {
			return fail(Err("INVALID_INPUT", "context takes no arguments", 0))
		}
		client, e := FromEnv()
		if e != nil {
			return fail(e)
		}
		result, err = client.Context(ctx)
	case "call":
		if len(args) < 2 {
			return fail(Err("INVALID_INPUT", "call requires an operation", 0))
		}
		f := flag.NewFlagSet("call", flag.ContinueOnError)
		f.SetOutput(io.Discard)
		fileName := f.String("input", "", "JSON file or stdin (-)")
		detail := f.Bool("detail", false, "detail")
		fields := f.String("fields", "", "output fields")
		jl := f.Bool("jsonl", false, "JSONL")
		if f.Parse(args[2:]) != nil || f.NArg() != 0 {
			return fail(Err("INVALID_INPUT", "Invalid call flags; use help", 0))
		}
		jsonl = *jl
		raw := []byte("{}")
		if *fileName != "" {
			reader := in
			if *fileName != "-" {
				file, e := os.Open(*fileName)
				if e != nil {
					return fail(Err("IO_ERROR", "Could not open input file", 0))
				}
				defer file.Close()
				reader = file
			}
			raw, err = io.ReadAll(io.LimitReader(reader, MaxMessage+1))
			if err != nil {
				return fail(err)
			}
		}
		input, e := DecodeInput(raw)
		if e != nil {
			return fail(e)
		}
		if *detail {
			input["detail"] = true
		}
		if *fields != "" {
			a := []any{}
			for _, s := range strings.Split(*fields, ",") {
				a = append(a, strings.TrimSpace(s))
			}
			input["fields"] = a
		}
		client, e := FromEnv()
		if e != nil {
			return fail(e)
		}
		result, err = client.Call(ctx, args[1], input)
	default:
		return fail(Err("INVALID_INPUT", "Unknown command; use help", 0))
	}
	if err != nil {
		return fail(err)
	}
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	if jsonl {
		if list, ok := result.(map[string]any); ok {
			if data, ok := list["data"].([]any); ok {
				for _, item := range data {
					if err := encoder.Encode(map[string]any{"record": item}); err != nil {
						return fail(err)
					}
				}
				meta := map[string]any{}
				for k, v := range list {
					if k != "data" {
						meta[k] = v
					}
				}
				if err := encoder.Encode(map[string]any{"pagination": meta}); err != nil {
					return fail(err)
				}
				return 0
			}
		}
		if list, ok := result.([]any); ok {
			for _, item := range list {
				if err := encoder.Encode(map[string]any{"record": item}); err != nil {
					return fail(err)
				}
			}
			return 0
		}
	}
	if err := encoder.Encode(result); err != nil {
		return fail(err)
	}
	return 0
}
