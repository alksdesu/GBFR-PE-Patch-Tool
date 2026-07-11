<p align="center">
  <img src="build/appicon.png" width="128" />
</p>

# GBFR 存档修改工具

Granblue Fantasy: Relink (碧蓝幻想：Relink) 存档与运行时修改工具，包含 PE 补丁、因子/祝福生成、副本次数查看、角色使用次数统计、杂项内存修改和怪物增强等功能。
<img src="https://img.shields.io/github/downloads/BitterG/GBFR-PE-Patch-Tool/total"/>

---

> **在大大的基础上加的一些功能如下**（本 Fork 新增，原项目：[BitterG/GBFR-PE-Patch-Tool](https://github.com/BitterG/GBFR-PE-Patch-Tool)）

### 召唤石编辑器

运行时编辑已解锁召唤石内嵌因子，列表式选择，即改即生效并触发游戏保存：

- 种类、阶级（Ⅰ/Ⅱ/Ⅲ）、主因子、主因子等级、副特性、副特性参数全部可改
- 主因子等级按游戏内真实上限 15 级封顶
- 副特性参数所见即所得，实时换算百分比（如输 5 显示「12%，最高 20%」）
- 内置 189 个召唤石、230 个主因子词条、22 个副特性完整档位，支持中文搜索

### 战斗辅助（开关类）

即开即用的运行时补丁，断开连接自动还原，含 53 项：

- 通用战斗：无限闪避、不被破防、自动精准格挡、瞬间破坏部位、无限连携时间、无限消耗品、无限 BOSS 破坏时间、解除召唤石限制、无视奥义槽、长按技能无限持续等
- 任务与生活品质：无限挑战时间、瞬开隐藏宝箱、冻结任务计时、自动开箱、跳过结算画面、终焉武器 100% 掉率、无视钥匙等

### 高级功能（代码注入）

分配代码洞并 Hook 游戏函数实现的复杂功能，含 28 项：

- 战斗：伤害修改（无敌、不死、一击必杀）、冷却修改、状态效果修改、自动精准闪避、眩晕修改、敌人模式修改
- 玩家与物品：玩家指针捕获、选中物品捕获、选中武器捕获
- 生活品质：动作速度修改、自定义过量精通

### 选中物品与武器编辑

在高级功能里开启对应捕获后，游戏内选中目标即可读取并编辑，写入后调用游戏保存函数持久化：

- 选中物品：数量、状态可改
- 选中武器：等级、突破、觉醒、经验、皮肤、HP、攻击、眩晕值、暴击率、幻影弹全部可改
- 武器四条词条与附魔词条（含附魔祝福石）均可改 ID 与等级，附魔为二级指针，自动判空
- 字段偏移与保存函数经逆向核对，写入范围与游戏原生保存逻辑一致

### 角色专属修改

20 个角色的专属机制，整合成选角色后展开的面板，一眼可见每个角色已开启的功能数：

- 例如伊德龙化形态、斗气槽、四重复仇、古兰奥义等级、伊欧观星、秘术漩涡、欧根穿甲弹、瞬间引爆、贝阿朵丽丝不灭之蓝、恩布拉斯克、冈达葛萨永恒之怒、尤达拉哈三重隐匿印等

### 其他

- DLC 无尽黄昏因子补充，补齐 DLC 新增因子数据
- 杂项界面整合，参数化功能改为折叠列表，角色功能改为芯片选择加展开，一屏纵览不再翻半页

---

## 功能
### 支持DLC的因子生成功能已完成（因子生成-新）
##### 演示视频 https://www.bilibili.com/video/BV1zaNn6zELm/

DLC 无尽黄昏更新后可能会有部分功能失效，我会抽时间尝试修复

### 存档相关

- **因子生成** — 搜索选择因子，配置等级、主/副特性，写入输出存档
- **祝福生成** — 搜索选择祝福，配置三个特性和等级，支持队列批量生成
- **副本次数查看** — 自动扫描存档槽位，查看任务/副本通关次数与存档摘要
- **原地修改** — 因子/祝福生成可选直接覆盖输入存档，建议先备份
- **中文翻译** — 因子、祝福、特性、任务名均有简体中文显示

### PE 补丁

- **挑战次数** — 修改任务挑战次数上限，不影响存档
- **点赞数值** — 修改被点赞时获得的数值，被点赞后生效，影响存档
- **自动识别** — 从 Steam 注册表和库目录自动定位游戏 exe
- **备份/恢复** — 补丁前创建 `.bak` 备份，仅 exe，一键恢复

### 运行时功能

- **角色使用次数** — 连接游戏进程，查看和修改角色使用次数
- **任务结算倒计时/零帧开箱** — 修改结算倒计时，设置为 `0` 可零帧开箱
- **脸部符文显示** — 隐藏或恢复紫色皮肤包脸部符文显示
- **在其他皮肤显示紫色符文** — 允许紫色符文在其他皮肤上显示
- **怪物增强** — 怪物多倍血、怪物伤害、昏厥条、Overdrive 状态、奥义接续计时、Link Time、蓝条/紫条控制等
- **挑战次数** — 开启后无视十次连续挑战限制
- **全称号解锁** — 开启后可解锁所有称号，可影响存档(目前持久化时机尚不明确)，
可领取任务奖励、佩戴指定称号、称号选择界面出现多个“未设置”是正常现象)
建议先备份一下存档再使用
- **巴武掉落** — 开启后，每一把至少会从黄金宝箱中获得一把未拥有巴武
- **全队伤害统计** — 根据怪物真实血量变动计算，没有溢出伤害，依赖怪物增强中的功能
- **上限突破** — 先扫描再打开游戏内突破界面后点刷新，切换角色后需刷新，修改后要点保存
- **检查更新** — 从 GitHub Releases 获取最新版本并打开发布页

## 使用说明

### 存档类功能

1. 切换到「因子生成」「祝福生成」或「副本次数」标签页
2. 点击「浏览」选择存档文件，或使用自动扫描的存档槽位
3. 配置需要生成或查看的内容
4. 写入前建议先备份存档

默认存档路径：

```text
C:\Users\<用户名>\AppData\Local\GBFR\Saved\SaveGames\
```

### PE 补丁

1. 关闭游戏
2. 切换到「补丁修改」标签页
3. 自动识别或手动选择 `granblue_fantasy_relink.exe`
4. 点击「备份」创建 `.bak`
5. 输入数值并点击「应用」
6. 启动游戏验证效果

### 运行时功能

1. 启动游戏并进入存档
2. 切换到「角色次数统计」「杂项」或「怪物增强」标签页
3. 连接或刷新游戏进程状态
4. 开启、应用或恢复需要的功能
5. 重启游戏后需要重新连接并重新设置

### 怪物增强说明

- 「怪物多倍血」和「鳄鱼多倍血」输入 `10` 表示等效 `10 倍血`
- 「怪物 Overdrive 状态」支持 `1 满红条`、`4 满黄条` 和「自动OD」
- 「锁定」会持续写入状态，「应用」只写入一次后恢复原始指令
- 「自动OD」会在非红条时写入一次满黄条，红条中不重复触发
- 「奥义接续计时」默认 `3 秒`，可输入自定义秒数并恢复默认
- 部分怪物增强功能依赖内置 `patch_core.dll`

## 实现简述

- PE 补丁直接修改 exe 中指定指令的立即数，并保留备份用于恢复
- 存档功能基于 FlatBuffer 解析与 XXHash64 校验写回
- 运行时功能通过打开游戏进程并读写内存实现
- 怪物增强中简单功能由 Go 直接写内存，复杂功能通过 `patch_core.dll` 注入并写入跳板或补丁
- `patch_core.dll` 仅输出调试信息，不弹出对话框

## 恢复方法

任选其一：

- 工具内点击「恢复」从 `.bak` 还原
- Steam → 游戏属性 → 本地文件 → 验证游戏文件完整性

## 环境要求

- Go 1.23+（必须 amd64 版本，游戏为 64 位进程）
- Node.js 16+
- [Wails CLI v2](https://wails.io/docs/gettingstarted/installation)
- Visual Studio / MSBuild（修改 `src_dll/patch_core` 后需要编译 DLL）

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

## 编译打包

```bash
# 安装前端依赖
cd frontend && npm install && cd ..

# 开发模式
wails dev

# 完整构建
wails build

# 仅构建 Go，跳过前端编译
wails build -s
```

如修改 `src_dll/patch_core`，先用 Visual Studio 构建 `Release x64`，确认输出覆盖：

```text
build/bin/patch_core.dll
```

### Windows 一键编译

项目根目录提供 `build-windows.bat`：

```powershell
.\build-windows.bat
```

如遇 `go: no such tool "compile"`，指定正确的 GOROOT：

```powershell
$env:GOROOT="D:\GO1.26.1"; wails build -s
```

构建产物：

```text
build/bin/GBFR PE Patch Tool.exe
```

## 数据说明

因子、祝福和任务翻译数据位于 `data/` 目录，并嵌入最终二进制：

| 文件 | 说明 |
|------|------|
| `sigils.json` | 因子定义 |
| `traits.json` | 因子特性定义 |
| `secondary-trait-rules.json` | 副特性兼容性规则 |
| `wrightstones.json` | 祝福定义 |
| `wrightstone_traits.json` | 祝福特性定义 |
| `quest_names_i18n.csv` | 任务 ID 到中文名映射 |

中文翻译主要位于：

- `sigil_locale.go`
- `wrightstone_locale.go`

## 项目结构

```text
.
├── app.go                         # PE 补丁、运行时内存修改、Steam 路径扫描、更新检测
├── main.go                        # Wails 入口、窗口配置、祝福 CLI 模式
├── save_app.go                    # 存档扫描、任务次数读取、中文任务名映射
├── save_parse.go                  # 存档 FlatBuffer 解析、摘要提取
├── sigil_data.go                  # 因子/特性数据加载
├── sigil_ctdata.go                # 因子合成表数据
├── sigil_gen.go                   # 因子生成业务逻辑
├── sigil_locale.go                # 因子/特性中文翻译
├── sigil_store.go                 # 因子槽位读写与校验
├── wrightstone_data.go            # 祝福/祝福特性数据加载
├── wrightstone_gen.go             # 祝福生成业务逻辑与 CLI 模式
├── wrightstone_locale.go          # 祝福/特性中文翻译
├── wrightstone_store.go           # 祝福槽位读写
├── damage_overlay_windows.go      # 伤害统计悬浮窗
├── data/                          # 嵌入式 JSON/CSV 数据
├── build/                         # 图标、manifest、内置 DLL 与构建产物
│   └── bin/
│       └── patch_core.dll         # 怪物增强注入 DLL
├── src_dll/
│   ├── patch_core.slnx            # patch_core Visual Studio 解决方案
│   ├── patch_core/                # patch_core DLL 源码
│   └── thirdparty/libmem/         # DLL 使用的 libmem 依赖
├── frontend/
│   ├── package.json
│   ├── vite.config.js
│   ├── wailsjs/                   # Wails 生成的前端绑定
│   └── src/
│       ├── main.js
│       ├── App.vue
│       ├── style.css
│       └── components/
│           ├── PatchTool.vue           # 主窗口与标签导航
│           ├── SigilGenerator.vue      # 因子生成
│           ├── WrightstoneGenerator.vue # 祝福生成
│           ├── SaveEditor.vue          # 副本次数
│           ├── CharaStats.vue          # 角色使用次数
│           ├── MiscTools.vue           # 杂项运行时修改
│           └── MonsterEnhance.vue      # 怪物增强
├── build-windows.bat              # Windows 构建脚本
├── wails.json                     # Wails 配置
├── go.mod
└── README.md
```

## 支持
如果本项目帮你省了时间或带来更多乐趣，欢迎请我喝杯咖啡。完全自愿，不是契约——捐赠不会影响功能优先级，也不会改变问题处理的顺序。
— 微信支付（扫码）
<p align="center">
  <img src="./QRcode.png" width="256" />
</p>

## 免责声明

本工具仅供学习研究使用。使用本工具修改游戏文件、存档或运行时内存所产生的一切后果由使用者自行承担。

存档因子相关部分解析方法来自 [GBFR-Sigil-Generator](https://github.com/Xzire91x/GBFR-Sigil-Generator)。

祝福添加相关部分解析方法来自 [GBFR-Wrightstone-Generator](https://github.com/Xzire91x/GBFR-Wrightstone-Generator)。

存档解析基于 [GBFRDataTools.SaveFile](https://github.com/Nenkai/GBFRDataTools/tree/master/GBFRDataTools.SaveFile)。
