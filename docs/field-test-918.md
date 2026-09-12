# 9/18 返校实测清单（v0.2.1 候选验证）

> 测试对象：**纯 fix 版 exe**（`fix/p0p1-pending` 分支，不含 refactor 重构）。
> 目的：验证 QueryEscape/脱敏/版本检查三项行为修复在真实校园网上的兼容性。

---

## 1. 出发前（家里准备）

```powershell
cd "D:\My Application\Auto_login"

# ① 测试主体：纯 fix 版（版本号仍显示 0.2.0，正常）
git checkout fix/p0p1-pending
go build -ldflags="-s -w -H windowsgui" -o CampusAutoLogin-fix.exe .

# ② 对照组：直接下载 GitHub Releases 的 v0.2.0 exe（新版若失败用于区分是回归还是网络问题）
```

- [ ] 两个 exe 拷进 U 盘（连同本清单）

---

## 2. 到校实测（按顺序做）

### 场景 A — 正常登录（核心，必测）
- [ ] 双击 `CampusAutoLogin-fix.exe` → **窗口应立即出现**（修复前未认证网络下会白等 5 秒）
- [ ] 登录 → 托盘变绿 + 成功气泡 → 能正常上网
- [ ] **结果：通过 / 失败**

### 场景 B — 日志脱敏（P0 验证，必测）
- [ ] 打开 `%LOCALAPPDATA%\CampusAutoLogin\logs\2026-09-18.log`
- [ ] 搜索 `URL:` → 应看到 `user_password=***` 和 `upass=***`
- [ ] **整个日志文件里绝不能出现明文密码或 base64 密码**（出现即 P0 回归，截图）

### 场景 C — 更新检查（有网时补测）
- [ ] 联网状态下重启一次程序，日志开头应出现 `Version check: current=0.2.0 remote latest=0.2.0 min=0.1.0`
- [ ] 远端 latest 仍是 0.2.0，所以不弹更新框是正常的；**只要这行出现就证明 master 分支 URL 修复生效**
- [ ] 未认证网络下启动没有这行是正常的（检查被静默跳过，已改为异步）

### 场景 D — 特殊字符密码（QueryEscape 核心验证，强烈建议）
- 新版修复的正是"密码 base64 后含 `+` 会变成空格"的问题
- [ ] 若你的密码本身含特殊字符：场景 A 通过即已覆盖
- [ ] 若是纯字母数字：登录成功后改一次密码为含 `+/=&` 的临时密码 → 退出重登 → 改回
- [ ] 对照：旧版 v0.2.0 用同一个特殊字符密码大概率失败——能直观看到修复效果

### 场景 E — 断线重连（可选）
- [ ] 登录成功后关闭再打开 WiFi → 心跳报断开 → 自动重连成功

---

## 3. 失败时的采集（发给 ZCode）

- [ ] 当天完整日志文件：`%LOCALAPPDATA%\CampusAutoLogin\logs\2026-09-18.log`
- [ ] 用的哪个 exe（fix 版 / 对照版 v0.2.0），两者结果分别是什么
- [ ] 失败弹窗/气泡上的错误原文
- [ ] 判定逻辑：**新版失败 + 旧版成功 = 新版引入回归**（大概率 QueryEscape，可秒回滚）；两者都失败 = 网络/服务端问题，与本次改动无关

---

## 4. 实测通过后（ZCode 执行）

1. merge 顺序：`fix/p0p1-pending` → `refactor/p2p3` → `master`，push
2. CHANGELOG 日期改实际发布日
3. `go build -ldflags="-s -w -H windowsgui"` 出正式 exe（应显示 v0.2.1）
4. GitHub Release 发布 v0.2.1（传 exe）
5. 蓝奏云上传 → **回填 version.json 的 download 新链接和提取码**（当前是旧链接占位）→ push
6. 提醒更新 `登录/` 本地文档

> 注意：version.json 推送后，所有 0.2.0 老用户会收到 0.2.1 更新提示——预期行为。
