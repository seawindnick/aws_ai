---
name: distill-memory
description: 将 memory/daily/ 的原始日志蒸馏为 memory/MEMORY.md 的持久条目
tools: Read, Write, Glob, Grep
---

# 记忆蒸馏

## 工作流程（两阶段）
- 阶段1：从 daily 日志提取结构化信号（decisions, lessons, patterns）
- 阶段2：频率门控 + 去重后写入 MEMORY.md

## 触发条件
memory/daily/ 下有 ≥3 个未标记 <!--distilled--> 的日志文件

## 提取三类条目

### 1. 关键决策（Key Decisions）
识别模式：decided to / chose / will use / going with / switched to
格式：- [YYYY-MM-DD] **决策标题** — 为什么这样选择（保留 why）

### 2. 教训（Lessons Learned）
识别模式：lesson learned / fixed by / root cause was / should have / next time
格式：- [YYYY-MM-DD] **教训标题** — 具体怎么避免

### 3. 复发模式（Recurring Patterns）
条件：同类事件在不同日志中出现 ≥2 次（频率门控）
格式：- [YYYY-MM-DD] **模式名** — 出现N次，建议动作

## 写入规则
- 追加（prepend）到 memory/MEMORY.md 对应 section
- 去重：如果 MEMORY.md 已有相同标题的条目，跳过不写
- 每条必须对未来会话可直接用（能指导判断，不是流水账）
- 蒸馏完给 daily 文件尾部加 <!--distilled YYYY-MM-DD-->

## 铁律
- 只提炼"学到了什么"，不照抄"发生了什么"
- 不删 daily（留待 TTL 过期或手动归档）
- 总 MEMORY.md 体积控制在 <30K token（超出时标记最旧条目为 dormant）
