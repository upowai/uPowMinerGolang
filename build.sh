DIR="builds/"

# remove builds dir if exists
if [ -d "$DIR" ]; then
    rm -r $DIR
fi

# compile for linux
# 64 bit
GOOS=linux GOARCH=amd64 go build -o $DIR/stand-alone-miner-linux64
# 32 bit
GOOS=linux GOARCH=386 go build -o $DIR/stand-alone-miner-linux32

echo "compiled for linux"

# compile for Ubuntu (or any Linux distribution) on ARM64
GOOS=linux GOARCH=arm64 go build -o $DIR/stand-alone-miner-linux-arm64

echo "compiled for Ubuntu ARM64"

# compile for windows
# 64 bit
GOOS=windows GOARCH=amd64 go build -o $DIR/stand-alone-miner-win64.exe
# 32 bit
GOOS=windows GOARCH=386 go build -o $DIR/stand-alone-miner-win32.exe

echo "compiled for windows"

# compile for macos
# amd 64 bit
GOOS=darwin GOARCH=amd64 go build -o $DIR/stand-alone-miner-macos-amd64
# arm 64 bit
GOOS=darwin GOARCH=arm64 go build -o $DIR/stand-alone-miner-macos-arm64

echo "compiled for macos"


