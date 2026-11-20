# FreeBSD 安装、权限和 FD 排障指南

本指南面向 FreeBSD `amd64`（x86-64）和 `arm64` 用户。telEgo 的 FreeBSD Release 是原生 FreeBSD 可执行文件，使用 kqueue，不是 Linux ELF，也不需要 Linux 兼容层。

## 1. 选择 Release

1. 用 `uname -m` 查看架构：FreeBSD `amd64` 对应 `freebsd_amd64`，`aarch64` 对应 `freebsd_arm64`。
2. 访问 [kmingee/telego Releases](https://github.com/kmingee/telego/releases)，选择已经包含 FreeBSD 附件的版本。
3. 下载 `telego_<版本>_freebsd_amd64.tar.gz` 或 `telego_<版本>_freebsd_arm64.tar.gz`，并下载同一版本的 `checksums.txt`。
4. 根据 `checksums.txt` 中对应文件的 SHA-256 与本机 `sha256 <文件>.tar.gz` 的结果核对，确认无误后再解压。

旧版本不会因为 CI 添加了 FreeBSD 构建而自动生成新附件；必须发布包含该修改的新 tag。如果 Releases 页面暂时没有新文件，可按下面方法从源码构建。

```sh
tar -xzf telego_<版本>_freebsd_amd64.tar.gz
./telego --help
```

其中 `<版本>` 是占位符，请替换为实际文件名中的版本号。`arm64` 用户也需替换文件名的架构部分。

## 2. 安装和配置文件权限

在解压目录下安装程序和示例配置：

```sh
install -m 0555 telego /usr/local/bin/telego
mkdir -p /usr/local/etc/telego
install -m 0600 config.example.toml /usr/local/etc/telego/config.toml
```

编辑 `/usr/local/etc/telego/config.toml`，替换 `[secrets]` 中的示例密钥，修改掩护站点，并设置监听地址。程序可通过 `telego generate <掩护主机名>` 生成新密钥。

示例先用非特权端口进行验证：

```toml
[general]
bind-to = "0.0.0.0:9443"
max-connections = 0

[secrets]
alice = "0123456789abcdef0123456789abcdef"

[tls-fronting]
mask-host = "www.google.com"
```

安装时确保配置文件**所有者是运行 telEgo 的用户**，权限为 `0600`。例如，若程序由专用 `telego` 用户启动，则应由管理员将文件所有权交给该用户。不要为了让程序能读文件而将权限改成 `0644`。

如果使用 Unix Domain Socket，其父目录也必须允许运行用户写入。示例 `/var/run/telego/telego.sock` 需要管理员提前创建合适的专用目录；不要把私有 socket 放到无保护的公共可写目录。

## 3. 监听端口和服务启动

FreeBSD 没有 Linux 的 systemd，因此仓库的 `make install`、`systemctl` 和 `.service` 文件不适用。建议先在前台运行，确认配置与网络连接均正常，再接入你自己的 FreeBSD `rc.d` 服务管理：

```sh
telego doctor -c /usr/local/etc/telego/config.toml
telego run -c /usr/local/etc/telego/config.toml -l
```

低于 1024 的端口（如 TCP 443）通常需要特权账号进行绑定。推荐使用单独的非特权服务用户、监听高端口（如 `9443`），并根据网络环境配置安全的端口转发或前置反向代理；不要仅因为要监听 443 就长期以 root 运行所有功能。

停止时向主进程发送 `SIGTERM` 或 `SIGINT`，预留至少 30 秒完成优雅退出。配置热更新可发送 `SIGHUP`，但 WEB/ME 的部分改动仍需重启。

## 4. FD 限额与兼容性修复

FreeBSD 与 Linux 都提供 `RLIMIT_NOFILE`。从包含本修复的版本开始：

- `general.max-connections = 0` 会根据 FreeBSD 进程的 FD 软限制推导最大连接数，不会再因为识别不到限额而变成无上限。
- `middle-end` 启动时可以读取 FreeBSD 的 FD 软/硬限制并输出容量提示。
- gnet 在多个监听地址部分创建失败，或监听 socket 已绑定但 keepalive 配置失败时，会关闭已打开的监听 FD。此类错误不会遗留占用端口的描述符。

可用以下命令排查：

```sh
ulimit -n             # 当前 shell 的 FD 软限制
procstat -f <PID>     # 列出指定运行中进程打开的 FD
```

`<PID>` 应替换为 telEgo 进程的实际 PID。请在服务管理器的启动上下文中设置 FD 限额，而不是只修改个人 shell 的 `ulimit`。FreeBSD 的登录类、用户和服务启动方式都可能影响最终限制。

每条直连连接往往需要两个 FD，此外还有监听端口、ME 链路、配置和指标等 FD；提高 `general.max-connections` 之前，应先核对实际 `RLIMIT_NOFILE`，并为其他用途保留足够余量。如果日志提示 `EMFILE`、`ENFILE`，先减少并发或提高系统允许的 FD 上限，而不是无限重试。

## 5. 源码构建与运行验证

在 FreeBSD 上安装 Go 1.27.4 或更新版本后：

```sh
git clone https://github.com/kmingee/telego.git
cd telego
make build
make test-gnet
```

`make test-gnet` 会运行此仓库的 gnet 补丁回归测试（包括监听 FD 清理路径），并不是完整上游 gnet 测试集。生产构建使用 `poll_opt,gc_opt`，分别对应优化的 kqueue poller 和连接矩阵。

在 Linux 上可交叉编译 FreeBSD 版本：

```sh
CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 \
  go build -trimpath -tags=poll_opt,gc_opt -o telego-freebsd-amd64 ./cmd/telego
```

交叉编译只能验证类型、构建标签和链接，不会运行 kqueue。正式部署时应在 FreeBSD 实机或虚拟机中额外验证：

1. `telego --help`、`telego doctor -c ...` 可以执行；
2. 监听 TCP 端口并与 Telegram 客户端建立连接；
3. 发送 `SIGTERM` 后监听端口释放且进程正常退出；
4. 反复启动、退出后，`procstat -f` 中的 FD 数不持续增加；
5. 绑定端口被占用或 keepalive 参数不合法时，不遗留占用的 socket。

## 6. 升级与回滚

保留当前工作的二进制和配置备份。升级前核对新包的架构与 SHA-256，停止服务，替换 `/usr/local/bin/telego`，再启动并检查日志及客户端连接。

如遇兼容问题，停止新进程，恢复旧版本二进制并重新启动。请勿在尚未核对新配置格式时覆盖唯一的 `config.toml`，也不应通过放宽密钥文件权限来尝试解决启动故障。

其他设置见 [中文配置参考](configuration.md) 和 [简体中文 README](../../README.zh-CN.md)。
