# Viper Configuration Example

A demonstration of using [Viper](https://github.com/spf13/viper) for application configuration management in Go. This example shows how to read configuration from YAML files and override values with environment variables.

## Features

- 📄 **YAML Configuration**: Read configuration from `config.yaml`
- 🔧 **Environment Variable Override**: Environment variables automatically override config file values
- 🏗️ **Structured Config**: Type-safe configuration using Go structs
- 🔄 **Automatic Env Mapping**: Dot notation in config maps to underscore in env vars (`server.port` → `SERVER_PORT`)

## Configuration Structure

```yaml
server:
  host: localhost
  port: 8080
  timeout: 10s
```

Maps to Go struct:

```go
type Config struct {
    Server struct {
        Host    string
        Port    int
        Timeout time.Duration
    }
}
```

## Usage

### Run with default config

```bash
go run .
```

Output:
```
Server: localhost:8080
Timeout: 10s
```

### Override with environment variables

```bash
# Override port using environment variable
export SERVER_PORT=9000
go run .
```

Output:
```
Server: localhost:9000
Timeout: 10s
```

### Override multiple values

```bash
export SERVER_HOST=0.0.0.0
export SERVER_PORT=3000
export SERVER_TIMEOUT=30s
go run .
```

Output:
```
Server: 0.0.0.0:3000
Timeout: 30s
```

## Key Features of Viper

### 1. Automatic Environment Variable Mapping

```go
viper.AutomaticEnv()
viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
```

This allows `server.port` in config to be overridden by `SERVER_PORT` environment variable.

### 2. Type-Safe Configuration

```go
var config Config
err = viper.Unmarshal(&config)
```

Viper automatically unmarshals the configuration into strongly-typed Go structs, including parsing durations like `10s` into `time.Duration`.

### 3. Multiple Configuration Sources

Viper supports reading from:
- Configuration files (YAML, JSON, TOML, etc.)
- Environment variables
- Command-line flags
- Remote config systems (etcd, Consul)

## File Structure

```
viper_config/
├── config.go       # Configuration setup and struct definition
├── config.yaml     # Default configuration file
└── main.go         # Example usage
```

## Why Use Viper?

✅ **12-Factor App Compliant**: Separate config from code  
✅ **Flexible**: Multiple config sources with priority order  
✅ **Type-Safe**: Strong typing with automatic unmarshaling  
✅ **Production-Ready**: Used by many popular Go projects  
✅ **Easy Testing**: Override config for different environments  

## Common Use Cases

- Application server configuration (host, port, timeouts)
- Database connection strings
- Feature flags
- API keys and secrets (combined with secure vaults)
- Multi-environment configuration (dev, staging, prod)

## Dependencies

```bash
go get github.com/spf13/viper
```

## Learn More

- [Viper GitHub Repository](https://github.com/spf13/viper)
- [The Twelve-Factor App - Config](https://12factor.net/config)
