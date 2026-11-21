# 会中「实时纪要」业内做法调研

调研日期：2026-11-21。来源以官方帮助中心、官方博客/更新日志、源码、论文为主；评测文、第三方博客只作线索，引用时单独标注。每条事实后附 URL。写着「未核实」的，是没找到一手来源的内容；只有营销页、没有帮助文档支撑的，标为「仅营销文案」。

---

## 0. 总览

| 产品 | 会中看到的 AI 内容 | 自动持续更新的纪要 | 按需「刚才/到目前为止」 | 会中用户输入 | 纪要能否回到原话 |
|---|---|---|---|---|---|
| Granola | 自己的笔记 + 实时转写面板；没有 AI 纪要 | 无（刻意不做） | Chat：会中 catch up | 手打笔记（影响会后纪要）、模板、Chat | 放大镜跳转写/笔记；不存音频 |
| Notion AI Meeting Notes | 实时转写 + 会中技能 | 无，停止后才生成 | 技能：Catch me up / Recap decisions | 会前 Notes、Instructions 模板、技能 | 引用悬停看片段、点击跳转写行 |
| Fireflies Live Assist | AI Notes + 转写 + Prep + AskFred | 有（官方说持续更新，频率未公开） | Catch Up（最近 1 分钟）/ Summarize So Far / Action Items | AskFred 提问、AI Skills | 笔记带时间戳（会后可跳） |
| Otter | 实时转写 + 实时摘要（Outline）+ AI Chat；2026-10 起 Live Assist 教练提示 | 有（频率未公开） | AI Chat 随时问 | 高亮、评论、AI Chat、模板 | Outline 点击跳转写源 |
| Fathom（新桌面版） | Live summary + Scratchpad | 有，按对话顺序累积 | 未核实 | Scratchpad（私有，影响个人纪要）、@同事 | 会后每条摘要链到录像时刻 |
| tl;dv | 录制界面 + 钉选按钮 | 未核实 | 未核实 | 钉选时刻→时间戳 + AI 小结 | 时间戳跳录像 |
| Jamie | Sidekick：笔记/实时转写/提问框 | 无，停止后生成 | Sidekick 自由提问（如「回顾最近 5 分钟」） | Scratchpad（不进入纪要）、标签 | 未核实 |
| Krisp | 实时转写 + 实时评论 | 无（帮助文档未提） | 未核实 | 实时评论 | 未核实 |
| Circleback | 私有笔记 | 无 | 未核实 | 私有笔记（影响 AI 纪要） | 会后点转写里的词跳录音 |
| Zoom AI Companion | Meeting questions 面板；My Notes 转写 + Ask AI | 无（会后出纪要） | 预设「Catch me up」「我的名字被提到了吗」 | My Notes 手打笔记、自定义提问 | 未核实 |
| Teams Copilot / Facilitator | Copilot 私聊；Facilitator 共享实时笔记 | Facilitator 有，开启几分钟后出现 | Copilot：晚到 5 分钟以上主动提示 catch up | 所有人可编辑 AI 笔记、@Facilitator、议程 | Copilot 回答带来源引用 |
| Google Meet | 「Take notes for me」铅笔图标；Ask Gemini | 有（"notes taken so far"），频率未公开 | Summary so far；Ask Gemini 私问 | 设置章节/长度、Ask Gemini | 会后文档带时间戳引用，点了跳转写 |
| 飞书会议 | 右侧「AI 总结」面板 + 实时字幕；AI 视图结构化图表 | 有，频率未公开 | 智能伙伴提问（总结要点、待办、发言人） | 提问、反馈 | 图表节点看来源；会后纪要原文溯源 |
| 腾讯会议 | AI 纪要：顶部概览（结论/待办）+ 每 2 分钟推送 | 有，每 2 分钟 | 「立即总结」；AI 小助手「刚刚说了什么」（1/3/5 分钟） | 提问、智能提醒（提到我/关键词/某人发言） | 会后纪要每条带时间戳跳录制 |
| 通义听悟 | 实时转写 | 消费端未见（未核实）；API 会后才算 | 未核实 | 改转写、标重点/问题/待办、摘到笔记 | 章节可跳；音字对应 |
| 讯飞听见 | 实时转写 + AI 洞察 + Chat | AI 洞察会中动态生成 | Chat 问答 | 重点标记、拍照、笔记区 | 洞察条目/自适应纪要可溯源原文 |

---

## 1. Granola

**1. 会中能看到什么。** 会中是一个空白笔记编辑器（Note Editor），用户自己打字，Granola 在后台实时转写；没有 AI 生成的纪要（https://docs.granola.ai/help-center/taking-notes/taking-notes-in-granola）。点聊天栏左边的波形按钮可以开关实时转写面板：灰色气泡是系统音频（对方），绿色是自己麦克风，能在面板里暂停、搜索、复制、删除片段（https://docs.granola.ai/help-center/taking-notes/transcription）。切到别的 App 时，屏幕右侧有一个浮动的会议指示器（同上）。

**2. 更新频率与按需摘要。** 会中没有自动更新的纪要。检测到通话结束或用户手动停止后，才自动生成「AI-enhanced notes」（https://docs.granola.ai/help-center/taking-notes/ai-enhanced-notes）。按需方面：官方把「During a meeting: Catch up on what you missed, recap action items」列为 Granola Chat 的用法（https://docs.granola.ai/help-center/getting-more-from-your-notes/chatting-with-your-meetings）。

这是有意的设计。CEO Chris Pedregal 在 Every 播客里说，早期的核心交互就是会中实时调 AI 写笔记，demo 很好看，但实际开会时很分心："if you have any kind of AI augmentation that happens real-time, you can't help but read what the AI is writing. And if it doesn't get it exactly right, then you can't help but want to fix it. And then you realize you've stopped paying attention…"（转录稿：https://every.to/podcast/transcript ，Exa 索引给出的就是这个地址；节目页：https://podwise.ai/episodes/8627781 ）。一篇二手中文转述说他们为此花了约 6 个月（https://techblog.summhub.com/ai/granola-ai-thinking-tool-revolution/ ，二手来源，"6 个月"这个数字未在一手转录稿中核实）。

**3. 结构及与会后定稿的关系。** 会后纪要的输入是：转写、会中手打的私有笔记、日历事件、用户资料。跟手打笔记对应的要点用黑色高亮（https://docs.granola.ai/help-center/taking-notes/ai-enhanced-notes）。用户可以用 Markdown 标题（如 "# Agenda"）、勾选框来引导最终结构（https://docs.granola.ai/help-center/taking-notes/taking-notes-in-granola）。默认用 auto 模板；会中或会后都能换模板，但不能设默认模板（https://docs.granola.ai/help-center/feature-requests）。

**4. 会中能做什么来影响纪要。** 手打笔记（"any notes you take are seen as important details to include"）、拖入图片、Markdown 结构（https://docs.granola.ai/help-center/taking-notes/taking-notes-in-granola）；会中换模板（https://docs.granola.ai/help-center/feature-requests）；Chat 提问。Snapshots 会自动截取会中共享屏幕里的幻灯片、图表并附进笔记（https://www.granola.ai/blog/some-ideas-take-a-few-fridays ，2026-10-06）。

**5. 能否点回原话/原音。** 每行旁边的放大镜能跳到转写或私有笔记里对应的位置（https://docs.granola.ai/help-center/taking-notes/ai-enhanced-notes）。Granola 不录、不存音频，"no audio files to play back later"，所以回不到原音（https://docs.granola.ai/help-center/taking-notes/transcription）。

**6. 用户抱怨。** "会中什么都不发生"出现在一篇竞品作者写的评测里（https://www.itsconvo.com/blog/granola-ai-review ，作者是竞品 Convo 的 CEO，只当线索）。另一位竞品联合创始人汇总的 Reddit 抱怨有：名字识别错、看着在录其实什么都没产出（https://anarlog.so/blog/granola-ai-complaints/ ，作者自述有利益关系）。针对实时纪要本身的一手用户抱怨：未找到。

---

## 2. Notion AI Meeting Notes

**1. 会中能看到什么。** 用 /meet 或 Meeting notes 标签页建块，点 Start transcribing 开始。说话人变化时转写会换行，桌面端可以标注说话人（https://www.notion.com/help/ai-meeting-notes）。转写进行中，笔记上会出现「会中技能」（skills）（同上）。

**2. 更新频率与按需摘要。** 摘要不实时生成。点 Stop 后才自动开始生成，并且至少需要 1 分钟录音（同上）。按需方面，内置的会中技能有 Catch me up、Suggest questions、Recap decisions；会后技能有 Make summary more detailed、Create follow-up tasks、Complete action items for me。会中和会后各自最多显示 12 个技能，用户自建的排在前面（同上）。

**3. 结构及与会后定稿的关系。** 会前可以在 Notes 区写议程，生成摘要时会参考。Instructions 可以按会议类型（sales call、standup 等）选结构，也可以自定义；改 Instructions 只影响新录音或恢复转写的部分，旧摘要要点 Retry summary 才会重生成（同上）。

**4. 会中能做什么来影响纪要。** 会前和会中写 Notes、选 Instructions、运行技能（同上）。

**5. 能否点回原话/原音。** 摘要里有引用，悬停能看到转写片段，点击跳到对应转写行（同上）。能否跳到原音：未核实。

**6. 用户抱怨。** 未找到针对会中体验的一手抱怨。

---

## 3. Fireflies（Live Assist / Real-Time Notes）

**1. 会中能看到什么。** Live Assist 面板（Web、移动、Chrome 扩展、桌面都有）包括：顶部会议标题和计时；四个默认按钮；动态话题卡片；左下角笔记图标打开 AI Notes / Transcript / Prep 三个标签；右侧 AskFred 对话（https://guide.fireflies.ai/articles/6032274417-learn-about-fireflies-live-assist-get-real-time-suggestions-answers-and-notes-live-during-the-meeting）。用机器人入会时，入会约 30 秒后会在会议聊天里发一条置顶的「View Realtime notes」链接（同上）。Chrome 扩展版不需要机器人，但只支持 Google Meet，会议开始约 10 秒后弹出面板（https://guide.fireflies.ai/articles/7077919424-explore-the-real-time-pane-on-the-fireflies-chrome-extension）。

**2. 更新频率、追加还是改写、按需摘要。** AI Notes "automatically creates structured notes with clear headings, timestamps, and bullet points as people talk"（Web 版文档）；移动端写的是 "Live AI Notes update continuously while the meeting is running"（https://guide.fireflies.ai/articles/5965172749-live-assist-on-fireflies-mobile-app-get-real-time-notes-live-during-meetings）。具体刷新间隔、是追加还是改写已显示内容：未核实。话题卡片 "refresh automatically to match the flow of your ongoing meeting"（Chrome 扩展文档）。另有一条实现层面的细节："Notes and transcripts update only when the tab is active"，标签页不在前台就不更新（Web 版文档）。

按需按钮有四个：Follow Up；**Catch Up (Last 1 min)**，总结最近 60 秒；**Summarize So Far**；Action Items。结果显示在右侧 AskFred 面板。动态话题卡片只给付费用户（Web 版文档）。

**3. 结构及与会后定稿的关系。** 实时 AI Notes 的结构是标题 + 时间戳 + 要点（Web 版文档）；会议结束后另做处理，生成 Notebook 里的摘要并发邮件（Chrome 扩展文档）。会后摘要可以带时间戳下载，"making it easy to jump back to the relevant part of the meeting"（https://guide.fireflies.ai/articles/3319752033-how-to-download-transcripts-summaries-and-meeting-recordings-from-fireflies）。实时笔记和会后摘要是不是同一份、会不会被覆盖：未核实。

**4. 会中能做什么来影响纪要。** 向 AskFred 提问（能记住上下文、能搜网页、可用 AI Skills 模板，按问题复杂度扣 AI credits）；分享 Live Assist 链接，主持人控制谁能看（Web 版文档）。手动编辑实时笔记：文档未提，未核实。

**5. 能否点回原话/原音。** 实时笔记带时间戳（Web 版文档）。会中点时间戳能不能跳转：未核实。

**6. 用户抱怨。** 一份第三方统计了 48 个 Reddit 线程里 85 人的观点，负面主要集中在机器人可见、日历扩散、积分扣费；"摘要有错、需要清理"有 8 人（https://metawhisp.com/blog/fireflies-ai-reddit-reviews/ ，竞品作者，只当线索）。针对 Live Assist 实时笔记的一手抱怨：未找到。

---

## 4. Otter（会中摘要、Otter AI Chat、Live Assist）

**1. 会中能看到什么。** Notetaker 入会后提供实时转写，以及 Otter Chat Q&A、Automated Live Summary、自动截取幻灯片（https://help.otter.ai/hc/en-us/articles/4425393298327-Otter-Notetaker-Overview ，2026-08-21 更新）。"Otter's proprietary AI will automatically generate a live summary during a recording and display it in the Outline section"（https://help.otter.ai/hc/en-us/articles/5093228433687-Conversation-Page-Overview）。2023 年的博客说实时摘要可以在 Web/移动端查看并分享给其他参会人，还能会中高亮、给同事分配待办（https://otter.ai/blog/live-transcribing-zoom-microsoft-teams-and-google-meet-using-otterpilot）。2026-10-27 发布的 Live Assist 在 Zoom/Teams/Meet 会议窗口里弹出小提示，私下给教练建议，"checks off your agenda as topics are covered"；企业客户先用，之后几周内推给全部付费用户（https://otter.ai/blog/otter-rethinks-the-meeting-new-ai-preps-you-coaches-you-and-follows-through ；https://otter.ai/live-assist ）。

**2. 更新频率与按需摘要。** 实时摘要的刷新间隔、追加还是改写：未核实（帮助中心没写）。按需：可以在 "anytime during the live meeting or post-meeting" 问 AI Chat（https://help.otter.ai/hc/en-us/articles/15114041061783-Asking-Otter-AI-Chat-questions ，2026-03 更新）。

**3. 结构及与会后定稿的关系。** 会后 Summary 页分 Overview/Gist、Action Items、Outline（按话题 + 支撑细节）。30 分钟会议结束后，定稿转写约 1.5–3 分钟、摘要约 2.5–6 分钟出来；会中转写是 draft（https://help.otter.ai/hc/en-us/articles/360048322493-Transcription-processing-time-FAQ ，2026-05-27）。老版帮助说摘要 "based on the content during the live meeting" 不能重生成（Conversation Page Overview）；后来上线了 Regenerate summary，会同时重写 Gist、Action Items、Outline，旧版本不保存、不能回退；Outline 和 Action Items 能手改，Gist 不能（https://help.otter.ai/hc/en-us/articles/25846455610263-Regenerate-the-summary）。

**4. 会中能做什么来影响纪要。** 高亮、评论、AI Chat；Live Assist 可以用 Agent Builder 导入团队 playbook（https://otter.ai/live-assist）。会议类型模板可以开关 Outline 等区块（https://help.otter.ai/hc/en-us/articles/31402572907415-Custom-Meeting-Type-Templates）。

**5. 能否点回原话/原音。** "Click on any part of the Outline to go directly to the associated source in the transcript"（同上）。

**6. 用户抱怨。** 2022 年 HN 高赞帖：Otter 机器人在用户关闭设置的情况下入会，并给所有人发了 "absurd, inaccurate 'outline'"（https://news.ycombinator.com/item?id=32751071）。这是对自动外发纪要的不满，不完全是会中体验。针对实时 Outline 跳动或滞后的一手抱怨：未找到。

---

## 5. Fathom

**1. 会中能看到什么。** 新的无机器人桌面版（目前 Mac、英文会议）提供 live summary 和 Scratchpad（https://help.fathom.video/en/articles/7573633 ，2026-04-15 更新；https://www.fathom.ai/whats-new ）。旧的机器人版没有实时转写，会后才出摘要（同一帮助文）。旧版在会中有 Highlight 按钮，新版暂时不支持，计划用会后 clips 替代（https://help.fathom.video/en/articles/295424 ，2026-06-04）。

**2. 更新频率、追加还是改写。** "The live summary builds in the order the conversation happens, so you can glance over, see what you missed, and jump back in"（https://www.fathom.ai/features/in-meeting-experience ，仅营销文案）。从字面看是按时间顺序累积追加，但没有文档说明是否会改写已经显示的内容，刷新间隔也未核实。

**3. 结构及与会后定稿的关系。** Scratchpad 是私有的，"shape your personal post-call summary"，不进共享摘要；在 Scratchpad 里 @同事可以把那条笔记直接推给对方（同上营销页）。2026-04-15 的新闻稿说个人笔记 "incorporated directly into summaries"（https://www.businesswire.com/news/home/20260415965820/en/ ）。会后摘要按主题分组（https://www.fathom.ai/features/ai-transcription ）。

**4. 会中能做什么来影响纪要。** Scratchpad 和 @提及（新版）；旧版可以在会中标 Highlight。Highlight 的行为：从对方开始说话的地方往前补，等你自己再开口时自动结束（https://help.fathom.video/en/articles/295424）。

**5. 能否点回原话/原音。** 会后 "Jump from any summary point to the exact moment it happened"（https://www.fathom.ai/features/ai-transcription ，营销页）。会中 live summary 能不能跳转：未核实。

**6. 用户抱怨。** 官方承认实时转写是长期热门需求（https://help.fathom.video/en/articles/7573633）。新版 live summary 的用户反馈：未找到。

---

## 6. tl;dv

**1–4.** 录制时可以点钉选图标（📌）标记时刻，生成带时间戳的笔记；Pro 计划会自动对这些时刻做 AI 摘要（https://intercom.help/tldv/en/articles/14434461-recording-meetings-with-google-meet ，2026-04-24）。「magic pin」会保存时间戳、总结这个时刻、打 AI 标签；Smart Topics 会 "automatically timestamps and summarizes key meeting moments during your meetings"，按话题分组，每个话题有 gist 和带时间戳的笔记（https://intercom.help/tldv/en/articles/7198123-ai-meeting-notes ）。但同一篇文档说这些摘要显示在录像左侧，会中能不能实时看到，文档没写清楚：未核实。官网写的是 "a ready-to-use summary the moment the call ends"（https://tldv.io/ ）。

**5.** 时间戳可以跳到录像（https://tldv.io/timestamp-zoom-meetings/ ）。

**6.** 未找到针对会中的一手抱怨。

---

## 7. Jamie

**1–2.** 桌面录制时显示一个小悬浮窗，点开是 Sidekick：会议标题/参会人/标签、Notes 和实时转写切换、Ask 框（例："Recap the last 5 minutes"、"What did they just say about the timeline?"），会前 briefing 也在这里显示（https://docs.meetjamie.ai/agent/sidekick ）。没有自动更新的实时纪要；停止后处理，依次经过 Transcribing → Speaker Matching → Writing Summary，并显示进度（https://docs.meetjamie.ai/getting-started/summary ）。

**3–4.** Scratchpad 是私有笔记，"will not be incorporated into the summary"，这一点和 Granola/Circleback 相反；可以会中加标签（https://docs.meetjamie.ai/getting-started/recorder ）。会后可以换模板重新生成（https://docs.meetjamie.ai/getting-started/templates ）。

**5–6.** 能否回到原话：未核实。抱怨：未找到。

---

## 8. Krisp

会中点通话指示条打开 Document 页，可以 "add real-time comments during the meeting"；点 CC 图标打开实时转写窗口（https://help.krisp.ai/hc/en-us/articles/30146816045980-Record-your-meetings-with-Krisp-AI-Note-Taker ）。帮助文档说 Meeting Notes（要点、待办）和 AI Chat 都是针对已录会议的（https://help.krisp.ai/hc/en-us/articles/8214720684956-AI-Meeting-Assistant-overview ；https://help.krisp.ai/hc/en-us/articles/18440374735644-Get-meeting-insights-with-Krisp-AI-Chat ）。官网 demo 里笔记边讲边生成（https://krisp.ai/ai-note-taker/ ），仅营销文案，帮助文档没有提会中 AI 笔记，标为未核实。回链和抱怨：未核实或未找到。

---

## 9. Circleback

会前、会中、会后都能加私有笔记；分享会议时笔记仍是私有的，但会用来改进 AI 纪要（例如标明某个只聊了 30 秒的话题其实最重要）（https://circleback.ai/docs/support/meetings/add-your-own-notes ）。更新日志里的 "Real-time meeting assistant responses"（2025-01-28）指的是问答流式输出，不是实时纪要（https://circleback.ai/releases/real-time-meeting-assistant-responses ）。会后核对流程：点转写里的词，播放器跳到那个时刻；可以生成 Share moment 链接；行动项带负责人（https://circleback.ai/blog/how-circleback-writes-meeting-notes ，2026-10-20）。会中有没有实时转写或 AI 内容：未核实。

---

## 10. Zoom AI Companion

**1–2.** **Meeting questions**：要主持人先开启。预设问题有 "Catch me up"、"Was my name mentioned?"、"What are the action items?"，也可以自定义问题。主持人决定谁能问、能问的范围（从 AI 开启时起，还是从提问人入会时起）。目前只有桌面端能提问；分组讨论室不支持（https://support.zoom.com/hc/en/article?id=zm_kb&sysparm_article=KB0057748 ）。回答下方可以 Send via Chat / Create Task / Create Doc（同上）。

**My Notes**：转写和用户自己的笔记并排显示，Ask AI 基于转写回答；会后生成的摘要以用户手记的话题为重点，转写只补充（https://library.zoom.com/ai/ai-whitepaper/my-notes ）。My Notes 可以用在 Teams、Meet 和线下会议（https://news.zoom.com/zooms-new-agentic-ai-capabilities-supercharge-focus-and-productivity/ ，2026-02-11）。

会中没有自动滚动更新的纪要；Meeting Summary 在会后生成（同上帮助）。

**3.** 会后摘要的结构：Quick recap、Next steps（每条以人名开头）、按话题分小标题的 Summary（https://hirekai.ai/blog/zoom-ai-meeting-notes ，第三方，引用 Zoom 白皮书，未直接核实）。

**5.** 会中回答能否跳回原话：未核实。

**6.** Zoom 社区帖："WILDLY inaccurate. Like, not even close"（2024-01，针对会后摘要，https://community.zoom.com/t5/Zoom-AI-Companion/Why-is-the-AI-Meeting-Summary-So-Inaccurate/m-p/162791 ）；"Zoom AI is Useless!"（2024-12，https://community.zoom.com/meetings-2/zoom-ai-is-useless-75154 ）。竞品 tl;dv 的评测说 in-meeting questions "often repeats summary points instead of real context"（https://tldv.io/blog/zoom-ai-companion-review/ ，有利益关系）。

---

## 11. Microsoft Teams：Copilot（个人）与 Facilitator（共享）

**1. 会中能看到什么。** Copilot 是私人对话面板，可以弹到单独窗口；别人看不到你的问答（https://support.microsoft.com/en-us/teams/copilot/catch-up-on-meetings-with-microsoft-365-copilot-in-teams ）。Facilitator 是共享代理：Notes 标签里是实时 AI 笔记，所有内部参会人都能看、能编辑；它还会贴议程、显示议程计时器、在会议过半和快结束时提醒总结决策和未决问题（https://support.microsoft.com/en-us/teams/copilot/facilitator-in-microsoft-teams-meetings ；https://techcommunity.microsoft.com/blog/microsoft365insiderblog/keep-meetings-focused-and-productive-with-facilitator-in-teams/4456029 ）。

**2. 更新频率与按需摘要。** "AI-generated notes won't appear immediately but will start a few minutes later based on the discussion captured in the transcript"（Facilitator 帮助）。一篇 MVP 博客说笔记 "created every few minutes, organised by topics and follow-up tasks"（https://robquickenden.blog/2024/11/facilitator-agent-live-ai-notes-in-teams-meetings-chat/ ，非官方；具体间隔未核实）。按需：入会晚 5 分钟以上、Copilot 已开启时，会主动推一条 catch up 通知；可以选预设提示，比如 "Where do we disagree on this topic?"（Copilot 帮助）。

**3. 结构及与会后定稿的关系。** Facilitator 的实时笔记存在 Loop 页里，会前、会中、会后都能编辑；会后 Recap 有两份：Notes 标签是会中实时笔记，AI summary 标签是会后基于全文转写重新生成的摘要（Facilitator 帮助）。也就是说，实时笔记和会后定稿是两份独立产物。Loop 文件生成已在 2026-07-31 停用，Facilitator 实时笔记不受影响（同上；https://learn.microsoft.com/en-sg/microsoftteams/facilitator-teams ）。关闭 Facilitator 只停止继续记，已记的保留（Facilitator 帮助）。

**4. 会中能做什么来影响纪要。** 直接改 AI 笔记。MVP 博客说笔记标注了哪些是 AI 生成、哪些是人改的（非官方）；官方 FAQ 说 "The notes generated are attributed to a participant, not Facilitator"（https://support.microsoft.com/en-us/teams/platform/frequently-asked-questions-about-agents-in-microsoft-teams ）。也可以 @Facilitator 加议程（Insider 博客）。Facilitator 的笔记只以转写为依据，不看会议聊天（同上博客，Known limitations）。

**5. 能否点回原话。** 会后在会议聊天里问 Copilot，"response will also cite which sources were used"（Copilot 帮助）。Facilitator 笔记能否点回转写：未核实。

**6. 用户抱怨。** Microsoft 社区帖：会议还剩 10 分钟时用 "Generate meeting notes" 得到的结果结构清楚、带转写链接；10 分钟后同样的提示只给出很短、没有小标题的摘要（https://techcommunity.microsoft.com/discussions/microsoft365copilot/why-are-copilots-responses-in-teams-better-in-the-middle-of-the-meeting-than-at-/4183968 ，2024-07）。一篇 Medium 文章记录 Copilot 把根本没参会的人写进了纪要（https://medium.com/@InfraSpectre/copilot-said-they-were-in-the-meeting-they-werent-1d56ec4a5e19 ，个人博客）。

---

## 12. Google Meet：「Take notes for me」与 Ask Gemini

**1. 会中能看到什么。** 开启后，所有人屏幕上出现蓝色铅笔图标，点开能看 "the meeting notes taken so far"（https://workspaceupdates.googleblog.com/2024/08/take-notes-for-me-google-meet-gemini.html ）。晚入会的人可以看 "Summary so far"（https://support.google.com/meet/answer/14754931?hl=en ）。Ask Gemini 是左下角的私人问答框，参会人达到两人就自动开启（https://support.google.com/meet/answer/16024610?hl=en ；入口位置在 2026-05 调整：https://workspaceupdates.googleblog.com/2026/05/ask-gemini-in-google-meet-is-becoming-more-easily-accessible-on-web.html ）。

**2. 更新频率与按需摘要。** "notes taken so far" 的刷新频率：未核实。Ask Gemini 能做：总结进行中的讨论或某人说了什么、列出要点/决策/待办；要追问入会之前的内容，前提是 take notes for me 已经开着（Ask Gemini 帮助）。Ask Gemini 只用实时字幕，会后不保存字幕；刷新页面或掉线会清空对话（同上）。

**3. 结构及与会后定稿的关系。** 会后文档在会议结束后不久生成，有两个标签：Quick notes（最重要的要点 + 待办）和 Full notes（Summary / Decisions / Next steps / Details）。Decisions 会标状态：Aligned、Needs Further Discussions、Disagreed、Shelved。长度可以选 Standard 或 Longer；Beta 用户能增减章节；在屏幕上停留至少 10 秒的演示内容会被截图放进笔记；推荐会议长度 15 分钟到 8 小时（https://support.google.com/meet/answer/14754931?hl=en ）。

**4. 会中能做什么来影响纪要。** 开始前或开始后调整接收人、长度、章节、语言；内部参会人可以随时停止和重启，把敏感讨论排除在外；检测到另一种语言连续说了 30 秒以上，会提示切换语言（同上）。

**5. 能否点回原话。** 同时开了转写时，笔记里带时间戳引用，比如 "(00:08:32)"，点了打开文档里的转写标签并跳到那里（https://workspaceupdates.googleblog.com/2025/02/transcript-citations-take-notes-for-me-google-meet.html ）。

**6. 用户抱怨。** Google 社区帖标题："Gemini didn't take notes during a Google Meet — I activated it but got only a vague summary"（https://support.google.com/meet/thread/ ，搜索结果里地址被截断，帖子正文未读，只当线索）。官方故障说明列出的原因有：网络问题、会议少于 15 分钟、违反使用政策（https://support.google.com/meet/answer/14754931?hl=en ）。

---

## 13. 飞书会议 / 飞书妙记

**1. 会中能看到什么。** 底部工具栏打开「AI 总结」开关后，右侧面板显示实时字幕和 AI 总结；关掉面板总结也不会停（https://www.larksuite.com/hc/zh-CN/articles/732542328457 ；镜像：https://www.fklzl.cnpc.com.cn/hc/zh-CN/articles/244959839578 ）。**AI 视图**（V7.50+，默认开启）把讨论实时整理成结构化图表，显示在视频宫格中。点图表节点可以看信息来源或点赞点踩反馈。只有主持人或联席主持人能关，关了所有人都看不到。会后内容会存进智能纪要文档的画板，可以导出。1v1 通话、网络研讨会、分组讨论、视频面试不支持（https://www.feishu.cn/hc/zh-CN/articles/092000440742 ，2026-08-04 更新）。

**2. 更新频率与按需摘要。** 刷新间隔、追加还是改写：未核实。按需：会中点「...」让智能伙伴总结要点、生成待办、总结发言人，也可以自由提问（https://www.feishu.cn/content/feishu-zhineng-huiyi-zhushou-tigao-jilu ，官网内容营销文，未标日期；文中说前提是开了录制，这条可能已过时，未核实）。

**3. 结构及与会后定稿的关系。** 关闭 AI 总结后，不仅不再总结，会后也不生成智能纪要（larksuite 帮助）。会后智能纪要文档包括会议信息、总结、待办、逐字稿，默认推送给所有参会人，所有人可读可编辑；开了录制的话，妙记里还有章节纪要（同上）。官方产品文还写了：待办识别责任人并能回溯原文、按会议类型「千会千面」、结论反转时自动判断最终结论、识别并过滤会前闲聊（https://www.feishu.cn/content/article/7577317993749269689 ，官网内容文）。

**4. 会中能做什么来影响纪要。** 提问；在 AI 视图节点上反馈（帮助文档）。手动编辑会中总结：未核实。

**5. 能否点回原话/原音。** 会中：AI 视图节点能看「信息来源」（帮助文档）。会后：「点击文字直接跳转到对应的视频或音频位置」（https://www.feishu.cn/content/article/7594772423082396858 ，官网内容文，2026-01-13）。

**6. 用户抱怨。** 一篇公众号实测：结构化的周例会，AI 摘要准确率大约 70%；"多人打断、话题跳来跳去，AI 基本就开始「蒙」了"；口头禅被如实写进摘要（http://mp.weixin.qq.com/s?__biz=MzY5NjIzODgyOA==&mid=2247483785&idx=1&sn=64b9753eab58234df4de4be421fec439 ，2026-05，个人作者，针对会后摘要）。

---

## 14. 腾讯会议（AI 纪要 / AI 小助手 / 元宝）

**1. 会中能看到什么。** 点工具栏的「AI 纪要」后，"上方气泡框内会智能提炼会议概览、结论和待办，跟随会议进程实时刷新总结"，"下方对话框内，每 2 分钟主动推送当前最新的会议纪要"。全会的客观总结和个人观点、短时话题上下分栏显示（https://meeting.tencent.com/news/aijy20250820.html ，官方，2025-08-25）。AI 小助手在会中可以弹独立窗口或收进侧边栏（https://meeting.tencent.com/support/topic/2075/index.html ）。

**2. 更新频率、追加还是改写、按需摘要。** 这是本次调研里唯一公开了间隔的产品：**每 2 分钟**。从官方描述看是两层：顶部概览会改写（"实时刷新"），下方是按时间追加的推送流。用户实测文进一步说明，下方推送是最近 2 分钟的内容，旁边还有「从会议开始到当前的所有内容概要」，并且有「立即总结」按钮可以手动触发（http://mp.weixin.qq.com/s?__biz=MzkyNjI3NjQ2MA==&mid=2247490370&idx=1&sn=a3a42e1f45a461b445a78911900f6c76 ；http://mp.weixin.qq.com/s?__biz=MzkwNDcyMjU2Mw==&mid=2247486077&idx=1&sn=7c359710329fb8b6007a1564f8e911e0 ，均为个人公众号，「立即总结」按钮官方文档未见）。AI 小助手：开了实时转写后，切到「提问会议内容」可以就参会期间的内容提问；没开转写会提示去找主持人（帮助中心）。官方宣传的示例问题有「刚刚说了什么」「前面 10 分钟说了什么」「我入会之前某某说了什么」（https://meeting.tencent.com/news/huiyuanshengji240129.html ；https://meeting.tencent.com/news/aixzsp250221.html ）。应用商店描述写的是「点击"刚刚说了什么"…回顾会上最近 1、3、5 分钟」（转载于 https://m.163.com/dy/article/IRI2QOO40511BTOQ.html 等，原始商店页未直接核实）。

**3. 结构及与会后定稿的关系。** 会中纪要包括概览、结论、待办（"谁、在什么时间点前、需要完成什么任务"），还有「会议氛围」解读（官方 2025-08 文）。会后可以一键导入元宝继续追问（同上）。录制场景下，「智能录制」生成总结、纪要（可选混元或 DeepSeek 模型）、待办、章节、发言人、话题，纪要和待办能编辑、能恢复初始版本（https://meeting.tencent.com/support/topic/2077/index.html ）。

**4. 会中能做什么来影响纪要。** 智能提醒：有人提到我、提到关键词、指定成员发言时私下提醒（帮助中心）；按会议性质选 AI 角色（https://meeting.tencent.com/ai-tab/ai-assistant/index.html ）；AI 托管：自己离开时让元宝代听（同页；https://www.geekpark.net/news/353910 ）。

**5. 能否点回原话/原音。** 2026-03 升级后，会后「纪要」里每条总结都附时间戳，点了跳到录制片段；用户可以用「+」把个人笔记锚定在某个讨论时刻（https://meeting.tencent.com/news/lzgb20260315.html ）。会中纪要条目能不能跳回：未核实（营销页写有「时间戳一键溯源」，没说明是会中还是会后）。

**6. 用户抱怨。** 没找到针对 2 分钟推送的负面一手反馈；找到的公众号多为正面体验文。

---

## 15. 通义听悟

**1–4. 消费端。** 官方帮助「高效会议」把会中能做的列为：实时转写、修改识别结果、编辑发言人、实时翻译、一键高亮「重点、问题和待办」、把识别结果摘到笔记。章节速览、发言总结、待办都放在「会后高效掌握要点」里（https://tingwu.aliyun.com/helpcenter/06ae13 ，页面未标日期）。消费端会中有没有 AI 实时总结：未核实。

**API 端。** 实时记录流程是先推流转写，"结束实时记录后进入智能纪要提取等 AI 能力阶段"（https://help.aliyun.com/zh/tingwu/interface-and-implementation ）。也就是说，大模型能力（章节速览、摘要、要点提炼）在实时场景下是会后算的。章节速览可以调粒度：Coarse 每小时约 4 章，General 约 6 章，Meticulous 约 12–15 章；标题长度也可调；输出带毫秒级 Start/End（https://help.aliyun.com/zh/tingwu/chapter-quick-view ）。全文摘要至少要 250 字（约 70 秒）；待办最多 6 条、每条 5–30 字（https://help.aliyun.com/zh/tingwu/features ）。

**5.** 音字对应回听；点章节速览跳到对应片段（「高效会议」帮助）。

**6.** 2023 年的媒体实测：关键词抓取过泛，ASR 错误会传到摘要里（如「大源模型」）（http://news.sohu.com/a/681576832_115565 ，媒体测评，年代较早）。

---

## 16. 讯飞听见

**1–2. AI 洞察。** 实时录音时默认开启，"在实时录音转写的基础上，同步对会议内容做语义分析"，按时间段抓取核心议题，输出四类信息：待办思路（含责任人、时间节点）、风险提示、矛盾冲突、提示建议。"以上洞察维度会根据实际会议内容动态生成"。会中还可以用 Chat 基于实时转写提问。导入的音视频不支持洞察（https://www.iflyrec.com/zixun/6a75a1cc.html ；https://www.iflyrec.com/zixun/6a75a1cd.html ，官方，2026-08-07）。刷新间隔：未核实。2023 年的会记 Web 教程写的是 AI 纪要、全文概览 "仅支持会议结束后使用"（https://huiji.iflyrec.com/huiji_help/655b2161.html ，已过时，仅说明演进）。

**3. 结构。** 会后 AI 纪要包括全文概要、主要内容、各发言人总结、待办（会记教程）。2025-11 上线的「自适应纪要」会把右侧笔记区的个人笔记、图片和转写合在一起生成纪要，生成后给出 5 个「拓展思考提示」（https://www.pingwest.com/a/308848 ；https://www.iflyrec.com/zhuanxie/690d8c4e.html ）。

**4. 会中输入。** 录音过程中可以随时标记重点、拍照、加批注（https://huiji.iflyrec.com/huiji_help/654ddcc0.html ；应用商店描述）。

**5. 回链。** 点洞察条目定位到原文（AI 洞察官方文）；自适应纪要能「溯源转写原文、用户笔记区的记录内容，或引用的图片」（pingwest 转载的官方通稿）。

**6. 抱怨。** 官方 FAQ 自己写了 "AI洞察结果…仅供用户参考，重要信息建议结合原文进行核实"。第三方用户的一手抱怨：未找到。

---

## 17. 公开的技术做法

### 17.1 触发条件和窗口

| 来源 | 触发 | 输入窗口 | 输出形态 |
|---|---|---|---|
| 腾讯会议 AI 纪要（官方文） | 固定每 2 分钟，另有手动「立即总结」（用户文） | 未公开 | 顶部概览改写 + 下方按时间追加 |
| VietMed 实时摘要论文 arXiv:2406.15888（Interspeech 2024） | 每 N=4–5 句（最多约 30 秒）出一次局部摘要；会后出全局摘要 | 只用这 N 句 | 局部摘要追加；会后全局摘要充当 "overrider" |
| Micro（Mastra 博客，基于 Recall.ai 实时转写） | 前几句后首次唤醒，之后每约 5 句新话或约 30 秒新内容一次；不在每个 partial 上调用 | 最近约 60 句，服务端再截到约 4k 字符 | 先流式显示原文本，结束后解析成结构化卡片 |
| MeetingBro 开源实现（源码） | rolling 每 60 秒；memory 每 120 秒；cumulative board 每 180 秒；各自有最少句数门槛 | rolling 看最近 180 秒；board 看 memory + 最近 120 秒原文 | 三层：增量播报、压缩记忆、稳定看板 |
| 议程感知实时摘要（Springer, 2025） | 每 Δ 句，或检测到议程切换 | 上一版摘要 + 新增话语 + 议程 | 已结束议程的章节冻结，只续写当前章节 |

来源：
- arXiv 2406.15888：https://arxiv.org/abs/2406.15888 。论文批评了「每句话都更新并修订摘要」的旧设计（含 utterance lookahead、information overrider 等组件）："The continuously updated summary creates confusion as users are unable to keep track, given the uncertainty of when a summary is completed"，而且计算成本更高。
- Mastra / Micro：https://mastra.ai/blog/building-low-latency-agents （2026-06-02）。要点："don't keep appending the whole transcript to the prompt. Use a rolling window plus durable state"；长会议维护一个 "compact running summary / open-issues / commitments object"；有更新的转写进来时，"aborting stale in-flight intelligence requests"；模型输出用小 maxOutputTokens、流式返回。
- MeetingBro：https://github.com/armpro24-blip/MeetingBro 。参数在 app/backend/meetingbro/session/manager.py（rolling_window_seconds=180、rolling_interval_seconds=60、memory_interval_seconds=120、cumulative_interval_seconds=180、summary_tail_seconds=120）；提示词在 app/backend/meetingbro/summarization/llm.py。这是一个 46 star 的小项目，只作为可读的参考实现。
- Springer：https://link.springer.com/article/10.1007/s44443-025-00304-y 。"S(t) should be updated progressively … rather than being computed from scratch each time"；已完成的议程章节 "We do not want to regenerate"。

### 17.2 提示词策略（来自源码，原文摘录）

MeetingBro 把「增量」和「累积」拆成两个提示词，避免一个提示词既要只写新内容、又要保持全局稳定：

- rolling_summary（增量播报）："You will receive two inputs: (1) the previous rolling summary, which is already done and must NOT be repeated or paraphrased, and (2) new transcript since that summary. Your job is ONLY to report what is genuinely new … new facts, updated numbers, new decisions, newly assigned actions, reversals … Write 2-4 sentences … If the new transcript contains nothing that wasn't already covered … say so in one sentence."
- meeting_memory（压缩记忆，不直接展示）："Update the previous memory using only the new transcript. Keep it concise and bounded. Preserve durable facts; remove resolved or redundant details." 固定标题：Topics / Decisions / Action Items / Open Questions / Important Facts。
- cumulative_meeting_summary（看板）："Use the compressed meeting memory as the durable context and the recent transcript as fresh evidence … Keep it compact, stable, and cumulative … If a section has no evidence yet, write '- None yet'."
- final（会后定稿）：输入同样是 memory 加最近一段原文，输出 JSON（chapters 带 time_start/time_end、action_items 带 assignee/due、final_summary），并要求用 vocabulary 校正人名和术语。

通用的「滚动摘要合并」写法（来自 Agent 上下文压缩代码，不是会议产品，但模式可以直接用）：把上一版摘要放进 <previous_summary>，要求 "Merge the new turns below INTO this summary's sections, in place: A Pending item that's now been answered moves to Resolved … Output the FULL updated summary using the same … format. Do not omit unchanged sections — repeat them verbatim"（https://github.com/cubeplexai/cubepi/blob/21f586b5/cubepi/middleware/compaction/summarizer.py ）。这种写法每次输出全文，代价是界面上内容会整体重排，适合「看板」层，不适合「播报」层。

### 17.3 结构化和防幻觉

- 每个条目都引用转写的 turn ID，对不上原文的条目直接丢弃："require each item to reference turn IDs. Reject items whose quoted span does not exist in the transcript"（https://acta.ai/blogs/live-transcription-to-searchable-meeting-intelligence ，厂商技术博客）。
- 通义听悟的自定义 Prompt 从 2025-05 起支持句子 ID，"方便原文内容定位"（https://help.aliyun.com/zh/tingwu/product-overview/ ）。
- 飞书宣称做了「结论反转」处理和闲聊过滤（见第 13 节），说明会中纪要必须能撤回或改写之前的结论；如果只追加，就会出现前后矛盾。

### 17.4 交互层面的公开教训

- Granola：实时生成的 AI 文本会把注意力从人身上拉走（第 1 节，Every 播客）。
- Heyalo（Show HN）："almost all the work went into making it talk less"（https://news.ycombinator.com/item?id=48816317 ）。
- Fireflies：只有标签页在前台时才更新（第 3 节），推测是为了省成本，这一点未核实。
- Teams Facilitator：开启后过几分钟才出第一版笔记（第 11 节），说明首次生成需要攒够上下文。

---

## 18. 对我们的启示

### 可以直接借鉴

1. **分成两层：「稳定看板」和「增量播报」。** 腾讯会议是顶部概览改写 + 下方每 2 分钟追加；MeetingBro 是 rolling + board；Teams 和 Google 会后都另出一份定稿。我们的界面可以拆成：上半部分是「当前状态」卡片（议题、决策、待办、未决问题，可以改写），下半部分是按时间追加的「阶段小结」流（只追加，不改写已显示的条目）。用户扫一眼看状态，想补课时往下翻。
2. **触发策略：按时间、按内容量取较大的那个，再加手动按钮。** 参考值：首次生成要等攒够上下文（Teams 是「几分钟后」；Micro 是首批几句之后）。之后每约 2 分钟或每约 5 句定稿句（取较晚者）更新一次；新内容低于阈值就跳过。加一个「现在总结」按钮（腾讯）。我们的 ASR 是按句返回的，可以直接用定稿句计数，不在 partial 结果上触发。
3. **按需摘要做成固定入口。** 最常见的三个是：「最近 1 分钟」（Fireflies）或「最近 1/3/5 分钟」（腾讯小助手）、「到目前为止」（Fireflies、Google）、「待办」。做成固定按钮，不要求用户打字。回答和自动纪要分开显示（Fireflies 放在右侧 AskFred 面板），不要污染主纪要。
4. **手打笔记是重要输入，但要说清楚它会去哪。** Granola、Circleback、讯飞自适应纪要、Zoom My Notes 会把手记当作强信号并入纪要；Jamie 的 Scratchpad 明确不并入；Fathom 的 Scratchpad 只进个人纪要。我们本地单用户，建议手记默认并入会后定稿并高亮来源（像 Granola 那样黑色高亮），会中不改写用户自己写的文字。
5. **每个条目都带时间锚点，能回到原句和原音。** Google 用 (00:08:32) 风格的引用，飞书 AI 视图节点可以看来源，腾讯 2026-03 起每条纪要都有时间戳，Otter 的 Outline 点了能跳转写，讯飞洞察能定位原文。我们本地有 SQLite 存句子和音频，可以让 LLM 输出句子 ID（参考听悟的「句子 ID」和 acta 的 turn ID 校验），丢弃引用不存在的条目，点击时同时定位转写和播放音频。Granola 不存音频、回不到原音，这是它的短板。
6. **增量提示词：只写新内容，并显式支持「反转」。** 播报层沿用 MeetingBro 的约束：上一版不许复述，只报新事实、数字变化、新决策、新分配、反转，没有新内容就说没有。状态层用 merge-in-place 写法：Pending 变成 Resolved，结论反转就覆盖。上下文 = 压缩记忆 + 最近约 2–3 分钟原文，不把全文塞进去（Micro、MeetingBro 都这么做）。
7. **会后定稿重新生成，和会中纪要分开。** Teams 是 Notes 和 AI summary 两份；Otter 会后基于定稿转写出摘要；VietMed 论文用全局摘要做 "overrider"。会后用完整转写（可以先切章节，参考听悟章节粒度：粗粒度每小时约 4 章，细粒度约 12–15 章）重新生成，会中的条目作为参考或保留为「会中时间线」。
8. **模板只决定结构，不在会中频繁切换。** Notion 的 Instructions、Google 的章节开关、Otter 的模板开关都是事先选好的；Notion 中途改了 Instructions 要 Retry 才生效。建议模板在开始前选定，中途改只影响会后定稿。

### 应该避开的坑

1. **刷新太频繁、整块重写，导致内容跳动。** VietMed 论文明确说「每句都更新」会让用户 "unable to keep track"；Granola 因为实时 AI 让人忍不住读、忍不住改，放弃了实时纪要。会中纪要要克制：增量区只追加；状态区改写时给变化的条目做轻量标记（比如一次性的高亮），不要整块闪烁。默认可以收起或弱化，让用户自己点开看。
2. **会中纪要和会后定稿互相覆盖、版本丢失。** Otter 的 Regenerate 不保留旧版本，也不能回退。我们应该保留会中快照，以及会后每次重生成的版本。
3. **在 partial 结果上触发 LLM，或者过时的请求结果晚到。** Micro 的做法是不在 partial 上调用；新上下文到了就中止还在跑的旧请求。我们桌面端也要处理并发和乱序：每次请求带 last_sentence_id，结果回来时如果已经过时就丢弃。
4. **早期上下文不足就出纪要，内容空泛或误判。** Teams 首版笔记要等几分钟；Google 推荐会议至少 15 分钟；听悟摘要至少 250 字。开头几分钟可以只显示转写，等内容量够了再出第一版。
5. **话题跳跃、多人插话时纪要质量明显下降。** 飞书用户实测：结构化会议约 70% 准确，乱的讨论会「蒙」。Copilot 在会议不同时刻对同一提示给出的质量差别很大。界面要允许用户纠错，比如删除某条或标记「不对」（飞书 AI 视图有节点反馈）；纠错要反馈进后续上下文，避免同样的错误在下一轮又出现。
6. **把 ASR 错误放大进纪要。** 听悟测评里的「大源模型」、Copilot 把没参会的人写进纪要，都属于这类。人名和术语用词表校正（MeetingBro 的 final 提示词用了 vocabulary）。责任人如果识别为「说话人 N」，就不要直接写成人名，Circleback 也专门提醒过会后要检查。
7. **标签页或窗口在后台时停止更新，用户切回来看到的是旧内容。** Fireflies 就是这样。如果我们为了省成本在窗口不可见时暂停生成，切回前台时要立刻补算一次，并显示「更新于 x 分钟前」。
8. **把纪要自动发给别人。** Otter 机器人把「不准确的 outline」发给全体参会人，引发了 HN 上的强烈不满。我们是本地单用户应用，默认不外发，分享必须由用户主动操作。

---

### 附：未能核实或存疑的点

- Otter 实时 Outline、Fireflies Live AI Notes、Fathom live summary、Google "notes taken so far"、飞书 AI 总结的**具体刷新间隔**，以及它们是追加还是改写：都没有官方文档说明。
- Teams Facilitator 的「每几分钟、按话题和后续任务组织、AI/人工编辑归属标注」来自 MVP 博客；官方只写了「几分钟后开始出现」，官方 FAQ 的归属说法（attributed to a participant）和博客描述不完全一致。
- 腾讯会议「立即总结」按钮来自用户公众号；「刚刚说了什么」的 1/3/5 分钟来自应用商店描述的转载。
- Granola「花了 6 个月」来自二手转述；一手转录稿的 every.to 具体地址以 Exa 索引为准。
- 通义听悟消费端会中有没有 AI 实时总结、Krisp 会中有没有 AI 笔记、Circleback 会中有没有实时转写、tl;dv 的 Smart Topics 会中是否可见、Jamie 纪要能否回到原话：都没找到一手说明。
- 针对「实时纪要」本身（跳动、太长）的一手用户抱怨很少；能找到的大多是对会后摘要准确性的抱怨，或者竞品写的评测。
