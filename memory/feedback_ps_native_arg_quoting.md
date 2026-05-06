---
name: PowerShell 5.1 native exe arg quoting 不可靠
description: UTF-8 字符 (≤≥—→中文) 通过 here-string @'...'@ 全过, 但 ASCII " 失败. commit msg / tag annotation 必走 -F file 模式
type: feedback
---

# PowerShell 5.1 Native Exe Arg Quoting

v0.2.9 实战发现: PowerShell 5.1 here-string `@'...'@` 含 ASCII `"` 字符
传给 native exe (git) 时, Windows 命令行 quoting 规则把 `"X"` 错误解析为
参数分隔符, 导致引号丢失. 但 UTF-8 字符 (≤ ≥ — → 中文) 全部安全.

直觉跟实际相反: 担心的 UTF-8 字符没事, 没担心的 ASCII `"` 反而出问题.

## 实测结果

通过 PS 5.1 here-string `@'...'@` + `git commit -m`:

| 字符类别 | 实测 | 备注 |
|---------|------|------|
| UTF-8 数学符号 (≤ ≥) | ✓ | v0.2.7+ 4 次实战通过 |
| UTF-8 标点 (— →) | ✓ | em dash, 箭头 |
| UTF-8 中文 | ✓ | 中文字符全过 |
| UTF-8 emoji (☰ 🌙) | ✓ | hamburger / weather emoji |
| ASCII `"` (双引号) | ✗ | **被 Windows command line tokenizer 吞** |
| ASCII `'` (单引号) | 未测 | 推测在 here-string 内会被当字面 |

## 根因 (双层)

1. **PowerShell 单引号 here-string** `@'...'@` 对 PS 自身 protect 字面
   (PS 不会展开 `$var` 等), 但 PS 不 escape 引号给 native exe.

2. **Windows 命令行 quoting** (CommandLineToArgvW) 把 `"X"` 解析为
   "begin quoted arg X end quoted arg", 导致内容中的 ASCII `"` 被消除.

= 即使你以为 here-string 已经 quote 了字面, 传给 native exe 那一刻仍然过
  Windows tokenizer, ASCII `"` 仍然被 strip.

## 可靠解法 — 用 -F file 模式

### 写无 BOM UTF-8 临时文件

PS 5.1 的 `Out-File` 默认是 UTF-8 BOM, 会污染 git 对文件首字节的判断.
必须用 .NET 显式无 BOM:

```powershell
$tempFile = Join-Path $env:TEMP "git-msg.txt"

[System.IO.File]::WriteAllText(
    $tempFile,
    $content,
    (New-Object System.Text.UTF8Encoding $false)   # $false = no BOM
)
```

### git commit / git tag 用 -F file

```powershell
# Commit
git commit -F $tempFile

# Annotated tag
git tag -a v0.X.Y -F $tempFile

Remove-Item $tempFile -Force
```

### 验证

写完文件后 grep 验证字符在:

```powershell
$verify = [System.IO.File]::ReadAllText($tempFile)
if ($verify -match '"管理后台"') { Write-Output "✓ ASCII quotes preserved" }
```

Commit 后用 `git log -1 --format='%B'` 验证字符保留.

## 适用范围

任何 PS 5.1 调用接受 multi-line UTF-8 内容的 native exe:
- `git commit -m` → 改用 `git commit -F file`
- `git tag -a -m` → 改用 `git tag -a -F file`
- 任何长字符串含 ASCII `"` 的 CLI 工具

短的 inline message 不含 ASCII `"` 时, `-m "..."` 仍可用 (但有风险).

## v0.2.9 实战案例

Commit 1st attempt SHA `54efc7b`:
- 用 `git commit -m @'...'@`
- commit body 含 `"管理后台"` 和 `"查看主页"` (ASCII `"` 包围中文)
- 结果: ASCII `"` 全部丢失, 显示为 `管理后台` (无引号)

修复:
- `git reset --soft HEAD~1` (保留 staged 改动, 不擅自 amend)
- 写临时文件用 WriteAllText + UTF8Encoding($false)
- `git commit -F $tempFile`

Commit 2nd attempt SHA `fa8ac1d`:
- ASCII `"` 完整保留 ✓
- UTF-8 (≤ ≥ — → 中文) 仍全过 ✓

## 跟其他 memory 的关系

- `feedback_no_unilateral_substitution.md`: "user 给的字面别擅自替换"
  → 处理 spec 字面字符的态度
- 本文 (`feedback_ps_native_arg_quoting.md`): "如何可靠传 user 字面到 native exe"
  → 处理传输的实操方法

两条互补: 不替换 (态度) + 用对方法传 (实操) = 字面字符 100% 保真.
