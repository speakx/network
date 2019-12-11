# 关于项目理的工具脚本
## 项目脚本
### go.sh
用来本机调试运行，go.sh 后面可以追加自定义参数，例如:`go.sh -test`

### build.sh
将 go 打包编译，可通过传入不同的参数编译出对应的平台版本  
默认关闭了C++依赖编译，如何工程里有C++混合编译，需要手动打开`CGO_ENABLED=1`  

### clone.sh
通过此脚本将Demo工程方便的完整复制为其他项目名称（修改go的import、修改根目录名称、修改go mod文件）

## 其他脚本
### ./shell/(configure.sh、gen-proto.sh)
configure.sh 自动引入pkg目录下和当前工程同级目录下，所有包含go.mod的工程  
gen-proto.sh proto编译脚本（会自动编译项目src目录下proto目录中的 *.proto 文件）  

## 关于 go mod
### 方便工程管理，可以脱离 GOPATH 的束缚来进行 mod 引用
### 如果发现 `go mod tidy` 卡住，可使用国内代理： `go env -w GOPROXY=https://goproxy.cn,direct`