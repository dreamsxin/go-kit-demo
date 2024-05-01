package cmd

import (
	"os/exec"
	"runtime"

	"github.com/sirupsen/logrus"
)

// 检查 protoc 命令
// https://pkg.go.dev/google.golang.org/protobuf/cmd/protoc-gen-go
// https://pkg.go.dev/google.golang.org/grpc/cmd/protoc-gen-go-grpc
func checkProtoc() bool {
	p := exec.Command("protoc")
	if p.Run() != nil {
		logrus.Error("Please install protoc first and than rerun the command")
		if runtime.GOOS == "windows" {
			logrus.Info(
				`Install proto3.
https://github.com/google/protobuf/releases
Update protoc Go bindings via
> go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.32
> go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.30

See also
https://github.com/grpc/grpc-go/tree/master/examples`,
			)
		} else if runtime.GOOS == "darwin" {
			logrus.Info(
				`Install proto3 from source macOS only.
> brew install autoconf automake libtool
> git clone https://github.com/google/protobuf
> ./autogen.sh ; ./configure ; make ; make install

Update protoc Go bindings via
> go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.32
> go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.30

See also
https://github.com/grpc/grpc-go/tree/master/examples`,
			)
		} else {
			logrus.Info(`Install proto3
sudo apt-get install -y git autoconf automake libtool curl make g++ unzip
git clone https://github.com/google/protobuf.git
cd protobuf/
./autogen.sh
./configure
make
make check
sudo make install
sudo ldconfig # refresh shared library cache.`)
		}
		return false
	}
	return true
}
