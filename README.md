# Synapse

This is an attempt to re-write the synapse code is Golang.

---

## Prerequisites

- **Go 1.20+** (or a similar, recent version)
- **Make** (commonly available on Linux/macOS; on Windows, you can install via [Chocolatey](https://chocolatey.org/) or use a compatible environment like [Git Bash](https://gitforwindows.org/) or WSL)

---

## Getting Started

1. **Clone the repository**:
   ```
   git clone https://github.com/apache/synapse-go.git
   ```

2. **Check your Go version (optional)**:
    ```
    go version
    ```

Ensure it meets the minimum requirement.

## Building & Packaging

1. **Install Dependencies**

The Makefile automatically fetches Go module dependencies (via go mod tidy) when you run make for the first time.

2. **Build**

To compile the Synapse binary for your local machine

```
make build
```


This fetches dependencies (if not already done).

Compiles the Go application and places the binary in the bin/ directory.

3. **Package**

To create a zip file (synapse.zip) containing the compiled binary and the required folder structure run:

```
make package
```

This will:

- Create a temporary synapse/ directory (in the project root) with:
bin/ containing the compiled synapse binary
artifacts/APIs
artifacts/Endpoints
- Zip everything into synapse.zip at the root of the project.
- Clean up the temporary folders and the bin/ directory.

4. **All-in-One (Default)**

Simply running **make** (or **make all**) will execute the following steps in order:

- deps — Installs and tidies Go dependencies.
- build — Builds the synapse binary in bin/.
- package — Creates the synapse.zip with the required folder structure.

```
make
```

5. **Clean**

If you want to remove all build artifacts and start fresh, run:

```
make clean
```

This deletes the bin/ folder and any synapse/ directories created during the packaging step.

**Customizing the Build**

If you need to cross-compile for multiple OS/architectures, you can add additional targets to the Makefile. For example:

```
build-linux:
    GOOS=linux GOARCH=amd64 go build -ldflags=$(LDFLAGS) -o bin/$(PROJECT_NAME) $(MAIN_PACKAGE)
```

Then run:

```
make build-linux
```

…and package as usual with:

```
make package
```

(Adjust paths and names as needed.)

## Running the server

After you unzip synapse.zip, you will see:

```
synapse/
├── bin/
│   └── synapse       # Compiled binary
└── artifacts/
    ├── APIs/
    ├── Sequences/
    ├── Inbounds/
    └── Endpoints/
```

Unzip the archive:

```
unzip synapse.zip
```

Run the binary:

```
cd synapse/bin
./synapse
```

(On Windows, it would be .\synapse.exe if compiled for Windows.)

## API Routing

Synapse-go includes a robust API routing system that automatically registers APIs when they are deployed.

### API Definition

APIs are defined using XML files placed in the `artifacts/APIs/` directory:

```xml
<api context="/api" name="MyAPI">
    <resource methods="GET" uri-template="/hello">
        <inSequence>
            <log level="full"/>
            <!-- Other mediators -->
        </inSequence>
        <faultSequence>
            <!-- Error handling mediators -->
        </faultSequence>
    </resource>
</api>
```

When Synapse starts, it:
1. Scans the `artifacts/APIs/` directory
2. Parses each API definition
3. Automatically registers routes with the HTTP server
4. Starts the HTTP server on the configured port (default: 8000)

### Customizing the HTTP Server

You can customize the HTTP server port by modifying the environment variable:

```
export SYNAPSE_HTTP_PORT=9000
./synapse
```

Or directly in code using the `NewDeployerWithConfig` function:

```go
deployer := deployers.NewDeployerWithConfig(deployers.DeployerConfig{
    BasePath:   "/path/to/artifacts",
    ListenAddr: ":9000", // Custom port
}, mediator)
```

**Contributing**

- Fork the repository

- Create your feature branch (git checkout -b feature/my-feature)

- Commit your changes (git commit -am 'Add some feature')

- Push to the branch (git push origin feature/my-feature)

- Create a new Pull Request

**License**

Apache 2