<p align="center">
  <img src=".github/logo.jpg" alt="telEgo 标志" width="200">
</p>

<h1 align="center">telEgo</h1>

<p align="center">使用 Go 编写的高性能 Telegram MTProxy，支持 FakeTLS、Telegram Middle-End 和原生 WEB 协议。</p>

<p align="center">
  <a href="README.md">English</a> | <a href="README.ru.md">Русский</a> | <b>简体中文</b>
</p>

> telEgo 通过模拟 TLS 流量提供 MTProxy 接入，也可将未识别的连接转发到伪装站点。请确保部署和使用符合所在地法规。

## 文档导航

- [快速开始](#快速开始)
- [下载安装](#下载安装)
- [配置说明](#配置说明)
- [FreeBSD 使用指南](docs/zh-CN/freebsd.md)
- [完整中文配置参考](docs/zh-CN/configuration.md)
- [中文文档目录](docs/zh-CN/README.md)
- [English README（全部功能与性能说明）](README.md)

## 功能简介

- **两种 MTProxy 协议**：自动识别 FakeTLS（`ee`）及原生 Obfuscated2（`dd`）。
- **TLS 伪装**：根据掩护站点的 TLS 特征构造响应；探测连接可转发到掩护站点。
- **Middle-End（ME）**：通过持续维护的 Telegram ME 连接池转发流量；ME 未就绪时自动使用直连路径。
- **原生 WEB 代理**：可选 HTTPS 或 WebSocket 载体，配合 Nginx 的真实 TLS 证书用于 Telegram Desktop。
- **连接防护**：支持握手超时、按 IP 限流、每用户 IP 数限制、重放防护与优雅退出。
- **可观测性**：结构化日志、Prometheus 指标、`doctor` TLS 检查和配置热加载。
- **平台**：Linux `amd64`、`arm64`、`arm/v7`；FreeBSD `amd64`、`arm64`。发布使用纯 Go 静态构建（`CGO_ENABLED=0`），无需 Linux 运行环境。

## 快速开始

1. 安装 `telego`：参见下文的 [下载安装](#下载安装)。
2. 生成 16 字节密钥以及 Telegram 代理链接：

   ```sh
   telego generate www.google.com
   ```

3. 创建 `config.toml`（将示例密钥替换为自己生成的值）：

   ```toml
   [general]
   bind-to = "0.0.0.0:443"
   log-level = "info"
   max-connections = 0

   [secrets]
   alice = "0123456789abcdef0123456789abcdef"

   [tls-fronting]
   mask-host = "www.google.com"
   ```

   `[tls-fronting].mask-host` 应与生成 FakeTLS 密钥时使用的掩护主机名一致。`443` 属于特权端口，FreeBSD 上使用非特权用户时可改用 `9443`。

4. 限制配置文件读取权限并运行：

   ```sh
   chmod 600 config.toml
   telego doctor -c config.toml
   telego run -c config.toml -l
   ```

   `-l` 会输出可用的 Telegram 代理链接。请妥善保管配置文件和密钥，不要将真实密钥提交到 Git。

## 下载安装

### 预编译 Release

在 [kmingee/telego Releases](https://github.com/kmingee/telego/releases) 中选择对应的压缩包：

| 平台 | 架构 | Release 文件格式 |
| --- | --- | --- |
| Linux | x86-64 | `telego_<版本>_linux_amd64.tar.gz` |
| Linux | ARM64 | `telego_<版本>_linux_arm64.tar.gz` |
| Linux | ARMv7 | `telego_<版本>_linux_armv7.tar.gz` |
| FreeBSD | x86-64 | `telego_<版本>_freebsd_amd64.tar.gz` |
| FreeBSD | ARM64 | `telego_<版本>_freebsd_arm64.tar.gz` |

压缩包内包含可执行文件、`LICENSE`、英文及简体中文 README，以及 `config.example.toml`。同一个 Release 提供 `checksums.txt`，可供下载后核对 SHA-256。首次包含 FreeBSD 文件的版本需要在合并相关构建改动后发布新 tag；旧 Release 不会自动补齐。

FreeBSD 手动安装示例（在已解压的目录执行）：

```sh
install -m 0555 telego /usr/local/bin/telego
mkdir -p /usr/local/etc/telego
install -m 0600 config.example.toml /usr/local/etc/telego/config.toml
```

配置完成后，可先以非特权端口和前台方式试运行。FreeBSD 不使用 systemd，请不要执行 Linux 的 `make install` 或 `systemctl` 示例。详见 [FreeBSD 指南](docs/zh-CN/freebsd.md)。

### 从源码构建

要求 Go 1.27.4 或更高版本，必须克隆整个仓库（包括本地修改的 `third_party/gnet`）：

```sh
git clone https://github.com/kmingee/telego.git
cd telego
make build
```

由于仓库使用了本地 `replace` 依赖，不能使用 `go install github.com/scratch-net/telego/cmd/telego@version` 单独安装。需要 FreeBSD 交叉编译时，在 Linux 开发机上执行：

```sh
CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 \
  go build -trimpath -tags=poll_opt,gc_opt -o telego-freebsd-amd64 ./cmd/telego
```

该命令仅验证能否生成目标平台二进制；完整的 kqueue/权限行为还需在真实 FreeBSD 主机上测试。

## 配置说明

参阅 [中文配置参考](docs/zh-CN/configuration.md) 或原始 [config.example.toml](config.example.toml)。

| 配置项 | 作用 | 注意事项 |
| --- | --- | --- |
| `general.bind-to` | 监听地址/端口，也支持 Unix Socket | 端口 443 可能需要特权权限 |
| `general.max-connections` | 全局连接上限 | `0` 按 Linux/FreeBSD `RLIMIT_NOFILE` 自动计算 |
| `general.max-connections-per-ip` | 单 IP 连接上限 | `0` 为不限制 |
| `general.trusted-proxy-cidrs` | 可信 PROXY 协议来源 | 使用反向代理时应明确设置 |
| `secrets` | 命名密钥 | 每个值为 32 位十六进制字符 |
| `tls-fronting.mask-host` | TLS 掩护主机名 | 与客户端的 FakeTLS 配置保持一致 |
| `middle-end.enabled` | 启用 ME 连接池 | 默认启用，未就绪时可直连 |
| `web-proxy.enabled` | 原生 WEB 代理 | 默认关闭，生产环境需独立配置 Nginx |
| `metrics.bind-to` | Prometheus 指标监听 | 建议仅绑定回环地址 |

### Telegram Middle-End

ME 默认开启，后台维护上游连接，未准备好时回退到 Telegram DC 直连。`[middle-end].max-connections` 默认值 `0` 表示最多 10,000 个连接，但仍受 `general.max-connections` 的较低值限制。查看 ME 的日志和 `telego_middleend_*` 指标可排查连接情况。

### 原生 WEB 代理

WEB 功能默认关闭。启用后，可通过 Nginx 的真实 TLS 证书与 Telegram Desktop 交互。`carrier` 支持 `https`、`https-lanes`、`websocket`、`websocket-lanes`；WebSocket 模式必须正确转发 HTTP/1.1 升级头。

现有的 Linux/Docker 网关自动安装脚本针对 Nginx/Docker 部署，并非原生 FreeBSD 安装脚本。详细参数参见 [英文 WEB 部署示例](examples/web-proxy/README.md) 与 [英文网关指南](examples/gateway/README.md)。

## 安全与运维

- 配置文件建议为 `0600`，将运行账号设为文件所有者；不要向其他用户暴露密钥。
- 未使用反向代理时不应启用 PROXY 协议；启用时务必配置可信源地址。
- 修改配置后可向进程发送 `SIGHUP`；部分功能（如 WEB 和 ME 配置）需要完整重启。
- `SIGTERM`/`SIGINT` 会触发最多约 25 秒的优雅退出；监督进程建议预留至少 30 秒。
- 连接数接近文件描述符上限时请同时查看 `ulimit -n`、实际 FD 占用与 `general.max-connections`，不能只提高应用级连接数。

## 开发与贡献

```sh
make test-gnet   # 本地 gnet 补丁回归测试
make test        # Linux 上完整测试入口
```

CI 还会交叉编译 FreeBSD `amd64` / `arm64`，并编译与 FD 限额相关的测试程序。交叉编译不等于原生运行测试。Release 发布由 `v*` tag 触发，且需要 `docs/releases/<tag>.md` 对应的更新说明。

许可证：[Apache-2.0](LICENSE)。
