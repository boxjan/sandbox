# SandBox

### 原本使用 C/C++ 开发的代码 已移入 c++ 分支。

本项目是 OnlineJudge 的 Judger 部分的 沙箱 实现，被设计为单独的可执行程序， 也同时提供 API 的方式与 Judger 整合。
设计过程中, 系统调用皆考虑在 docker 容器内运行，倘若在操作系统上运行，可能会造成严重的安全问题！！

## 说明
使用Go语言进行实现

2019-11-30 进度更新： 暂停开发，由于主要开发人员准备跑路了。
他说目前时间、内存、线程限制均已经实现，暂未对结果做判断。
系统调用限制方面，只有简单的框架，自带限制还未编写，自定义系统调用限制。

## 构建说明


## 使用说明
使用 ``` --help ``` 或 ``` -h ``` 获取帮助.

## 项目测试


## 第三方库
