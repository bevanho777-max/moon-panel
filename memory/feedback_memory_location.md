---
name: Memory 写入位置
description: Moon Panel 项目的 memory 只写 P:\moon-panel\memory\，不写 Claude 自动 memory 目录
type: feedback
---

Moon Panel 项目的 memory 文件应当写到 **P:\moon-panel\memory\**（项目内目录，跟随 NAS SMB 共享走），**不要**写到 Claude 默认的 `C:\Users\mingc\.claude\projects\<encoded-path>\memory\`。

**Why:** 用户把项目永久迁到群晖 SMB 共享 P:\moon-panel（NAS 本地 `/volume5/code/moon-panel`），希望 memory 跟着项目走 —— NAS 多端访问、版本控制、备份都更顺。`C:\Users\mingc\.claude\projects\d--Projects-moon-panel\memory\` 是迁移前的旧 memory，保留作备份但**只读不写**。

**How to apply:**
- 新 memory 文件全部写到 `P:\moon-panel\memory\`
- 更新 MEMORY.md 索引时也只更新 P 上的版本
- 永远不动 `C:\Users\mingc\.claude\projects\d--Projects-moon-panel\memory\` 下的文件
- 新会话如果系统提示 memory 目录是 `C:\Users\mingc\.claude\projects\p--moon-panel\memory\`，要切换到 P:\moon-panel\memory\ 读取已有 memory，并继续往 P 上写
