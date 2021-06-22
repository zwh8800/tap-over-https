# tap-over-https

通过websocket协议建立vpn链接，服务端可以部署在openwrt等linux软路由，客户端目前支持mac、linux，方便在公司等外部环境访问家庭内网。

## 使用方法

编译/安装

```bash
go install github.com/zwh8800/tap-over-https@latest
```

启动服务端

```bash
tap-over-https -s -addr :8012 -i br-lan
```
-s 参数代表以服务器模式启动
-addr 指定websocket绑定端口
-i 后跟一个linux桥接网卡，客户端连接过来后会被桥接到这个网卡上

启动客户端
```bash
tap-over-https -addr ws://www.baidu.com/vpn
```

-addr 指定服务端的地址，需要以ws://或wss://为开头

macOS需先安装tap网卡驱动
下载地址：https://sourceforge.net/p/tuntaposx
