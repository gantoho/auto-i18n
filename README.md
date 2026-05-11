# auto-i18n

自动化国际化翻译工作流工具。只需 2 条命令（或一个 Web 页面），即可完成从 JSON 提取文案 → 生成 Excel 翻译模板 → 翻译回填的完整流程。

## 痛点

传统 i18n 工作流中，开发者需要：
- 手动从 JSON 中逐个找出可翻译文案
- 手动创建 Excel/表格给翻译人员
- 翻译完成后手动将内容逐条填入各语言 JSON
- 人工维护 JSON 结构与翻译内容的一致性

**auto-i18n** 将这些重复劳动全部自动化。

## 工作流程

```
                         ┌─────────────────────┐
                         │   about_us_en.json   │
                         │  (开发者编写源语言)    │
                         └──────────┬──────────┘
                                    │
                                    ▼
                    ┌─────────────────────────────┐
                    │  auto-i18n extract           │
                    │  auto-i18n server            │
                    │  ✓ 递归遍历 JSON             │
                    │  ✓ 自动过滤媒体路径/邮箱等   │
                    │  ✓ 从文件名识别源语言         │
                    │  ✓ 字段顺序与源文件一致       │
                    └──────────┬──────────────────┘
                               │
                               ▼
                    ┌──────────────────────────────────────────┐
                    │  about_us.xlsx                            │
                    │  (翻译模板, 无 key)                       │
                    │  en │ Simplified Chinese │ Japanese │ Korean │
                    └──────────────────┬───────────────────────┘
                               │
                               ▼
                    ┌──────────────────────┐
                    │  翻译人员填写各语言列   │
                    │  打开 Excel 直接填写   │
                    └──────────┬───────────┘
                               │
                               ▼
                    ┌─────────────────────────────┐
                    │  auto-i18n generate          │
                    │  auto-i18n server            │
                    │  ✓ 按行顺序映射翻译          │
                    │  ✓ 回填到原始 JSON 结构      │
                    │  ✓ 保持非翻译字段原样        │
                    │  ✓ 字段顺序与源文件一致       │
                    └──────────┬──────────────────┘
                               │
                    ┌──────────┼──────────┐
                    ▼          ▼          ▼
           about_us_zh-CN.json  ja.json  ko.json
```

## 安装

### 方式一：直接下载二进制

从 [Releases](../../releases) 页面下载对应平台的二进制文件，放到 `PATH` 目录即可。

### 方式二：从源码编译

```bash
git clone <your-repo-url>
cd auto-i18n
go build -o dist/auto-i18n .
```

需要 Go 1.21+。

编译产物输出到 `dist/` 目录，由 `.gitignore` 统一忽略。

### 使用 Makefile 构建（推荐）

项目提供了 Makefile，封装了常用构建目标，无需记忆复杂的 `-o` 和交叉编译参数。

> **Windows 用户**：如需使用 `make`，参考以下安装方式：
> 1. 以管理员身份打开 PowerShell，执行 `Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))` 安装 Chocolatey
> 2. 然后执行 `choco install make` 安装 make
> 3. 完成后重启终端即可使用 `make` 命令

```bash
# 构建当前平台二进制
make build

# 构建所有平台
make build-all

# 运行测试
make test

# 清理构建产物
make clean
```

所有编译产物统一输出到 `dist/` 目录。

### 交叉编译到其他平台

Go 支持一键交叉编译，在 Windows 上即可编译出 Linux / macOS 版本。

```bash
# 使用 Makefile（推荐，全平台通用）
make build-all
```

或手动指定：

```powershell
# Windows PowerShell（以 Linux amd64 为例）
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -ldflags="-s -w" -o dist/auto-i18n-linux .
```

```bash
# Linux / macOS（以 Linux amd64 为例）
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/auto-i18n-linux .
```

> **部署到 Linux 服务器时**：如果运行报错 `cannot execute binary file`，说明二进制架构与服务器不匹配。请在服务器上执行 `uname -m` 查看架构，然后选择对应的构建目标：
> - `x86_64` → `make build-linux`（GOARCH=amd64）
> - `aarch64` / `arm64` → `make build-linux-arm64`（GOARCH=arm64）

`-ldflags="-s -w"` 可去掉调试符号，减小约 30% 体积。生成的二进制为静态编译，不依赖任何外部库，可直接在目标系统运行。

## 快速开始（命令行）

### 第 1 步：提取可翻译文案

假设有一个源语言 JSON 文件 `about_us_en.json`：

```json
{
  "banner": {
    "title": "Hello Banner",
    "content": "This is the about us banner content",
    "bgimg_src": "/images/about-banner.png"
  },
  "sections": [
    {
      "title": "Our Mission",
      "desc": "We aim to provide the best service",
      "icon_src": "/images/mission.png"
    }
  ],
  "footer": {
    "copyright": "© 2025 Company Name",
    "company_link": "https://company.com",
    "email": "contact@company.com"
  }
}
```

执行提取命令：

```bash
auto-i18n extract about_us_en.json -t "Simplified Chinese,Japanese,Korean"
```

- 自动从文件名识别源语言为 `en`
- `bgimg_src`（`_src` 后缀）、`company_link`（`_link` 后缀）、`email` 自动过滤
- 生成 `about_us_en.xlsx`

生成的 Excel 表格（无 key_path 列，通过行顺序映射）：

| en | Simplified Chinese | Japanese | Korean |
|----|-------------------|----------|--------|
| Hello Banner | | | |
| This is the about us banner content | | | |
| Our Mission | | | |
| We aim to provide the best service | | | |
| ... | | | |

> **提示**：如果不指定 `-t` 参数，生成的 xlsx 只有源语言列，翻译人员可以自行在 Excel 中插入新列填写。

### 第 2 步：翻译人员填写

将 xlsx 文件发给翻译人员，翻译人员直接用 Excel/WPS 打开，在各语言列中填写翻译内容。

| en | Simplified Chinese | Japanese | Korean |
|----|-------------------|----------|--------|
| Hello Banner | 你好横幅 | こんにちは | 안녕하세요 |
| This is the about us banner content | 这是横幅内容 | ... | ... |
| Our Mission | 我们的使命 | ... | ... |
| ... | ... | ... | ... |

### 第 3 步：生成各语言 JSON

翻译完成后，执行生成命令：

```bash
auto-i18n generate about_us_en.xlsx
```

程序自动：
- 从 xlsx 表头读取语言列表（`en` 为源语言，`Simplified Chinese`、`Japanese`、`Korean` 为目标语言）
- **自动在同目录寻找原始 JSON 文件**：优先找 `about_us_en_en.json`，再找 `about_us_en.json`
- 重新从源 JSON 提取字段路径，按行位置一对一映射
- 生成各语言 JSON 文件（文件名使用语言代码，如 `about_us_cn.json`）

输出：

```
  ✓ about_us_zh-CN.json
  ✓ about_us_ja.json
  ✓ about_us_ko.json
```

生成的 `about_us_zh-CN.json`（字段顺序与源文件完全一致）：

```json
{
  "banner": {
    "title": "你好横幅",
    "content": "这是关于我们的横幅内容",
    "bgimg_src": "/images/about-banner.png"
  },
  "sections": [
    {
      "title": "我们的使命",
      "desc": "我们致力于提供最好的服务",
      "icon_src": "/images/mission.png"
    }
  ],
  "footer": {
    "copyright": "© 2025 公司名称",
    "company_link": "https://company.com",
    "email": "contact@company.com"
  }
}
```

所有非翻译字段（媒体路径、链接、邮箱）被完整保留，JSON 结构与源文件完全一致。

## 快速开始（Web UI）

auto-i18n 内置了 Web 界面，提供更直观的操作方式。

### 启动服务

```bash
# 默认端口 8080
auto-i18n server

# 也可以指定端口
auto-i18n server -p 3000
```

访问 `http://localhost:8080`（或指定的端口）即可看到操作界面。

### 提取翻译模板

1. 点击「提取文案」标签
2. 上传 JSON 文件（拖拽或点击）
3. 选择目标语言（点击标签或手动输入）
4. 点击「生成 xlsx 模板」→ 自动下载

### 生成 JSON 文件

1. 点击「生成 JSON」标签
2. 上传翻译完成的 xlsx 文件
3. 上传原始 JSON 文件
4. 点击「生成 JSON 文件」→ 自动下载 ZIP 包

## 全部命令参考

### `help`

查看帮助信息。

```bash
# 查看所有命令
auto-i18n help
auto-i18n --help
auto-i18n -h

# 查看子命令的详细帮助
auto-i18n help extract
auto-i18n help generate
auto-i18n help server
```

### `extract`

从 JSON 文件提取可翻译文案，生成 xlsx 翻译模板。

```bash
auto-i18n extract <json_file> [flags]
```

参数：

| 参数 | 说明 |
|------|------|
| `<json_file>` | 源语言 JSON 文件路径（必需） |
| `-t, --target-langs` | 目标语言列表，逗号分隔，如 `Simplified Chinese,Japanese,Korean` |
| `-s, --split-tags` | 拆分含 HTML 标签的文案为多段分别翻译（默认不拆分） |
| `-h, --help` | 查看 extract 子命令帮助 |

示例：

```bash
# 只生成源语言列，翻译人员自行添加其他语言列
auto-i18n extract home_en.json

# 带目标语言列
auto-i18n extract home_en.json -t "Simplified Chinese,Japanese,Korean"

# 拆分 HTML 标签文案
auto-i18n extract home_en.json -t "Simplified Chinese" -s
```

### `generate`

从翻译完成的 xlsx 生成各语言 JSON 文件。

```bash
auto-i18n generate <xlsx_file> [flags]
```

程序会自动在同目录下寻找原始 JSON 文件，查找顺序为：

1. `{xlsx文件名}_{源语言}.json`，例如 `about_us_en_en.json`
2. `{xlsx文件名}.json`，例如 `about_us_en.json`

参数：

| 参数 | 说明 |
|------|------|
| `<xlsx_file>` | 翻译完成的 xlsx 文件路径（必需） |
| `-o, --output-dir` | JSON 输出目录（默认与 xlsx 同目录） |
| `-h, --help` | 查看 generate 子命令帮助 |

示例：

```bash
# 输出到当前目录
auto-i18n generate about_us_en.xlsx

# 指定输出目录
auto-i18n generate about_us_en.xlsx -o ./output
```

### `server`

启动 Web 服务，通过浏览器进行操作。

```bash
auto-i18n server [flags]
```

参数：

| 参数 | 说明 |
|------|------|
| `-p, --port` | 服务端口号（默认 8080） |
| `-h, --help` | 查看 server 子命令帮助 |

示例：

```bash
# 使用默认端口 8080
auto-i18n server

# 自定义端口
auto-i18n server -p 3000
```

启动后控制台会输出访问地址，按 `Ctrl+C` 停止服务。

### `version`

显示版本信息。

```bash
auto-i18n version
```

### `completion`

生成 shell 自动补全脚本。

```bash
# 生成 PowerShell 补全
auto-i18n completion powershell > _auto-i18n.ps1

# 生成 bash 补全
auto-i18n completion bash > /etc/bash_completion.d/auto-i18n

# 生成 zsh 补全
auto-i18n completion zsh > /usr/local/share/zsh/site-functions/_auto-i18n

# 生成 fish 补全
auto-i18n completion fish > ~/.config/fish/completions/auto-i18n.fish
```

## API 接口

启动 `server` 后，提供以下 HTTP API：

| 端点 | 方法 | 说明 |
|------|------|------|
| `/` | GET | Web UI 页面 |
| `/api/health` | GET | 健康检查 |
| `/api/extract` | POST | 上传 JSON → 下载 xlsx |
| `/api/generate` | POST | 上传 xlsx+JSON → 下载 ZIP |

### `/api/health`

**响应**：`{"status":"ok"}`

### `/api/extract`

**请求格式**：`multipart/form-data`

| 字段 | 类型 | 说明 |
|------|------|------|
| `file` | file | 源语言 JSON 文件 |
| `langs` | string | 目标语言，逗号分隔（可选） |
| `splitTags` | string | 设为 `true` 时拆分含 HTML 标签的文案（可选） |

**响应**：xlsx 文件下载（`Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`）。

### `/api/generate`

**请求格式**：`multipart/form-data`

| 字段 | 类型 | 说明 |
|------|------|------|
| `xlsx` | file | 翻译完成的 xlsx 文件 |
| `json` | file | 原始 JSON 文件 |

**响应**：ZIP 包下载（内含各语言 JSON 文件）。

## 自动过滤规则

程序会自动识别以下内容，不会将其列为需要翻译的文案：

### 键名过滤

检查 JSON 对象的键名是否以特定后缀结尾，匹配则跳过整个字段：

```
_src    _link   _no     _url    _path   _href
_img    _icon   _class  _id     _key    _mail
email
```

### 值内容自动检测

检查字符串值是否匹配以下规则，匹配则跳过：

| 类型 | 匹配规则 | 示例 |
|------|----------|------|
| 空字符串 | 长度为零 | `""` |
| 纯数字 | 整数或小数 | `12345`, `3.14`, `-1` |
| 邮箱 | `xxx@xxx.xxx` 格式 | `user@company.com` |
| URL | 以 `http://` 或 `https://` 开头 | `https://example.com` |
| 绝对路径 | 以 `/` 开头且包含文件扩展名 | `/images/banner.png` |
| 相对路径 | 以 `./` 或 `../` 开头且包含文件扩展名 | `./path/file.pdf` |

### JSON 字段顺序

- 程序使用 `json.Decoder` 的 Token API 按**文档顺序**遍历 JSON
- 提取的文案顺序与源文件完全一致
- 生成的目标语言 JSON 也保持相同的字段顺序
- 数组元素按索引顺序遍历

## 常见问题

### 源语言不是 en 怎么办？

程序通过文件名最后一部分来识别源语言，以下命名均有效：

| 文件名 | 源语言 |
|--------|--------|
| `about_us_en.json` | en |
| `home_zh-CN.json` | zh-CN |
| `contact_ja.json` | ja |
| `intro_fr.json` | fr |

> 如果文件名末尾不是语言代码，程序也能正常工作，只是生成的 xlsx 中源语言列名会使用文件名的最后一段内容。

### 为什么 xlsx 中没有 key_path 列？

从 v0.2 版本开始去掉了 key_path 列。程序通过**行顺序**建立映射关系——第 1 行数据对应 JSON 遍历的第 1 个文案，第 2 行对应第 2 个，以此类推。这样翻译人员看到的是更干净的表格，只需从左到右填写即可。

### xlsx 中的语言列名为什么是英文全名而不是代码？

xlsx 表头使用语言英文全名（如 `Simplified Chinese`、`Japanese`），方便翻译人员直接识别。生成 JSON 文件时，会自动将语言名称映射回代码作为文件名（如 `about_us_cn.json`、`about_us_ja.json`）。

支持的语言及其映射关系：

| 语言代码 | 英文全名 |
|----------|----------|
| `cn` | Simplified Chinese |
| `tw` | Traditional Chinese |
| `jp` | Japanese |
| `kr` | Korean |
| `fr` | French |
| `de` | German |
| `ru` | Russian |
| ... | ... |

完整列表请查看 [lang.go](internal/lang/lang.go)。

如果传入的代码不在支持列表中（如 `en`），会原样保留不变。

### 如果源 JSON 更新了，之前翻译过的内容怎么处理？

当前版本需要重新执行一次完整流程（extract → 翻译 → generate）。增量更新功能在计划中。

### JSON 中嵌套了数组/对象怎么办？

程序会对 JSON 进行深度优先递归遍历，无论嵌套多深都能正确处理。数组元素会以 `sections.0.title`、`sections.1.title` 的形式在内部管理路径，翻译人员无需关心这些细节。

### generate 时提示找不到原始 JSON 文件？

程序会自动在同目录下按以下顺序查找：

1. `{xlsx文件名}_{源语言}.json`（例如 xlsx 是 `about_us_en.xlsx`，源语言是 `en`，则找 `about_us_en_en.json`）
2. `{xlsx文件名}.json`（例如 `about_us_en.json`）

如果文件名不符合这些规则，可以将 JSON 文件重命名后再试。

### 翻译人员需要在 Excel 中做什么？

只需打开 xlsx 文件，在对应语言的空白列中填入翻译内容即可。不需要接触任何 JSON 文件或命令行。

### 如何查看某个子命令的详细帮助？

```bash
auto-i18n extract --help
auto-i18n generate --help
auto-i18n server --help
```

## 技术栈

- **语言**: Go
- **CLI**: [cobra](https://github.com/spf13/cobra)
- **Excel**: [excelize](https://github.com/xuri/excelize/v2)
- **JSON 操作**: [sjson](https://github.com/tidwall/sjson)
- **Web UI**: 纯 HTML/CSS/JS（嵌入到二进制，无外部依赖）
