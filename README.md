# braid-scaffold
braid scaffold repository


``` go
/*

components		        | 游戏内的通用组件 (规则引擎, 日期模块, 错误处理, 日志 ...
└── actors		        | 各种可复用计算单元（mail, rank, chat, gate ...
    └── core #braid#	| 分布式系统，actor模型，addressbook中心化地址管理

---------------------

event_handlers (control
	- 事件处理函数
states (model
	- 计算单元的状态维护

*/

```


### 默认支持 jaeger 链路追踪
> 使用 docker 安装 jaeger

```shell
$ docker run -d --name jaeger \
  -p 16686:16686 \
  -p 14268:14268 \
  jaegertracing/all-in-one:latest
```

[![image.png](https://i.postimg.cc/wTVhQhyM/image.png)](https://postimg.cc/XprGVBg6)


<div style="display: flex; align-items: center; margin: 1em 0;">
  <div style="flex-grow: 1; height: 1px; background-color: #ccc;"></div>
  <div style="margin: 0 10px; font-weight: bold; color: #666;">测试机器人</div>
  <div style="flex-grow: 1; height: 1px; background-color: #ccc;"></div>
</div>

### 通过测试机器人验证 braid 提供的服务器接口
> 使用上面的脚手架工程

```shell
$ cd you-project-name/testbots

# 1. 运行机器人服务器
$ go run main.go

# 2. 下载 gobot 编辑器（最新版本
https://github.com/pojol/gobot/releases

# 3. 运行 gobot 编辑器
$ run gobot_editor_[ver].exe or .dmg

# 4. 进入到 bots 页签
# 5. 将 testbots 目录中的 testbot.bh 文件拖拽到 bots 页面中
# 6. 选中 testbot 机器人，点击 load 加载 testbot
# 7. 点击左下角按钮，构建机器人实例
# 8. 点击单步运行按钮，查看机器人和 braid 服务器交互情形
```

[测试机器人 Gobot](https://github.com/pojol/gobot)
[![image.png](https://i.postimg.cc/LX5gbV34/image.png)](https://postimg.cc/xJrdkMZB)
