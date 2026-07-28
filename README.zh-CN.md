<div align="center">

# lazytui

一个通用的、键盘驱动的**本地 CRUD TUI 模板** —— lazy 系列工具(lazyssh / lazytmux / ...)的脚手架。

[English](./README.md) | **[简体中文](./README.zh-CN.md)**

</div>

---

`lazytui` 是一个模板,不是成品。它提供了一套完整、有主见的 TUI,用于管理一组本地条目 —— 列出、搜索、过滤、创建、编辑、删除、置顶、打标签 —— 底层是原子写入的 JSON 存储,外层是六边形架构。你可以直接拿它管理通用的 `Item` 记录,也可以用 scaffold 脚本几分钟内生成一个专用工具(书签、代码片段、hosts 等)。

UI 里的每一个快捷键、应用内帮助面板(`?`)、底部提示行,以及下方的快捷键表,都派生自同一个唯一来源:`internal/adapters/ui/keybindings.go` 中的 `keyBindings` 切片。只需在那儿加一次,所有展示面就保持同步。

---

## ✨ 功能特性

### 条目管理
- 📜 列出本地 JSON 存储中的条目,置顶的常用项始终排在顶部。
- ➕ 从 UI 创建新条目。
- ✏️ 就地编辑条目(名称、标签、备注)。
- 🗑️ 安全删除条目(带确认)。
- 📌 置顶 / 取消置顶,让常用项始终排在顶部。
- 🏷️ 给条目打标签,方便分组与查找。

### 快速导航
- 🔍 按名称模糊搜索。
- ↕️ 置顶条目始终浮顶。
- 🏷️ 按标签过滤(`f`,多选,OR 匹配)。
- 🧩 详情面板,按分组、带标签的 Section 展示每个条目。

### 工作流
- 💾 原子 JSON 存储:`~/.lazytui/items.json` —— 进程崩溃也不会留下写了一半的文件。
- 🔄 在后台刷新列表状态。
- ❓ 应用内帮助(`?`)—— 双列分组面板,列出全部快捷键,与本 README 同出一源。

---

## 🔒 工作原理

`lazytui` 是一个纯 Go 二进制,**没有任何外部运行时依赖**。与 `lazytmux`(调用系统 `tmux`)不同,它只读写自己的状态:

- 条目数据存放于 `~/.lazytui/items.json`。
- 日志写入 `~/.lazytui/lazytui.log`。
- 所有写入都是原子的(先写临时文件再 rename),因此即使崩溃也不会损坏存储。

因为没有系统二进制依赖,Homebrew formula **不声明** `depends_on` —— `brew install` 完全自包含。

---

## 📷 应用截图

> 截图存放在 [`./docs/resources/`](./docs/resources/),为可选项 —— 模板刚 checkout 时该目录可能为空。把 `list.png`、`search.png`、`help.png` 等放进去即可填充本节。

<div align="center">

| 仪表盘 | 帮助 |
| --- | --- |
| <i>./docs/resources/list.png</i> | <i>./docs/resources/help.png</i> |

</div>

---

## 📦 安装

### Option 1: Homebrew(macOS)

```bash
brew install maybewaityou/tap/lazytui
```

`lazytui` 是自包含的 Go 二进制,Homebrew 不会再装别的东西。

> **较新版 Homebrew(5.1.15+/6.0):** 第三方 tap 默认不受信任。若安装时报 `Refusing to load formula ... from untrusted tap`,先信任该 tap(只需一次):
>
> ```bash
> brew trust maybewaityou/tap
> ```

### Option 2: `go install`

```bash
go install github.com/maybewaityou/lazytui/cmd@latest
```

### Option 3: 从 Release 下载二进制

从 [GitHub Releases](https://github.com/maybewaityou/lazytui/releases) 下载预编译二进制。下面这段脚本会自动检测最新版本,并按你的系统拉取对应的 tarball(darwin/linux × amd64/arm64):

```bash
# 检测最新版本
LATEST_TAG=$(curl -fsSL https://api.github.com/repos/maybewaityou/lazytui/releases/latest | jq -r .tag_name)

# 把 OS / 架构归一化成 Release 资源名(darwin/linux × amd64/arm64)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
esac

# 下载 + 解压 + 安装
curl -LJO "https://github.com/maybewaityou/lazytui/releases/download/${LATEST_TAG}/lazytui_${OS}_${ARCH}.tar.gz"
tar -xzf lazytui_${OS}_${ARCH}.tar.gz
sudo mv lazytui /usr/local/bin/

# 享受!
lazytui
```

### Option 4: 从源码构建

```bash
# 克隆仓库
git clone https://github.com/maybewaityou/lazytui.git
cd lazytui

# 构建(会先执行 fmt + go vet)
make build
sudo mv bin/lazytui /usr/local/bin/

# 或者不安装、直接运行
make run
```

### 快照二进制(可选)

`make build-all` 通过 [goreleaser](https://goreleaser.com) 生成交叉编译快照(linux/darwin × amd64/arm64):

```bash
make build-all
```

---

## ⌨️ 快捷键

下表完全按照 [`internal/adapters/ui/keybindings.go`](./internal/adapters/ui/keybindings.go) 中 `keyBindings` 切片的分组排列,逐条列出该切片的每一项。应用内 `?` 帮助面板与底部提示行同样派生自该切片 —— 本 README 是第三个消费者,三者绝不会漂移。

### Navigate(导航)

| 按键     | 动作               |
| -------- | ------------------ |
| `↑↓`     | 上下移动           |
| `Enter`  | 打开详情           |
| `←/→`    | 列表 / 详情切换焦点 |
| `/`      | 搜索               |
| `q`      | 退出               |

### Item(条目)

| 按键 | 动作   |
| ---- | ------ |
| `a`  | 新建   |
| `e`  | 编辑   |
| `d`  | 删除   |
| `r`  | 刷新   |

### Filter(过滤)

| 按键 | 动作        |
| ---- | ----------- |
| `f`  | 按标签过滤  |

### Metadata(元数据)

| 按键 | 动作          |
| ---- | ------------- |
| `p`  | 置顶 / 取消置顶 |
| `n`  | 编辑备注      |
| `c`  | 清空标签      |

### Other(其他)

| 按键 | 动作           |
| ---- | -------------- |
| `?`  | 帮助           |

**在条目表单中:**

| 按键           | 动作                |
| -------------- | ------------------- |
| `↑↓`           | 切换字段            |
| `Tab/Shift+Tab`| 在字段间切换        |
| `Enter`        | 提交(保存)         |
| `Shift+Enter`  | 换行(仅 Note 字段) |
| `Esc`          | 取消                |

提示:底部的状态栏会显示你上一次操作的结果。

---

## 🏗 架构

采用六边形架构(端口与适配器):

```
cmd/main.go                       → cobra 根命令,装配依赖
internal/core/domain/             → Item 领域模型
internal/core/ports/              → Repository / Service 端口
internal/core/services/           → 业务逻辑
internal/adapters/data/store/     → ~/.lazytui/items.json(原子 JSON CRUD)
internal/adapters/ui/             → tview TUI(Tokyo Night)
internal/logger/                  → zap → ~/.lazytui/lazytui.log
```

store 适配器是一个通用 JSON CRUD 引擎;若要把 `lazytui` 改造成非本地后端,实现 `ports.Repository` 并在 `cmd/main.go` 里换掉该适配器即可。

---

## 🧰 作为模板使用(scaffold 新工具)

`lazytui` 自带一个 scaffold 脚本,会拷贝整棵模板树,并改写占位符(`lazytui` → 你的工具名,`Item` → 你的实体),从而生成一个新的 lazy 系列工具。

```bash
./scripts/scaffold.sh --name <tool> --entity <Entity> --dir <target>
```

示例 —— 生成一个实体为 `Bookmark` 的 `lazybookmark` 工具:

```bash
./scripts/scaffold.sh --name lazybookmark --entity Bookmark --dir ../lazybookmark
cd ../lazybookmark
go mod tidy && go build ./...
```

参数:

| Flag        | 含义                                                              |
| ----------- | ----------------------------------------------------------------- |
| `--name`    | 小写工具名,如 `lazybookmark`(模块后缀 + 二进制名)                |
| `--entity`  | PascalCase 实体名,如 `Bookmark`(领域类型;复数 = `+s`)           |
| `--dir`     | 生成目标目录(不存在则创建)                                       |

改写顺序很关键(module path → `LAZYTUI` → `Item` → `items` → `item` → `lazytui`),且 tview 自带的 `*Item*` API 方法会被临时 token 屏蔽,不会被误改写。完整定制清单(领域字段、详情 Section、表单字段、快捷键、Repository 适配器)见 [`docs/template-guide.md`](./docs/template-guide.md)。

---

## 🏗 构建

```bash
make build       # fmt + vet + build → bin/lazytui
make run         # 带版本 ldflags 的 go run
make test        # race + cover
make lint        # golangci-lint
make build-all   # goreleaser 交叉编译快照
```

---

## 🤝 参与贡献

欢迎贡献!

- 如果你发现了 bug 或有新功能想法,请[提一个 issue](https://github.com/maybewaityou/lazytui/issues)。
- 如果你愿意贡献代码,fork 仓库后提交 pull request。

### 语义化提交信息

本项目遵循语义化提交。请将你的 commit / PR 标题写成:

- `type(scope): 简短描述`

常见 type:`feat`、`fix`、`improve`、`refactor`、`docs`、`test`、`ci`、`chore`、`revert`。
scope 可选(例如 `ui`、`cli`、`core`)。

---

## 📄 许可证

基于 [Apache License 2.0](./LICENSE) 发布。

---

## 🙏 致谢

- 基于 [tview](https://github.com/rivo/tview) + [tcell](https://github.com/gdamore/tcell)、[cobra](https://github.com/spf13/cobra) 与 [zap](https://go.uber.org/zap) 构建。
- 架构与交互语言继承自 lazy 系列工具([lazyssh](https://github.com/Adembc/lazyssh)、lazytmux)。
- 主题:Tokyo Night。
