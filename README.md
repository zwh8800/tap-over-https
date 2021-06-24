# tap-over-https

<a href="https://sm.ms/image/yx9KtBOobZehzH1" target="_blank"><img align="left" width="100" hspace="5" vspace="5" src="https://i.loli.net/2021/06/24/yx9KtBOobZehzH1.png" ></a>通过websocket协议建立vpn链接，服务端可以部署在openwrt等linux软路由，客户端目前支持Windows、mac、linux（有GUI版本），方便在公司等外部环境访问家庭内网。

<br><br><br><br><br>

## 截图
<a href="https://sm.ms/image/OfpDhonMQjiBHJG" target="_blank"><img width="200" src="https://i.loli.net/2021/06/24/OfpDhonMQjiBHJG.jpg" ></a>
<a href="https://sm.ms/image/wB8mCbJjTzhauyW" target="_blank"><img width="200" src="https://i.loli.net/2021/06/24/wB8mCbJjTzhauyW.jpg" ></a>

## 使用方法

#### 编译/安装

方式1: 通过源码安装

```bash
# 命令行版本
go install github.com/zwh8800/tap-over-https/cmd/tap-over-https@latest

# GUI版本
go install github.com/zwh8800/tap-over-https/cmd/tap-over-https-gui@latest
```

方式2: 下载预编译包

https://github.com/zwh8800/tap-over-https/releases


#### 启动服务端

```bash
tap-over-https -s -addr :8012 -i br-lan
```
-s 参数代表以服务器模式启动

-addr 指定websocket绑定端口

-i 后跟一个linux桥接网卡，客户端连接过来后会被桥接到这个网卡上

-ip-start 后跟一个ip地址，代表分配给客户端的起始ip

-ip-end 后跟一个ip地址，代表分配给客户端的终止ip

#### 启动客户端
```bash
tap-over-https -addr ws://www.baidu.com/vpn
```

-addr 指定服务端的地址，需要以ws://或wss://为开头

#### macOS需先安装tap网卡驱动
下载地址：https://sourceforge.net/p/tuntaposx

#### windows需先安装tap网卡驱动
下载地址：http://build.openvpn.net/downloads/releases/

win7需下载老版本：https://build.openvpn.net/downloads/releases/tap-windows-9.9.2_3.exe

## 安全性
本身不具备安全性，websocket协议全是明文的，为了安全性使用时可以前面加一个nginx/caddy，配置上https，再加上个http basic auth，能比明文裸奔强一些吧（大概
