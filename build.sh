GOOS=windows GOARCH=arm64 go build -tags webifyvideo -o build/webifyvideo-windows-arm64.exe
GOOS=windows GOARCH=amd64 go build -tags webifyvideo -o build/webifyvideo-windows-amd64.exe
GOOS=darwin GOARCH=arm64 go build -tags webifyvideo -o build/webifyvideo-macos-arm64
GOOS=darwin GOARCH=amd64 go build -tags webifyvideo -o build/webifyvideo-macos-amd64
GOOS=linux GOARCH=arm64 go build -tags webifyvideo -o build/webifyvideo-linux-arm64
GOOS=linux GOARCH=amd64 go build -tags webifyvideo -o build/webifyvideo-linux-amd64

GOOS=windows GOARCH=arm64 go build -tags extractsubs -o build/extractsubs-windows-arm64.exe
GOOS=windows GOARCH=amd64 go build -tags extractsubs -o build/extractsubs-windows-amd64.exe
GOOS=darwin GOARCH=arm64 go build -tags extractsubs -o build/extractsubs-macos-arm64
GOOS=darwin GOARCH=amd64 go build -tags extractsubs -o build/extractsubs-macos-amd64
GOOS=linux GOARCH=arm64 go build -tags extractsubs -o build/extractsubs-linux-arm64
GOOS=linux GOARCH=amd64 go build -tags extractsubs -o build/extractsubs-linux-amd64
