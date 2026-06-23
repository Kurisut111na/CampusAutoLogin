# Changelog

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
