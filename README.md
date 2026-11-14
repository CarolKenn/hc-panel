[[ 简体中文 ]](https://sun-panel-doc.enianteam.com/zh_cn/introduce/project.html) |
[[ English ]](https://sun-panel-doc.enianteam.com/introduce/project.html)

<div align=center>

<img src="./doc/images/logo.png" width="100" height="100" />

# Sun-Panel

[![Github](https://img.shields.io/badge/Github-123456?logo=github&labelColor=242424)](https://github.com/hslr-s/sun-panel)
[![Gitee](https://img.shields.io/badge/Gitee-123456?logo=gitee&labelColor=c71d23)](https://gitee.com/hslr/sun-panel)
[![docker](https://img.shields.io/badge/docker-123456?logo=docker&logoColor=fff&labelColor=1c7aed)](https://hub.docker.com/r/hslr/sun-panel) 
[![Bilibili](https://img.shields.io/badge/Bilibili-123456?logo=bilibili&logoColor=fff&labelColor=fb7299)](https://space.bilibili.com/27407696/channel/collectiondetail?sid=2023810)
[![YouTube](https://img.shields.io/badge/YouTube-123456?logo=youtube&labelColor=ff0000)](https://www.youtube.com/channel/UCKwbFmKU25R602z6P2fgPYg)
<br>
[![GitHub User's stars](https://img.shields.io/github/stars/hslr-s%2Fsun-panel?style=flat&logo=github)](https://github.com/hslr-s/sun-panel)
[![github downloads](https://img.shields.io/github/downloads/hslr-s/sun-panel/total.svg?logo=github)](https://github.com/hslr-s/sun-panel/releases)
[![docker pulls](https://img.shields.io/docker/pulls/hslr/sun-panel.svg?logo=docker)](https://hub.docker.com/r/hslr/sun-panel)

[[ 中文文档 ]](https://sun-panel-doc.enianteam.com/zh_cn) |
[[ Document ]](https://sun-panel-doc.enianteam.com) |
[[ Demo ]](http://sunpaneldemo.enianteam.com) 

A server, NAS navigation panel, Homepage, Browser homepage.
<br>
一个服务器、NAS导航面板、Homepage、浏览器首页。

</div>

## 🐳 Docker 部署（快速开始）

> 本仓库基于 `sun-panel` v1.8.1 构建，额外提供「完整数据备份 / 恢复」功能。
> 镜像由 GitHub Actions 自动构建，推送到 GitHub Container Registry（GHCR）。

**镜像地址**

```
ghcr.io/carolkenn/hc-panel:1.8.1-export
```

> [!NOTE]
> GHCR 的镜像包默认是私有的。首次拉取前请先登录，密码处填 GitHub Personal Access Token（需 `read:packages` 权限）：
>
> ```bash
> docker login ghcr.io -u <你的GitHub用户名>
> ```
>
> 如果想免登录拉取，可在仓库的 Package 设置里把可见性改为 `Public`。

### 方式一：docker compose（推荐）

新建 `docker-compose.yml`，内容如下：

```yaml
services:
  hc-panel:
    image: ghcr.io/carolkenn/hc-panel:1.8.1-export
    container_name: hc-panel
    volumes:
      - ./conf:/app/conf           # 配置文件（端口、数据库、上传路径）
      - ./uploads:/app/uploads     # 上传的图标 / 壁纸等文件
      - ./database:/app/database   # SQLite 数据库
    ports:
      - 3002:3002
    restart: always
```

启动：

```bash
docker compose up -d
```

### 方式二：docker run

```bash
docker run -d \
  --name hc-panel \
  --restart always \
  -p 3002:3002 \
  -v $(pwd)/conf:/app/conf \
  -v $(pwd)/uploads:/app/uploads \
  -v $(pwd)/database:/app/database \
  ghcr.io/carolkenn/hc-panel:1.8.1-export
```

> Windows PowerShell 中把 `$(pwd)` 换成绝对路径，例如 `-v D:/hc-panel/conf:/app/conf`。

### 访问与初始账号

浏览器打开 `http://<服务器IP>:3002`，初始管理员账号：

| 账号 | 密码 |
| ------------ | ------------ |
| `admin@sun.cc` | `12345678` |

首次登录后请立刻修改密码。如需改端口，编辑 `./conf/conf.ini` 中的 `[base] http_port` 并重启容器。

### 目录挂载说明

| 宿主机目录 | 容器内路径 | 说明 |
| ------------ | ------------ | ------------ |
| `./conf` | `/app/conf` | 配置文件 `conf.ini`，首次启动自动生成 |
| `./database` | `/app/database` | SQLite 数据库文件 |
| `./uploads` | `/app/uploads` | 上传的图标、壁纸等静态文件 |

> 这三个目录必须持久化。**数据导出功能生成的备份包内含 `files/` 与 `uploads/` 的相对路径一一对应，只备份数据库会丢失图片资源。**

### 升级与备份

```bash
docker compose pull
docker compose up -d
```

升级前建议先进入面板的「导入导出 → 完整数据备份」导出一份 ZIP，出问题时可原地恢复。

### 从源码构建（可选）

```bash
git clone https://github.com/CarolKenn/hc-panel.git
cd hc-panel
docker build -t hc-panel:1.8.1-export .
```

![](./doc/images/main-dark.png)

> [!IMPORTANT]
> In order to maintain the livelihood, the author added some [`PRO`] (https://pro.sun-panel.top) function, so the project temporarily entered a closed source state.; At present, the latest version of the open source is `v1.3.0`, [Please see the latest version of closed source](https://github.com/hslr-s/sun-panel/releases).; When the modular technology is developed, the separation of the PRO and the programs will be opened again, and the closed source will have no effect on ordinary users.; Let's look forward to open source again, and at the same time, we are welcome to supervise and review the security of the program.
> 
> 作者为了维持生计，增加了一些 [`PRO`](https://pro.sun-panel.top) 功能，所以项目暂时进入闭源状态。目前开源最新版本为`v1.3.0`，[闭源最新版本请查看](https://github.com/hslr-s/sun-panel/releases)。待开发出模块化技术，然后对PRO和主程序进行分离会再次开源，闭源对普通用户没有任何影响。我们一起期待再次开源吧，同时也欢迎各位大佬对程序的安全性进行监督和审查。

## 😎 Features

- 🍉 Clean interface, powerful functionality, low resource consumption
- 🍊 Easy to use, visual operation, zero-code usage
- 🍠 One-click switch between internal and external network modes
- 🍵 Supports Docker deployment (compatible with Arm systems)
- 🎪 Supports multi-account isolation
- 🎏 Supports viewing system status
- 🫙 Supports custom JS, CSS
- 🍻 Simple usage without the need to connect to an external database
- 🍾 Rich icon styles for free combination, supports [Iconify icon library](https://icon-sets.iconify.design/)
- 🚁 Supports opening small windows in the webpage (some third-party websites may block this feature)

## 🖼️ Preview Screenshots

**Various styles, freely combined**

![](./doc/images/icon-small-new.png)
![](./doc/images/transparent-info.png)
![](./doc/images/transparent-small.png)
![](./doc/images/solid-color-info.png)
![](./doc/images/full-color-small.jpg)

**Built-in small windows**

![](./doc/images/window-ssh.png)
![](./doc/images/window-xunlei.png)



## 🐳 Deployment tutorial
本仓库的 Docker 部署方式见文首「🐳 Docker 部署（快速开始）」一节；
群晖、威联通、宝塔等其他平台的详细教程可参考官方文档 [Deployment Tutorial](https://sun-panel-doc.enianteam.com/usage/quick_deploy.html)。

## 🍵 Donate

> Open-source development is not easy. If you feel that my project has helped you, you are welcome to [donate](./doc/donate.md) or buy me a cup of tea☕ (please leave your nickname or name in the note if possible). Your support is my motivation, thank you.


<a href="https://www.paypal.me/hslrs">
<img height="60" src="./doc/images/donate/paypal.png" target="_blank"></img> 
</a>


|   |   |
| ------------ | ------------ |
| <img height="300" src="./doc/images/donate/weixin.png"/> |  <img height="300" src="./doc/images/donate/alipay.png" /> |

## 🏖️ Communication group & community

Author：**[红烧猎人](https://blog.enianteam.com/u/sun/content/11)**

[Github Discussions](https://github.com/hslr-s/sun-panel/discussions)

QQ交流群，进不去可以点上方连接联系作者

<img src="./doc/images/qq_group_qr2.png"  height="350" />

## ❤️ Thanks

- [Roc](https://github.com/RocCheng)
- [jackloves111](https://github.com/jackloves111)
- [Rock.L](https://github.com/gitlyp)


---

[![Star History Chart](https://api.star-history.com/svg?repos=hslr-s/sun-panel&type=Date)](https://star-history.com/#hslr-s/sun-panel&Date)
