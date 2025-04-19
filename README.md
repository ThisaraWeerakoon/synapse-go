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
4. Starts the HTTP server on the configured port (default: 8290)

### CORS Support

Synapse-go provides built-in CORS (Cross-Origin Resource Sharing) support for APIs. You can configure CORS settings for each API using the `<cors>` element in the API definition.

Example API with CORS configuration:

```xml
<api context="/api" name="MyAPI">
    <!-- Configure CORS for this API -->
    <cors enabled="true"
          allow-origins="https://example.com,https://app.example.com"
          allow-methods="GET,POST,PUT,DELETE,PATCH,OPTIONS"
          allow-headers="Content-Type,Authorization,X-Requested-With,Accept"
          expose-headers="X-Request-ID,X-Response-Time"
          allow-credentials="true"
          max-age="3600" />
          
    <!-- API Resources -->
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

#### CORS Configuration Options

| Attribute | Description | Default |
|-----------|-------------|---------|
| `enabled` | Enable/disable CORS support for this API | `false` |
| `allow-origins` | Comma-separated list of allowed origins (domains) | `*` (all origins) |
| `allow-methods` | Comma-separated list of allowed HTTP methods | `GET,POST,PUT,DELETE,OPTIONS,PATCH` |
| `allow-headers` | Comma-separated list of allowed HTTP headers | `Origin,Content-Type,Accept,Authorization` |
| `expose-headers` | Comma-separated list of headers to expose to clients | (none) |
| `allow-credentials` | Whether to allow credentials (cookies, auth) | `false` |
| `max-age` | Cache duration for preflight responses in seconds | `86400` (24 hours) |

#### CORS Features

- **Automatic OPTIONS Handling**: Synapse automatically creates OPTIONS method handlers for all API resources when CORS is enabled.
- **Origin Validation**: Requests from unauthorized origins are rejected with a 403 Forbidden response.
- **Wildcard Support**: You can use `*` to allow all origins, or `*.example.com` to allow all subdomains of example.com.
- **Preflight Requests**: OPTIONS requests are handled correctly with appropriate CORS headers.

### Swagger Documentation

Synapse-go provides built-in Swagger/OpenAPI documentation for your APIs. The documentation is automatically generated from your API definitions and accessible through special URLs.

#### Accessing Swagger Documentation

For any API named `<API_NAME>`, you can access its Swagger documentation at:

- **YAML Format**: `http://localhost:8290/<API_NAME>?swagger.yaml`
  - Example: `http://localhost:8290/FoodAPI?swagger.yaml`

- **JSON Format**: `http://localhost:8290/<API_NAME>?swagger.json`
  - Example: `http://localhost:8290/FoodAPI?swagger.json`

- **HTML UI**: `http://localhost:8290/<API_NAME>?swagger.html`
  - Example: `http://localhost:8290/FoodAPI?swagger.html`

If your API has a version specified (e.g., `version="1.0"` in the API definition), the documentation URLs include the version:

- `http://localhost:8290/<API_NAME>/<API_VERSION>?swagger.yaml`
- `http://localhost:8290/<API_NAME>/<API_VERSION>?swagger.json`
- `http://localhost:8290/<API_NAME>/<API_VERSION>?swagger.html`

#### Generated Documentation Includes

The automatically generated Swagger documentation includes:

- API basic information (name, description, version)
- All endpoints (resources) defined in the API
- HTTP methods supported by each endpoint
- Path parameters extracted from URI templates
- Response definitions

#### Swagger UI

The HTML documentation URL (`?swagger.html`) provides an interactive Swagger UI interface where you can:

- Browse all API endpoints
- Expand operations to see details
- View request/response schemas
- Test API endpoints directly from the browser

This makes it easy to share API documentation with developers or test your APIs without additional tools.

**Contributing**

- Fork the repository

- Create your feature branch (git checkout -b feature/my-feature)

- Commit your changes (git commit -am 'Add some feature')

- Push to the branch (git push origin feature/my-feature)

- Create a new Pull Request

**License**

Apache 2