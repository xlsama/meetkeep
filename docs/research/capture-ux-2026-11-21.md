# 会议记录应用「采集与录入」业内实现调研

调研日期：2026-11-21。范围：A 按 App 录音、B 线下会议（只有麦克风）、C 上传/导入、D 录制过程交互。

写法约定：
- 每条事实后附一手来源（官方文档、帮助中心、changelog、源码、开发者技术博客）。评测/媒体文章只作线索，出现时会标「线索」。
- 「未核实」：找过但没有一手来源确认。「仅营销文案」：只有官网宣传，没有文档。
- 「推断」：根据一手资料推出来的结论，不是原文。

---

## A. 按 App 录制音频

### A1. macOS：Core Audio process tap 与 ScreenCaptureKit

#### Core Audio process tap（macOS 14.2+）

- **接口形态**：用 `CATapDescription` 描述一个 tap，调 `AudioHardwareCreateProcessTap` 创建；tap 本身不出数据，要放进一个（可设为私有的）聚合设备（aggregate device），再像读麦克风一样用 IOProc 读。可配置：抓哪些进程（`processes`）、公有/私有（`isPrivate`）、是否把被抓进程静音（`muteBehavior`，静音后该进程的声音只进 tap 不出扬声器）、mono/stereo 混缩、`isExclusive`（列表语义反转为「排除这些进程」）、指定输出设备与 stream。Apple 示例要求 macOS 14.2+。（来源：https://developer.apple.com/documentation/coreaudio/capturing-system-audio-with-core-audio-taps）
- **按进程 = 按 Core Audio 进程对象，不是 PID**：先用 `kAudioHardwarePropertyTranslatePIDToProcessObject` 把 PID 转成 AudioObjectID。进程只有在连上 Core Audio（打开过音频设备）之后才有进程对象，所以「刚启动、还没出过声」的 App 抓不到。（来源：https://victorantos.com/posts/macos-process-taps-tapelark/ ；https://github.com/insidegui/AudioCap）
- **一个 tap 可同时抓多个进程**（`processes` 是数组，`stereoMixdownOfProcesses:` 接收进程列表），也可以「全局抓、排除若干进程」（exclusive）。cpal 的 macOS loopback 就是用空进程列表 + `setExclusive(true)` 抓整机输出。（来源：https://developer.apple.com/documentation/coreaudio/capturing-system-audio-with-core-audio-taps ；https://victorantos.com/posts/macos-process-taps-tapelark/ ；https://github.com/RustAudio/cpal/blob/master/src/host/coreaudio/macos/loopback.rs）
- **权限**：Info.plist 必须有 `NSAudioCaptureUsageDescription`；第一次从含 tap 的聚合设备开始录音时系统弹窗，授予「系统音频录制」权限。（来源：https://developer.apple.com/documentation/coreaudio/capturing-system-audio-with-core-audio-taps） 在系统设置里对应「隐私与安全性 › 屏幕与系统音频录制」下的「System Audio Recording Only（仅系统录音）」一档，比 ScreenCaptureKit 需要的「Screen Recording」窄。（来源：https://systemaudiokit.lucaspiera.com/docs/permissions ；https://victorantos.com/posts/macos-process-taps-tapelark/）
- **没有公开 API 查询/申请该权限**：AudioCap 和 SystemAudioKit 都靠 dlopen 私有 TCC 框架，调 `TCCAccessPreflight` / `TCCAccessRequest`（服务名 `kTCCServiceAudioCapture`），并说明不能用于 Mac App Store。（来源：https://github.com/insidegui/AudioCap ；https://github.com/insidegui/AudioCap/blob/main/AudioCap/ProcessTap/AudioRecordingPermission.swift ；https://systemaudiokit.lucaspiera.com/docs/permissions）
- **被拒绝 = 静音**：权限被拒时所有调用仍返回 noErr，IOProc 只送全零数据；没有 `NSAudioCaptureUsageDescription` 时连弹窗都没有，只有静音。实践做法是检测「来源在播放但持续数字静音」并提示用户。（来源：https://victorantos.com/posts/macos-process-taps-tapelark/ ；https://systemaudiokit.lucaspiera.com/docs/permissions）
- **弹窗是阻塞的**：创建 tap/聚合设备不弹窗，第一次创建 IOProc 或 `AudioDeviceStart` 时弹，并阻塞到用户回答，要放到非主线程并加超时。（来源：https://victorantos.com/posts/macos-process-taps-tapelark/）
- **授权绑定代码签名**：ad-hoc 签名每次构建签名都变，授权会丢；Developer ID 签名的 App 能保留授权。（来源：https://systemaudiokit.lucaspiera.com/docs/permissions） 反例：Splitwave 未签名，README 写明每次更新后 Screen Recording 权限都会被重置，要用户手动删掉再加回。（来源：https://github.com/Horuse/Splitwave）
- **macOS 26 新增**：`CATapDescription.bundleIDs` 和 `processRestoreEnabled` 两个属性。（来源：https://sdkdiffs.furnacecreek.org/macos-26.0/CoreAudio.html ；https://developer.apple.com/documentation/coreaudio/catapdescription/bundleids） 语义（是否按 bundle ID 自动覆盖同一 App 的新进程、进程重启后是否自动恢复）Apple 文档页没有正文，**未核实**。
- **「谁在出声」可以直接问 coreaudiod**：从 14.2 起，`kAudioHardwarePropertyProcessObjectList` 列出所有音频活跃进程，每个进程对象有 `kAudioProcessPropertyPID`、`kAudioProcessPropertyBundleID`、`kAudioProcessPropertyIsRunningInput`、`kAudioProcessPropertyIsRunningOutput`（正在播放）。不需要辅助功能权限，也不用私有 API。（来源：https://macnotetaker.com/blog/which-app-is-using-mic-coreaudio-process-objects ；https://docs.rs/objc2-core-audio/latest/objc2_core_audio/constant.kAudioProcessPropertyIsRunningOutput.html）
- **浏览器不是自己出声**：Chrome 的声音来自 helper 进程（bundle ID 形如 `com.google.Chrome.helper`），Safari 来自 WebKit GPU 进程（`com.apple.WebKit.GPU`，任何内嵌 WebKit 的 App 都会报这个）。要按「App」展示，需要按 bundle ID 前缀把 helper 归并回父 App；WebKit GPU 进程只能「大概是 Safari」。（来源：https://macnotetaker.com/blog/which-app-is-using-mic-coreaudio-process-objects ；https://victorantos.com/posts/macos-process-taps-tapelark/）
- **最低系统版本口径不一**：Apple 示例写 14.2；AudioCap 写 14.4（「With macOS 14.4, Apple introduced new API…」）；cpal 的系统 loopback 写 macOS > 14.6；Granola 要求 14.2；知了要求 14.4。（来源：https://developer.apple.com/documentation/coreaudio/capturing-system-audio-with-core-audio-taps ；https://github.com/insidegui/AudioCap ；https://github.com/RustAudio/cpal/blob/master/CHANGELOG.md ；https://docs.granola.ai/help-center/getting-started/setting-up-granola-for-the-first-time ；https://github.com/wquguru/cicada/blob/master/README.zh-CN.md） 14.2–14.4 之间具体差异**未核实**。
- **运行中的坑（来自一个 2026-10 上线的 tap 录音 App 的开发博客）**：Swift 闭包注册的属性监听器无法移除（每次桥接成新 block），要用 C 函数指针 API；IOProc 线程里不能分配/加锁/打日志；长时间录制压测中，系统合盖/菜单睡眠无法被 idle-sleep assertion 阻止（`IOPMLib.h` 明确写了），恢复后要记录断档；每个聚合设备用新 UID，因为销毁是异步的、复用 UID 会冲突。（来源：https://victorantos.com/posts/macos-process-taps-tapelark/）

#### ScreenCaptureKit（macOS 13+ 有音频）

- `SCStreamConfiguration.capturesAudio` 打开音频；内容过滤器可按 App/窗口过滤，`excludesCurrentProcessAudio` 排除自己。（来源：https://developer.apple.com/documentation/screencapturekit/scstreamconfiguration/capturesaudio）
- 需要「屏幕录制」权限（比 tap 宽，用户看到的是录屏授权）。（来源：https://systemaudiokit.lucaspiera.com/docs/permissions ；https://victorantos.com/posts/macos-process-taps-tapelark/）
- macOS 15 起，不经系统选择器（`SCContentSharingPicker`）直接抓屏的 App 会周期性弹出「…is requesting to bypass the system private window picker and directly access your screen and audio」的二次确认；开发者论坛里有「每隔几天」的报告，媒体报道正式版为每月一次。（来源：https://developer.apple.com/forums/tags/screencapturekit?page=2&sortBy=newest ；https://developer.apple.com/documentation/screencapturekit/sccontentsharingpicker）（频率出自 TidBITS 报道，属线索：https://tidbits.com/2024/08/19/apple-reduces-excessive-sequoia-permission-requests-shifts-to-monthly/；Apple 无正式文档说明频率，**未核实**。）
- 实际采用：OBS 30+ 的「macOS Audio Capture」源（macOS 13+）可选「整机」或「单个应用」（来源：https://obsproject.com/kb/macos-desktop-audio-capture-guide）；Splitwave 的 macOS 按 App 捕获走 ScreenCaptureKit（来源：https://github.com/Horuse/Splitwave）；TranscribeX 打开 App 选择器时请求「屏幕与系统音频录制」权限（来源：https://www.transcribex.io/docs/app-recording/）；Krisp 的屏幕录制模式在 macOS 上要求屏幕录制权限（来源：https://help.krisp.ai/hc/en-us/articles/30146816045980-Record-your-meetings-with-Krisp-AI-Note-Taker）；Notion 要求「system audio and screen recording permissions」（来源：https://www.notion.com/help/ai-meeting-notes）。

#### 对比小结

| 维度 | Core Audio tap | ScreenCaptureKit |
|---|---|---|
| 最低系统 | 14.2（各家口径 14.2–14.6） | 13（音频） |
| 权限 | 系统设置里的「仅系统录音」一档；无公开检查 API | 屏幕录制；macOS 15 有周期性二次确认 |
| 粒度 | 进程对象列表（可多进程）/全局排除 | App、窗口、显示器过滤 |
| 拒绝时表现 | 静默零数据 | 有可查询的授权状态（Splitwave 用 `CGPreflightScreenCaptureAccess` 非弹窗检查）（来源：https://github.com/Horuse/Splitwave/blob/main/docs/CONCEPT.md） |
| 视频 | 无 | 有 |
| 能否把被抓进程静音 | 能（muteBehavior） | 未查到 |

#### Granola、Notion 怎么做

- Granola（macOS）：**不能**只抓单个 App，抓的是混合后的系统音频，会议期间播放的音乐也会进转写，哪怕走的是另一个输出设备。（来源：https://docs.granola.ai/help-center/taking-notes/transcription） 要求 macOS 14.2+，权限为麦克风 + 屏幕与系统音频录制。（来源：https://docs.granola.ai/help-center/getting-started/setting-up-granola-for-the-first-time ；https://docs.granola.ai/help-center/troubleshooting/transcription-issues）
- Notion：桌面端用「system audio + mic」，要求 macOS 13+，需要「system audio and screen recording permissions」。（来源：https://www.notion.com/help/ai-meeting-notes）

### A2. Windows：Process Loopback API

- **接口**：`ActivateAudioInterfaceAsync` 传 `VIRTUAL_AUDIO_DEVICE_PROCESS_LOOPBACK`，激活参数 `AUDIOCLIENT_ACTIVATION_PARAMS`（类型 `AUDIOCLIENT_ACTIVATION_TYPE_PROCESS_LOOPBACK`），里面的 `AUDIOCLIENT_PROCESS_LOOPBACK_PARAMS` 只有两个字段：`TargetProcessId`、`ProcessLoopbackMode`。（来源：https://learn.microsoft.com/en-us/windows/win32/api/mmdeviceapi/nf-mmdeviceapi-activateaudiointerfaceasync ；https://learn.microsoft.com/en-us/windows/win32/api/audioclientactivationparams/ns-audioclientactivationparams-audioclient_process_loopback_params）
- **包含/排除进程树**：`PROCESS_LOOPBACK_MODE_INCLUDE_TARGET_PROCESS_TREE`（只要该进程及其子进程的渲染流）/ `..._EXCLUDE_TARGET_PROCESS_TREE`（除该进程树外的全部系统声音）。注意没有「只要这个进程、不含子进程」的官方模式（wasapi-rs 文档把 `include_tree=false` 描述为「只抓目标进程」，与微软枚举定义不一致，**存疑**）。（来源：https://learn.microsoft.com/en-us/windows/win32/api/audioclientactivationparams/ne-audioclientactivationparams-process_loopback_mode ；https://docs.rs/wasapi/latest/wasapi/struct.AudioClient.html）
- **不绑定输出设备**：抓的是进程的渲染流，不依赖某个端点，用户换耳机/扬声器不需要重建多个 IAudioClient；目标进程没有渲染流时得到静音。（来源：https://learn.microsoft.com/en-us/samples/microsoft/windows-classic-samples/applicationloopbackaudio-sample/）
- **一次激活只对应一个 PID**：要录多个 App，就开多个客户端自己混音。佐证：OBS 每个「Application Audio Capture」源各管一个应用，指南让用户为每个应用重复添加。（来源：https://obsproject.com/kb/application-audio-capture-guide） 并发客户端数量上限**未核实**。
- **能力缺失**：`GetMixFormat` 返回 `E_NOTIMPL`，必须自己指定格式（官方示例硬编码 44.1 kHz/16 bit/双声道）（来源：https://learn.microsoft.com/en-us/answers/questions/1125409/loopbackcapture-activateaudiointerfaceasync-with-v）；wasapi-rs 列出这种模式下不可用的方法：`get_mixformat`、`is_supported`、`get_device_period`、`calculate_aligned_period_near`、`get_current_padding` 等，`get_buffer_size` 返回离谱大值；只能 Capture + Shared。（来源：https://docs.rs/wasapi/latest/wasapi/struct.AudioClient.html）
- **最低版本口径冲突（重要）**：
  - 结构体/枚举文档写「Minimum supported client: Windows 10 Build 20348」，官方示例 README 写「requires Windows 10 build 20348 or later」。（来源：https://learn.microsoft.com/en-us/windows/win32/api/audioclientactivationparams/ns-audioclientactivationparams-audioclient_process_loopback_params ；https://learn.microsoft.com/en-us/windows/win32/api/audioclientactivationparams/ne-audioclientactivationparams-process_loopback_mode ；https://learn.microsoft.com/en-us/samples/microsoft/windows-classic-samples/applicationloopbackaudio-sample/）
  - `ActivateAudioInterfaceAsync` 文档写「Starting with Windows 10 Build 20438」。（来源：https://learn.microsoft.com/en-us/windows/win32/api/mmdeviceapi/nf-mmdeviceapi-activateaudiointerfaceasync）
  - OBS 写「Windows 10 (Version 2004 or later) and Windows 11」；Splitwave、知了也写 Win10 2004+。（来源：https://obsproject.com/kb/application-audio-capture-guide ；https://github.com/Horuse/Splitwave ；https://github.com/wquguru/cicada/blob/master/README.zh-CN.md）
  - screenpipe 源码把门槛定为 build 20348（注释「Windows 11」），低于此直接报错、不静默回退到整机 loopback。（来源：https://github.com/screenpipe/screenpipe/blob/fe296feb/crates/screenpipe-audio/src/core/process_tap/windows.rs）
  - 微软问答里有用户反映官方示例「only worked on Windows 11」。（来源：https://learn.microsoft.com/en-us/answers/questions/1125409/loopbackcapture-activateaudiointerfaceasync-with-v）
  - 结论：真实下限**存疑**（20348 是 Server 2022 的 build 号，Windows 10 2004 是 19041）。稳妥做法是运行时尝试激活，失败再降级。
- **多进程 App（浏览器、Electron、WebView2）**：OBS 取所选窗口所属进程的 PID 并固定用 INCLUDE_TARGET_PROCESS_TREE。（来源：https://github.com/obsproject/obs-studio/blob/master/plugins/win-wasapi/win-wasapi.cpp） screenpipe 的注释说明：检测到在用麦克风的常是工具子进程（Chrome Audio Service、Electron helper、WebView2 子进程），要先沿父链走到 App 根进程再用 include tree。（来源：https://github.com/screenpipe/screenpipe/blob/fe296feb/crates/screenpipe-audio/src/core/process_tap/windows.rs）
- **兼容性例外**：Granola Windows 版「大多数会议 App 无论从哪个设备出声都能听到」（推断是按进程抓），但 Webex、WhatsApp、8x8 Work、3CX 改为监听 Windows 默认输出设备，Teams 也可能回退；会议 App 走非默认设备时就听不到对方。（来源：https://docs.granola.ai/help-center/troubleshooting/transcription-issues） OBS 也说明部分应用（如 Valorant 语音）与 Application Audio Capture 不兼容，要用虚拟声卡。（来源：https://obsproject.com/kb/application-audio-capture-guide）
- **Teams 回声**：Windows 上 Teams 桌面版外放时，对方的话会在麦克风轨再出现一次；Granola 的回声消除检测不到 Teams 的这种播放方式，已报给微软和 Chromium，建议戴耳机或用网页版。（来源：https://docs.granola.ai/help-center/troubleshooting/transcription-issues）
- **权限**：Windows 不弹系统音频授权。Granola：Windows 上权限自动授予（只有麦克风隐私开关）；Krisp：Windows 屏幕录制无需额外权限。（来源：https://docs.granola.ai/help-center/getting-started/setting-up-granola-for-the-first-time ；https://docs.granola.ai/help-center/troubleshooting/transcription-issues ；https://help.krisp.ai/hc/en-us/articles/30146816045980-Record-your-meetings-with-Krisp-AI-Note-Taker）
- **选择器数据来源（推断，基于 API 文档）**：音频会话状态有 Active（有流在跑）/Inactive/Expired，系统音量合成器 Sndvol 同时显示 Active 和 Inactive 会话，Expired 后才消失（来源：https://learn.microsoft.com/en-us/windows/win32/api/audiosessiontypes/ne-audiosessiontypes-audiosessionstate）；`IAudioMeterInformation` 提供峰值电平（端点级，未静音前的电平）（来源：https://learn.microsoft.com/en-us/windows/win32/api/endpointvolume/nn-endpointvolume-iaudiometerinformation）。Splitwave 用 `windows` crate 做音频会话枚举和 exe 图标提取。（来源：https://github.com/Horuse/Splitwave）

### A3. Rust 里用哪个 crate

| 需求 | 可选 | 状态/证据 |
|---|---|---|
| Windows 按进程 loopback | `wasapi`（HEnquist/wasapi-rs） | 有 `AudioClient::new_application_loopback_client(process_id, include_tree)` 和 `record_application.rs` 示例。（来源：https://docs.rs/wasapi/latest/wasapi/struct.AudioClient.html ；https://docs.rs/crate/wasapi/latest/source/examples/record_application.rs） |
| Windows 按进程 loopback（自己写） | `windows` crate 原始绑定 | screenpipe、Splitwave 都直接用 `windows` crate 调 `ActivateAudioInterfaceAsync`。（来源：https://github.com/screenpipe/screenpipe/blob/fe296feb/crates/screenpipe-audio/src/core/process_tap/windows.rs ；https://github.com/Horuse/Splitwave） |
| Windows 整机 loopback | `cpal` / `wasapi` | Hyprnote 用 `wasapi` 对默认渲染设备做整机 loopback（不按进程）。（来源：https://github.com/fastrepl/hyprnote/blob/6ac06c35/crates/audio/src/speaker/windows.rs） |
| `cpal` 按进程 | 不支持 | changelog 只有 macOS「loopback recording (recording system audio output) on macOS > 14.6」，实现为全局 exclusive tap；ScreenCaptureKit host PR #894 自 2024 年至今未合并。（来源：https://github.com/RustAudio/cpal/blob/master/CHANGELOG.md ；https://github.com/RustAudio/cpal/blob/master/src/host/coreaudio/macos/loopback.rs ；https://github.com/RustAudio/cpal/pull/894） |
| macOS process tap | `objc2-core-audio`、`cidre` | cpal 用 `objc2-core-audio` 的 `CATapDescription`；`cidre` 有 `core_audio::TapDesc` 与 tap 封装。（来源：https://github.com/RustAudio/cpal/blob/master/src/host/coreaudio/macos/loopback.rs ；https://docs.rs/cidre/latest/cidre/core_audio/struct.TapDesc.html） |
| macOS ScreenCaptureKit | 自写 Swift 静态库 | Splitwave 明确不用 `screencapturekit` crate（其 `swift build` 在 CLT 上因 PackageDescription ABI 不匹配失败），改为 build.rs 里 `swiftc` 编自己的 Swift 静态库再 FFI。（来源：https://github.com/Horuse/Splitwave/blob/main/docs/CONCEPT.md） |

- 已知坑：cpal 0.18.2 WASAPI 主机在端点通知回调线程里把线程池线程初始化成 STA，之后激活默认设备报 `RPC_E_CHANGED_MODE`，只能重启进程。（来源：https://github.com/RustAudio/cpal/issues/1402）

### A4. 选择器交互：各家长什么样

**Audio Hijack（Rogue Amoeba）**
- 以「会话」组织，管线由 Application 源块 → Recorder → Output 组成；新建会话先选模板（如「Web Audio」模板默认选 Safari）。（来源：https://rogueamoeba.com/support/manuals/audiohijack/?print=true）
- 源选择器里有「Special Sources」分组：Background Sounds、Control Center、Finder、Notification Center、Siri、Sound Effects、Spoken Content、VoiceOver，每项背后是若干系统进程/bundle ID。按住 Option 点选择器会多出「Running Processes」子菜单，列出所有可能出声的进程（「not all are likely to produce audio」）。（来源：https://rogueamoeba.com/support/knowledgebase/?showArticle=AH-HiddenSources）
- 4.4+ 源块有「Audio Device Selection」：只抓发往默认输出的声音 / 发往所有输出 / 指定设备和声道。普通 App 默认第一项，**识别为 VoIP 的 App 默认「所有输出设备」**。（来源：https://www.rogueamoeba.com/support/knowledgebase/?showArticle=AudioHijack-AudioDeviceSelection）
- 浏览器多个标签同时出声时**无法分离单个页面**；官方给的绕法是用 Safari「添加到程序坞」把网站做成独立 Web App，再单独选它。（来源：https://www.rogueamoeba.com/support/knowledgebase/?product=Audio+Hijack&showArticle=Misc-WebAppAudio）
- Recorder 自动在 2 GB 或 24 小时处切文件；有静音阈值，可删除静音、按静音切文件或停止录音。（来源：https://rogueamoeba.com/support/manuals/audiohijack/?print=true）
- 选择器里是否显示「正在出声」标记、是否显示图标：**未核实**（手册截图无文字说明）。

**Loopback（Rogue Amoeba）**
- 虚拟设备编辑器的 Sources 列点「+」，从应用或物理设备中选；每个源有开关、音量滑杆、静音按钮；应用源有「Mute when capturing」勾选框，默认开（被抓的 App 不再从扬声器出声）。（来源：https://www.rogueamoeba.com/support/manuals/loopback/?page=sources）

**OBS Studio**
- Windows：OBS 28 起有「Application Audio Capture (BETA)」源，下拉列出窗口（代码里 `ms_fill_window_list(..., INCLUDE_MINIMIZED, ...)`，包含最小化窗口），另有「窗口匹配优先级」：标题 / 窗口类 / 可执行文件；标题会变的应用（Chrome、Spotify）建议按可执行文件匹配。每个应用一个源；用了按 App 源要关掉全局桌面音频，避免回声。OBS 30.1 起窗口/游戏捕获有「Capture Audio (BETA)」勾选框。（来源：https://obsproject.com/kb/application-audio-capture-guide ；https://github.com/obsproject/obs-studio/blob/master/plugins/win-wasapi/win-wasapi.cpp）
- macOS：「macOS Audio Capture」源可选整机或单个应用。（来源：https://obsproject.com/kb/macos-desktop-audio-capture-guide）

**Splitwave（Horuse/Splitwave，MIT，Tauri + Svelte + Rust）**
- 节点图：麦克风、系统音频、按 App 捕获、WAV 文件作为输入节点，接效果链再接扬声器或录音。三平台「App icons in the picker」都打勾。macOS 用 ScreenCaptureKit，Windows 用 Process Loopback（Win10 2004+），Linux 用 PipeWire。（来源：https://github.com/Horuse/Splitwave）
- 设计原则：后端不支持时直接报错，不偷偷换成别的采样率/设备/格式；某路输入超过 150 ms 没数据就补零继续。（来源：https://github.com/Horuse/Splitwave/blob/main/docs/CONCEPT.md）

**Windows 音量合成器**
- 系统 Sndvol 按「音频会话」列出应用，会话 Active 和 Inactive 都显示，Expired 才移除。（来源：https://learn.microsoft.com/en-us/windows/win32/api/audiosessiontypes/ne-audiosessiontypes-audiosessionstate）
- OBS 官方指南提到可在 Windows 音量合成器里把任意应用的输出设备指到 VB-Cable（即 Win11 支持按 App 指定输出设备）。（来源：https://obsproject.com/kb/application-audio-capture-guide） 微软官方帮助页**未取到**，新版合成器的具体布局（图标、电平条）**未核实**。

**支持选 App 的会议记录工具**
- **TranscribeX（macOS）**：首页「Recording App」卡片 → 「Select an App to record」面板，只列**正在运行**的 App，有「Search apps…」搜索和「Refresh」；选中后进入「App Audio Recording」面板，勾「Microphone」决定是否同时录麦克风，并可选具体麦克风；点 Record 开始。检测到的会议 App（Zoom、Teams、Slack、Discord、Webex、浏览器）有独立卡片、可免费录；任意 App 选择是 Pro 功能。停止：页面 Stop 或菜单栏「Stop Recording <App 名>」，收尾时菜单显示「Saving App Recording…」。可选「Suppress Speaker Echo in Mic」（外放时把麦克风里回灌的对方声音静音）。（来源：https://www.transcribex.io/docs/app-recording/）
- **MacWhisper**：不让用户直接选进程，而是维护一个「被观察的应用」下拉多选列表：Arc、Brave、Chime、Chrome、Discord、Edge、Firefox、Safari、Skype、Slack、Teams、Telegram、Webex、WhatsApp、Zoom；FaceTime 因系统隐私限制不支持。检测到会议弹「Meeting Detected」通知，10 秒不操作自动消失；对方所有人当作一个说话人。需要屏幕录制 + 麦克风权限。（来源：https://docs.macwhisper.com/article/30-record-meetings）
- **Jamie**：没有录音源选择器，但有「Excluded apps」排除列表（「Select an app」添加），被排除的应用不触发提醒和自动开始。（来源：https://docs.meetjamie.ai/getting-started/notifications ；https://docs.meetjamie.ai/getting-started/auto-start-recording）
- **Granola / Notion / 知了**：都没有按 App 选择器。知了「直接听系统声音」，并识别会议软件的通话状态用来自动结束。（来源：https://docs.granola.ai/help-center/taking-notes/transcription ；https://www.notion.com/help/ai-meeting-notes ；https://github.com/wquguru/cicada/blob/master/README.zh-CN.md ；https://cicada.indaloop.app/changelog/）
- 录制中能否切换所选 App：上面各家文档都没有描述，**未核实**。Krisp 的屏幕录制模式写明录制中可以切换到别的屏幕。（来源：https://help.krisp.ai/hc/en-us/articles/30146816045980-Record-your-meetings-with-Krisp-AI-Note-Taker）

---

## B. 线下会议（只有麦克风）

### Otter
- **入口**：网页右上「Import」旁的「Record」；手机 App 右下角蓝色麦克风图标。新记录默认标题「Note」，录中录后都能改名。（来源：https://help.otter.ai/hc/en-us/articles/360048269733-Record-a-conversation）
- **会中**：实时转写，需要联网；回放条上有 Pause / Resume / Stop。断网时继续录、缓存在浏览器或手机里，恢复网络后补传、补转写，期间不显示实时文字；建议上行至少 512 kbps。（来源：https://help.otter.ai/hc/en-us/articles/360048269733-Record-a-conversation ；https://help.otter.ai/hc/en-us/articles/31672594631063-Getting-the-most-out-of-Otter-ai-Best-Practices-for-in-person-recordings）
- **说话人**：说话人识别在**录音停止后**才开始，会中**不能**标注说话人（Zoom 会议用 Notetaker 机器人才有实时名字）。新声音显示为「Speaker 1/2/3」，给一个 Speaker # 打标签后同编号的所有段落自动更新；识别不出的是「Unknown speaker」。手动打过标签的显示绿勾，并用于以后的会议自动识别；工作区成员的声纹可共享。（来源：https://help.otter.ai/hc/en-us/articles/360048465453-Tagging-speaker-names-in-a-conversation ；https://help.otter.ai/hc/en-us/articles/21665587209367-Speaker-Identification-Overview）
- **会后改名/合并**：改名在所有会议生效；「Rematch」用最新声纹给旧会议里的 Speaker # 重新匹配；2026-07 起灰度「Speaker Management」：多选说话人合并到一个、可撤销合并（Undo merge）、永久隐藏（不可恢复）。文档提示：摘要、大纲、Otter Chat、搜索不一定反映改名，需要「Regenerate Summary」。（来源：https://help.otter.ai/hc/en-us/articles/360048465453-Tagging-speaker-names-in-a-conversation ；https://help.otter.ai/hc/en-us/articles/40586357592471-Speaker-Management）

### Granola（手机端，iOS/Android）
- **入口**：新建笔记；即将开始的会议点「start notes」；iOS 锁屏 Live Activity 上的「Start Granola」；Apple Watch（watchOS 11+）；Android 快捷设置磁贴。不会自动转写。（来源：https://docs.granola.ai/help-center/ios/transcription ；https://docs.granola.ai/help-center/ios/getting-started）
- **会中**：笔记卡片上随麦克风跳动的音柱 + 计时；iOS 锁屏 Live Activity 显示「taking notes」和时长，灵动岛显示图标和计时；Android 常驻通知「Taking notes」带暂停/继续/结束。卡片上可暂停/继续，点「end」才定稿并生成笔记（手机端较慢，完成后推送通知）。（来源：https://docs.granola.ai/help-center/ios/transcription ；https://docs.granola.ai/help-center/ios/getting-started）
- **说话人**：转写里标「Speaker A / Speaker B」，官方说不保证 100% 准，靠增强笔记理解上下文弥补。（来源：https://docs.granola.ai/help-center/ios/transcription）
- **限制**：录手机当前输入设备（连了蓝牙耳机就录耳机麦，位置不对会几乎没声）；手机抓不到其他 App 的声音，所以不能记线上会议；Android 不足约 5 分钟的录音可能生成不了笔记，约 10 小时以上可能处理失败；音频会中临时缓存，转写完成后从 Granola 及第三方删除。（来源：https://docs.granola.ai/help-center/ios/transcription ；https://docs.granola.ai/help-center/ios/getting-started ；https://docs.granola.ai/help-center/consent-security-privacy/getting-consent）

### Notion AI Meeting Notes
- **入口**：桌面端任意页面输入 `/meet`，或侧边栏「Meeting notes」标签、新建按钮、首页「Upcoming events」卡片；加入视频通话时桌面端会弹通知。（来源：https://www.notion.com/help/ai-meeting-notes）
- **会中**：点「Start transcribing」开始——**点这个按钮即视为确认所有参会人已同意录音和转写**；点「Stop」结束并自动生成摘要（至少 1 分钟音频），之后可在块顶部滑块图标里「Resume transcription」。说话人变化时另起一行。（来源：https://www.notion.com/help/ai-meeting-notes）
- **手机/浏览器**：手机只录手机麦克风，抓不到手机本身播放的声音，适合线下；浏览器只录麦克风。（来源：https://www.notion.com/help/ai-meeting-notes）
- **说话人**：桌面端 1 对 1 线上会议效果最好（麦克风 = 我，系统音频 = 对方，关联日历时能写出对方名字）；多人共用一个麦克风、多人会议、浏览器、手机上「可能没有可靠的说话人标签」。Google Meet 装插件、Mac 上的 Zoom 开权限后可以拿到参会人名字。（来源：https://www.notion.com/help/ai-meeting-notes）

### Plaud
- **没有实时转写**：官方明确「No」，录完后整体处理。（来源：https://support.plaud.ai/hc/en-us/articles/54643893661977-Does-Plaud-support-real-time-transcription） 默认 AutoFlow 在录音同步到云端后自动转写+摘要，需超过 200 词（约 5 分钟语音）。（来源：https://support.plaud.ai/hc/en-us/articles/62437655066649-Automatic-transcription-and-summaries）
- **说话人**：转写时自动区分为 Speaker 1/2…；只有用户自己可以录声纹（朗读至少 30 秒）后自动标名，其他人每次都要手动命名，**名字只对这一条录音有效**。（来源：https://support.plaud.ai/hc/en-us/articles/54027338385177-Auto-speaker-labeling ；https://support.plaud.ai/hc/en-us/articles/50635937755161-Name-Speakers）
- **命名界面（v4.0+）**：文件头部有「Speakers」标签，点开「Name speakers」弹窗：列出每位说话人、**总发言时长**和**可播放的转写片段**，逐个输入名字；关联了日历时可从参会人里选（Plaud 不自动对应参会人与说话人）；确认后改名/合并会同步更新摘要。（来源：https://support.plaud.ai/hc/en-us/articles/63429907932569-Meeting-info-and-Speaker-Labels-for-v4-0-x-or-later）
- **旧版（v3.30 及以前）**：点转写里任意说话人标签改名，可选「只改这一段」或「所有同标签段落」；改名后摘要不变，要重新生成；名字不能清回默认标签；**重新转写会把所有名字重置为 Speaker 1/2**。（来源：https://support.plaud.ai/hc/en-us/articles/50635937755161-Name-Speakers）

### 飞书妙记（录音）
- **入口**：妙记页「录音」按钮；桌面端导航栏顶部或移动端顶部「+ 创建 › 开始录音转写（或 实时语音记录）」。（来源：https://www.feishu.cn/hc/zh-CN/articles/386045971891） 手机还可加桌面小组件、锁屏小组件、iOS 18 锁屏底部快捷入口、iPhone 15 Pro 操作按钮「实时转写」：未在录音时点一下打开飞书并**自动开始录音**，录音中点一下回到当前录音页。（来源：https://www.feishu.cn/hc/zh-CN/articles/831830859410）
- **会中**：实时转写，自动识别中/英/日等语言，可实时翻译；「暂停」/再点继续/「结束录音」；手机端左上角「<」把录音缩成浮窗继续录；可「分享」邀请协作者实时看文字（协作者听不到声音）；有智能纪要权限时实时生成总结，结束时选「保留音频」或「仅保留文字」。手机录音单次上限 4 小时。（来源：https://www.feishu.cn/hc/zh-CN/articles/386045971891）
- **说话人**：V7.44 起手机录音实时转写支持说话人识别拆分，成员授权声纹后可显示姓名。（来源：https://www.feishu.cn/hc/zh-CN/articles/772105146309）
- **会后改名/拆分**：桌面端「··· › 重新识别说话人」：选一个需要拆分的参与者、勾「选择说话人数」并填人数，拆出「说话人 1/2/3」，悬停改名；仅限创建 30 天内的妙记，每位参会者只能拆一次，识别期间不能编辑/翻译/导出；关闭页面不影响后台识别，重开显示剩余时间。（来源：https://www.feishu.cn/hc/zh-CN/articles/812241214493） OpenAPI 侧有「说话人替换」：把某个 speaker_id 的全部发言改归属到某个飞书用户（open_id）。（来源：https://open.feishu.cn/.well-known/skills/lark-meeting/references/lark-minutes-speaker-replace.md）

### 通义听悟
- **入口与会中**：会前打开「开启实时记录」；会中可直接改识别结果、**编辑发言人**，并把常一起开会的同事设为「常用发言人」；可开实时翻译（双语对照/纯译文）；一键高亮「重点 / 问题 / 待办」；摘取到笔记。（来源：https://tingwu.aliyun.com/helpcenter/06ae13）
- **结束**：点「结束录音」时**选择本次会议发言人数**并确认，然后生成记录；会后可按发言人筛选、看每人发言总结。（来源：https://tingwu.aliyun.com/helpcenter/06ae13）
- **能力边界**：说话人分离只能区分不同发音人，不能识别身份，会后改名。（来源：https://help.aliyun.com/zh/tingwu/faq-about-basic-usage） API 侧离线说话人区分可选「不分离 / 2 人 / 多人」；实时 16k 最多三个声道。（来源：https://help.aliyun.com/zh/tingwu/features ；https://help.aliyun.com/zh/tingwu/faq-about-basic-usage）

### Apple 语音备忘录
- iPhone 12 及以后，支持英/西/葡/意/法/德/日/韩/简体中文/繁体中文；录音时在波形上方上滑可看实时转写（当前词高亮），可暂停/继续；录完可查看、复制转写，搜索可搜到转写内容并定位到波形。（来源：https://support.apple.com/guide/iphone/view-a-transcription-iph00953a982/ios）
- 官方文档**没有提到说话人区分**；是否存在说话人标签**未核实**（倾向于没有）。

### 补充：Jamie、知了
- **Jamie 手机端**：可录线下会议，不能录电话；会后在「Speakers」列表逐个播放短音频片段并输入名字，可从最近联系人里选；识别过的人以后自动认出；文档说明识别出的人数可能多于或少于实际人数，同一人被拆成两个时「填同一个名字」即可。（来源：https://docs.meetjamie.ai/getting-started/mobile-app ；https://docs.meetjamie.ai/getting-started/identify-speaker）
- **知了（开源，Apache-2.0）**：线下点「打开麦克风」，混合会议用同一个按钮：电脑声音标「线上」，麦克风标「现场」，合在一份记录里；按钮在菜单栏面板、纪要窗口和字幕条都能点，每次启动默认关闭；外放窜音时线上有人说话会暂停听桌边，戴耳机/会议麦时两边同时记。说话人用本机 WeSpeaker ResNet34-LM 声纹区分，可改名或合并；Mac 上写名字后点「记住这个声音」（先确认对方知道），以后从第一句就按名字标，不够确定时提示「可能是 TA」由用户确认；声纹只存本机。认不出是谁的短句单独成段；可选每句都显示说话人。听电脑 1 分钟没声音会问要不要改听会议室。（来源：https://github.com/wquguru/cicada/blob/master/README.zh-CN.md ；https://cicada.indaloop.app/changelog/）

### 单麦克风说话人分离的公开准确率（DER，越低越好）

商业会议产品（Otter、Plaud、飞书、听悟、Granola）**都没有公开说话人分离准确率**，以下是开源模型在公开数据集上的数字：

| 数据集（单通道/远场） | pyannote 3.1 | pyannote community-1 | pyannoteAI precision-2 | DiariZen-Large-s80-v2 |
|---|---|---|---|---|
| AMI SDM（英文会议，单远场麦） | 22.7 | 19.9 | 15.6 | 13.9 |
| AliMeeting（中文会议，远场单通道） | 24.5 | 20.3 | 15.2 | 10.8 |
| AISHELL-4（中文会议，转单声道） | 12.2 | 11.7 | 11.4 | 10.1 |
| NOTSOFAR-1 单通道 | — | — | — | 16.7 |

- pyannote 数据：无 collar、不跳过重叠语音、不告诉说话人数（2025-09 更新）。（来源：https://huggingface.co/pyannote/speaker-diarization-community-1）
- DiariZen 数据：无 collar、无领域自适应、各数据集同一套聚类超参。（来源：https://github.com/BUTSpeechFIT/DiariZen）
- pyannoteAI 的基准页列了 DIHARD 10 个场景（259 条录音、约 67 小时、9.3% 重叠语音），对比 AssemblyAI、Deepgram、ElevenLabs、Soniox、Speechmatics，但具体数值是页面动态渲染，**未取到**。（来源：https://www.pyannote.ai/benchmark）
- 解读：远场单麦会议的 DER 普遍在 10%–20%，重叠和短句是主要错误来源，所以各家都把「会后改名/合并/拆分」做成必备功能。

---

## C. 上传 / 导入已有录音

### 各家规格

| 产品 | 格式 | 大小/时长上限 | 进度与失败处理 | 视频 |
|---|---|---|---|---|
| Otter | 音频 AAC/MP3/M4A（含语音备忘录）/WAV/WMA/OGG；视频 AVI/MOV/MPEG/MP4/WMV/MPG/MKV/M4P/3GP | 单文件 5 GB，受套餐时长额度限制 | 网页拖拽或浏览，可一次拖多个，各自独立上传处理；上传完成后可关窗口，进度在左下角，可开邮件通知；手机端顶部有上传进度条，支持从其他 App 分享导入 | 直接收视频；排错文档建议大视频先导出纯音频再传 |
| Plaud | Web：MP3/MP4/WAV/OGG/RMVB/RM/DIVX/TS/M2TS/3GP/F4V 等；手机：MP3/MP4/WAV/AAC/3GP 等 | 单文件 24 小时、500 MB（Web）；**超过 5 小时自动拆成多个文件** | 先转码再导入；重复文件跳过并提示 | 收视频 |
| Notion | AAC、M4A、MP3、WAV | 未写明 | 会议笔记块设置里「Upload audio」 | 不支持视频（格式表无视频） |
| Granola | — | — | **不支持**导入预录音频 | — |
| 飞书妙记 | 帮助中心未列格式；OpenAPI/CLI 列：wav/mp3/m4a/aac/ogg/wma/amr，视频 avi/wmv/mov/mp4/m4v/mpeg/ogg/flv | 帮助中心：单文件 4 GB，单次最多 20 个；CLI 文档：6 小时、6 GB | 上传弹窗拖拽，需为每个文件选语言或勾「自动识别文件语言」才能提交；一小时最多约 25 个，超了提示「请求过于频繁」；「处理中/转写中…99%」卡住时排查网络中断或存储空间满 | 收视频 |
| 通义听悟 | 音频 MP3/WAV/M4A/WMA/AAC/OGG/AMR/FLAC/AIFF；视频 MP4/WMV/M4V/FLV/RMVB/DAT/MOV/MKV/WEBM/AVI/MPEG/3GP/OGG | 6 GB 且 6 小时；实时记录 24 小时 | 未查到进度/重试细节 | 收视频，视频只识别首声道 |
| 知了 | 「文件 › 导入录音或视频…」 | 未写明 | 1.9.0 起导入文件也做说话人区分；可用保存的录音「重新转写」（旧记录先备份） | 收视频 |
| Krisp | — | 单次录音最长 4 小时，超出需退出重进会议开新段 | — | — |

来源：Otter （来源：https://help.otter.ai/hc/en-us/articles/360047733574-Import-an-audio-or-video-file ；https://help.otter.ai/hc/en-us/articles/360051342914-Fix-problems-importing-files-to-Otter）；Plaud （来源：https://support.plaud.ai/hc/en-us/articles/63431016171545-File-import-on-Plaud-Web-for-v4-0-x-or-later ；https://support.plaud.ai/hc/en-us/articles/51038386639897-Audio-import ；https://support.plaud.ai/hc/en-us/articles/63429919103385-File-import-for-v4-0-x-or-later）；Notion （来源：https://www.notion.com/help/ai-meeting-notes）；Granola （来源：https://docs.granola.ai/help-center/taking-notes/transcription）；飞书 （来源：https://www.feishu.cn/hc/zh-CN/articles/386045971891 ；https://www.feishu.cn/hc/zh-CN/articles/778360370936 ；https://github.com/larksuite/cli/blob/HEAD/skills/lark-minutes/references/lark-minutes-upload.md）；听悟 （来源：https://help.aliyun.com/zh/tingwu/features ；https://help.aliyun.com/zh/tingwu/faq-about-basic-usage）；知了 （来源：https://github.com/wquguru/cicada/blob/master/README.zh-CN.md ；https://cicada.indaloop.app/changelog/）；Krisp （来源：https://help.krisp.ai/hc/en-us/articles/12216182124956-FAQ-about-Krisp-Recording-feature）。

补充细节：
- Otter 排错文档：上传大文件（>200 MB）时电脑休眠会中断上传，建议临时关闭休眠；手机「Data Saver」会限制只在 Wi-Fi 且前台时上传。（来源：https://help.otter.ai/hc/en-us/articles/360051342914-Fix-problems-importing-files-to-Otter）
- 飞书：录制会议生成妙记最长 16 小时，超过则没有文字记录（会议录制本身最长 24 小时）。（来源：https://www.feishu.cn/hc/zh-CN/articles/386045971891 ；https://www.feishu.cn/hc/zh-CN/articles/778360370936）
- 通义听悟 API：音频文件大多只识别首声道（8k 音频为双声道）。（来源：https://help.aliyun.com/zh/tingwu/features）

### Azure Fast Transcription + MAI-Transcribe-2 的限制（Microsoft Learn）

- **调用方式**：Fast Transcription 接口 `POST /speechtotext/transcriptions:transcribe?api-version=2025-10-15`，`definition` 里 `enhancedMode.enabled=true`、`enhancedMode.model="MAI-Transcribe-2"`；可选 `diarization.enabled`、`modelOptions.timestamps`（word/segment/none）、`transcribeStyle`（verbatim 默认 / clean）、`phraseList.phrases`、`locales`（只能填一个，不填自动识别）；60 种语言，含 `zh`、`yue`。MAI-Transcribe-1 已于 2026-08-20 弃用。页面最后更新 2026-09-24。（来源：https://learn.microsoft.com/en-us/azure/ai-services/speech-service/mai-transcribe）
- **MAI 页面写的格式**：WAV、MP3、FLAC；大小和时长让读者去看 REST 参考里的 `audio` 参数。（来源：https://learn.microsoft.com/en-us/azure/ai-services/speech-service/mai-transcribe）
- **开 diarization 的时长备注**：功能表里写「Supports shorter recordings than transcription without diarization; see the note that follows this table」，开头也说「If you enable speaker diarization, see the note on recording length in the next section」。但截至 2026-11-21 抓取，**页面上那条备注本身没有渲染出来，没有具体数字**。（来源：https://learn.microsoft.com/en-us/azure/ai-services/speech-service/mai-transcribe）
- **几处文档的上限互相矛盾**：
  - REST 参考（2025-10-15）`audio` 参数：音频**短于 2 小时、小于 250 MB**。（来源：https://learn.microsoft.com/en-us/rest/api/speechtotext/transcriptions/transcribe?view=rest-speechtotext-2025-10-15）
  - 配额页 Fast transcription 与 LLM Speech：**< 500 MB、< 5 小时/文件**，600 次/分钟（与批量转写共享）。（来源：https://learn.microsoft.com/en-us/azure/ai-services/speech-service/speech-services-quotas-and-limits）
  - Fast transcription 使用文档：< 5 小时、< 500 MB；格式 WAV、MP3、OPUS/OGG、FLAC、WMA、AAC、WAV 封装的 ALAW/MULAW、AMR、WebM、SPEEX。（来源：https://learn.microsoft.com/en-us/azure/ai-services/speech-service/fast-transcription-create）
  - 用户引用的早期 MAI 页面写「less than 300 MB」，微软社区版主回复也引用了 300 MB。（来源：https://learn.microsoft.com/en-us/answers/questions/5995065/mai-transcribe-2-speaker-diarization-fails-on-reco）
  - 批量转写（非 Fast）开 diarization 的上限是 240 分钟/文件。（来源：https://learn.microsoft.com/en-us/azure/ai-services/speech-service/speech-services-quotas-and-limits）
- **实测问题（微软问答，2026-09）**：
  - 一位用户：MAI-Transcribe-2 原生 diarization 在 31:20 成功、33:15 起稳定 503（`diarization_unavailable … Diarization service returned error code 400`），关掉 diarization 47 分钟整段可转；9 月 14 日复测，33:15 和 47:44 都成功了，没有确认服务端修复。（来源：https://learn.microsoft.com/en-us/answers/questions/5994805/mai-transcribe-2-diarization-returns-503-on-audio）
  - 另一位用户：开 diarization 时约 15 分钟以上就失败，返回 408 Timeout / 500 / 503，不在文档定义的错误码里；同一份 73 分钟、70 MB 的录音关掉 diarization 能成功。（来源：https://learn.microsoft.com/en-us/answers/questions/5995065/mai-transcribe-2-speaker-diarization-fails-on-reco）
  - 微软社区版主答复：文档里查不到 MAI 专属的 diarization 时长上限，判断为 preview 服务的限制或缺陷；分块可以作为临时办法，但**各块独立产出的说话人编号不能当成同一个人**；如要全程一致，先用 MAI 只转写、再单独跑说话人分离更稳。（来源：https://learn.microsoft.com/en-us/answers/questions/5994805/mai-transcribe-2-diarization-returns-503-on-audio ；https://learn.microsoft.com/en-us/answers/questions/5995065/mai-transcribe-2-speaker-diarization-fails-on-reco）
- **建议的工程取值（推断）**：按最严口径设计——单请求 < 2 小时、< 250 MB，上传前转成 FLAC 或 MP3 单声道；开原生 diarization 时把每块控制在 15 分钟以内，或者不开原生 diarization，自己做分离。

### 超长文件切块后，说话人编号跨块不一致怎么处理

1. **两阶段聚类（全局重聚类）**：NeMo 的长音频方案：超过 45 分钟就切成等长段（如 4 小时切 8×30 分钟），每段先独立分离，并保存每个说话人的若干个 embedding；第二步只拿每段每人的前 E 个 embedding 做全局聚类，把各段的局部标签映射到全局说话人。（来源：https://github.com/NVIDIA/NeMo/pull/6813）
2. **带约束的层次聚类**：IberSpeech 2024 论文：每块每个说话人提一个 x-vector（ECAPA-TDNN），在余弦距离矩阵上做带约束的 AHC（同一块内的不同说话人不能合并），把局部标签合成全局标签；VBx 相对 DER 改善 13.2%，DiaPer 改善 48.75%；pyannote 本身已经有类似机制，所以没有提升。（来源：https://www.isca-archive.org/iberspeech_2024/alvareztrejos24_iberspeech.pdf）
3. **转写和分离分开做**：ASR 关掉分离、整段转写，再对整段音频单独跑分离（如 pyannote community-1），用词级时间戳对齐。微软问答里的两位用户都在用这个办法。（来源：https://learn.microsoft.com/en-us/answers/questions/5994805/mai-transcribe-2-diarization-returns-503-on-audio ；https://learn.microsoft.com/en-us/answers/questions/5995065/mai-transcribe-2-speaker-diarization-fails-on-reco） pyannote community-1 专门提供「exclusive speaker diarization」输出，就是为了方便和转写时间戳对齐，也支持 `num_speakers` / `min_speakers` / `max_speakers`。（来源：https://huggingface.co/pyannote/speaker-diarization-community-1）
4. **用声纹锚定身份**：给已知的人建声纹，每块按声纹相似度直接认人（Otter 打过标签的说话人、飞书授权声纹、Plaud 用户本人声纹、知了「记住这个声音」）。（来源：https://help.otter.ai/hc/en-us/articles/360048465453-Tagging-speaker-names-in-a-conversation ；https://www.feishu.cn/hc/zh-CN/articles/772105146309 ；https://support.plaud.ai/hc/en-us/articles/54027338385177-Auto-speaker-labeling ；https://cicada.indaloop.app/changelog/）
5. **重叠窗口 + 匈牙利匹配**（相邻块重叠几十秒，按重叠区的标签一致性做二分图匹配）：业内常见说法，但这次**没找到一手来源**，列为未核实。

---

## D. 录制过程的交互

### Granola（桌面）
- **开始**：点会议通知；会议开始时间过后从首页打开该会议；右上「+ New Note」开临时会议；提前打开即将开始的会议，到点自动开始转写。没打开笔记就不会在后台转写。（来源：https://docs.granola.ai/help-center/taking-notes/transcription）
- **指示**：录音时笔记底部有绿色动态音柱；其他 App 在前台时，屏幕**右侧**出现悬浮「live meeting indicator」，点它回到笔记，拖底部把手可移动位置，可在「Settings › Preferences › General」关掉。（来源：https://docs.granola.ai/help-center/taking-notes/transcription）
- **暂停/继续**：转写面板里「Stop」暂停（不结束会议）、「Resume」继续；会议结束、笔记生成后还能在同一笔记上 Resume，新音频追加到原转写，再手动重新生成笔记。可删除单段转写。（来源：https://docs.granola.ai/help-center/taking-notes/transcription）
- **自动停止**：(1) 检测到通话结束：综合转写长度、通话 App 是否还在活动、日历结束时间；(2) **15 分钟没有新音频**；(3) **电脑睡眠**。临时会议没有日历结束时间，只能靠通话 App 释放麦克风来判断；挂断后还有音频的话 15 分钟计时不触发，会一直录。**连续会议**时 Zoom 可能一直占着麦克风，两场会被并成一份转写，要手动在中间点停止。macOS 14.2 以下非管理员用户收不到通话结束信号。（来源：https://docs.granola.ai/help-center/taking-notes/transcription）
- **设备**：转写窗口底部选麦克风，「Auto」跟随会议 App 正在用的麦克风，否则用系统默认；蓝牙耳机麦被占用时耳机会降成低音质模式，建议改用笔记本内置麦。（来源：https://docs.granola.ai/help-center/troubleshooting/transcription-issues）
- **故障提示**：连不上转写服务时显示橙色警告「Granola is having trouble transcribing」，设置里有网络连通性检查并可复制报告给 IT；菜单栏/托盘菜单有「Restart Granola」。桌面端不存音频。（来源：https://docs.granola.ai/help-center/troubleshooting/transcription-issues ；https://docs.granola.ai/help-center/consent-security-privacy/getting-consent）
- **同意**：会中自动在会议聊天里发消息 + 「Granola Watermark」在自己的摄像头画面上叠可见标记，用户或管理员可开启；企业版有日历邀请里的「Heads Up」页面；企业可开「Allow participants to stop transcription」，参会者从 Heads Up 或聊天里的「See who is transcribing」页面点「Stop Granola for everyone」，只有停止的人能恢复（桌面版需高于 7.795.0，2026-11-13）。（来源：https://docs.granola.ai/help-center/consent-security-privacy/transparency-solutions/introduction ；https://docs.granola.ai/help-center/consent-security-privacy/stop-granola-transcription）

### Notion
- **开始**：`/meet`、侧边栏、日历入口、加入视频通话时的桌面通知。点「Start transcribing」= 确认已获得所有人同意。（来源：https://www.notion.com/help/ai-meeting-notes）
- **停止/继续**：「Stop」结束并生成摘要（至少 1 分钟音频）；离开会议时可自动停止，依据「会议 App 在线状态和多种音频信号（如是否有音频活动），不使用音频内容本身」；块顶部滑块图标里「Resume transcription」。（来源：https://www.notion.com/help/ai-meeting-notes）
- **同意提示**：
  - 文字：滑块菜单里编辑同意文案，复制到会议聊天。
  - 语音：开始转写前点块底部的扬声器按钮，**用电脑扬声器播放**同意语音（提示要取下耳机、打开扬声器），可按语言选声音。
  - 自动：可设置每次开始转写时自动播放；工作区所有者可强制全员开启（2026-03-12 上线），企业组织所有者可在组织层面「交给各工作区 / 默认开启 / 强制开启」。
  - Google Meet 插件：开始转写时自动在 Meet 聊天里发并置顶一条消息，并在摄像头画面上显示录制标记。（来源：https://www.notion.com/help/ai-meeting-notes ；https://www.notion.com/releases/2026-03-12）
- **排障**：反复弹出屏幕/系统音频授权时，建议撤销授权、重启电脑后重新授权；可开启「保存会议音频」以便反馈转写问题（推断默认不存）。（来源：https://www.notion.com/help/ai-meeting-notes）

### Krisp
- **模式**：AI Note Taker 下选「Transcribe only / Audio Recording / Screen Recording / Off」；会前或会中从 App 里开始；**可随时暂停**录音和转写；屏幕录制可在 App 和小组件上暂停/继续。（来源：https://help.krisp.ai/hc/en-us/articles/30146816045980-Record-your-meetings-with-Krisp-AI-Note-Taker）
- **指示与入口**：会议开始时显示 Krisp 小组件（设置「Show widget」）；桌面 App 有「call indication bar」，点开进入文档页，左下 CC 图标打开实时转写窗口；菜单栏可预览即将开始的日历事件。（来源：https://help.krisp.ai/hc/en-us/articles/13882967746588-AI-Meeting-Assistant-settings ；https://help.krisp.ai/hc/en-us/articles/30146816045980-Record-your-meetings-with-Krisp-AI-Note-Taker）
- **采集方式**：建议在会议软件里同时选「Krisp Microphone」和「Krisp Speaker」，即通过 Krisp 的虚拟音频设备拿到双方声音。（来源：https://help.krisp.ai/hc/en-us/articles/30146816045980-Record-your-meetings-with-Krisp-AI-Note-Taker）
- **规则**：少于 30 秒的会议不生成录音/转写/笔记；单次录音最长 4 小时，超出需退出重进会议开新段；会中存储满了不会截断，允许超额。（来源：https://help.krisp.ai/hc/en-us/articles/12216182124956-FAQ-about-Krisp-Recording-feature）
- **同意**：「Settings › Privacy & Consent › Notify others you're using Krisp」；文档提醒有的平台会拦截聊天消息或音频提示，要另行告知。（来源：https://help.krisp.ai/hc/en-us/articles/30146816045980-Record-your-meetings-with-Krisp-AI-Note-Taker）

### Jamie
- **开始**：检测到加入视频通话时弹通知「Start Jamie」；也可在 App 导航栏或 macOS 菜单栏「Join & Start Jamie」手动开始；默认用系统默认麦克风。（来源：https://docs.meetjamie.ai/getting-started/recorder）
- **自动开始**（默认关）：「麦克风被已知会议 App 激活」+「日历上有会议（5 分钟内开始或 10 分钟内已开始）」同时满足才开始，开始前有可取消的倒计时；未识别的 App（语音备忘录、听写工具）不触发；同一日历事件只自动开始一次，停止或取消后重连、刷新等候室都不会再触发；线下会议可关掉「需要会议 App 占用麦克风」条件，到点直接开录。（来源：https://docs.meetjamie.ai/getting-started/auto-start-recording）
- **会中**：屏幕上出现可拖动的紧凑悬浮小组件，点组件（不是停止按钮）展开完整会议视图；可**随时换麦克风且不中断录音**；默认停在私人「Scratchpad」标签（不进摘要）。macOS 菜单栏在录音时显示已录时长，优先于「下一场会议」倒计时。（来源：https://docs.meetjamie.ai/getting-started/recorder ；https://docs.meetjamie.ai/getting-started/menu-bar）
- **停止**：小组件或展开视图底部的红色 Stop；**不足 5 分钟停止会警告**；录音中关闭 App 会二次确认，退出时保存当前录音。（来源：https://docs.meetjamie.ai/getting-started/recorder）
- **自动停止**（默认「只提醒」）：同时满足「麦克风不再被占用」「日历会议进入最后四分之一（会议结束后该条件持续为真）」「至少 30 秒无语音（30 秒–5 分钟可调）」才停或提醒；没有关联日历时，要求麦克风持续空闲且无语音，再额外等 5–15 分钟（默认 10 分钟）兜底，**兜底即使在只提醒模式下也会真的停**，防止忘记的会话录几个小时；停止前有倒计时可取消；提醒可「本场不再提醒」。（来源：https://docs.meetjamie.ai/getting-started/auto-stop-recording）
- **暂停**：文档里**没有找到**暂停/继续功能，**未核实**。
- **同意**：会前自动发「录音告知邮件」（可选提前 10 分钟到 48 小时，默认 4 小时，含 GDPR 法律依据、保留期、反对方式）；会议聊天告知（Beta，Teams/Meet/Zoom）。（来源：https://docs.meetjamie.ai/getting-started/recording-consent ；https://docs.meetjamie.ai/getting-started/notifications）

### 知了（Cicada）
- **开始**：手动点开始听，Mac 上实时纪要窗口立刻出现；开机或会议软件来电**自动开始**时，等第一句话再出现窗口。（来源：https://cicada.indaloop.app/changelog/）
- **常驻入口**：菜单栏/顶栏面板，Windows 托盘；1.7.0 起 Dock 常驻，**记录中 Dock 图标右上角有橙点**，右键 Dock 图标可开始/停止记录、打开纪要/记录/设置；纪要窗口点黄灯收进菜单栏，会议照常记。（来源：https://cicada.indaloop.app/changelog/）
- **悬浮条**：字幕条只在有人说话时出现，鼠标划过自动变透明；「打开麦克风」按钮在字幕条上也有。（来源：https://github.com/wquguru/cicada/blob/master/README.zh-CN.md ；https://cicada.indaloop.app/changelog/）
- **自动结束**：识别 Zoom、腾讯会议、飞书、Teams、Lark 国际版、企业微信、Webex 等挂断后自动结束并整理纪要；很久没人说话也会自动结束（Windows 可在「设置 › 结束会议」改规则；Android 可改为直接结束或一直记）。1.10.1 修复：允许过麦克风但从没用过的应用不再被当成在通话。（来源：https://cicada.indaloop.app/changelog/）
- **设备变化**：「打开麦克风」旁的小箭头换麦，开着麦换也不中断；拔掉外接麦自动改用剩下的麦克风并提示，没有可用麦克风时，「电脑和会议室」改为只听电脑，「只听会议室」则暂停，**不会悄悄换到耳机或虚拟麦克风**；选蓝牙耳机时提醒只听得到戴着的人。（来源：https://cicada.indaloop.app/changelog/）
- **静默失败处理**：1.10.1 前，Windows 音频服务重启/拔插扩展坞后会议看着还在记，其实一个字都没录，10 分钟后被当成没人说话自动结束；现在会自动重连，一直接不上才停下并说明原因；需要用户处理的问题在纪要窗口、字幕条、菜单栏图标（橙点）提示，纪要窗口收起 10 秒后发一次系统通知。（来源：https://cicada.indaloop.app/changelog/）
- **隐私**：Mac 上共享屏幕、录屏、截图都拍不到纪要窗口（可手动关闭）。（来源：https://github.com/wquguru/cicada/blob/master/README.zh-CN.md）
- **暂停、休眠/合盖处理**：文档和更新日志里**没有找到**，**未核实**。

### 其他产品的相关细节
- **飞书妙记**：暂停/继续/结束三个按钮；手机端缩成浮窗不中断。（来源：https://www.feishu.cn/hc/zh-CN/articles/386045971891）
- **Otter**：Pause/Resume/Stop；断网继续缓存录音，联网后补传。（来源：https://help.otter.ai/hc/en-us/articles/360048269733-Record-a-conversation ；https://help.otter.ai/hc/en-us/articles/31672594631063-Getting-the-most-out-of-Otter-ai-Best-Practices-for-in-person-recordings）
- **Audio Hijack**：静音检测可删静音段/切新文件/停止录音；2 GB 或 24 小时自动切文件。（来源：https://rogueamoeba.com/support/manuals/audiohijack/?print=true）
- **MacWhisper**：会议检测通知 10 秒自动消失；可从「Meeting Ended」通知停止。（来源：https://docs.macwhisper.com/article/30-record-meetings）
- **TranscribeX**：菜单栏「Stop Recording <App>」；由会议检测启动的录音在会议结束时自动停止。（来源：https://www.transcribex.io/docs/app-recording/）

### 横向对比：合盖/休眠、设备切换、长时间静音

| 情况 | 已知做法 |
|---|---|
| 电脑睡眠 | Granola：睡眠即自动停止。（来源：https://docs.granola.ai/help-center/taking-notes/transcription） macOS 上 idle-sleep assertion 拦不住合盖、苹果菜单睡眠、低电量睡眠（`IOPMLib.h`），恢复后应记录断档。（来源：https://victorantos.com/posts/macos-process-taps-tapelark/） 其他家：**未核实** |
| 上传中休眠 | Otter：会导致上传失败，建议临时关休眠。（来源：https://help.otter.ai/hc/en-us/articles/360051342914-Fix-problems-importing-files-to-Otter） |
| 长时间静音 | Granola 15 分钟；Jamie 无日历时 30 秒静音 + 10 分钟兜底（可调 5–15 分钟）；知了「很久没人说话」（之前版本有 10 分钟的描述）。（来源：https://docs.granola.ai/help-center/taking-notes/transcription ；https://docs.meetjamie.ai/getting-started/auto-stop-recording ；https://cicada.indaloop.app/changelog/） |
| 麦克风切换 | Jamie、知了录音中可换且不中断；Granola「Auto」跟随会议 App。（来源：https://docs.meetjamie.ai/getting-started/recorder ；https://cicada.indaloop.app/changelog/ ；https://docs.granola.ai/help-center/troubleshooting/transcription-issues） |
| 输出设备切换 | Windows Process Loopback 不绑定端点；Audio Hijack 对 VoIP App 默认抓所有输出设备。（来源：https://learn.microsoft.com/en-us/samples/microsoft/windows-classic-samples/applicationloopbackaudio-sample/ ；https://www.rogueamoeba.com/support/knowledgebase/?showArticle=AudioHijack-AudioDeviceSelection） |
| 音频服务重置 | 知了：自动重连，失败才停并说明原因。（来源：https://cicada.indaloop.app/changelog/） |

---

## 对我们的启示

### 可以直接借鉴的做法

**采集层（A）**
1. **macOS 默认用 Core Audio tap，不用 ScreenCaptureKit 录音**。权限更窄（「仅系统录音」），没有 macOS 15 的周期性二次确认，一个 tap 就能装多个进程，还能「全局抓、排除自己」。只有将来要录画面时才引入 ScreenCaptureKit。最低版本定 14.4 比较稳（各家口径 14.2–14.6）。（来源：https://developer.apple.com/documentation/coreaudio/capturing-system-audio-with-core-audio-taps ；https://victorantos.com/posts/macos-process-taps-tapelark/ ；https://systemaudiokit.lucaspiera.com/docs/permissions ；https://github.com/RustAudio/cpal/blob/master/CHANGELOG.md）
2. **Windows 用 Process Loopback + INCLUDE_TARGET_PROCESS_TREE**。用户选的是「App」，内部要找到 App 的根进程（浏览器/Electron/WebView2 先沿父链往上走）；选多个 App 就开多个客户端，在 Rust 里重采样后混音。格式自己定（建议 48 kHz/float/双声道，或直接 16 kHz 单声道），不要调 GetMixFormat。版本不够或激活失败时**不能静默退回整机 loopback**，要明确让用户选「录全部系统声音」（screenpipe 的做法）。（来源：https://github.com/screenpipe/screenpipe/blob/fe296feb/crates/screenpipe-audio/src/core/process_tap/windows.rs ；https://github.com/obsproject/obs-studio/blob/master/plugins/win-wasapi/win-wasapi.cpp ；https://learn.microsoft.com/en-us/answers/questions/1125409/loopbackcapture-activateaudiointerfaceasync-with-v ；https://docs.rs/wasapi/latest/wasapi/struct.AudioClient.html）
3. **Rust 选型**：Windows 用 `wasapi` crate（已封装 application loopback），要细控就直接用 `windows` crate；macOS 用 `objc2-core-audio`（cpal 也用它）或 `cidre` 写 tap。不要指望 cpal 做按进程捕获。cpal 用在 WASAPI 上要注意 COM 线程模型的坑（#1402）。（来源：https://docs.rs/wasapi/latest/wasapi/struct.AudioClient.html ；https://github.com/screenpipe/screenpipe/blob/fe296feb/crates/screenpipe-audio/src/core/process_tap/windows.rs ；https://github.com/RustAudio/cpal/blob/master/src/host/coreaudio/macos/loopback.rs ；https://docs.rs/cidre/latest/cidre/core_audio/struct.TapDesc.html ；https://github.com/RustAudio/cpal/issues/1402）
4. **「默认整机、按 App 是进阶」**：Granola 在 macOS 上只录混合系统音频也能成立，Windows 上才按进程；TranscribeX 把「任意选 App」做成付费功能，免费的是「检测到的会议 App」卡片。建议默认选项是「全部系统声音（排除本应用）+ 麦克风」，选择器里再提供按 App。（来源：https://docs.granola.ai/help-center/taking-notes/transcription ；https://docs.granola.ai/help-center/troubleshooting/transcription-issues ；https://www.transcribex.io/docs/app-recording/）

**选择器（A4）**
5. **只列正在发声或有音频会话的 App，并把子进程归并到父 App**。macOS：`kAudioHardwarePropertyProcessObjectList` + `IsRunningOutput` + BundleID，按前缀归并 `*.helper`；Windows：枚举音频会话（Active/Inactive），按进程树聚合到 exe，图标从 exe 提取（Splitwave 已这么做）。每行：图标 + App 名 + 是否正在出声的电平/动态标记 + 复选框（多选）。顶部固定「全部系统声音」。列表要能自动刷新，或像 TranscribeX 那样给一个刷新按钮（未出过声的 App 在 macOS 上没有进程对象，抓不到）。（来源：https://macnotetaker.com/blog/which-app-is-using-mic-coreaudio-process-objects ；https://learn.microsoft.com/en-us/windows/win32/api/audiosessiontypes/ne-audiosessiontypes-audiosessionstate ；https://github.com/Horuse/Splitwave ；https://www.transcribex.io/docs/app-recording/ ；https://victorantos.com/posts/macos-process-taps-tapelark/）
6. **浏览器整体算一个 App，在界面上说清楚不能只录一个标签页**（Audio Hijack 官方也做不到）。需要只录某个网页会议时，引导用户用「Safari 添加到程序坞」或 Chrome「安装为应用」开独立窗口（后者是推断）。（来源：https://www.rogueamoeba.com/support/knowledgebase/?product=Audio+Hijack&showArticle=Misc-WebAppAudio）
7. **VoIP 类 App 抓所有输出设备**（Audio Hijack 4.4 的默认），避免用户把会议音频切到耳机后漏录。Windows 的 Process Loopback 本身不绑定端点，天然满足。（来源：https://www.rogueamoeba.com/support/knowledgebase/?showArticle=AudioHijack-AudioDeviceSelection ；https://learn.microsoft.com/en-us/samples/microsoft/windows-classic-samples/applicationloopbackaudio-sample/）

**线下（B）**
8. **一键开始 + 按句实时滚动**，说话人先用「说话人 1/2」占位；会后命名用 Plaud v4 式的弹窗：每位说话人的总发言时长 + 2–3 段可播放片段 + 输入名字（可从日历参会人选）。（来源：https://support.plaud.ai/hc/en-us/articles/63429907932569-Meeting-info-and-Speaker-Labels-for-v4-0-x-or-later ；https://docs.meetjamie.ai/getting-started/identify-speaker）
9. **改名只改一处、全局生效**（Otter：给一个 Speaker # 打标签，同编号全部更新）；提供「合并」和「撤销合并」（Otter 2026-07 新增）；再提供「拆分」：选中一个说话人并指定人数重新识别（飞书），或在结束时问一句「这次几个人在说话」（通义听悟），把人数传给分离模型的 `num_speakers`。（来源：https://help.otter.ai/hc/en-us/articles/360048465453-Tagging-speaker-names-in-a-conversation ；https://help.otter.ai/hc/en-us/articles/40586357592471-Speaker-Management ；https://www.feishu.cn/hc/zh-CN/articles/812241214493 ；https://tingwu.aliyun.com/helpcenter/06ae13 ；https://huggingface.co/pyannote/speaker-diarization-community-1）
10. **声纹记忆必须用户主动触发并提示征得对方同意**（知了的「记住这个声音（先确认对方知道）」），不确定时只提示「可能是 TA」，由用户确认。（来源：https://cicada.indaloop.app/changelog/）
11. **说话人名字和转写结果分开存**（SQLite 里用 speaker_id → 名字的映射表），这样重新转写、重新生成摘要不会丢名字（Plaud v3 重新转写会清空所有名字，是反例），改名后自动或提示重新生成摘要（Otter、Plaud 都要求手动重新生成）。（来源：https://support.plaud.ai/hc/en-us/articles/50635937755161-Name-Speakers ；https://help.otter.ai/hc/en-us/articles/40586357592471-Speaker-Management）

**上传（C）**
12. **本地先转码再上传**：用 ffmpeg 抽音轨，转成 16 kHz 单声道 FLAC/MP3，视频大文件也就几十 MB（Plaud「先转码再导入」，Otter 建议先导出音频）。进度分两段显示：「上传 x%」→「转写中」。上传完成后允许关窗口，后台继续处理并在完成后系统通知（Otter、飞书）。上传期间阻止系统休眠（Otter 的失败原因之一）。（来源：https://support.plaud.ai/hc/en-us/articles/63431016171545-File-import-on-Plaud-Web-for-v4-0-x-or-later ；https://help.otter.ai/hc/en-us/articles/360051342914-Fix-problems-importing-files-to-Otter ；https://help.otter.ai/hc/en-us/articles/360047733574-Import-an-audio-or-video-file）
13. **批量导入时每个文件独立排队、独立重试**（Otter），并对单文件上限、批量数量、频率限制给出明确错误（飞书：单次 20 个、每小时约 25 个、99% 卡住时的排查方向）。（来源：https://help.otter.ai/hc/en-us/articles/360047733574-Import-an-audio-or-video-file ；https://www.feishu.cn/hc/zh-CN/articles/386045971891 ；https://www.feishu.cn/hc/zh-CN/articles/778360370936）
14. **Azure MAI-Transcribe-2**：按最严口径做客户端校验（< 2 小时、< 250 MB；格式只发 WAV/MP3/FLAC），超出就本地切块；开原生 diarization 时每块不超过约 15 分钟；跨块用「每块每人取几个 embedding → 全局聚类」统一编号（NeMo 两阶段、IberSpeech 带约束 AHC），或者干脆关掉 MAI 的 diarization，整段本地跑一次 pyannote community-1，再用词级时间戳对齐。408/500/503 都要当成「可能是 diarization 失败」，降级为「先出无说话人转写，再补分离」。（来源：https://learn.microsoft.com/en-us/rest/api/speechtotext/transcriptions/transcribe?view=rest-speechtotext-2025-10-15 ；https://learn.microsoft.com/en-us/answers/questions/5994805/mai-transcribe-2-diarization-returns-503-on-audio ；https://learn.microsoft.com/en-us/answers/questions/5995065/mai-transcribe-2-speaker-diarization-fails-on-reco ；https://github.com/NVIDIA/NeMo/pull/6813 ；https://www.isca-archive.org/iberspeech_2024/alvareztrejos24_iberspeech.pdf ；https://huggingface.co/pyannote/speaker-diarization-community-1）

**录制交互（D）**
15. **三层状态可见**：主窗口录音条（计时 + 电平），可拖动的悬浮指示条（Granola 在右侧、Jamie 小组件可拖动），菜单栏/托盘图标在录音时显示计时或橙点（Jamie、知了）。悬浮条只在本应用不在前台时出现（Granola 的做法），并且可以关掉。（来源：https://docs.granola.ai/help-center/taking-notes/transcription ；https://docs.meetjamie.ai/getting-started/recorder ；https://docs.meetjamie.ai/getting-started/menu-bar ；https://cicada.indaloop.app/changelog/）
16. **自动停止用多信号 + 倒计时 + 默认只提醒**：会议 App 不再占用麦克风、日历会议接近结束、持续静音，三者同时满足才停；没有日历时加长兜底（Jamie 默认 10 分钟，Granola 15 分钟），兜底要真停。连续会议的问题要单独处理：检测到新的日历事件开始时，提示「拆成新会议」（Granola 的坑）。（来源：https://docs.meetjamie.ai/getting-started/auto-stop-recording ；https://docs.granola.ai/help-center/taking-notes/transcription）
17. **暂停 ≠ 停止**：暂停不生成纪要，停止才定稿；停止后还能「继续」，新音频追加到同一份记录（Granola Resume、Notion Resume transcription）。不足 N 分钟时停止要二次确认（Jamie 5 分钟）。（来源：https://docs.granola.ai/help-center/taking-notes/transcription ；https://www.notion.com/help/ai-meeting-notes ；https://docs.meetjamie.ai/getting-started/recorder）
18. **设备变化不中断、不静默换设备**：麦克风拔掉就回退到剩下的设备并提示，没有就只录系统声音；绝不自动选中虚拟设备或蓝牙耳机麦（知了）；选蓝牙麦时提示音质会下降、只能收到戴耳机的人（Granola、知了）。（来源：https://cicada.indaloop.app/changelog/ ；https://docs.granola.ai/help-center/troubleshooting/transcription-issues）
19. **对「静默失败」做主动检测**：macOS tap 被拒或 Windows 音频服务重置都表现为「全零/没有数据」。规则：来源在出声（IsRunningOutput / 会话 Active）但持续数字静音 N 秒 → 提示并尝试重建；重建失败就停下并说明原因（知了 1.10.1、Tapelark）。（来源：https://victorantos.com/posts/macos-process-taps-tapelark/ ；https://cicada.indaloop.app/changelog/）
20. **同意**：本地应用加不了会议聊天机器人，可以学 Notion：「开始」按钮旁写明「开始即表示已告知参会人」，提供「复制告知文案」和「用扬声器播放告知语音」两个快捷操作；线下场景的语音告知最自然。（来源：https://www.notion.com/help/ai-meeting-notes）

### 应避开的坑
- **权限被拒 = 静音且不报错**（macOS tap）：上线首日就要有静音检测和「去设置」引导；Info.plist 少了 `NSAudioCaptureUsageDescription` 连弹窗都没有。（来源：https://victorantos.com/posts/macos-process-taps-tapelark/ ；https://systemaudiokit.lucaspiera.com/docs/permissions）
- **开发版 ad-hoc 签名会让 TCC 授权反复失效**，测试时用固定 designated requirement 或 Developer ID 签名；Splitwave 未签名，用户每次更新都要重新授权。（来源：https://systemaudiokit.lucaspiera.com/docs/permissions ；https://github.com/Horuse/Splitwave）
- **权限弹窗会阻塞调用线程**：不要在主线程或 UI 线程启动 tap。（来源：https://victorantos.com/posts/macos-process-taps-tapelark/）
- **App 还没出过声就拿不到进程对象**（macOS）：选择器里要允许「等它出声后自动开始捕获」，或者先用全局 tap。（来源：https://victorantos.com/posts/macos-process-taps-tapelark/）
- **Windows 最低版本文档自相矛盾**（19041 / 20348 / 20438）：用运行时探测，不要写死版本号。（来源：https://learn.microsoft.com/en-us/windows/win32/api/audioclientactivationparams/ns-audioclientactivationparams-audioclient_process_loopback_params ；https://learn.microsoft.com/en-us/windows/win32/api/mmdeviceapi/nf-mmdeviceapi-activateaudiointerfaceasync ；https://obsproject.com/kb/application-audio-capture-guide）
- **部分 Windows App 不走按进程路径**（Granola 点名 Webex、WhatsApp、8x8、3CX，OBS 点名部分游戏语音），要有回退到默认输出设备的方案并告知用户；**Teams 桌面版外放回声**会让同一句话出现两次，需要自己的回声消除或在 UI 上建议戴耳机。（来源：https://docs.granola.ai/help-center/troubleshooting/transcription-issues ；https://obsproject.com/kb/application-audio-capture-guide）
- **会议 App 一直占着麦克风导致连续会议被合并**、挂断后仍有音频导致不停（Granola）。（来源：https://docs.granola.ai/help-center/taking-notes/transcription）
- **说话人数量偏差是常态**（Jamie 文档原话：可能多可能少），UI 要把「合并/拆分」放在显眼位置，不要假定分离是对的。（来源：https://docs.meetjamie.ai/getting-started/identify-speaker）
- **超过 5 小时自动拆成多个文件**（Plaud）会把一场会议割裂成多份记录、说话人不连续；我们切块应在内部完成，对用户仍是一条记录。（来源：https://support.plaud.ai/hc/en-us/articles/63431016171545-File-import-on-Plaud-Web-for-v4-0-x-or-later）
- **Azure 返回的错误码误导**：diarization 失败时可能给 408/500/503，而不是 `AudioLengthLimitExceeded`；重试同样的请求无用，要降级。（来源：https://learn.microsoft.com/en-us/answers/questions/5995065/mai-transcribe-2-speaker-diarization-fails-on-reco ；https://learn.microsoft.com/en-us/answers/questions/5994805/mai-transcribe-2-diarization-returns-503-on-audio）
- **改名不同步到摘要/搜索**（Otter、Plaud v3）。（来源：https://help.otter.ai/hc/en-us/articles/40586357592471-Speaker-Management ；https://support.plaud.ai/hc/en-us/articles/50635937755161-Name-Speakers）
- **合盖睡眠拦不住**：恢复后要续接同一场记录并在时间轴标出断档，不要悄悄新建一场或丢掉前半段。（来源：https://victorantos.com/posts/macos-process-taps-tapelark/ ；https://docs.granola.ai/help-center/taking-notes/transcription）

---

## 没查到或存疑的点
- macOS 26 `CATapDescription.bundleIDs` / `processRestoreEnabled` 的确切语义（只有 SDK diff，Apple 文档页没有正文）。
- Core Audio tap 的真实最低版本（14.2 / 14.4 / 14.6 口径不一）。
- Windows Process Loopback 的真实最低 build（19041 / 20348 / 20438），以及单进程可并发的客户端数量上限。
- wasapi-rs 把 `include_tree=false` 说成「只抓目标进程」，与微软枚举定义（只有 include/exclude 进程树）不一致。
- Audio Hijack 选择器里是否显示图标、是否标出正在出声；Windows 11 新版音量合成器的官方说明页没取到。
- 各家录制中能否切换所抓的 App：文档都没提。
- MAI-Transcribe-2 开 diarization 的官方时长上限：页面引用的备注没有渲染出来，社区实测在 15–33 分钟出过失败，后来又能成功，没有官方确认。
- macOS 15 屏幕录制二次确认的频率（月度）只有媒体报道，Apple 无正式文档。
- Jamie、知了的暂停功能，以及它们对合盖/休眠的处理：文档未提。
- Apple 语音备忘录是否区分说话人：官方文档没提。
- 商业产品（Otter、Plaud、飞书妙记、通义听悟、Granola）的说话人分离准确率：都没有公开；pyannoteAI 基准页的商业 API 对比数值是动态渲染，没取到。
- 通义听悟、Notion 上传的进度展示和失败重试细节。
- 「重叠窗口 + 匈牙利匹配」做跨块对齐：没找到一手来源。
