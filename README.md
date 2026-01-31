# ztime

`ztime` is a standalone Go binary that mimics `zsh`'s `time` reserved word behavior. It allows you to time commands and format the output using the `TIMEFMT` environment variable, just like in `zsh`.

## Features

- **Customizable Output**: Supports the `TIMEFMT` environment variable with standard `zsh` specifiers.
- **Signal Forwarding**: Forwards signals (SIGINT, SIGTERM, etc.) to the child process.
- **Exit Code Transparency**: Returns the same exit code as the executed command (including 128+n for signals).
- **Resource Usage**: Captures and displays detailed resource usage via `syscall.Rusage`.

## Usage

```bash
ztime <command> [arguments...]
```

### Example

```bash
ztime sleep 1
# Output: sleep 1  0.00s user 0.00s system 0% cpu 0:01.01 total
```

### Custom Format

```bash
TIMEFMT="elapsed: %E, cpu: %P" ztime sleep 1
# Output: elapsed: 1.01s, cpu: 0%
```

## Supported Specifiers

| Specifier | Description |
| :--- | :--- |
| `%J` | Command and arguments |
| `%U` | CPU seconds in user mode |
| `%S` | CPU seconds in system mode |
| `%E` | Elapsed wall time in seconds |
| `%*E` | Elapsed wall time in `mm:ss.SS` format |
| `%P` | CPU percentage |
| `%M` | Maximum resident set size (KB) |
| `%W` | Number of swaps |
| `%F` | Major page faults |
| `%R` | Minor page faults |
| `%I` | Input operations |
| `%O` | Output operations |
| `%w` | Voluntary context switches |
| `%c` | Involuntary context switches |

## Building

Requires Go 1.25+.

```bash
go build -o ztime ./src
```

## Testing

```bash
go test -v ./...
```
