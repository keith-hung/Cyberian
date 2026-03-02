<p align="center">
  <img src="cyberian-logo.png" alt="Cyberian Logo" width="200">
</p>

<h1 align="center">Cyberian</h1>

<p align="center">
  職場生產力 CLI 工具集 — 工時表、打卡、JIRA、行事曆，一站搞定
</p>

---

## 簡介

Cyberian 是一個 monorepo，包含兩個 Go CLI 應用程式與一個 Claude Code plugin，將日常職場操作整合為可透過 AI 助手驅動的技能。

| 元件                   | 說明                                                |
|------------------------|-----------------------------------------------------|
| **timecard-cli**       | 工時表管理 — 透過 web scraping 操作工時系統          |
| **wedaka-cli**         | 打卡出勤 — 透過 REST API 操作 WeDaka 系統           |
| **Claude Code Plugin** | 將上述工具及 JIRA、Outlook 行事曆封裝為 4 個 AI 技能 |

## 專案結構

```
├── timecard-cli/         Go CLI — 工時表管理
├── wedaka-cli/           Go CLI — 打卡出勤
├── .claude-plugin/       Claude Code plugin
│   ├── plugin.json           Plugin manifest（4 skills）
│   ├── skills/               Skill 定義（每個 skill 一份 SKILL.md）
│   └── scripts/              Launcher scripts（自動下載 CLI binary）
├── .claude/              Claude Code 專案設定
│   └── settings.json.example    環境變數範本
└── dev/                  開發筆記（gitignored）
```

## 建置與執行

兩個 CLI 皆為零依賴的 Go module（Go 1.25，僅使用 stdlib）。

```bash
# 建置 timecard-cli
cd timecard-cli && go build -o timecard .

# 建置 wedaka-cli
cd wedaka-cli && go build -o wedaka .
```

## CLI 工具

### timecard-cli — 工時表管理

透過 web scraping 操作工時系統，支援 cookie-based session 自動管理。

**指令：**

| 指令         | 說明                 |
|--------------|----------------------|
| `projects`   | 列出可用專案         |
| `activities` | 列出活動項目         |
| `timesheet`  | 查看當週工時表       |
| `summary`    | 工時摘要             |
| `save`       | 儲存工時（draft only） |
| `version`    | 版本資訊             |

**環境變數：** `TIMECARD_BASE_URL`、`TIMECARD_USERNAME`、`TIMECARD_PASSWORD`

### wedaka-cli — 打卡出勤

REST API client，操作 WeDaka 出勤系統。

**指令：**

| 指令            | 說明             |
|-----------------|------------------|
| `clock-in`      | 上班打卡         |
| `clock-out`     | 下班打卡         |
| `timelog`       | 查看出勤記錄     |
| `check-workday` | 檢查是否為工作日 |
| `version`       | 版本資訊         |

**環境變數：** `WEDAKA_API_URL`、`WEDAKA_USERNAME`、`WEDAKA_EMP_NO`、`WEDAKA_DEVICE_ID`

### 共通設計

- 所有輸出皆為 JSON 格式（stdout 為成功結果，stderr 為錯誤訊息）
- 密碼透過 `--pass-stdin` pipe 輸入，不使用 command-line flag
- Exit codes：`0` 成功、`1` 一般錯誤、`2` 認證/設定、`3` 驗證、`4` 網路

## Claude Code Plugin

Plugin 註冊了 4 個 skill，可透過 Claude Code 自然語言觸發：

| Skill                | 觸發關鍵字                 |
|----------------------|----------------------------|
| **timecard**         | 工時表、timesheet、填工時    |
| **wedaka**           | 打卡、clock in/out、出勤     |
| **jira**             | JIRA issue、專案狀態、sprint |
| **outlook-calendar** | 會議、行事曆、schedule       |

### 設定

1. 複製環境變數範本：
   ```bash
   cp .claude/settings.json.example .claude/settings.local.json
   ```
2. 編輯 `.claude/settings.local.json`，填入各服務的認證資訊
3. Launcher scripts 會在首次執行時自動從 GitHub Releases 下載對應平台的 binary

## 授權

[MIT License](LICENSE)
