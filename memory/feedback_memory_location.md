---
name: Memory 写入位置规则
description: 项目 memory 唯一权威 c:\moon-panel-dev\memory\（git tracked），P:\moon-panel\memory\ 已废弃为 legacy 备份
type: feedback
---

# Memory 写入位置规则

Moon Panel 项目的 memory 文件写入位置规则（v0.2.7 ship 后更新）。

## 唯一权威: c:\moon-panel-dev\memory\

- 这是 git tracked 目录（跟代码同 repo, GitHub remote = origin）
- 跟随 release 一起 push 到 GitHub
- 跨机器 / contributor clone 时自动继承
- AI-collaboration 全程可 audit（符合 c:\MEMORY.md intro 设计意图）

## 写入规则

1. 新 memory 文件全部写到 `c:\moon-panel-dev\memory\`
2. 更新 MEMORY.md 索引时只更新 `c:\moon-panel-dev\memory\MEMORY.md`
3. 写完后跟代码改动同 commit（memory chore 可单独 commit，也可跟 release commit 合并）
4. 不公开内容（含敏感路径如 D:\）走 `.gitignore` 排除
   - 例: `feedback_d_drive_residue.md` 在 `.gitignore:102`

## P:\moon-panel\memory\ — 已废弃 (legacy 备份)

- 历史: v0.2.5 之前曾作为 memory 主目录（基于 SMB 假设）
- 现状: 已废弃，仅保留作只读 legacy 备份，防遗漏
- ★ 严禁 ★ 写入 `P:\moon-panel\memory\`（写了不会 git push，会丢失）
- 已发生过的丢失案例: 2026-05-06 v0.2.7 release 期间写到 P:\ 的
  `feedback_no_unilateral_substitution.md` 没进 release，后续 chore 才发现并 sync
- 未来某天确认无遗漏可清理

## Auto memory dir (系统 prompt 默认)

- 默认路径 `C:\Users\<user>\.claude\projects\<hash>\memory\` 不使用
- 项目 memory 全走 `c:\moon-panel-dev\memory\`
