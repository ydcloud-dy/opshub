# Task 任务中心插件

<p align="center">
  <img src="https://img.shields.io/badge/Plugin-Task-FF6B6B?style=flat&logo=task" alt="Task">
  <img src="https://img.shields.io/badge/Version-1.0.0-blue?style=flat" alt="Version">
  <img src="https://img.shields.io/badge/Status-Stable-green?style=flat" alt="Status">
</p>

---

## 概述

Task 任务中心插件提供强大的任务编排与执行能力，支持脚本执行、批量操作、文件分发、任务模板管理等功能，帮助运维团队高效完成日常运维工作。

---

## 功能特性

### 执行任务

| 功能 | 描述 |
|:-----|:-----|
| 脚本执行 | 支持 Shell、Python、Perl 等多种脚本语言 |
| 批量执行 | 同时在多台主机上执行任务 |
| 实时日志 | 实时显示执行输出和错误信息 |
| 执行统计 | 统计成功/失败主机数量和执行耗时 |
| 超时控制 | 设置执行超时时间，避免任务卡死 |

### 模板管理

| 功能 | 描述 |
|:-----|:-----|
| 模板定义 | 创建可复用的任务模板 |
| 参数配置 | 支持模板参数化，执行时动态传入 |
| 分类管理 | 按类型（脚本/Ansible/模块）分类 |
| 版本控制 | 保存模板历史版本 |

### 文件分发

| 功能 | 描述 |
|:-----|:-----|
| 批量分发 | 同时向多台主机分发文件 |
| 目录分发 | 支持整个目录的分发 |
| 权限设置 | 设置目标文件的权限和所有者 |
| 进度显示 | 实时显示传输进度 |
| 完整性校验 | 自动校验文件 MD5 |

### 执行历史

| 功能 | 描述 |
|:-----|:-----|
| 历史记录 | 保存所有任务执行记录 |
| 日志查看 | 查看详细执行日志 |
| 结果统计 | 按主机统计执行结果 |
| 数据导出 | 支持导出 CSV/Excel |

---

## 安装与启用

### 通过管理界面启用

1. 登录 OpsHub 系统
2. 进入「插件管理」-「插件列表」
3. 找到「Task」插件
4. 点击「启用」按钮
5. 刷新页面，左侧菜单出现「任务中心」

### 通过 API 启用

```bash
# 启用插件
curl -X POST http://localhost:9876/api/v1/plugins/task/enable \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# 禁用插件
curl -X POST http://localhost:9876/api/v1/plugins/task/disable \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

---

## 使用指南

### 执行任务

#### 方式一：直接执行脚本

1. 进入「任务中心」-「执行任务」
2. 选择执行方式：**脚本执行**
3. 编写脚本内容或选择已有模板
4. 选择目标主机（支持多选）
5. 设置执行参数（超时时间等）
6. 点击「执行」按钮
7. 实时查看执行日志和结果

#### 方式二：使用任务模板

1. 进入「任务中心」-「执行任务」
2. 选择执行方式：**使用模板**
3. 选择任务模板
4. 填写模板参数（如有）
5. 选择目标主机
6. 点击「执行」按钮

### 创建任务模板

#### 步骤

1. 进入「任务中心」-「任务模板」
2. 点击「新建模板」按钮
3. 填写模板信息：

| 字段 | 说明 |
|:-----|:-----|
| 模板名称 | 模板的显示名称 |
| 模板编码 | 唯一标识符（英文） |
| 模板描述 | 模板用途说明 |
| 分类 | script / ansible / module |
| 平台 | linux / windows |
| 脚本内容 | 实际执行的脚本 |
| 超时时间 | 执行超时秒数 |

4. 定义模板参数（可选）
5. 点击「保存」

#### 模板参数示例

在脚本中使用 `{{参数名}}` 占位符：

```bash
#!/bin/bash
# 示例：软件安装模板

# 参数: PACKAGE_NAME - 要安装的软件包
# 参数: SERVICE_NAME - 服务名称

echo "开始安装 {{PACKAGE_NAME}}..."
apt-get update
apt-get install -y {{PACKAGE_NAME}}

echo "启动服务 {{SERVICE_NAME}}..."
systemctl start {{SERVICE_NAME}}
systemctl enable {{SERVICE_NAME}}

echo "安装完成！"
systemctl status {{SERVICE_NAME}}
```

执行时系统会提示输入参数值。

### 文件分发

#### 单文件分发

1. 进入「任务中心」-「文件分发」
2. 点击「新建分发任务」
3. 上传要分发的文件
4. 选择目标主机
5. 设置分发参数：
   - **目标路径**: 文件存放位置
   - **文件权限**: 如 755、644
   - **所有者**: user:group 格式
6. 点击「开始分发」

#### 批量文件分发

1. 选择多个文件或整个目录
2. 配置分发参数
3. 选择目标主机组
4. 点击「开始分发」
5. 查看分发进度和结果

### 查看执行历史

1. 进入「任务中心」-「执行历史」
2. 筛选条件：
   - 时间范围
   - 执行状态
   - 执行用户
   - 任务名称
3. 点击记录查看详细日志
4. 支持按主机查看执行结果

---

## 脚本语言支持

### Shell 脚本

最常用的脚本类型，支持 Bash、Sh 等：

```bash
#!/bin/bash
set -e  # 遇到错误立即退出

# 检查服务状态
echo "检查 nginx 服务状态..."
systemctl status nginx

# 获取系统信息
echo "系统信息："
uname -a
cat /etc/os-release

# 磁盘使用情况
echo "磁盘使用情况："
df -h
```

### Python 脚本

适合复杂的数据处理和系统管理：

```python
#!/usr/bin/env python3
import os
import sys
import subprocess

def check_disk_usage():
    """检查磁盘使用率"""
    result = subprocess.run(['df', '-h'], capture_output=True, text=True)
    print(result.stdout)

def check_memory():
    """检查内存使用"""
    with open('/proc/meminfo', 'r') as f:
        for line in f:
            if 'Mem' in line:
                print(line.strip())

if __name__ == '__main__':
    print("=== 系统检查 ===")
    check_disk_usage()
    check_memory()
```

### Perl 脚本

支持文本处理和系统管理：

```perl
#!/usr/bin/perl
use strict;
use warnings;

# 读取并分析日志文件
my $error_count = 0;
open(my $fh, '<', '/var/log/nginx/error.log') or die "Cannot open: $!";
while (<$fh>) {
    $error_count++ if /error/i;
}
close($fh);

print "发现 $error_count 个错误\n";
```

---

## 数据库表

Task 插件使用以下数据库表：

| 表名 | 说明 |
|:-----|:-----|
| `job_templates` | 任务模板 |
| `job_tasks` | 任务执行记录 |
| `ansible_tasks` | Ansible 任务 |

---

## 权限管理

### 平台级权限

在「系统管理」-「角色管理」中配置：

| 权限 | 说明 |
|:-----|:-----|
| 执行任务 | 允许执行脚本和任务 |
| 模板管理 | 创建、编辑、删除模板 |
| 文件分发 | 允许进行文件分发操作 |
| 查看历史 | 查看执行历史记录 |

### 主机级权限

任务只能在用户有权限的主机上执行：

1. 进入「资产管理」-「权限配置」
2. 为角色分配主机访问权限
3. 用户只能看到和操作有权限的主机

---

## 最佳实践

### 脚本编写建议

| 建议 | 说明 |
|:-----|:-----|
| 使用 Shebang | 始终在脚本开头指定解释器 (`#!/bin/bash`) |
| 错误处理 | 使用 `set -e` 或检查命令返回值 |
| 日志输出 | 添加 echo 输出便于调试 |
| 参数验证 | 验证输入参数的有效性 |
| 超时保护 | 对长期运行的命令设置超时 |
| 幂等设计 | 脚本可重复执行不产生副作用 |

### 模板管理建议

| 建议 | 说明 |
|:-----|:-----|
| 命名规范 | 使用清晰的模板名称和描述 |
| 参数化 | 尽可能使用参数提高复用性 |
| 分类整理 | 按用途分类管理模板 |
| 文档说明 | 为复杂模板添加使用说明 |
| 定期审查 | 定期更新和测试模板 |

### 安全建议

| 建议 | 说明 |
|:-----|:-----|
| 权限最小化 | 按需分配权限，避免过度授权 |
| 敏感数据 | 避免在脚本中硬编码密码 |
| 命令注入 | 对用户输入进行验证和转义 |
| 审计日志 | 定期检查执行历史和操作日志 |

---

## 常见问题

### Q: 脚本执行超时怎么办？

**A:**
1. 在模板设置中增加超时时间
2. 检查脚本是否有死循环或等待用户输入
3. 对耗时操作添加后台执行 (`nohup command &`)

### Q: 如何让脚本以特定用户身份运行？

**A:** 在主机凭证中配置 SSH 用户，脚本将以该用户身份执行。如需切换用户，可在脚本中使用 `sudo -u username command`。

### Q: 文件分发失败怎么办？

**A:** 检查：
1. 目标主机是否在线
2. SSH 连接是否正常
3. 目标目录是否有写权限
4. 磁盘空间是否充足

### Q: 如何导出执行历史？

**A:** 在执行历史页面，选择时间范围后点击「导出」按钮，支持 CSV 和 Excel 格式。

### Q: 批量执行时部分主机失败怎么办？

**A:**
1. 查看执行历史中每台主机的详细日志
2. 失败的主机可以单独重新执行
3. 检查失败主机的网络连接和权限

---

## 相关文档

- [Shell 脚本教程](https://www.gnu.org/software/bash/manual/)
- [Python 文档](https://docs.python.org/)
- [Ansible 文档](https://docs.ansible.com/)
- [OpsHub 主文档](../../README.md)
- [部署指南](../deployment.md)
