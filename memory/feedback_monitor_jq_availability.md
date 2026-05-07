---
name: Monitor jq 可用性教训 (v0.2.13 Ship 1)
description: bash CI/release Monitor 用外部 jq binary 必须先验证可用, 否则 polling silent fail, STUCK_WARNING 是 false positive
type: feedback
---

# Monitor jq 可用性教训 (v0.2.13 Ship 1)

## 教训
bash CI/release Monitor 用外部 jq binary 时, 必须先验证 jq 可用.
否则 polling loop 内 jq 全 silent fail, status 永远空,
STUCK_WARNING 是 false positive (CI 实际已 success).

**Why**: v0.2.13 Ship 1 实例: bash 环境无 jq (exit 127), Monitor 5min 触发 STUCK_WARNING 时 CI 实际 2m38s 已 SUCCESS. Monitor 永远拿不到 status="completed" 信号.

**How to apply**: 写 release/CI Monitor spec 时强制要求 gh `--jq` flag 或 PowerShell `ConvertFrom-Json`, 启动前 dry-run 验证 status parse 成功.

## 根因 (v0.2.13 Ship 1 实例)
- bash 环境无 jq (exit code 127, "command not found")
- Monitor polling loop: `data=$(gh run view ... 2>/dev/null) && jq '.status' <<<"$data"`
- jq 不存在 → silent fail → status 始终空 → loop 永远不识别 "completed"
- 5min STUCK_WARNING 触发, 但 CI 实际 2m38s 已 success

## 修法

### 方案 A (推荐): gh CLI 内置 --jq flag
gh CLI 自带 jq 解析能力, 不依赖外部 binary:

```bash
status=$(gh run view "$RUN_ID" --json status --jq '.status')
conc=$(gh run view "$RUN_ID" --json conclusion --jq '.conclusion // "null"')
```

### 方案 B: PowerShell raw JSON parse (Ship 1 验证可行)
跨平台 fallback:

```powershell
$data = gh run view $RUN_ID --json status,conclusion,jobs | ConvertFrom-Json
$status = $data.status
```

### 方案 C: 启动前探测
```bash
if ! command -v jq &>/dev/null; then
  echo "WARN: jq not found, falling back to gh --jq"
  USE_GH_JQ=1
fi
```

## 应用历史
- v0.2.13 Ship 1 (CI watch): false STUCK_WARNING, Bevan 手动 raw JSON 验证 CI success
- v0.2.13 Ship 2 (release watch): 修法应用 (gh --jq flag), 顺利完成 7m39s

## 不要重犯
下次 v0.2.x release 时:
- 写 Monitor spec 时 ★ 明确要求 gh --jq flag 或 PowerShell raw JSON ★
- 不要假设 bash 有 jq
- 启动 Monitor 前 dry-run 验证 status parse 能成功
