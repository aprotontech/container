set -ex

GOLANG_TAR=.devcontainer/go1.22.6.linux-amd64.tar.gz

yum install -y wget vim make git


if [ ! -f $GOLANG_TAR ]; then
    wget --timeout=120 https://golang.google.cn/dl/go1.22.6.linux-amd64.tar.gz -O $GOLANG_TAR
fi
tar zxf $GOLANG_TAR -C /usr/local/
GOPATH=$(/usr/local/go/bin/go env GOPATH)
echo 'export PATH=$PATH:/usr/local/go/bin:'$GOPATH/bin>>~/.bashrc

. ~/.bashrc
go env -w GOPROXY=https://goproxy.cn,direct
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
