#### Option 1: Real LLM Build (Recommended)
This builds with real LLM inference using llama.cpp. Requires `llama-cpp` to be installed:

```bash
# Install llama.cpp first
brew install llama.cpp  # macOS
# OR
sudo apt install llama-cpp  # Ubuntu/Debian

# Build with real LLM support
go build -tags llama -o poaid cmd/poaid/*.go

# Alternative for Linux (if shell expansion fails):
go build -tags llama -o poaid cmd/poaid/main.go cmd/poaid/cli.go
```

#### Option 2: Stub LLM Build (Fast Testing)
This builds with stub LLM for fast testing without LLM dependencies:

```bash
# Build with stub LLM (no external dependencies)
go build -o poaid cmd/poaid/*.go

# Alternative for Linux (if shell expansion fails):
go build -o poaid cmd/poaid/main.go cmd/poaid/cli.go
```

#### Option 3: GPU Acceleration (macOS)
For macOS with Metal GPU acceleration:

```bash
CGO_LDFLAGS="-framework Metal -framework Foundation" go build -tags llama -o poaid cmd/poaid/*.go

# Alternative for Linux (if shell expansion fails):
CGO_LDFLAGS="-framework Metal -framework Foundation" go build -tags llama -o poaid cmd/poaid/main.go cmd/poaid/cli.go
```

**Note**: The `cmd/poaid/*.go` pattern includes all Go files in the cmd/poaid directory (main.go and cli.go). If your shell doesn't expand the `*.go` pattern correctly, use the explicit file listing instead. 