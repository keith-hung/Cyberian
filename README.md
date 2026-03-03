<p align="center">
  <img src="cyberian-logo.png" alt="Cyberian Logo" width="200">
</p>

<h1 align="center">Cyberian</h1>

<p align="center">
  Claude Code plugin — 工時表、打卡、JIRA、行事曆，用自然語言搞定職場日常
</p>

---

## 快速開始

### 1. 選擇安裝方式

依使用情境選擇適合的 scope，安裝 plugin 並設定環境變數：

| Scope | Plugin 設定位置 | 環境變數設定位置 | 適用情境 |
|-------|----------------|-----------------|---------|
| `user`（預設） | `~/.claude/settings.json` | `~/.claude/settings.json` 或 shell export | 個人使用，跨專案通用 |
| `project` | `.claude/settings.json`（進版控） | `.claude/settings.local.json`（不進版控）或 shell export | 團隊共享 plugin，各自設定認證 |
| `local` | `.claude/settings.local.json`（不進版控） | `.claude/settings.local.json`（不進版控）或 shell export | 僅限此專案，不進版控 |

---

#### 方式 A：User scope（推薦）

所有專案皆可使用，環境變數只需設定一次。

**安裝：**

```bash
claude plugin marketplace add keith-hung/Cyberian
claude plugin install cyberian@cyberian-marketplace
```

**設定環境變數** — 將 `env` 區塊加入 `~/.claude/settings.json`：

```jsonc
{
  // ...existing settings...
  "env": {
    "TIMECARD_BASE_URL": "https://your-timecard-server.example.com/TimeCard/",
    "TIMECARD_USERNAME": "your_username",
    "TIMECARD_PASSWORD": "your_password",
    "WEDAKA_API_URL": "https://your-wedaka-api.example.com",
    "WEDAKA_USERNAME": "your_username",
    "WEDAKA_EMP_NO": "your_employee_number",
    "WEDAKA_DEVICE_ID": "your_device_uuid",
    "JIRA_SERVER_URL": "https://your-company.atlassian.net",
    "JIRA_USER_EMAIL": "you@example.com",
    "JIRA_API_TOKEN": "your_api_token",
    "JIRA_PROJECT": "PROJ",
    "JIRA_BOARD": "My Board",
    "OUTLOOK_ICS_URL": "https://outlook.office365.com/owa/calendar/.../reachcalendar.ics"
  }
}
```

---

#### 方式 B：Project scope（團隊共享）

Plugin 設定進版控，團隊成員 clone 後自動啟用。認證資訊各自設定於 local scope。

**安裝（由一位成員執行並 commit `.claude/settings.json`）：**

```bash
claude plugin marketplace add keith-hung/Cyberian --scope project
claude plugin install cyberian@cyberian-marketplace --scope project
```

**設定環境變數** — 複製範本後填入個人認證資訊（每位成員各自操作）：

```bash
cp .claude/settings.json.example .claude/settings.local.json
```

編輯 `.claude/settings.local.json`，填入對應欄位。此檔案已被 gitignore，不會進入版控。

---

#### 方式 C：Local scope（單一專案）

Plugin 與環境變數都僅限此專案，不進版控。

**安裝：**

```bash
claude plugin marketplace add keith-hung/Cyberian
claude plugin install cyberian@cyberian-marketplace --scope local
```

**設定環境變數** — 複製範本後填入認證資訊：

```bash
cp .claude/settings.json.example .claude/settings.local.json
```

編輯 `.claude/settings.local.json`，填入對應欄位。

---

#### 環境變數替代方案：Shell export

任何 scope 都可以改用 shell profile（`~/.zshrc`、`~/.bashrc`）export 環境變數，不需修改 settings.json：

```bash
# ~/.zshrc or ~/.bashrc
export TIMECARD_BASE_URL="https://your-timecard-server.example.com/TimeCard/"
export TIMECARD_USERNAME="your_username"
export TIMECARD_PASSWORD="your_password"
# ...依需求加入其他變數
```

> **提示：**
> - 只需設定你要使用的服務，不需要全部填寫
> - 優先順序：shell export > local > project > user
> - 環境變數範本請參考 `.claude/settings.json.example`

<details>
<summary>環境變數一覽</summary>

| 服務               | 環境變數                                                                 |
|--------------------|--------------------------------------------------------------------------|
| **TimeCard** 工時表 | `TIMECARD_BASE_URL`、`TIMECARD_USERNAME`、`TIMECARD_PASSWORD`            |
| **WeDaka** 打卡     | `WEDAKA_API_URL`、`WEDAKA_USERNAME`、`WEDAKA_EMP_NO`、`WEDAKA_DEVICE_ID` |
| **JIRA**            | `JIRA_SERVER_URL`、`JIRA_USER_EMAIL`、`JIRA_API_TOKEN`、`JIRA_PROJECT`、`JIRA_BOARD` |
| **Outlook 行事曆**  | `OUTLOOK_ICS_URL`                                                        |

</details>

安裝完成後，直接用自然語言對 Claude Code 說話即可：

```
「幫我查這週的工時表」
「打卡上班」
「列出我的 JIRA issues」
「今天有什麼會議？」
```

CLI binary 會在首次觸發時自動從 GitHub Releases 下載，無需手動安裝。

---

## Plugin 技能一覽

| Skill                | 功能             | 觸發關鍵字                   |
|----------------------|------------------|------------------------------|
| **timecard**         | 工時表管理       | 工時表、timesheet、填工時     |
| **wedaka**           | 打卡出勤         | 打卡、clock in/out、出勤      |
| **jira**             | JIRA 操作        | JIRA issue、專案狀態、sprint  |
| **outlook-calendar** | 行事曆查詢       | 會議、行事曆、schedule        |

## 系統需求

- [Claude Code](https://claude.ai/code) CLI
- macOS (amd64/arm64)、Linux (amd64/arm64)、WSL 皆支援
- 各服務的帳號與存取權限

---

## 進階資訊

<details>
<summary><strong>專案結構</strong></summary>

```
├── timecard-cli/         Go CLI — 工時表管理
├── wedaka-cli/           Go CLI — 打卡出勤
├── .claude-plugin/       Claude Code plugin
│   ├── plugin.json           Plugin manifest（4 skills）
│   ├── marketplace.json      Marketplace manifest
│   ├── skills/               Skill 定義（每個 skill 一份 SKILL.md）
│   └── scripts/              Launcher scripts（自動下載 CLI binary）
├── scripts/              建置腳本
│   └── build.sh              跨平台 cross-compile
├── .github/workflows/    CI/CD
│   └── release.yml           Push tag 自動 build + release
├── .claude/              Claude Code 專案設定
│   └── settings.json.example    環境變數範本
└── dev/                  開發筆記（gitignored）
```

</details>

<details>
<summary><strong>CLI 工具詳細說明</strong></summary>

### timecard-cli — 工時表管理

透過 web scraping 操作工時系統，支援 cookie-based session 自動管理。

| 指令         | 說明                 |
|--------------|----------------------|
| `projects`   | 列出可用專案         |
| `activities` | 列出活動項目         |
| `timesheet`  | 查看當週工時表       |
| `summary`    | 工時摘要             |
| `save`       | 儲存工時（draft only） |
| `version`    | 版本資訊             |

### wedaka-cli — 打卡出勤

REST API client，操作 WeDaka 出勤系統。

| 指令            | 說明             |
|-----------------|------------------|
| `clock-in`      | 上班打卡         |
| `clock-out`     | 下班打卡         |
| `timelog`       | 查看出勤記錄     |
| `check-workday` | 檢查是否為工作日 |
| `version`       | 版本資訊         |

### 共通設計

- 所有輸出皆為 JSON 格式（stdout 為成功結果，stderr 為錯誤訊息）
- 密碼透過 `--pass-stdin` pipe 輸入，不使用 command-line flag
- Exit codes：`0` 成功、`1` 一般錯誤、`2` 認證/設定、`3` 驗證、`4` 網路

</details>

<details>
<summary><strong>從原始碼建置</strong></summary>

兩個 CLI 皆為零依賴的 Go module（Go 1.25，僅使用 stdlib）。

```bash
# 單一平台建置（開發用）
cd timecard-cli && go build -o timecard .
cd wedaka-cli && go build -o wedaka .

# 跨平台建置（產出至 dist/）
./scripts/build.sh v0.1.0
```

### Release 流程

Push version tag 後 GitHub Actions 會自動 cross-compile 並建立 Release：

```bash
git tag v0.1.0
git push origin v0.1.0
```

支援 6 個平台：linux/darwin/windows × amd64/arm64。

</details>

## 授權

[MIT License](LICENSE)
