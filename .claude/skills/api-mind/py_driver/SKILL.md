---
name: api-mind-py-driver
description: |
  api-mind 的 Python 语言用例驱动。负责 Python 语言测试用例的生成、执行与报告。
  由 api-mind 主 SKILL 分发调用，不直接面向用户。
---

# py_driver

Python 语言用例生成与执行驱动（待实现）。

## 职责

- `generate`：将 case.md 转为 Python 测试用例代码
- `execute`：执行 Python 测试用例，失败时自动修复
- `report`：根据 Python 用例执行结果生成测试报告

> 具体实现待补充。
