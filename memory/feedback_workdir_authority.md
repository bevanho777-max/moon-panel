---
name: 工作目录权威性核对
description: 三件证据法核对 git 工作目录是否真权威, 防止改错副本 / 推错 repo
type: feedback
---

# 工作目录权威性核对

v0.2.7 ship 时发现 memory 写"源码权威 P:\moon-panel\" 已过时.
实际权威在 c:\moon-panel-dev\ (本地 NTFS git repo, remote = GitHub origin).
为防止下次再发生类似混乱, 沉淀本规则.

## 三件证据法 (核对工作目录)

任何疑似工作目录混乱时 (cwd 跟 memory 不一致 / 多副本并存 / 跨机器恢复),
跑这三个命令验证:

```powershell
git -C <候选目录> remote -v          # 验证 remote URL (应 = GitHub origin)
git -C <候选目录> rev-parse HEAD     # 验证 HEAD commit
git -C <候选目录> reflog --date=iso -10  # 反推哪些 commit 是本目录直接 commit
```

判断:
- reflog 出现 `commit:` = 本目录直接 commit, 是真权威
- reflog 出现 `pull:` / `fetch:` = 别处 commit 同步过来的, 别处可能是真权威
- remote = GitHub origin URL = 跟仓库连通
- HEAD = 期望 SHA = 跟最新 release 一致

## v0.2.7 实测结论 (3 候选目录)

- `c:\moon-panel-dev\`: 
  - remote = `https://github.com/bevanho777-max/moon-panel.git` ✓
  - HEAD = `de03d7e` (v0.2.7) ✓
  - reflog: v0.2.5/2.6/2.7 全部 `commit:` (直接 commit, 是真权威)
  
- `P:\moon-panel\`: 
  - 不是 git repo (无 `.git` 目录)
  - 仅文件树, 不参与 git 流程
  - Memory 写入此处不会 push (已废弃, 见 feedback_memory_location.md)
  
- `C:\moon-build\frontend\`: 
  - 不是 git repo
  - 纯 build workspace (F-lite 验证副本)
  - 仅在 SMB 场景使用 (见 feedback_delivery_routine.md)

→ `c:\moon-panel-dev\` 是真权威
→ 详见 v0.2.7 ship commit message + 工作目录核对流程

## 何时跑核对

- 跨机器恢复 (新 PC / VM / 笔电淘汰后) 第一次 ship 前
- Memory 提示工作目录跟 cwd 不一致时
- Claude Code session 报错 "cannot change to ..." 之类路径错误
- 任何 "我以为在 X 但 ls 失败" 的情况

跑核对的代价 (~30 秒) 远小于改错副本 / 推错 repo 的代价.
