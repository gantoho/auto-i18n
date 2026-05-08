# auto-i18n

自动化国际化翻译工作流工具。只需 2 条命令，即可完成从 JSON 提取文案 → 生成 Excel 翻译模板 → 翻译回填的完整流程。

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
                    │  ✓ 递归遍历 JSON             │
                    │  ✓ 自动过滤媒体路径/链接等    │
                    │  ✓ 从文件名识别源语言         │
                    └──────────┬──────────────────┘
                               │
                               ▼
                    ┌──────────────────────┐
                    │  about_us_en.xlsx     │
                    │  (翻译模板)           │
                    └──────────┬───────────┘
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
                    │  ✓ 读取 Excel 翻译内容       │
                    │  ✓ 回填到原始 JSON 结构      │
                    │  ✓ 保持非翻译字段原样        │
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
go build -o auto-i18n .
```

需要 Go 1.21+。

## 快速开始

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
    "company_link": "https://company.com"
  }
}
```

执行提取命令：

```bash
auto-i18n extract about_us_en.json -t zh-CN,ja,ko
```

- 自动从文件名识别源语言为 `en`
- `bgimg_src`（`_src` 后缀）、`company_link`（`_link` 后缀）被自动过滤
- 生成 `about_us_en.xlsx`

生成的 Excel 表格：

| key_path | en | zh-CN | ja | ko |
|----------|----|-------|----|----|
| banner.title | Hello Banner | | | |
| banner.content | This is the about us banner content | | | |
| sections.0.title | Our Mission | | | |
| sections.0.desc | We aim to provide the best service | | | |
| footer.copyright | © 2025 Company Name | | | |

> **提示**：如果不指定 `-t` 参数，生成的 xlsx 只有源语言列，翻译人员可以自行在 Excel 中插入新列填写。

### 第 2 步：翻译人员填写

将 xlsx 文件发给翻译人员，翻译人员直接用 Excel/WPS 打开，在各语言列中填写翻译内容。

### 第 3 步：生成各语言 JSON

翻译完成后，执行生成命令：

```bash
auto-i18n generate about_us_en.xlsx
```

程序自动：
- 从 xlsx 表头读取语言列表（`en` 为源语言，`zh-CN`、`ja`、`ko` 为目标语言）
- 在同目录寻找原始 JSON 文件 `about_us_en.json`
- 生成各语言 JSON 文件

输出：

```
  ✓ about_us_zh-CN.json
  ✓ about_us_ja.json
  ✓ about_us_ko.json
```

生成的 `about_us_zh-CN.json`：

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
    "company_link": "https://company.com"
  }
}
```

所有非翻译字段（媒体路径、链接）被完整保留，JSON 结构与源文件完全一致。

## 命令参考

### `extract`

从 JSON 文件提取可翻译文案，生成 xlsx 翻译模板。

```bash
auto-i18n extract <json_file> [flags]
```

参数：

| 参数 | 说明 |
|------|------|
| `<json_file>` | 源语言 JSON 文件路径（必需） |
| `-t, --target-langs` | 目标语言列表，逗号分隔，如 `zh-CN,ja,ko` |

示例：

```bash
# 只生成源语言列，翻译人员自行添加其他语言列
auto-i18n extract home_en.json

# 带目标语言列
auto-i18n extract home_en.json -t zh-CN,ja,ko,fr,de
```

### `generate`

从翻译完成的 xlsx 生成各语言 JSON 文件。

```bash
auto-i18n generate <xlsx_file> [flags]
```

参数：

| 参数 | 说明 |
|------|------|
| `<xlsx_file>` | 翻译完成的 xlsx 文件路径（必需） |
| `-o, --output-dir` | JSON 输出目录（默认与 xlsx 同目录） |

示例：

```bash
# 输出到当前目录
auto-i18n generate about_us_en.xlsx

# 指定输出目录
auto-i18n generate about_us_en.xlsx -o ./output
```

### `version`

显示版本信息。

```bash
auto-i18n version
```

## 常见问题

### 源语言不是 en 怎么办？

程序通过文件名最后一部分来识别源语言，以下命名均有效：

| 文件名 | 源语言 |
|--------|--------|
| `about_us_en.json` | en |
| `home_zh-CN.json` | zh-CN |
| `contact_ja.json` | ja |
| `intro_fr.json` | fr |

> 如果文件名末尾不是语言代码，程序也能正常工作，只是在生成的 xlsx 中源语言列名会使用默认值。

### 如何自定义哪些字段不参与翻译？

程序默认跳过以下后缀的键名：

```
_src, _link, _no, _url, _path, _href, _img, _icon, _class, _id, _key
```

此外，以下值会被自动跳过：
- 以 `http://` 或 `https://` 开头的 URL
- 以 `/`、`./`、`../` 开头且包含文件扩展名的路径
- 纯数字值

### JSON 中嵌套了数组/对象怎么办？

程序会对 JSON 进行深度优先递归遍历，无论嵌套多深都能正确处理。

数组元素会以 `array.0`、`array.1` 的形式展平到 Excel 中，回填时自动恢复为数组结构。

### 翻译人员需要在 Excel 中做什么？

只需打开 xlsx 文件，在对应语言的空白列中填入翻译内容即可。不需要接触任何 JSON 文件或命令行。

## 技术栈

- **语言**: Go
- **CLI**: [cobra](https://github.com/spf13/cobra)
- **Excel**: [excelize](https://github.com/xuri/excelize/v2)
