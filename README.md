# rbi
mac cgo
```bash
 CC=x86_64-linux-musl-gcc CXX=x86_64-linux-musl-g++ CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CGO_LDFLAGS="-static" go build -v -ldflags="-w -s" -trimpath  -o rbi
```