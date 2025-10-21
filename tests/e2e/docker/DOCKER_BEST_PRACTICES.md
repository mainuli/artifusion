# Docker Best Practices Used in Tests

## CMD Instruction - JSON Array Format

### ✅ Correct (JSON Array)
```dockerfile
CMD ["cat", "/hello.txt"]
CMD ["node", "server.js"]
CMD ["python", "-m", "http.server"]
```

### ❌ Incorrect (Shell Form)
```dockerfile
CMD cat /hello.txt
CMD node server.js
CMD python -m http.server
```

## Why JSON Array Format?

### 1. Proper Signal Handling
**Shell form**:
```dockerfile
CMD cat /hello.txt
```
- Runs as: `/bin/sh -c "cat /hello.txt"`
- Shell (PID 1) receives signals, not your app
- SIGTERM/SIGINT may not reach your process
- Container may not stop gracefully

**JSON array form**:
```dockerfile
CMD ["cat", "/hello.txt"]
```
- Runs directly as: `cat /hello.txt` (no shell wrapper)
- Your process is PID 1
- Receives signals directly
- Stops gracefully on `docker stop`

### 2. Predictable Behavior
**Shell form**:
- Shell variable expansion: `$HOME`, `$PATH`, etc.
- Command substitution: `` `date` ``
- Globbing: `*.txt`
- Can have unexpected side effects

**JSON array form**:
- No variable expansion
- No command substitution
- No globbing
- Predictable, explicit behavior

### 3. No Build Warnings
JSON array form follows Docker best practices and produces no warnings:
```
✓ Image built successfully (no warnings)
```

Shell form produces BuildKit warning:
```
⚠️  JSONArgsRecommended: JSON arguments recommended for CMD
```

## Real-World Examples

### Node.js Application
```dockerfile
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
CMD ["node", "server.js"]  # ✅ Correct
```

### Python Application
```dockerfile
FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt
COPY . .
CMD ["python", "-m", "uvicorn", "main:app", "--host", "0.0.0.0"]  # ✅ Correct
```

### Go Application
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o /app/server

FROM alpine:latest
COPY --from=builder /app/server /server
CMD ["/server"]  # ✅ Correct
```

## ENTRYPOINT vs CMD

### ENTRYPOINT + CMD (Recommended Pattern)
```dockerfile
ENTRYPOINT ["python", "app.py"]
CMD ["--port", "8080"]
```
- ENTRYPOINT defines the executable
- CMD provides default arguments
- Can override CMD at runtime: `docker run myapp --port 9000`

### CMD Only (Simpler Pattern)
```dockerfile
CMD ["python", "app.py", "--port", "8080"]
```
- Complete command in CMD
- Entire command can be overridden at runtime

## When Shell Form Is Acceptable

Shell form is OK for simple cases where you WANT shell features:

```dockerfile
# Need shell variable expansion
CMD echo "Starting app at $(date)"

# Need command chaining
CMD nginx && tail -f /var/log/nginx/access.log

# Need piping
CMD cat /config.txt | grep setting
```

But these can usually be rewritten as JSON array with explicit shell:
```dockerfile
CMD ["sh", "-c", "echo 'Starting app at $(date)'"]
CMD ["sh", "-c", "nginx && tail -f /var/log/nginx/access.log"]
CMD ["sh", "-c", "cat /config.txt | grep setting"]
```

## Test Script Fix

Our Docker E2E test was updated to follow best practices:

**Before** (produces warning):
```bash
cat > Dockerfile << 'EOF'
FROM alpine:latest
RUN echo "Hello from Artifusion!" > /hello.txt
CMD cat /hello.txt  # ❌ Shell form
EOF
```

**After** (clean build):
```bash
cat > Dockerfile << 'EOF'
FROM alpine:latest
RUN echo "Hello from Artifusion!" > /hello.txt
CMD ["cat", "/hello.txt"]  # ✅ JSON array form
EOF
```

**Result**: No warnings during `docker build`

## Signal Handling Test

You can test proper signal handling:

**Shell form** (slow shutdown):
```dockerfile
FROM alpine
CMD sleep 30
```
```bash
docker run -d --name test myimage
time docker stop test  # Takes ~10s (waits for SIGKILL timeout)
```

**JSON array form** (fast shutdown):
```dockerfile
FROM alpine
CMD ["sleep", "30"]
```
```bash
docker run -d --name test myimage
time docker stop test  # Takes ~10s (sleep ignores SIGTERM, but receives it)
```

For apps that handle SIGTERM properly:
```dockerfile
FROM node:18-alpine
COPY server.js .
CMD ["node", "server.js"]  # Stops immediately on SIGTERM
```

## References

- [Dockerfile best practices](https://docs.docker.com/develop/develop-images/dockerfile_best-practices/)
- [CMD instruction](https://docs.docker.com/engine/reference/builder/#cmd)
- [ENTRYPOINT instruction](https://docs.docker.com/engine/reference/builder/#entrypoint)
- [Container signals](https://docs.docker.com/engine/reference/commandline/stop/)

## Quick Checklist

When writing Dockerfiles:
- ✅ Use JSON array format for CMD: `CMD ["executable", "arg1", "arg2"]`
- ✅ Use JSON array format for ENTRYPOINT
- ✅ Combine ENTRYPOINT + CMD for flexibility
- ✅ Handle signals properly in your application (SIGTERM for graceful shutdown)
- ✅ Test `docker stop` to verify graceful shutdown
- ❌ Avoid shell form unless you specifically need shell features
- ❌ Don't use `CMD command arg1 arg2` syntax
