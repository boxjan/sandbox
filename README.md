# SandBox
本项目是 OnlineJudge 的 Judger 部分的 沙箱 实现, 被设计为单独的可执行程序, 也同时提供 API 的方式与 Judger 整合

## 说明
通过 Linux 提供的函数, 对通过 沙箱运行的 程序进行一定的资源限制

## 构建说明
此项目使用 Cmake 进行 Makefile 的环境配置, 通过 ``` cmake . ``` 构建 makefile 后, 再通过 ``` make``` 进行编译.

## 项目测试
项目中 test/bad_code 文件夹下包含沙箱测试的部分样例, 参考于 [Judger QingdaoU](https://github.com/QingdaoU/Judger/), 通过 ``` make test ``` 进行测试

## 第三方库
- [argh](https://github.com/adishavit/argh.git) 用于处理输入参数