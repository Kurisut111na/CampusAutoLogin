# Changelog

## [0.2.1] - 2026-09-18（预计发布日，实测通过后生效）

### Fixed
- 版本检查 URL 指向不存在的 `main` 分支，自 v0.2.0 起始终 404（静默失效）
- 登录密码写入日志文件（`Portal URL` / `Old API URL` 现已脱敏为 `***`）
- 密码双重加密导致永久损坏：解密失败（换用户/换机器）时保留密文，下次保存被再加密一层；现改为清空并提示重输
- "上次登录"时间失真：每次保存配置都被刷成当前时间；现仅在登录成功时写入
- 退出时丢失最后一次配置修改（防抖定时器尚未触发进程即退出）；新增退出前同步落盘
- 登录"成功但无法上网"时无限重试：重试耗尽的提前返回只退出了异步闭包，重试链继续；现最多 2 次后停止
- 保存配置时 `AfterFunc` 回调与 UI 线程的数据竞争；重试计数器改用原子操作
- 日志目录创建失败时 `GetLogger()` 返回 nil 导致 panic

### Changed
- 版本检查移入后台执行，启动不再被 5 秒超时阻塞（此前每次冷启动在未认证网络下白等）
- 托盘三色图标改为内嵌预渲染资产（`assets/`），删除 217 行运行时 ICO/BMP 编码器
- 登录 URL 参数保持不转义，新增注释说明原因（避免后人"好心修坏"）

### Performance
- 图标资产移除 128/256px 尺寸（托盘显示用不到），单文件缩减约 1MB

## [0.2.0] - 2026-06-23

### Fixed
- Portal v4.0 `AC认证失败`：修复 `wlan_user_ip` 使用本地 IP 而非 BRAS 提供的 IP 导致 AC 拒绝认证
- Portal v4.0 登录 URL 补全 `wlan_area_id` 参数、MAC 使用带横线格式
- 僵尸会话检测：新增双引擎登出功能 + post-login 网络验证 + 自动重试（最多 2 次）
- Captive portal 劫持导致心跳误启动：`CheckInternetAccess` 只接受 2xx 且检查最终 URL host

### Added
- `ACInfo` 新增 `UserIP`、`MACRaw`、`AreaID` 字段
- `fetchACInfo()` 访问外部 URL 触发 captive portal 重定向获取 BRAS 参数
- `portalV4Logout()` / `oldAPILogout()` / `Logout()` 登出功能
- `CheckInternetAccess()` HTTP GET 网络连通性验证

## [0.1.2] - 2026-06-22

### Changed
- 更新赞赏码图片

## [0.1.1] - 2026-06-06

### Fixed
- 补完校园网运营商选项（之前遗漏，现支持移动/联通/电信/校园网四种）
- 旧 API 校园网登录使用正确的 `R6=0` + `terminal_type=1` 参数
- 登录请求自动附加 IPv6 全局单播地址

### Changed
- README 补充「校园网和运营商有什么区别」FAQ
- 界面结构图更新运营商选项列表

## [0.1.0] - 2026-06-03

- 首个发布版本
