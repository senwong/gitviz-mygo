# Gitviz MyGo

使用 Go 和 [MyGo](https://github.com/egoist/mygo) 构建的原生桌面 Git 图形客户端。

这是 Gitviz 的 Go 实现，与 [Rust 版](https://github.com/senwong/gitviz) 独立维护。

## 功能

- 多仓库切换、按名称搜索仓库，支持打开 VS Code `.code-workspace` 文件。
- 提交关系图，展示本地分支、远程分支和 tag；支持提交与引用搜索高亮。
- 行内提交详情与文件树，点击文件打开大幅 diff。
- 文本 diff 搜索、多行选择与复制；图片修改前后预览。
- 分支、提交和 tag 的右键操作。
- 后台 Fetch、远程分支和 tag 更新检查，以及当前分支的 ahead/behind 状态。
- 多套明暗主题、仓库设置、上次打开仓库恢复。

## 构建与运行

环境：macOS、Git、Go 1.27.3 或更新版本。MyGo 版本固定在 `go.mod` 中。

```sh
./script/build

# 打开仓库或工作区
./build/gitviz-mygo /path/to/repository
./build/gitviz-mygo /path/to/project.code-workspace
```

不传入路径时，恢复上次使用的仓库和工作区。

## 验证

```sh
# 格式检查、go vet、race 测试及构建
./script/verify
```

部分真实仓库预览和完整主题截图检查需要显式设置测试环境变量；普通验证使用临时测试仓库。

## macOS 应用打包

```sh
./script/package
open build/gitviz.app
```

应用使用本机临时签名，产物位于 `build/`，不纳入版本管理。

修改 `assets/AppIcon.svg` 后，可运行 `./script/icon` 重新生成图标；这一步需要 `rsvg-convert` 和 macOS 的 `iconutil`。

## 常用快捷键

| 快捷键 | 操作 |
| --- | --- |
| ⌘P | 仓库选择 |
| ⌘⇧P | 命令面板 |
| ⌘B | 分支选择 |
| ⌘F | 搜索提交或当前文件 diff |
| ⌘G / ⌘⇧G | 下一个 / 上一个匹配 |
| ⌘R | 刷新 |
| ⌘, | 设置 |
| ⌘W | diff 中返回提交图；提交图中关闭窗口 |
| ⌘Q | 退出应用 |

设置保存在 `~/.config/gitviz-mygo/`。Fetch 更新远程引用和 tag，不会自动将远程提交合并进当前本地分支。
