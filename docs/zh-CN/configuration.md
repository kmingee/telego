# 配置参考（简体中文）

telEgo 使用 TOML 格式。完整默认值和所有可用选项见 [config.example.toml](../../config.example.toml)。未知键、负数连接限制以及部分不合法的端口/时间设置会使启动失败，便于尽早发现拼写错误。

## 基础配置

```toml
[general]
bind-to = "0.0.0.0:9443"   # 示例采用非特权端口
log-level = "info"
max-connections = 0       # Linux/FreeBSD 根据 FD 上限自动计算
max-connections-per-ip = 100
handshake-timeout = "5s"

[secrets]
alice = "0123456789abcdef0123456789abcdef"

[tls-fronting]
mask-host = "www.google.com"
```

必须将密钥替换为 `telego generate www.google.com` 生成的值。`secrets` 中每个用户对应一个 16 字节的密钥，以 32 位十六进制字符表示。

| 字段 | 含义 |
| --- | --- |
| `general.bind-to` | 监听的 IP/端口，也可为 `unix:///路径/文件.sock` |
| `general.log-level` | `trace`、`debug`、`info`、`warn` 或 `error` |
| `general.max-connections` | 进程允许的最大前端连接数，`0` 为按 FD 限额自动计算 |
| `general.max-connections-per-ip` | 单 IP 最大连接数，`0` 为不限制 |
| `general.max-ips-per-user` | 单用户可使用的 IP 个数，`0` 为不限制 |
| `general.handshake-timeout` | 握手超时，默认 5 秒；允许 0（默认值）或 1 秒至 1 分钟 |
| `tls-fronting.mask-host` | FakeTLS 的 SNI 和掩护站点主机名 |
| `tls-fronting.splice-host` | 识别失败的客户端流量所转发到的主机，默认与 `mask-host` 相同 |
| `tls-fronting.mask-sni-safelist` | 允许跟随 SNI 转发的额外域名白名单；默认空 |

## 文件描述符和并发上限

每条直连客户端链路通常需要两个 FD（客户端和 Telegram DC）。在 Linux 和 FreeBSD 上，`general.max-connections = 0` 使用进程 `RLIMIT_NOFILE` 的软限制计算：

```text
max_connections = (soft_limit - min(256, soft_limit / 2)) / 2
```

例如软限制为 1024 时，自动上限约为 384 条。仍需给 ME、监听端口、日志、Web 和指标保留余量；不要直接把上限设为 FD 软限制。`[middle-end].max-connections` 可进一步降低上限，但不会提高全局上限。

在 FreeBSD 上可以执行 `ulimit -n` 查询当前 shell 的 FD 软限制。实际由服务管理器启动时，须在**服务进程的启动环境**中设置限额；从另一个终端修改 `ulimit` 不会影响正在运行的进程。

## 反向代理和来源地址

仅在确实有 HAProxy/Nginx 发送 PROXY 协议头时启用 `general.proxy-protocol`。当监听 TCP 时，务必将 `general.trusted-proxy-cidrs` 限制在受信任的反向代理地址，例如：

```toml
[general]
proxy-protocol = true
trusted-proxy-cidrs = ["127.0.0.1/32"]
```

否则直接连接者可能伪造客户端 IP。`[web-proxy].trusted-proxy-cidrs` 控制 WEB 侧对 `X-Forwarded-For` 的信任，是另一个独立配置项。

## Middle-End

ME 默认启用，适用于 Telegram 上游链路；若启动阶段尚未准备好，会暂时使用直连：

```toml
[middle-end]
enabled = true
# proxy-tag = "<Telegram 发放的 32 位十六进制标签>"
# max-connections = 0  # 0 表示默认最多 10000，仍受 general 限制
```

如果不需要 ME，可设置 `enabled = false`。SOCKS5 的全局配置是 `[upstream].socks5`，ME 可以使用 `[middle-end].socks5` 单独覆盖。ME 参数修改通常需要重启。

## WEB 代理

WEB 默认关闭，启用时需要前置 Nginx 与真实 TLS 证书：

```toml
[web-proxy]
enabled = true
carrier = "websocket"
hostname = "proxy.example.com"
bind-to = "127.0.0.1:8080"
trusted-proxy-cidrs = ["127.0.0.1/32"]
```

`websocket` 使用单个多路复用载体；`websocket-lanes` 按 stream 建立 WebSocket；`https` 和 `https-lanes` 使用 fetch/长轮询。WebSocket 需要 Nginx 转发 `Upgrade` 和 `Connection` 请求头；所有请求均应转发给 WEB 后端。详见 [英文 WEB 部署示例](../../examples/web-proxy/README.md)。

## 监控和诊断

```toml
[metrics]
bind-to = "127.0.0.1:9090"
path = "/metrics"
diagnostics = false
```

不要直接将调试和监控端口暴露到公网。常用命令：

```sh
telego doctor -c config.toml
telego run -c config.toml -l
```

配置文件应由运行账号持有且设置 `chmod 600 config.toml`。不同配置项是否能通过 `SIGHUP` 热更新有所不同，WEB/ME 更改应安排完整重启。
