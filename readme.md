# vnet-v2

> 使用链路绑定的多链路vnet升级版本，使用上与v1版本一致

golang编写的 **虚拟组网工具**

> 由一个部署在公共网的 route 端和部署在私网的多个 vnet 端组成
>
> 流量可以由一个 vnet 端转发至另一个 vnet 端，以达成虚拟组网的目的
>
> 使用注册服务路由的方式来进行流量流转

**route端[组件]**

> 转发vnet的流量
>
> 维护一些配置项

**vnet端[组件]**

> 通过配置可以将一些本地服务注册到route端
>
> 可查询route端维护的所有服务信息，并开启某一个服务在本地的映射
>
> 本地应用可访问本vnet端开启的这个映射服务，来达成访问远端vnet注册的远端服务的目的

## 使用

### route端

route端部署在有公网ip[或公共网域]的机器上

**配置文件**

config.ini

```ini
[route] ; route端的配置，是route端的监听设置，也是vnet端连接route端的设置
ip=127.0.0.1 ; ip地址，必填项
port=7749 ; 端口号，必填项
```

>ip 为必填项，route端监听的ip，如果是 127.0.0.1，则只能本机器访问，如果是 0.0.0.0 则外网可访问
>
>port 为必填项，route端监听的端口，默认为 7749

**运行route**

```sh
./route
```

> 默认读取运行环境目录下的 config.ini

或指定配置文件

```
./route config.ini
```

> 配置文件可更改路径及文件名

### vnet端

vnet端部署在需要虚拟组网的私网机器中

**配置文件**

config.ini

```ini
[route] ; route端的配置，是route端的监听设置，也是vnet端连接route端的设置
ip=127.0.0.1 ; ip地址，必填项
port=7749 ; 端口号，必填项

[vnet] ; vnet端的配置
name=xiao ; vnet名称
group=default ; 分组，只能共享同分组连接组网 ；不写则默认 default
terminal=true ; 是否开启命令行交互 ; 不开启则完全以配置文件为根据提供服务，startserve和startuse配置不生效 ; 开启则可在命令行界面进行额外配置，且startserve和startuse配置会生效 ; 不写则默认 false
startserve=false ; 是否启动时自动开启serve服务 ; 不写则默认 true
startuse=false ; 是否启动时自动映射use服务 ; 不写则默认 true
allow_unserve=false ; 是否允许被访问本vnet端未注册的服务 ; 不写则默认 false
unserve_remote=false ; 是否允许被访问本vnet端非localhost地址的未注册服务 ; 不写则默认 false
unserve_num=1 ; 允许被访问未注册服务总数量 ; 不写则默认 1

[serve.xrdp] ; 远程服务配置，serve字为标志字，后面的字符为服务名称，在配置文件中不可重复
serveid=999 ; 服务唯一标识，取值范围[1,499]，其中[500,999]为静态标识，服务端不会更改，其他为动态标识，如果冲突，会被服务端动态修改 ; 可不写此项，默认000，在启动后，服务端自动分配
type=tcp ; 服务协议，vnet只支持tcp传输协议，基于tcp的其他协议也可借tcp链路传输 ; 可不写此项
ip=127.0.0.1 ; ip地址 ; 可不写此项，默认127.0.0.1
port=3389 ; 端口号，必填项
name=xrdp ; 服务名称，name配置项不存在的话，以serve标志字后的服务名称为name配置名称 ; 可不写此项
info=Microsoft remote desktop protocol ; 服务补充信息 ; 可不写此项
active=false ; 是否激活，其作用在于是否在vnet端启动时被自动化处理，如果不激活，则需在命令行手动处理 ; 可不写此项，默认激活

[use.xrdp] ; 映射服务配置，use字为标志字，后面的字符为服务名称，在配置文件中不可重复，use服务名称无实际意义，仅作区分其他use配置意义
serveid=999 ; 服务唯一标识，必填项，取值范围[1,999]，是识别和获取远程服务的唯一根据，获取到则开启映射，为获取到则待命 ; 不写或写000则不会开启此项配置
ip=127.0.0.1 ; ip地址 ; 可不写此项，默认127.0.0.1
port=33389 ; 端口号，映射服务监听端口 ; 可不写此项，默认使用原端口
active=false ; 是否激活，其作用在于是否在vnet端启动时被自动化处理，如果不激活，则需在命令行手动处理 ; 可不写此项，默认激活
```

>[route] 为需要连接的route端的地址
>
>> ip 为必填项，route端监听的ip，vnet端访问route端的ip地址
>>
>> port 为必填项，route端监听的端口，默认为 7749，vnet端访问route端的port端口
>
>[vnet] 为本端vnet的一些配置：
>
>> vnet名称：
>>
>> - name=xiao：提供简单的名称以帮助分辨，主要在需要辨别服务来源和请求未注册服务的场景使用
>>
>> vnet分组：
>>
>> - group=default：使用分组名称来区别，同时区分开不同的分组，仅可见同分组的服务和vnet终端
>>
>> vnet终端形态：
>>
>> - terminal=false：简单形态，仅使用配置文件中的配置，不提供其他终端命令操作
>> - terminal=true：命令行形态，相比简单形态，额外提供其他终端命令进行操作
>>
>> 访问未注册服务：
>>
>> - allow_unserve=false：不允许访问未注册服务
>> - allow_unserve=true：允许访问未注册服务，收到请求后先尝试建立无serveid服务，然后向route端申请和注册这个服务，如果成功则向请求方返回这个serveid，否则返回空serveid
>>
>> 允许未注册服务为远端：
>>
>> - unserve_remote=true：允许未注册服务ip为非127.0.0.1的unserve执行
>> - unserve_remote=false：仅允许未注册服务ip为127.0.0.1的unserve执行
>>
>> 允许访问多少个未注册服务：
>>
>> - unserve_num=1：由提供方计数，超过的一律响应为准备失败
>
>[serve.] 本端vnet向route端注册的服务
>
>> 注意：[serve.] 使用时因为中括号内的字符不能重复，所以需要在小数点后面增加一些字符，充当这个serve的名称，如 [serve.xrdp]、[serve.ssh]
>>
>> serveid=999 ; 服务唯一标识，取值范围[1,499]，其中[500,999]为静态标识，服务端不会更改，其他为动态标识，如果冲突，会被服务端动态修改 ; 可不写此项，默认000，在启动后，服务端自动分配
>>
>> type=tcp ; 服务协议，vnet只支持tcp传输协议，基于tcp的其他协议也可借tcp链路传输 ; 可不写此项
>>
>> ip=127.0.0.1 ; ip地址 ; 可不写此项，默认127.0.0.1
>>
>> port=3389 ; 端口号，必填项
>>
>> name=xrdp ; 服务名称，name配置项不存在的话，以serve标志字后的服务名称为name配置名称 ; 可不写此项
>>
>> info=Microsoft remote desktop protocol ; 服务补充信息 ; 可不写此项
>>
>> active=false ; 是否激活，其作用在于是否在vnet端启动时被自动化处理，如果不激活，则需在命令行手动处理 ; 可不写此项，默认激活
>
>[use.] 本端vnet将使用的映射服务
>
>> 注意：[use.] 使用时因为中括号内的字符不能重复，所以需要在小数点后面增加一些字符，充当这个use的名称，如 [use.xrdp]、[use.ssh]
>>
>> serveid=999 ; 服务唯一标识，必填项，取值范围[1,999]，是识别和获取远程服务的唯一根据，获取到则开启映射，未获取到则待命 ; 不写或写000则不会开启此项配置
>>
>> ip=127.0.0.1 ; ip地址 ; 可不写此项，默认127.0.0.1
>>
>> port=33389 ; 端口号，映射服务监听端口 ; 可不写此项，默认使用原端口
>>
>> active=false ; 是否激活，其作用在于是否在vnet端启动时被自动化处理，如果不激活，则需在命令行手动处理 ; 可不写此项，默认激活
>
>注意：serve和use中的serveid最大只能是3位字符，为数字的0~999，vnet默认将500以上的serveid当做静态处理，使用期间不会被更改，而500以下的serveid在运行时遇到冲突，则可能会被更改和重新分配

**运行vnet**

```sh
./vnet
```

> 默认读取运行环境目录下的 config.ini

或指定配置文件

```
./vnet config.ini
```

> 配置文件可更改路径及文件名



## 开机启动

建议使用vnet.sh或vnet.cmd来使用

linux vnet.sh

```sh
cd /home/xiao/my/vnet
./vnet config.ini
```

windows vnet.cmd

```
cd /d D:\my\vnet
vnet.exe config.ini
```

自启动：

linux可选择新建service来自启动

> 自行查询service文件新建和使用方法

windows可选择将vnet.cmd的快捷方式放到启动文件夹

> win+R，输入shell:startup回车，就是启动文件夹



## 使用场景示例

> 在办公室连接家中的主机：使用远端ssh服务，以及临时使用远程jupyter工具

- 处于家中的主机A开启ssh服务

- A运行vnet终端，配置文件为

```ini
[route]
ip=10.12.140.180
port=7749

[vnet]
name=remote-ssh
group=default-ssh
terminal=false
startserve=true
startuse=false
allow_unserve=true
unserve_remote=false
unserve_num=3

[serve.ssh]
serveid=700
type=tcp
ip=127.0.0.1
port=22
name=ssh
info=home-ssh
active=true
```

- ip 为10.12.140.180[胡编的]的公网服务器运行route端，配置文件为

```ini
[route]
ip=0.0.0.0
port=7749
```

- 位于公司办公室的主机B运行vnet终端，配置文件为

```ini
[route]
ip=10.12.140.180
port=7749

[vnet]
name=local-ssh
group=default-ssh
terminal=true
startserve=false
startuse=true
allow_unserve=false
unserve_remote=false
unserve_num=3

[use.ssh]
serveid=700
ip=127.0.0.1
port=1122
active=true
```

- 在主机B中连接ssh服务

```sh
ssh user@127.0.0.1 -p 1122
```

- 在主机B中还想访问主机A中的jupyter服务，但是[show serves]发现主机A没有注册这个服务

首先，远程ssh中打开jupyter服务

```sh
jupyter notebook
```

然后，vnet终端中使用unserve

```sh
>: show serves

请求route的服务列表 ...
route服务列表:
serveid vnet    name    type    ip      port    info
700     01      ssh     tcp     127.0.0.1       22    home-ssh
-  1 个vnet终端, 共注册 1 个服务 -

>: unserve

    to vnetid: 01
    ip(127.0.0.1):
    port: 8888
unservemap_local [[01 map[127.0.0.1:8888:000]]]
已向 vnetid: 01 的vnet终端请求其未注册的 127.0.0.1:8888 服务，等待回应 ...
注意：unserve服务保质期为 3min ...
成功获取到 vnetid: 01 的vnet终端的服务 - 已注册serveid: 001 继续开启 ...

    ip(127.0.0.1):
    port: 18888
newunserveuse {001 127.0.0.1 18888 true}

>:
```

- 此时浏览器中访问localhost:18888，则可以成功打开jupyter

**其他场景举例**

> 目前我使用过的场景：

- windows的远程桌面
- code-serve 这个是微软vscode的web版，部署在wsl中，超好用
- filemanager 这个是简单的前端文件管理工具，可以查看，下载，上传，删除文件，在线视频就是用这个测试的
- 当然还有众多的内网web服务
- **多端虚拟组网**：通过多vnet端部署，可让这些客户机组建一张虚拟局域网
- **内网穿透**：可以在公网服务器上同时部署一个route端和一个vnet端来实现经典的内网穿透
- **合并远程局域网**：在互不可访问的多个局域网中部署vnet，可使多个不同局域网中的服务互通，达到局域网合并效果



## vnet终端交互

配置文件中[vnet]terminal=true开启本vnet端的终端交互

- help 

> reload  重新加载配置，可指定配置文件
> serve   本地服务操作，创建、执行、开始、暂停、删除等
> use     映射服务操作，创建、执行、开始、暂停、删除等
> show    显示信息，本端vnet、本端serve、本端use、route端serves、route端vnets等
> ping    测试与route端的回传延迟
> unserve 请求非注册服务
> exit    退出本端vnet

- reload

> reload不指定配置文件的话，则默认读取运行环境目录下的config.ini
>
> 不会全部关闭现有的所有serve或use，只会对变动项做出反应

- serve

> 注册服务配置，使用 serve open|close|new|run|delete
>
> open 则打开一个active=false的serve，并注册
>
> close 则关闭一个注册的serve，并active=false
>
> new 新建一个serve，并active=false
>
> run 新建一个serve，并active=true，并注册
>
> delete 删除一个serve，并注销

- use

> 映射服务配置，使用 use open|close|new|run|delete
>
> open 则打开一个active=false的use，并监听
>
> close 则关闭一个监听的use，并active=false
>
> new 新建一个use，并active=false
>
> run 新建一个use，并active=true，并监听
>
> delete 删除一个use，并解除监听

- show

> 查询route端维护的相关信息和展示本vnet端的相关信息
>
> 使用 show serve|use|serves|vnet|vnets
>
> serve 展示本vnet端的所有serve配置
>
> use 展示本vnet端的所有use配置
>
> serves 查询route端维护的所有serve配置
>
> vnet 展示本vnet端的相关配置
>
> vnets 查询route端维护的所有vnet配置

- ping

> 向route端发送心跳，并记录回传时间
>
> 注意，如果vnet忙，则这个时间包含消息流转时的排队[阻塞]时间

- unserve

> 请求对方vnet端的未注册服务，需要对方vnet端配置中[vnet]allow_unserve=true
>
> 会要求输入对方vnetid，ip，port
>
> ip不输入则默认 127.0.0.1，vnetid必须是已和route端连接的vnet终端
>
> 如果对方vnet端开启unserve成功，则本端vnet还会继续要求输入开启映射服务的ip和port
>
> 如果开启成功，则对方vnet会保留这个unserve 3分钟时间，未使用则自动注销
>
> 被使用后，当所有连接被关闭，则这个unserve会在1分钟后自动注销
>
> 注意：因为延迟和消息阻塞的因素，提示失败时也可能是开启延时，可以稍后使用 [show serves]查询

- exit

> 关闭本端vnet

## 使用的其他外部库

- gopkg.in/ini.v1  用来读取ini配置文件

## 升级

v2版本相比v1版本更新了流量转发链路方式，相比与v1版本使用单链路路由分发，v2版本将每个工作链路单独区别，避免了终端一个链路流量拥堵而导致这个终端全链路被迫等待的问题，而且v1版本每帧流量头部用于识别的固定操作码也省去了，对于route端来说，管理众多vnet端的链路建立和销毁会更加简单。

## 注意事项

- 所有连接使用tcp链路，因此vnet只支持tcp相关流量的转发，例如http协议也可，udp类型则不行。
- vnet有分组规则，即使是连接同一个route端，但服务仅同组vnet终端可见。只需要设置同一个group名就能处于同一个分组。
- vnet端和route端设计支持流量加密，但没有实现，有需求可在代码中的注释加密方法处自行实现
- 目前vnet没有密码 。。。，但可使用复杂分组名称简单识别



## 后续

没有后续了

v2版本已实现，使用链接绑定来传递流量简明优雅，也避免了v1版本单链路可能导致的流量堵塞问题(当然不看在线视频的话v1版本还是很稳定的)

route端依旧尽量简单，使其能很好的在配置简陋的服务端运行，将更多的能力使用在转发流量上

最后

用另外的一个portsmap工具，一个filemanager工具，来搭配vnet-v2使用，可以使vnet-v2虚拟融合多个远程局域网的功能更完善哦。。。
