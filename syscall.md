# SYSTEM CALL

系统调用(System Call) 说明

### 综述
白名单参考来源 [Docker](https://github.com/moby/moby/blob/master/profiles/seccomp/default.json)
考虑实际使用时是在docker容器中, 故所有系统调用规则皆基于 docker 的调用白名单再进行限制, 
一旦不是在在docker中使用, 则需要修改黑名单机制的规则

系统白名单共分为 4 个级别: 
- 无限制
- 编译用
- 严格限制
- 温和限制

### 无限制 
无需指定 -p 参数

### 编译用 
禁止网络连接和部分设置属性, 黑名单基于docker的白名单构建

使用 ```-p compile``` 指定, 

### 温和级别
允许使用 clone 用于多线程, 有部分语言会使用到, 比如ruby等, 黑名单基于docker构建

使用 ```-p gentle``` 指定 

### 严格级别
这个级别可以在系统中直接使用. 在 docker 白名单的基础上, 进一步限制了系统调用, 满足基本需求

使用 ```-p strict``` 指定