# xeh-project

The ultra-flexible XML-based multi-runtime orchestrator programming language. `xeh` allows developers to write and coordinate multiple backend languages (Python, Go, Rust, Ruby, Java, etc.) inside a single unified XML syntax, leveraging the full power of OS-level pipeline communication.

## Features

- **Unified Multi-Runtime**: Run Python, Go, Rust, and more side-by-side within a single `.xeh` source file.
- **Dynamic Engine Assignment**: Map customized XML tags to any language execution commands via `set.json`.
- **Named Memory Space System**: Isolate and manage variable spaces via strict JSON object structures and stream packet passing.
- **OS-Level Pipe Communication**: Real-time memory streaming directly via standard input/output pipelines.
- **No Constraints**: 100% freedom over your project structures, language platforms, and runtime configurations.

## Architecture

```text
[app.xeh] (XML Source with Named Memory Space)
    │
    ▼ (XML Parse / Dynamic Routing)
[xeh Core Engine] (Go-powered Orchestrator) ── Loaded by ── [set.json]
    │
    ├── Pipe Memory Stream (Real-time Packet JSON Passing)
    ▼
[Subprocesses] (Python/Flask, Go, Rust, Java, etc. running concurrently)
```

## Getting Started

### Prerequisites

- [Go](https://go.dev) (1.18 or higher)
- Runtimes you want to use (e.g., Python, Ruby) installed on your system path.

### Installation & Run

1. Clone the repository:
   ```bash
   git clone https://github.com
   cd xeh-lang
   ```

2. Run the `xeh` engine using the sample source code:
   ```bash
   go run main.go
   ```

3. Check version and license information via CLI options:
   ```bash
   go run main.go --version
   go run main.go --license
   ```

## Configuration (`set.json`)

All configurations, metadata, language commands, and custom engines are unified in `set.json`. You can easily add any backend runtime without modifying the core engine code.

```json
{
  "meta": {
    "name": "xeh-core-engine",
    "version": "9.9.0",
    "license": "MIT",
    "charset": "UTF-8"
  },
  "runtimes": {
    "py": { "command": "python", "args": ["-u", "{src}"] },
    "go": { "command": "go", "args": ["run", "{src}"] }
  },
  "engines": {
    "ai-logic": { "type": "py", "src": "plugins/ai_core.py" }
  }
}
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
