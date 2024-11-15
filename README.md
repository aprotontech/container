# container
Container的学习和实验


这个仓库用于从零到一的实现容器运行时的学习和研究。
详细的文档可以参考: [https://www.aproton.tech/](https://www.aproton.tech/)

## How To Build & Test
```shell
# build
make

# test pull
./bin/sctr image pull docker.io/library/nginx:latest

# test run
./bin/sctr run -d docker.io/library/nginx:latest
```