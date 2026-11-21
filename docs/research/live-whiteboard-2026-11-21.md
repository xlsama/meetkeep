# 实时会议白板/图示：业内实现调研

调研日期：2026-11-21。标注规则：每条事实后附来源；「源码」指我在本地只读检查过的仓库（链接带 commit）；「营销」指只有产品宣传页、没有文档或代码佐证；「未核实」指查不到一手来源。

## 0. 速览表

| 产品/项目 | 触发频率 | 输出中间格式 | 增量策略 | 画布 | 开源/许可 |
|---|---|---|---|---|---|
| Shimmer | 每 60 秒（转写≥200 字符） | Mermaid 文本 | 把最近 3 个版本喂回去，「EVOLUTION MODE」整张重写 | mermaid.js 渲染 SVG，只读 | 未开源（只读过线上 JS 包） |
| MindMesh | 每次 final 转写 + 1.5 秒静默，1 秒冷却 | JSON 节点/边（稳定 key） | 模型返回全图，服务端 diff 成 patch；变动 >30% 改为整图替换 | React Flow | 无许可证 |
| conversation-canvas | 最后一段 final 转写后 3 秒防抖 | 28 种 Action 的 Zod 联合类型 | 只追加 Action + 三重去重 | tldraw v5 + 自定义 shape | 无许可证；tldraw 需 key |
| 知了 Cicada（鱼骨图） | 每 45 秒 | 整份纪要 JSON（鱼骨图是话题的一个字段） | 冻结旧话题，只允许改最后 2 个 | SwiftUI 原生绘制 | Apache-2.0 |
| Agent Draw | 用户框选区域并说完一段话 | tldraw agent action（工具调用） | 每次在新区域新增，不改旧内容 | tldraw | 演示代码 MIT/Apache；SDK 需 key |
| tldraw agent 模板 | 用户发消息，单轮最多 20 次后续请求 | 结构化 action + 截图 + 简化 shape | agent 读画布后增删改 | tldraw | 模板 MIT；SDK 为 tldraw license |
| BUD | 语音对话 | LLM 写 JS，批量执行画布操作 | 对话式编辑 | Excalidraw | 未开源 |
| Wavelength | 按住麦克风时实时生成，松开后做一次清理 | 未公开 | 推断出的连线画成琥珀色虚线，用户确认后转实线 | 自研（未核实） | 商业 |
| TalkGraph | 「每隔几秒」（营销） | 带类型的卡片 | 未公开 | 自研（未核实） | 商业 |
| D-Sket Canvas | 250ms 音频块，关键词出现 ≥3 次才建节点 | KeyBERT 关键词 + 向量聚类（不用 LLM） | 相似度 ≥0.85 合并到已有节点 | 自研无限画布 | 商业 |
| meetgraph | 每 15 秒做一次增量纪要 | Pydantic 纪要 → RDF 知识图谱 | 只摘要新增文本，再追加去重合并 | PyQt QGraphicsView + 确定性力导向布局 | 自定义许可（NOASSERTION） |
| 通义听悟 | 会后（实时任务结束后才出结果） | MindMap 树 JSON（Title/Topic，最多 4 层） | 无，一次生成 | 由调用方渲染 | 商业 API |
| 飞书 | 会中「AI 视图」实时；会后纪要配图 | 未公开 | 未公开 | 飞书画板 | 商业 |
| Miro AI | 用户发提示词 | Mermaid → 原生可编辑图 | 「在原图上改」，不重新起稿 | Miro | 商业 |
| FigJam AI | 用户发提示词 | 未公开 | 改提示词后重新生成 | FigJam | 商业 |
| Zoom Whiteboard | 按需，可用会议上下文 | 未公开 | 一次性生成 + 简化/扩展 | Zoom Whiteboard | 商业 |
| Excalidraw 文字生成图 | 用户发提示词 | Mermaid → mermaid-to-excalidraw | 对话式，每次一张新图 | Excalidraw（MIT） | MIT |

---

## 1. Shimmer（Lev Dubinets，shimmerdiagrams.com）

**来源说明**：作者的博客与 HN 帖只讲了概况（https://dubinets.io/shimmer-launch/ ，https://news.ycombinator.com/item?id=47533840 ）。GitHub 上找不到公开仓库（`gh search repos` 没有结果）。下面的细节来自线上前端包 `https://shimmerdiagrams.com/assets/index-BKYgtCnu.js`（2026-11-21 抓取，只读，未运行）。这是生产代码，可能随时变化，不算官方文档。

### 管线（3 步 LLM + 校验重试）
- 转写在浏览器本地完成：Whisper base 或 Voxtral Mini 4B Realtime（WebGPU），也可以用 Web Speech API（博客；JS 包里的模型 ID 是 `onnx-community/whisper-base` 和 `onnx-community/Voxtral-Mini-4B-Realtime-2602-ONNX`）。
- 默认参数（JS 包 `regen` 配置）：`interval_ms: 60000`（每 60 秒一次），`min_transcript_length: 200`，`evolution_context_depth: 3`（带上最近 3 个历史版本），`max_retries: 2`，`retry_strategy: auto_retry`。官网写着「Every 60 seconds, an LLM reads your transcript」（https://shimmerdiagrams.com/ ）。
- 本地 LLM 默认 `onnx-community/Qwen3.5-2B-ONNX`，可选 4B；BYOK 时默认 `gpt-5` / `claude-sonnet-4-6` / Ollama `llama3`（JS 包）。
- 步骤 1 `diagram_type_classifier`：输入**完整原始转写**，让模型「select the mermaid diagram type that best represents the DOMINANT STRUCTURE」，在 flowchart / sequenceDiagram / mindmap / timeline / stateDiagram-v2 / erDiagram / classDiagram 中选一种，「Respond with ONLY the diagram type name」。解析时做子串匹配，失败就回退到 `mindmap`。只有用户手动指定类型（`diagram_type_hint`）时才跳过这一步。所以没指定类型的话，**每个周期都会重新分类**，图类型可能来回跳（这是根据代码的推断）。
- 步骤 2 `transcript_processor`：「Distill this transcript into the structural elements needed for a {{diagram_type}} diagram」。提示词里有 SALIENCE RULES（反复讨论的话题、决定、具体流程/架构/数据模型、依赖与阻塞）和 NOISE TO DISCARD（寒暄、跑题、重复、观点、口头禅）。输出三段：CORE TOPIC、KEY ELEMENTS（5–12 个）、RELATIONSHIPS。
- 步骤 3 `mermaid_generator`：「Include 5-15 nodes」，「Use simple alphanumeric node IDs」，mindmap「max 3 levels deep」，「Output ONLY the mermaid code block」。如果首行和选定类型不一致，代码会强制改写首行。
- 校验：用 `mermaid.parse()` 校验，失败就把 `validation_error` 放进提示词重试，最多 2 次（system 换成「You are an expert at generating valid mermaid diagram syntax. Fix any syntax errors.」）。
- 只有内容和上一版不同时才写入新版本；版本历史存 IndexedDB（JS 包；博客也提到「full transcript and diagram version history」）。

### 增量更新
- 不打补丁。生成器前面会拼上 `EVOLUTION MODE` 前缀：「Update the existing diagram with new information — do not regenerate from scratch… If the diagram is growing too large (>15 nodes), consolidate less important nodes rather than just appending… If no meaningful new information has been discussed since the last version, reproduce the same diagram exactly」（JS 包）。也就是靠提示词约束 LLM 整张重写 Mermaid 文本。
- 位置稳定性：完全交给 Mermaid 自动布局，每次重排。没有用户编辑节点的能力，渲染结果是只读 SVG + svg-pan-zoom（JS 包里只有 `mermaid.render`，没有编辑器）。
- 作者在 Agent Draw 的 HN 帖下评论：「Tldraw seems to be a better medium for this than mermaid」（https://news.ycombinator.com/item?id=48805475 ）。

---

## 2. MindMesh（asher14333/MindMesh，黑客松项目）

源码：https://github.com/asher14333/MindMesh/blob/b7210c16408177741b02b5ff9fabd179a6a53904/README.md 。没有 LICENSE 文件（GitHub API 返回 license=NONE），**不能直接复用代码**。

### 管线
- 浏览器 Web Speech API 采集 partial/final 转写，通过 WebSocket 发给 FastAPI（README）。
- 触发（https://github.com/asher14333/MindMesh/blob/b7210c16408177741b02b5ff9fabd179a6a53904/backend/app/services/trigger_engine.py ，https://github.com/asher14333/MindMesh/blob/b7210c16408177741b02b5ff9fabd179a6a53904/backend/app/config.py ）：
  - 每收到一条 `SpeechFinalEvent` 并且有未处理文本，就触发；
  - 后台每 250ms 检查一次静默，距离上个音频块 ≥ `pause_threshold_seconds=1.5` 秒也触发；
  - 每次生成后冷却 `generation_cooldown_seconds=1.0`；
  - LLM 超时 8 秒，重试 1 次。
- 先跑规则分类器（`intent_classifier.py`），再调 LLM（默认 `gpt-4o-mini`，`temperature=0.2`，`response_format=json_object`）（https://github.com/asher14333/MindMesh/blob/b7210c16408177741b02b5ff9fabd179a6a53904/backend/app/services/model_orchestrator.py ）。
- user prompt 包含：「New transcript chunk」（只有增量）、当前图类型、**完整的上一版图 JSON**、会议上下文摘要。
- 系统提示词要点：「DO NOT copy transcript sentences into nodes」；四种图类型 flowchart / mindmap / timeline / orgchart 及其触发线索（"first/then"、"either/option"、"X will do Y"、"types of"）；一对多句式必须做成分支，不能连成链；「Max 12 nodes, max 16 edges, max 56 chars per label」；「The SAME concept must always use the SAME key across calls — keys are stable IDs」；「Return the FULL canonical graph」；闲聊返回 `noop`；"actually/no/instead" 标为 `correction`；只有话题根本变化时才 `replace`。
- 响应带置信度：`action=noop` 且 `confidence<0.65` 时直接丢弃（https://github.com/asher14333/MindMesh/blob/b7210c16408177741b02b5ff9fabd179a6a53904/backend/app/services/pipeline.py ）。
- 用 `request_id` 丢弃过期响应：`ai_response.request_id == state.last_request_id`（pipeline.py）。

### 增量更新与位置稳定
- 模型每次返回全图，服务端按稳定 key 和上一版做 diff，生成 `add_node/update_node/remove_node/add_edge/...` 操作（https://github.com/asher14333/MindMesh/blob/b7210c16408177741b02b5ff9fabd179a6a53904/backend/app/services/diagram_generator.py ）。
- 结构变动上限：增删节点数超过 `max(2, 30% 当前节点数)` 时放弃 patch，改为整图 replace。`patch.base_version` 和服务端版本不一致时也走 replace（diagram_generator.py、pipeline.py）。
- 布局在服务端（https://github.com/asher14333/MindMesh/blob/b7210c16408177741b02b5ff9fabd179a6a53904/backend/app/services/render_adapter.py ）：新增节点按索引放进固定网格（流程图横排，间距 240+40；思维导图根节点居中，其余 3 列；组织图第一行根节点）。`update_node` 如果没带位置就**保留原位置**。有 `branch` 边时整图重新做树状布局（父节点在上，子节点居中排在下方）。
- 用户手动编辑：前端 React Flow 允许拖动，拖完只写回本地 store（https://github.com/asher14333/MindMesh/blob/b7210c16408177741b02b5ff9fabd179a6a53904/Client/components/process-canvas.tsx 第 245–253 行）。我没看到同步回后端的代码，所以后端下一次 replace 会覆盖用户的拖动（推断，未运行验证）。

---

## 3. conversation-canvas（aravindsriraj/conversation-canvas，Milan AI Week 2026 黑客松）

源码：https://github.com/aravindsriraj/conversation-canvas/blob/f99130559d2399df67491b86b5509e41d4b41b84/README.md 。没有 LICENSE（license=NONE）。README 写的是 tldraw v3，但 package.json 实际是 `tldraw ^5.0.1`（https://github.com/aravindsriraj/conversation-canvas/blob/f99130559d2399df67491b86b5509e41d4b41b84/package.json ）。

### 管线
- STT：Speechmatics realtime，带说话人分离（README）。
- 缓冲：只收 final 段；滑动窗口 90 秒；最后一段 final 之后 **3 秒防抖** 触发一次 tick（https://github.com/aravindsriraj/conversation-canvas/blob/f99130559d2399df67491b86b5509e41d4b41b84/lib/orchestrator/buffer.ts ，https://github.com/aravindsriraj/conversation-canvas/blob/f99130559d2399df67491b86b5509e41d4b41b84/server/room.ts 第 101–105 行）。
- 每个房间一把互斥锁：上一个 tick 没结束就跳过本次，等下次防抖用更新的窗口（room.ts 注释说 MODE B 一个 tick 要 8–12 秒）。
- 两级模型：
  1. `gemini-3.1-flash-lite` 二分类：MODE A（被动记录）还是 MODE B（对画布发命令，如「把提案改成红色」）。注释称约 $0.075/M 输入 token、p95 150–300ms，作者自己的说法，没有基准数据。注释里还提到之前的正则分类器「kept missing turns of phrase」（https://github.com/aravindsriraj/conversation-canvas/blob/f99130559d2399df67491b86b5509e41d4b41b84/lib/orchestrator/classifier.ts ）。
  2. MODE A 用 `gemini-3-flash-preview` 的 `streamObject` 单次调用，temperature 0.2，边流式解析边逐条广播 Action。MODE B 交给多步 ReAct voice agent（https://github.com/aravindsriraj/conversation-canvas/blob/f99130559d2399df67491b86b5509e41d4b41b84/lib/orchestrator/loop.ts ）。
- 提示词（https://github.com/aravindsriraj/conversation-canvas/blob/f99130559d2399df67491b86b5509e41d4b41b84/lib/orchestrator/prompt.ts ）：被动模式只抽「提案/决定/承诺/阻塞/开放问题」五种卡片和 `supports/counters/depends_on/decides/blocks/contradicts` 六种关系；「NEVER duplicate a decision for the same topic… use update_card」；「ID DISCIPLINE: … USE ITS EXISTING ID」；布局只给语义提示（`below/right_of/grid/cluster_with`）；「RE-EMIT GUARD: the RECENT_ACTIONS section lists everything already emitted… Speechmatics just re-finalized a partial」。输入还包括画布快照（id、类型、摘要）、说话人表、最近 25 条 Action、长期记忆（超过 50 条消息后由 flash-lite 压缩）。

### 增量更新、位置、人工编辑
- **只追加 Action**，从不重生成整张图。服务端三重去重：(from,to,kind) 连线去重、L3 组件单例、文本重叠去重。重复的 decision 会改写成对原卡片的 `update_card`（loop.ts `StreamingDedup`）。
- 位置由确定性布局器决定，LLM 不给坐标（https://github.com/aravindsriraj/conversation-canvas/blob/f99130559d2399df67491b86b5509e41d4b41b84/lib/actions/layout-resolver.ts ）：
  1. 按类型分泳道：提案 y=80、决定 y=300、承诺/阻塞/问题 y=520、组件 y=760，便签单独一列；
  2. LLM 的 `below/right_of` 提示如果引用了真实存在的 shape，就覆盖默认位置；
  3. 碰撞检测：向下按 GAP=64 逐步挪，最多 20 次。注释写道：「shipped overlapping cards because Gemini's hints are inconsistent」。
- 已有 shape 只在 `createShapeIfMissing` 时创建，不会被重新定位（https://github.com/aravindsriraj/conversation-canvas/blob/f99130559d2399df67491b86b5509e41d4b41b84/lib/actions/apply.ts ）。所以用户拖过的位置不会被覆盖。但 `update_card`（包括去重后合成的那种）会覆盖卡片文字，没有「用户改过就锁定」的机制（推断）。

---

## 4. 知了 Cicada 的鱼骨图（wquguru/cicada）

源码：https://github.com/wquguru/cicada/blob/8450cb9d43f1367ac9ef1cc0c0d5a128272c70b0/Sources/Cicada/Notes.swift 。许可证 Apache-2.0（https://github.com/wquguru/cicada/blob/8450cb9d43f1367ac9ef1cc0c0d5a128272c70b0/LICENSE ）。macOS 原生 Swift/SwiftUI，另有 Windows/Linux/Android 版。

### 管线
- 鱼骨图不是单独的管线，而是实时纪要 JSON 里某个话题的一个可选字段。每次纪要更新都是一次 LLM 调用，返回整份纪要 JSON（Notes.swift 第 110–112 行注释：「every ~45 s with new speech, one Ling call sends the current notes plus the new lines and gets the whole updated notes back as JSON. A failed or odd reply never replaces good notes」）。
- 触发：每 5 秒检查一次；第一次出纪要的条件是 ≥3 行或已过 20 秒，之后每 45 秒一次（`interval = 45`，Notes.swift 第 157、262 行）。
- 模型：蚂蚁 Ling `Ling-3.1-flash`，也可以换成自建的 OpenAI 兼容服务；README 称「开会一小时大概 0.15 到 1 元人民币」（https://github.com/wquguru/cicada/blob/8450cb9d43f1367ac9ef1cc0c0d5a128272c70b0/README.zh-CN.md ）。
- 鱼骨图相关提示词（Notes.swift 第 577–591 行）：JSON schema 中 `"fishbone": null or {"question": ..., "branches": [{"label": "2-6 words", "detail": "one short line", "t": seconds, "evidence": ["L5"]}]}`；规则「Add a fishbone only to a topic that discusses causes, reasons or factors (3-5 branches); otherwise null」；「Every topic, point, branch… needs evidence: the ids of the lines that say it. Never state anything the transcript does not say」。
- 解析时 branches 少于 2 条就丢弃整个鱼骨图；分支时间取它引用的转写行的时间，**不信模型自己给的 t**，因为 changelog 记录过模型给出的 t 超过了会议结束时间（Notes.swift 第 733–741 行；https://github.com/wquguru/cicada/blob/8450cb9d43f1367ac9ef1cc0c0d5a128272c70b0/CHANGELOG.md ）。
- 渲染：SwiftUI `FishboneView`，竖向排布适配窄窗口（问题在顶部，分支在脊柱两侧交替）；没有转写证据支撑的分支画成虚线；点分支可以跳到原话（https://github.com/wquguru/cicada/blob/8450cb9d43f1367ac9ef1cc0c0d5a128272c70b0/Sources/Cicada/NotesViews.swift 第 568–627 行）。可复制为图片或文字（CHANGELOG）。

### 增量与稳定性（值得照抄）
- 冻结旧内容：实时更新只允许改最后 `editableTopics = 2` 个话题，更早的话题只以标题形式发给模型，并要求「settled: kept as they are, do not return them」（Notes.swift 第 167、307 行及提示词）。注释写明原因：以前每次重写全部话题，「about an hour in, it was cut off at max_tokens or fell into a loop and the notes stopped updating」。
- 防过早切分话题：当前话题开始不到 2 分钟时，提示词明确写「do not start a new topic in this update」。
- 防顺序跳动：`keepingOrder` 让术语保持旧顺序，因为「the model often returns the same names reshuffled, which would make the chips jump on every update」（Notes.swift 第 88–91 行）。
- 话题数上限：实时纪要话题过多时，合并时间最短的相邻两个已定话题，不再全塞进最后一个话题（注释：以前最后一个话题会变成「最新…又…补充…」的长文）。
- 会后最终版：从空白重新整理全部内容；超过 100 分钟的会议按 60 分钟一段分别整理再合并（Notes.swift 第 615 行起）。
- 人工编辑：`applyEdited` 用编辑后的版本作为下一次实时更新的 base（Notes.swift 第 216–221 行）；但最终版从空白重写，会中的手改是否保留**未核实**。

---

## 5. Agent Draw（ritza-co/tldraw-agent-draw-demo）与 tldraw agent 模板

### tldraw agent 模板（tldraw/tldraw 仓库 templates/agent）
源码：https://github.com/tldraw/tldraw/blob/8a6bedb85cfb6236730deae10fcbb51cca5a1466/templates/agent/README.md 。模板本身是 MIT（https://github.com/tldraw/tldraw/blob/8a6bedb85cfb6236730deae10fcbb51cca5a1466/templates/agent/LICENSE.md ），依赖的 tldraw SDK 不是（见第 13 节）。
- 每个 action 是一次工具调用，浏览器边收流边执行，然后把结果和新的画布视图发回模型；模型不再调用工具时循环结束，「A turn has at most 20 follow-up requests」（README）。
- 模型能看到的「prompt parts」：用户消息、选区、用户视口、上下文区域、**用户最近的操作**（UserActionHistoryPartUtil）、agent 视口截图、视口内 shape 的简化格式（FocusedShape：`_type/x/y/w/h/color/fill/text/note`）、视口外 shape 的聚类摘要、会话历史、画布 lint 结果（README；https://github.com/tldraw/tldraw/blob/8a6bedb85cfb6236730deae10fcbb51cca5a1466/templates/agent/shared/format/FocusedShape.ts ）。
- 系统提示词规则（https://github.com/tldraw/tldraw/blob/8a6bedb85cfb6236730deae10fcbb51cca5a1466/templates/agent/worker/prompt/sections/rules-section.ts ）：每个 shape 带 `note` 字段（用户看不见），记录意图，方便后续轮次理解；箭头必须用 `fromId/toId` 绑定；「Be sure not to create arrows twice」；文本要预估占用空间。
- 默认模型通过 OpenRouter/Anthropic/Google/OpenAI 提供，README 写「We recommend Anthropic for best results」。

### Agent Draw（在模板上加「框选区域 + 说话」）
源码：https://github.com/ritza-co/tldraw-agent-draw-demo/blob/35555bdaa63f1cb6cbf6c1004e3090985564611a/README.md ；技术文章：https://techstackups.com/articles/tldraw-agent-draw/ （2026-06-24）。
- 交互：用 Agent draw 工具拖出一个矩形，松手开始录音；点 Stop 或拖下一个矩形就结束录音。多个区域进 FIFO 队列串行处理，状态依次为 Recording → Queued → Transcribing → Drawing（README）。
- STT：整段音频 POST 到 Worker，再转发给 Mistral `voxtral-mini-transcribe-2507`（https://github.com/ritza-co/tldraw-agent-draw-demo/blob/35555bdaa63f1cb6cbf6c1004e3090985564611a/worker/routes/transcribe.ts ）。**不是流式**。文章说早期版本试过边说边画，「It was laggy and the value was fuzzy, so we cut it」。
- 提示词（https://github.com/ritza-co/tldraw-agent-draw-demo/blob/35555bdaa63f1cb6cbf6c1004e3090985564611a/client/capture/requestDrawInArea.ts ，https://github.com/ritza-co/tldraw-agent-draw-demo/blob/35555bdaa63f1cb6cbf6c1004e3090985564611a/worker/prompt/sections/rules-section.ts ）：以「CAPTURED AREA REQUEST」开头；先判断该用哪种视觉形式（指名的图形 / 单个插画 / 带标签的图 / 图表 / 只有用户明确要求时才写原文）；「Build the COMPLETE result in this turn」；「The single most common failure mode is dumping the spoken sentence verbatim as one block of text」；所有 shape 必须放在区域内。
- 用 `agent.prompt` 完整循环，不用单轮 `agent.request`，否则模型画一个 shape 就停（文章）。
- 提速：去掉 `setMyView` 和 `review` 两个 action 后，「roughly halved the number of model calls per capture」（文章）。
- 默认模型 `google/gemini-2.5-flash`（OpenRouter）（README）。文章对比：claude-opus-4.8 两个任务都画得好；claude-haiku-4.5 流程图可以，插画退化成基本图形；gemini-2.5-flash-lite「seems to give up early」。
- 增量：每次只在新区域新增内容，不改旧内容；用户编辑不会被覆盖（设计如此）。
- HN 上有用户反馈「two circles connected by an arc」画到一半崩了，作者回复弱模型问题多（https://news.ycombinator.com/item?id=48805475 ）。作者还说「I love mermaid but have struggled to get my agents to be any good at it」（同一帖）。

---

## 6. BUD（budcanvas.app，OpenAI Build Week 黑客松）

只有 Devpost 和官网，没有源码，**以下都是作者自述**。
- 管线：`Cartesia STT → GPT-5.6-Terra (Responses API) → Cartesia TTS → Canvas Tools`；从 WebRTC 改成了可打断的分段管线；用户开口时立即取消 TTS 和进行中的 API 请求，并回滚状态（https://devpost.com/software/bud-h0lngs ）。
- 「Programmatic Tool Calling」：模型不逐个调工具，而是写 JS 在 OpenAI 托管的 V8 沙箱里批量执行画布操作。作者称「Multi-step builds dropped from 1–2 minutes to under 10 seconds」（同上，没有基准数据）。
- 「semantic scene compiler」负责把 LLM 输出转成间距合理的 Excalidraw 元素（同上）。
- 有矛盾：Product Hunt「built with」写 canvas 用的是 Excalidraw React 包（https://www.producthunt.com/products/bud-voice-canvas/built-with ）；macOS 下载页却要求「Accessibility permissions to interact with Excalidraw on your behalf」和「the official Excalidraw workspace open in your web browser」（https://www.budcanvas.app/download ），说明桌面版可能通过辅助功能操作 excalidraw.com。实际是哪种**未核实**。

---

## 7. Wavelength（wavelengthflow.com）

- 模型：「open-weight models on Groq: gpt-oss for structuring and Whisper for speech to text」，开启零数据保留（https://wavelengthflow.com/faq.html ）。
- 流程：按住麦克风时实时生成图，「a final cleanup pass tidies it when you let go」（https://wavelengthflow.com/voice-to-diagram.html ）。没有公开提示词和触发间隔（未核实）。
- 关键 UX：「Solid arrows are connections stated directly in the speech. Dashed amber arrows are inferences, left unconfirmed until you accept or correct them」（https://wavelengthflow.com/demo.html ）。这是本次调研中处理幻觉连线最明确的产品设计。
- 节点类型：actor、step、system、decision、data store；布局可选纵向、横向、按负责人分泳道；**明确不做** UML、ERD、网络拓扑（https://wavelengthflow.com/ ）。
- 可导入 Visio/draw.io/Mermaid（「in the browser with no AI call」）；可导出 Mermaid、BPMN 2.0、draw.io、PPTX 等；有 MCP server（106 个工具，包括 add_nodes/update_edges 等细粒度编辑）（https://wavelengthflow.com/ ）。
- 价格：免费版每月 10 分钟麦克风时长；Pro $7/月（同上）。
- 编辑方式：语音、文字指令、右键、手动拖拽（同上）。增量算法未公开。

## 8. TalkGraph（talkgraph.ai）

- 技术栈：「Deepgram converts the live audio into text」；结构化用「an approved language model」（**未披露具体模型**）；Supabase 存储，Vercel 托管（https://talkgraph.ai/security ）。
- 行为（营销页）：「every few seconds structures what you've said into cards」；卡片类型有 idea/fact/decision/action/question/risk，还能根据口述数字生成 chart；「Hedged thoughts ("maybe we should…") land as faded drafts until you commit to them out loud」；停顿时会问一个短问题；卡片可拖动、合并、重命名（https://talkgraph.ai/voice-to-mind-map ）。
- 免费 250 节点/月，「about 2 hours of talking」（同上）。增量、布局、冲突处理均**未核实**。

## 9. D-Sket Canvas（canvas.d-sket.dev，韩国 DataSketchers）

- 会议实时思维导图（官方技术博客 https://data-sketchers.com/blog/realtime-meeting-mindmap ，2026-03-22）：浏览器 MediaRecorder 每 250ms 一个音频块，经 WebSocket 发给自建 Whisper v2 服务；**并行**跑韩语 KeyBERT 关键词抽取和 embedding 聚类；WebRTC VAD 过滤静音；PyAnnote 实时说话人分离，前 30 秒用于校准。
- 生成节点的规则（不是 LLM）：同一关键词出现 ≥3 次才成为候选；embedding 相似度 ≥0.85 就并入已有节点；节点按说话人着色；「회의 끝나고 보면 평균 8~15개의 노드」（会议结束平均 8–15 个节点），用户再手动整理位置（同上）。
- 博客引用用户访谈：「지연이 0.5초 이상이면 신뢰가 깨진다」（延迟超过 0.5 秒信任就崩），所以优先响应速度而非准确率（同上）。
- 另有提示词一键生成流程图/思维导图/ERD，号称「Under 10 seconds」（营销，https://canvas.d-sket.dev/ ）。画布是自研无限画布（https://sinbum.com/blog/canvas-buildlog-01 ）。

## 10. meetgraph（sensein/meetgraph）

注意：这个项目做的是**会后知识图谱**，不是会中白板。我只找到这一个叫 meetgraph 的项目；如果你们指的是别的同名项目，**未找到**。
- PyQt6 桌面应用，采集麦克风和系统音频，本地 Whisper/MLX 或 OpenAI 兼容 STT（https://github.com/sensein/meetgraph/blob/1f039ad6a9764c17716d83b3ec9a730d0ff274a0/README.md ）。
- 实时纪要：定时器 15 秒（https://github.com/sensein/meetgraph/blob/1f039ad6a9764c17716d83b3ec9a730d0ff274a0/meeting_transcriber/ui.py 第 488 行）；有了 running summary 后**只摘要新增文本**，新增不足 40 字符就跳过（ui.py 第 4039–4048 行）；然后 `merge_summaries` 按话题名小写去重、只追加（https://github.com/sensein/meetgraph/blob/1f039ad6a9764c17716d83b3ec9a730d0ff274a0/meeting_transcriber/agent.py 第 388 行起）。
- Graph 标签页：自写 Fruchterman-Reingold 力导向布局，节点初始化在圆周上，所以「the same graph lays out the same way each refresh」（ui.py 第 6286 行起），用确定性布局保证位置稳定。
- 导出 RDF（JSON-LD/Turtle），自带 MCO 本体（README）。

---

## 11. 通义听悟（脑图/思维导图）

- API：`Summarization.Types` 里加 `MindMap`，返回 `MindMapSummary[]`，每个节点 `{Title, Topic: [...]}` 递归；「目前思维导图最多生成四个层级」；「您需要将结果传递给前端框架，渲染出思维导图图片」，即只给数据，**不负责渲染**（https://help.aliyun.com/zh/tingwu/large-model-summary ）。
- 时机：可以在创建实时会议时设置，也可以会后基于同一 TaskId 重跑；结果在任务完成后通过 OSS 链接下载（同上 FAQ 和示例）。所以**不是会中实时更新的脑图**。
- 依赖：全文摘要依赖章节标题，开启全文摘要时会自动触发章节速览（同上）。
- 规模：2024-03-26 release notes 写「支持2万字以内的导图生成（约80分钟音频）」（https://help.aliyun.com/zh/tingwu/release-notes ）。消费端 App 宣传「最多支持五级xmind脑图」（https://startup.aliyun.com/info/1083329.html ，2024-03），和 API 的四层不一致，现状**未核实**。
- 基于通义千问（同 large-model-summary 文档）。

## 12. 飞书（妙记章节 / 智能纪要 / 会中 AI 视图）

- **会中 AI 视图**（飞书 V7.50+，帮助中心 2026-08-04 更新）：「AI 会实时、自动提炼会议信息，以结构化图表形式展示」；点节点可以看信息来源，或用「手指」图标反馈；默认开启，只有主持人可关；会后同步进智能纪要文档的**画板**，可「一键导出 AI 视图」；不支持 1 对 1、网络研讨会、分组讨论、视频面试（https://www.feishu.cn/hc/zh-CN/articles/092000440742 ）。具体有哪些图表类型、多久更新、如何增量，**官方未披露**。
- **章节**：「智能伙伴能精准对会议做章节切分，并对各章节做总结，支持点击跳转对应章节」（飞书官网文章，营销性质，https://www.feishu.cn/content/article/7577317993749269689 ）。
- **可视化纪要**：同一篇官网文章写「AI 自动为你生成可视化表格、流程图、饼图」和「会议封面图」。媒体报道称复杂议题会生成思维导图（https://post.smzdm.com/p/a3r2ve7k/ ，转载硅星人，非官方）。竞品 Ai好记的对比文说「飞书妙记不支持根据音视频直接自动生成AI思维导图」（https://aihaoji.com/blog/aihaoji-vs-feishu-minutes ，竞品自述，有利益冲突）。说法互相矛盾：**妙记是否能生成思维导图未核实**，能确认的是「AI 视图 + 纪要画板」。
- 飞书官方另一篇文章展示了通过飞书 CLI + Agent + skill 把会议妙记画成可编辑「飞书画板」的「一图流」做法（https://www.feishu.cn/content/article/7647452839320718324 ）。

## 13. Miro AI / FigJam AI / Zoom Whiteboard

**Miro**
- 生成方式：「Miro AI writes the diagram or mindmap directly onto the canvas as an editable Mermaid-based Diagram」；追问修改时「Miro AI edits the existing diagram in place rather than regenerating a draft」（https://help.miro.com/hc/en-us/articles/28782102127890-Miro-AI-with-Diagrams-and-mindmaps ）。
- 模型（Miro AI reference，2026-06-01 更新）：创建流程图用 GPT-5.4 + GPT-5.4 mini；编辑流程图/思维导图/ERD 用 GPT-5.4 + GPT-5.4 mini + Claude Sonnet 4.6；创建思维导图和 ERD 用 GPT-5.4 mini；手绘图数字化用 Claude Sonnet 4.6 + Haiku 4.5；整板布局用 Claude Opus 4.7（https://help.miro.com/hc/en-us/articles/20970362792210-Miro-AI-reference ）。创建和编辑用不同的模型组合。
- 颜色：没有指定时用默认配色，给模糊描述时映射到 Miro 的无障碍柔和色板（同 Diagrams 文档）。
- 是否能从会议转写实时生成：**未找到官方文档**，以上都是提示词驱动。

**FigJam AI**
- 「Currently, FigJam AI can create flow charts, Gantt charts, and org charts」，同一页还列出思维导图、时间线的提示示例；修改方式是「Adjust your prompt and click Make to create a new visual」，即重新生成（https://help.figma.com/hc/en-us/articles/18706554628119-Make-boards-and-diagrams-with-FigJam-AI ）。
- 思维导图节点可以「Generate ideas」用 AI 扩展（https://help.figma.com/hc/en-us/articles/18917944627095-Create-mindmaps-in-FigJam ）。没有会议实时生成。

**Zoom Whiteboard**
- AI Companion 可生成 stickies / table / text / mindmap，结果可 Refine，思维导图有 Simplify / Expand（https://support.zoom.com/hc/en/article?id=zm_kb&sysparm_article=KB0057610 ，2026-10-17 更新）。
- 2025-09-21：「Generate whiteboard from AI Companion meeting content… generated using meeting data, including title, description, roster, and attachments」，也可以把以往会议摘要转成白板；2026-04-19：AI 聊天面板可以「creates canvas objects from prompts or Zoom meeting context like transcripts」（https://support.zoom.com/hc/en/article?id=zm_kb&sysparm_article=KB0063731 ）。都是**用户触发的一次性生成**，不是持续更新。

## 14. Excalidraw：文字生成图 与 mermaid-to-excalidraw

### 文字生成图
- 前端（MIT 开源）把对话 POST 到 `/v1/ai/text-to-diagram/chat-streaming`（SSE），LLM 返回 Mermaid，再用 `@excalidraw/mermaid-to-excalidraw` 转成元素插入画布。后端闭源（https://github.com/excalidraw/excalidraw/blob/master/excalidraw-app/components/AI.tsx ）。
- 解析失败时有「AI Repair」：把原 Mermaid 和报错拼成「Fix the error in this Mermaid diagram…」重发（https://github.com/excalidraw/excalidraw/blob/master/packages/excalidraw/components/TTDDialog/TextToDiagram.tsx 第 152–162 行）。
- 限额：免费编辑器每天 10 次 AI 请求；Plus 每天 100 次；Plus 可让管理员配置 OpenAI/Anthropic/Gemini/Grok/OpenRouter 的 BYOK（https://plus.excalidraw.com/ai ）。
- 许可：excalidraw/excalidraw 是 **MIT**（GitHub API license=MIT）。excalidraw 包 0.18.0 依赖 `@excalidraw/mermaid-to-excalidraw: 3.6.0`（packages/excalidraw/package.json）。

### mermaid-to-excalidraw（问题 5）
源码：https://github.com/excalidraw/mermaid-to-excalidraw/blob/4187e848b2bf13f26eafb2ab0526daad74240f56/README.md ，npm latest = **3.6.0**（2026-11-04 发布），MIT。
- **v3 起不再依赖 Mermaid**，改用 Pele（https://pele.run ）在无 DOM 渲染的情况下解析和布局，读取 Pele 输出的 SVG 几何属性再转成 Excalidraw 元素。仍需要 `DOMParser`，Node 下要配 jsdom（README；https://github.com/excalidraw/mermaid-to-excalidraw/blob/4187e848b2bf13f26eafb2ab0526daad74240f56/CHANGELOG.md v3.0.0；https://github.com/excalidraw/mermaid-to-excalidraw/blob/4187e848b2bf13f26eafb2ab0526daad74240f56/src/pele/svg.ts ）。v3 的布局和 v2 不同。v2 仍在 `release/2.x` 分支维护（README）。
- **转成可编辑元素的类型**（以源码 https://github.com/excalidraw/mermaid-to-excalidraw/blob/4187e848b2bf13f26eafb2ab0526daad74240f56/src/diagrams/index.ts 的 `DIAGRAM_TYPES` 为准）：
  - `flowchart`（包括 `swimlane-beta` 泳道；节点映射为矩形/圆角/椭圆/菱形，**六边形、圆柱、平行四边形等其他形状都变成矩形**；子图变成带标题的框；`click` 变成元素链接）；
  - `sequenceDiagram`（参与者、生命线、激活条、note、loop/alt/opt/par/critical/break、autonumber）；
  - `classDiagram`（成员、注解、泛型、note、关系、namespace 画成框）；
  - `erDiagram`（属性、PK/FK、基数标记）；
  - `stateDiagram`（复合状态、并发区、choice/fork/join）；
  - `gantt`、`quadrantChart`、`timeline`：3.5.0（2026-11-04）起转成元素（commit 记录「feat: gantt chart support (#120)」「quadrant chart support (#121)」「timeline support (#122)」，均早于 3.5.0 发布）。
  - 官方文档页 https://plus.excalidraw.com/docs/mermaid-to-excalidraw/supported-diagrams **还没更新**，仍写着「Gantt charts… inserted as an SVG image」；以 README 和源码为准。
- **退化成单张 SVG 图片的类型**：除上述 8 类以外 Pele 支持的其他类型，**包括 mindmap**、pie、journey、gitGraph 等（README：「Other types are inserted as an image」；`isConvertedDiagramType` 之外的类型走 `graphImage`）。已支持的类型如果转换失败也会退化成图片（文档页）。pie、journey 等具体是否都能被 Pele 渲染，我没逐个验证，**未核实**。
- 样式：支持 `style/classDef/class` 颜色；`elbow: true` 可把流程图和状态图的边画成直角折线（README；CHANGELOG）。
- **对我们的影响**：Shimmer 式「LLM 出 Mermaid」管线里的 mindmap，用这个库转出来只能是图片，不能编辑。要做可编辑脑图，必须自己做树布局，或者让 LLM 输出 flowchart。

## 15. tldraw 许可证现状（问题 4）

- SDK 默认许可只允许开发环境使用；生产环境必须有 license key，类型有 100 天试用、商业（年付，「value-based pricing」，没有公开价格）、hobby（非商业，需审批，**必须显示 "made with tldraw" 水印**）（https://tldraw.dev/community/license ，https://tldraw.dev/pricing ，https://tldraw.dev/sdk-features/license-key ）。
- 技术强制：生产环境（HTTPS、非 localhost、`NODE_ENV=production`）没有有效 key 时，「after five seconds, stops rendering the editor」；key 在客户端离线校验，编码了允许的域名（https://tldraw.dev/sdk-features/license-key ）。
- **对开源项目**：「If you wish to include tldraw in an open source project… the SDK itself must remain under its original license. This means that you and your downstream users will require their own trial, commercial, or hobby license in order to use the SDK in production」（https://tldraw.dev/community/license ）。
- 桌面应用：文档没有提 Electron/Tauri。按检测规则，`file://` 或自定义协议不是 HTTPS，SDK 可能把它当成开发环境而不拦截，但**许可条款仍禁止在面向用户的生产环境使用**（LICENSE.md：「Not to use the Software in Production Environments」，https://github.com/tldraw/tldraw/blob/8a6bedb85cfb6236730deae10fcbb51cca5a1466/LICENSE.md ）。官方对桌面端的立场**未核实**。
- 对照：Excalidraw 是 MIT；mermaid-to-excalidraw 是 MIT；React Flow（@xyflow/react）是 MIT（MindMesh 用的就是它；许可证我没有在本次调研中单独核对，**未核实**）。

---

## 16. 延迟与成本：公开数据汇总

| 数据 | 来源性质 | 来源 |
|---|---|---|
| Shimmer 每 60 秒重生成，3 次串行 LLM 调用 + 最多 2 次校验重试 | 线上 JS 包 | 第 1 节 |
| MindMesh：静默 1.5 秒触发、1 秒冷却、LLM 超时 8 秒 | 源码 | https://github.com/asher14333/MindMesh/blob/b7210c16408177741b02b5ff9fabd179a6a53904/backend/app/config.py |
| conversation-canvas：3 秒防抖；flash-lite 分类 p95 150–300ms；MODE B 一个 tick 8–12 秒 | 源码注释（作者自述） | https://github.com/aravindsriraj/conversation-canvas/blob/f99130559d2399df67491b86b5509e41d4b41b84/lib/orchestrator/classifier.ts ，https://github.com/aravindsriraj/conversation-canvas/blob/f99130559d2399df67491b86b5509e41d4b41b84/server/room.ts |
| Cicada：45 秒一次；每小时 0.15–1 元人民币 | 源码 + README | 第 4 节 |
| BUD：多步构图从 1–2 分钟降到 <10 秒 | 黑客松自述 | https://devpost.com/software/bud-h0lngs |
| Agent Draw：去掉 2 个 action 后模型调用次数约减半 | 技术博客 | https://techstackups.com/articles/tldraw-agent-draw/ |
| D-Sket：250ms 音频块；串行 >1 秒、并行后约 250ms；延迟 >0.5 秒信任就崩 | 官方技术博客 | https://data-sketchers.com/blog/realtime-meeting-mindmap |
| Excalidraw：免费 10 次/天，Plus 100 次/天 | 官方 | https://plus.excalidraw.com/ai |
| 通义听悟：导图支持 2 万字以内（约 80 分钟） | 官方 release notes | https://help.aliyun.com/zh/tingwu/release-notes |
| Wavelength：免费 10 分钟/月麦克风时长，Pro $7/月 | 官方 | https://wavelengthflow.com/ |

各产品**端到端「说完到图更新」的实测延迟**：没有一家公开，**未核实**。

## 17. 已知失败模式（有出处的）

1. **随会议变长而崩**：全量重写的输出会随会议增长，大约 1 小时后撞上 max_tokens 或陷入循环，更新停止（Cicada 注释，https://github.com/wquguru/cicada/blob/8450cb9d43f1367ac9ef1cc0c0d5a128272c70b0/Sources/Cicada/Notes.swift 第 304–306 行）。Shimmer 用「>15 nodes consolidate」限制规模；MindMesh 硬上限 12 节点/16 边。
2. **最后一个话题越写越长**：话题数到上限后，新内容全塞进最后一个话题，摘要变成「最新…又…补充…」（Cicada 注释）。
3. **重复节点**：STT 重新 finalize 同一句话导致重复抽取（conversation-canvas RE-EMIT GUARD 和 dedup 注释）。
4. **元素乱跳**：模型每次返回顺序打乱，标签芯片跳动（Cicada `keepingOrder`）；Mermaid 整图重排导致位置不稳定（Shimmer 架构决定，作者也认为 tldraw 更合适）。
5. **布局重叠**：LLM 给的位置提示不一致，卡片重叠（conversation-canvas layout-resolver 注释）。
6. **幻觉时间戳**：模型给的时间超过会议结束时间（Cicada CHANGELOG）。
7. **幻觉连线/内容**：Wavelength 用虚线标出推断出的连线；Cicada 没有证据行支撑的分支画虚线，提示词强制「Never state anything the transcript does not say」。
8. **把原话整句贴上画布**：Agent Draw 称这是「single most common failure mode」。
9. **弱模型半途放弃或崩溃**：Agent Draw 文章和 HN 反馈。
10. **Mermaid 语法错误**：Shimmer 和 Excalidraw 都需要「报错回灌重试」；Agent Draw 作者说 agent 写 Mermaid 写不好。
11. **图类型来回跳**：Shimmer 每轮重新分类（代码推断）；MindMesh 类型变化时直接整图 replace。
12. **话题切得太碎**：模型太早开新话题，Cicada 要在提示词里明确禁止。

---

## 18. 对我们的启示

### 可直接借鉴的做法
1. **中间表示用「带稳定 ID 的语义图 JSON」，不要让 LLM 直接出坐标或 Mermaid**（MindMesh 的 key 规则，conversation-canvas 的 Action 联合类型 + Zod 校验）。LLM 只决定节点、边和类型，坐标交给确定性布局器。需要 Mermaid 导出时由 JSON 生成。
2. **服务端 diff + 结构变动阈值**：模型返回全图（实现简单、模型不易出错），服务端按 ID diff 成 patch；变动超过约 30% 时提示「结构已重组」，再整体替换（MindMesh）。
3. **冻结已定内容**：只允许 LLM 改最近 N 个话题/子树，更早的只以标题作上下文（Cicada `editableTopics=2`）。这样同时解决 token 增长、输出不稳定和用户编辑被覆盖三个问题。对按句转写的长会议最关键。
4. **证据绑定**：每个节点/分支带上它引用的转写句 ID，时间从证据行取，没有证据就虚线显示，点击跳回原句（Cicada + Wavelength）。我们本地 SQLite 已经存了逐句转写，天然支持。
5. **触发策略**：「final 句子 + 静默约 1.5 秒」或「防抖 3 秒」触发小步更新，加上「每 45–60 秒或话题切换时」做一次整理；单会话互斥，在途请求未完成时丢弃新 tick，并用 request_id 丢弃过期响应（MindMesh、conversation-canvas、Cicada）。
6. **便宜模型先做路由**：先用 flash-lite 级模型判断「有没有新结构 / 是不是对画布的命令 / 该用哪种图」，再决定要不要调大模型（conversation-canvas classifier）。图类型一旦确定就锁住，或要求高置信度才切换，避免每轮重分类。
7. **布局器做成「按类型分泳道 + LLM 相对位置提示 + 碰撞检测」**，已有节点永不移动（conversation-canvas）。脑图/树用确定性布局，同样的输入出同样的结果（meetgraph 圆周初始化 + 力导向；或 d3-hierarchy/ELK）。
8. **人工编辑优先**：节点上记 `userEdited/locked`，LLM 的 update 遇到锁定字段就跳过；把「用户最近的操作」放进提示词（tldraw agent 的 UserActionHistory），让模型知道哪些内容是人改过的。
9. **图类型做少而精**：会议高频的是思维导图/话题树、流程图、决定-行动-风险卡片、时间线。鱼骨图只在出现「原因/因素」讨论时作为话题附件生成，并要求 3–5 个分支（Cicada）。Wavelength 也明确只做流程图，不做 UML/ERD。
10. **提示词要点可以抄**：「不要把原句搬进节点」、一对多必须分支、闲聊返回 noop、更正词（actually/不对/改成）标为 correction、「没有新信息就原样返回」（MindMesh、Shimmer、Agent Draw）。
11. **画布选型**：
    - 需要可编辑、可开源、可商用：**Excalidraw（MIT）** 或 **React Flow（@xyflow/react）** 加自研布局；
    - 只做只读预览：Mermaid 渲染（实现最快，但不稳定、不可编辑）；
    - 想复用 Mermaid 生态：mermaid-to-excalidraw 3.6 能把 flowchart/sequence/class/ER/state/gantt/quadrant/timeline 转成可编辑元素，**但 mindmap 只能出图片**。

### 应避开的坑
1. **tldraw**：开源项目也不能把生产授权转给用户。每个下游用户要自己申请 license（hobby 带水印、需审批），生产环境无 key 5 秒后停止渲染；桌面端条款未明确。开源桌面应用基本不能选 tldraw 作主画布（tldraw.dev/community/license）。
2. **整张图每轮让 LLM 重写**（Shimmer 模式）：位置乱跳，用户编辑会丢，长会议 token 失控。Shimmer 作者自己也觉得 Mermaid 不是好载体。
3. **只在会后生成**（通义听悟 MindMap API、Zoom 一次性生成）不满足「会中实时」需求，但可以作为会后「最终整理」参考：听悟的 4 层树结构和「先章节后摘要」的依赖关系值得借鉴。
4. **规则或关键词法**（D-Sket 的 KeyBERT ≥3 次）延迟低，但只能得到关键词云式的脑图，表达不出关系和决定；可以作为 LLM 结果回来前的「占位节点」。
5. **不做去重**：按句转写会反复 finalize 和更正，必须做 ID 层面和文本相似度去重，并把最近已发出的操作喂回模型（conversation-canvas RE-EMIT GUARD）。
6. **不展示推断与证据**：会议图会被当作「会议记录」转发，推断的连线必须视觉区分（Wavelength 琥珀色虚线），否则信任受损。
7. **会后最终版覆盖会中手改**：Cicada 的最终版从空白重写。我们若做「会后整理」，必须把用户锁定的节点作为不可改的输入。
8. **轻信模型给的时间戳和数字**：时间一律从证据句推出（Cicada CHANGELOG）。

---

## 附：没查到或存疑的点
- Shimmer 未开源，细节来自线上 JS 包，可能随版本变化；「每轮都重新分类」是代码推断，没有实际运行验证。
- TalkGraph、Wavelength、飞书 AI 视图、Miro/FigJam/Zoom 的触发频率、增量算法、布局算法均未公开。
- 飞书妙记/智能纪要是否生成思维导图：官方帮助中心只确认「AI 视图」（结构化图表）和纪要中的流程图/饼图等，思维导图只有媒体报道，竞品说法相反。
- 通义听悟脑图层级：API 文档写 4 层，2024 年宣传写 5 级 xmind，现状未核实。
- BUD 是嵌入 Excalidraw 包，还是通过辅助功能操作 excalidraw.com，官方说法矛盾。
- tldraw 对 Electron/Tauri 桌面应用的授权立场没有官方说明。
- mermaid-to-excalidraw 对 pie/journey/gitGraph 等的退化行为没有逐个运行验证。
- 「meetgraph」只找到 sensein/meetgraph（会后知识图谱），如果指的是别的同名项目，未找到。
- 各产品端到端延迟没有公开的实测数据。
