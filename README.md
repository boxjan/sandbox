# SandBox
本项目是 OnlineJudge 的 Judger 部分的 沙箱 实现， 被设计为单独的可执行程序， 也同时提供 API 的方式与 Judger 整合。
设计过程中, 系统调用皆考虑在 docker 容器内运行，倘若在操作系统上运行，可能会造成严重的安全问题！！

## 说明
通过 Linux 提供的函数, 对通过 沙箱运行的 程序进行一定的资源限制. 提供一个黑名单的系统调用, 但 **非常不建议** 直接在系统中使用, 
关于系统系统调用详情见 [syscall.md](syscall.md).

## 构建说明
需要系统包含 ```libseccomp-dev``` 包， 可用 yum 或 apt安装
此项目使用 Cmake 进行 Makefile 的环境配置, 通过 ``` cmake . ``` 构建 makefile 后, 再通过 ``` make ``` 进行编译.

## 使用说明
使用 ``` --help ``` 或 ``` -h ``` 获取帮助.

## 项目测试
项目中 test/bad_code 文件夹下包含沙箱测试的部分样例, 参考于 [QingdaoU Judger](https://github.com/QingdaoU/Judger/), 
通过 ``` make test ``` 进行测试

## 第三方库
- [getopt](https://github.com/r-lyeh-archived/getopt.git) 用于处理输入参数