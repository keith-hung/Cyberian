<p align="center">
  <img src="cyberian-logo.png" alt="Cyberian Logo" width="200">
</p>

<h1 align="center">Cyberian</h1>

<p align="center">
  Claude Code plugin — 工時表、打卡、JIRA、行事曆、Azure DevOps，用自然語言搞定職場日常
</p>

---

## 這是什麼

Cyberian 是一套 [Claude Code](https://claude.ai/code) plugin，把日常的職場系統包成 skill。安裝後不需記任何指令，直接用自然語言對 Claude Code 說話即可：

```
「幫我查這週的工時表」
「打卡上班」
「列出我的 JIRA issues」
「把這張 JIRA 的 Reviewers 欄位設成 John」
「今天有什麼會議？」
「列出 Azure DevOps 的 PR」
「我有哪些 PR 待處理？」
```

Claude 會自動挑選對應的 skill，背後透過各服務的 CLI 或 API 完成。

---

## 技能一覽

| Skill                | 功能                  | 觸發關鍵字                                                            |
|----------------------|-----------------------|----------------------------------------------------------------------|
| **nouveau-timecard** | 工時系統工時填報      | 工時表、timesheet、填工時、工時填報、同步休假                          |
| **wedaka**           | 打卡出勤              | 打卡、clock in/out、出勤                                              |
| **jira**             | JIRA 操作             | JIRA issue、專案狀態、sprint、custom fields                           |
| **outlook-calendar** | 行事曆查詢            | 會議、行事曆、schedule                                                |
| **azuredevops**      | Azure DevOps 操作     | PR、pull request、code review、my PRs、Azure DevOps、PR image attachments |
| **change-password**  | 變更 on-prem AD 密碼（自動選路徑） | 改密碼、change password、AD 密碼、密碼過期、domain password             |
| **flashback**        | 從本機逐字稿估算專案工時 | 工時回推、time attribution、worklog、花多少時間、how long did I spend  |
| **read-email** \*    | 解析 .eml / .msg 郵件 | 讀郵件、parse .eml、開啟 .msg、取出附件、信件內容                      |

> \* **read-email** 是本 marketplace 內的**獨立 plugin**（原始碼於 [keith-hung/claude-read-email](https://github.com/keith-hung/claude-read-email)），需另外安裝且有不同的執行需求 — 詳見下方「安裝與設定」中的「額外安裝：read-email」。

---

## 快速開始

最短路徑：安裝 plugin，設定你要用的服務，然後直接說話。

**1. 安裝 plugin**

```bash
claude plugin marketplace add keith-hung/Cyberian
claude plugin install cyberian@cyberian-marketplace
```

**2. 設定你要用的服務**（只設需要的，不必全填）。以工時填報為例，在 `~/.claude/settings.json` 的 `env` 區塊加入：

```jsonc
{
  "env": {
    "TIMECARD_BASE_URL": "https://your-timecard-server.example.com/",
    "TIMECARD_USERNAME": "your_username",
    "TIMECARD_PASSWORD": "your_password"
  }
}
```

其他服務的變數見下方「安裝與設定」中的「環境變數一覽」。

**3. 直接用自然語言操作**

```
「幫我查這個月的工時表」
```

各 skill 的 binary 會在首次觸發時自動下載，無需手動安裝（細節見下方「進階資訊」中的「各 skill 的執行方式」）。

---

## 安裝與設定

<details>
<summary><strong>三種安裝 scope（個人 / 團隊 / 單一專案）</strong></summary>

依使用情境選擇適合的 scope：

| Scope          | Plugin 設定位置                           | 環境變數設定位置                                         | 適用情境                      |
|----------------|-------------------------------------------|----------------------------------------------------------|-------------------------------|
| `user`（預設） | `~/.claude/settings.json`                 | `~/.claude/settings.json` 或 shell export                | 個人使用，跨專案通用          |
| `project`      | `.claude/settings.json`（進版控）         | `.claude/settings.local.json`（不進版控）或 shell export | 團隊共享 plugin，各自設定認證 |
| `local`        | `.claude/settings.local.json`（不進版控） | `.claude/settings.local.json`（不進版控）或 shell export | 僅限此專案，不進版控          |

**方式 A：User scope（推薦）** — 所有專案皆可使用，環境變數只需設定一次。

```bash
claude plugin marketplace add keith-hung/Cyberian
claude plugin install cyberian@cyberian-marketplace
```

環境變數加入 `~/.claude/settings.json` 的 `env` 區塊（參考下方「環境變數一覽」）。

**方式 B：Project scope（團隊共享）** — Plugin 設定進版控，團隊成員 clone 後自動啟用；認證資訊各自設定於 local scope。

```bash
claude plugin marketplace add keith-hung/Cyberian --scope project
claude plugin install cyberian@cyberian-marketplace --scope project
cp .claude/settings.json.example .claude/settings.local.json   # 各成員填入個人認證
```

`.claude/settings.local.json` 已被 gitignore，不會進入版控。

**方式 C：Local scope（單一專案）** — Plugin 與環境變數都僅限此專案，不進版控。

```bash
claude plugin marketplace add keith-hung/Cyberian
claude plugin install cyberian@cyberian-marketplace --scope local
cp .claude/settings.json.example .claude/settings.local.json
```

</details>

<details>
<summary><strong>環境變數一覽</strong></summary>

| 服務                          | 環境變數                                                                                                                                            |
|-------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------|
| **Nouveau Timecard** 新版工時 | `TIMECARD_BASE_URL`、`TIMECARD_USERNAME`、`TIMECARD_PASSWORD`、選用 `TIMECARD_INSECURE`                                                            |
| **WeDaka** 打卡               | `WEDAKA_API_URL`、`WEDAKA_USERNAME`、`WEDAKA_EMP_NO`、`WEDAKA_DEVICE_ID`                                                                           |
| **JIRA**                      | `JIRA_SERVER_URL`、`JIRA_USER_EMAIL`、`JIRA_API_TOKEN`、`JIRA_PROJECT`、`JIRA_BOARD`                                                               |
| **Outlook 行事曆**            | `OUTLOOK_ICS_URLS`                                                                                                                                 |
| **Azure DevOps**              | `AZDO_BASE_URL`、`AZDO_COLLECTION`、`AZDO_DOMAIN`、`AZDO_USERNAME`、`AZDO_PASSWORD`、`AZDO_PROJECT`*、`AZDO_REPO`*、`AZDO_API_VERSION`*、`AZDO_INSECURE`* |
| **Change Password** 密碼變更  | `CHPW_BASE_URL`、`CHPW_USERNAME`、選用 `CHPW_INSECURE`（僅離線自助服務入口路徑需要；本機 AD 變更免設定）                                          |

（標 `*` 為選用）

**替代方案：Shell export** — 任何 scope 都可改用 shell profile（`~/.zshrc`、`~/.bashrc`）export，不需修改 settings.json：

```bash
# ~/.zshrc or ~/.bashrc
export TIMECARD_BASE_URL="https://your-timecard-server.example.com/"
export TIMECARD_USERNAME="your_username"
export TIMECARD_PASSWORD="your_password"
# ...依需求加入其他變數
```

> **提示：**
> - 只需設定你要使用的服務，不需要全部填寫
> - 優先順序：shell export > local > project > user
> - 完整範本請參考 `.claude/settings.json.example`

</details>

<details>
<summary><strong>額外安裝：read-email（獨立 plugin）</strong></summary>

`read-email` 與上方六個內建 skill 不同，是 marketplace 內的**獨立 plugin**，需另外安裝：

```bash
claude plugin install read-email@cyberian-marketplace
```

執行需求：[uv](https://docs.astral.sh/uv/) 與 Python 3.10+。它不需環境變數，透過 uv 執行 Python 解析器處理 `.eml` / `.msg` 檔案。

</details>

---

## 系統需求

- [Claude Code](https://claude.ai/code) CLI
- macOS (amd64/arm64)、Linux (amd64/arm64)、WSL 皆支援
- 各服務的帳號與存取權限
- `read-email` 另需 [uv](https://docs.astral.sh/uv/) 與 Python 3.10+

---

## 進階資訊

<details>
<summary><strong>各 skill 的執行方式</strong></summary>

| Skill            | 實作                                                          | binary 來源                              |
|------------------|---------------------------------------------------------------|------------------------------------------|
| nouveau-timecard | Go CLI（HTTP + HTML 解析）                                     | 首次觸發自 GitHub Releases 自動下載，快取於 `scripts/.cache/` |
| wedaka           | Go CLI（REST API）                                             | 首次觸發自 GitHub Releases 自動下載       |
| azuredevops      | Go CLI（REST API）                                             | 首次觸發自 GitHub Releases 自動下載       |
| change-password  | Path A: PowerShell（ADSI，本機免 OTP，僅 Windows）；Path B: Go CLI（chpw-cli，互動 `-i` 或兩段式，App/SMS OTP，Windows/Mac 皆可） | Path A 隨 skill 附帶，無需下載；Path B 首次觸發自 GitHub Releases 自動下載 |
| jira             | 第三方 [jira-cli](https://github.com/ankitpokhrel/jira-cli)   | launcher 自動下載並依 `JIRA_*` 初始化設定 |
| outlook-calendar | 直接讀取 ICS 訂閱（`OUTLOOK_ICS_URLS`）                        | 無 binary，由 skill 邏輯處理              |
| read-email       | 獨立 plugin，Python 解析器                                     | 透過 uv 執行，需 Python 3.10+             |

</details>

<details>
<summary><strong>專案結構</strong></summary>

```
├── timecard-cli/         Go CLI — 舊版工時表（已自 plugin 退役，原始碼保留供參考）
├── nouveau-timecard-cli/ Go CLI — 工時系統工時填報（草稿）
├── wedaka-cli/           Go CLI — 打卡出勤
├── azuredevops-cli/      Go CLI — Azure DevOps PR 管理
├── chpw-cli/             Go CLI — 密碼變更（自助服務入口：互動 -i 或兩段式，App/SMS OTP）
├── .claude-plugin/       Claude Code plugin metadata
│   ├── plugin.json           Plugin manifest
│   └── marketplace.json      Marketplace manifest（含外部 read-email plugin）
├── skills/               Skill 定義（每個 skill 一份 SKILL.md）
│                         jira 走 jira-cli、outlook-calendar 讀 ICS，皆無自帶 Go CLI
├── scripts/              Launcher scripts + 建置腳本
│   ├── build.sh              跨平台 cross-compile
│   └── *-launcher.sh/.ps1    自動下載 CLI binary
├── .github/workflows/    CI/CD
│   └── release.yml           Push tag 自動 build + release
├── .claude/              Claude Code 專案設定
│   └── settings.json.example    環境變數範本
└── dev/                  開發筆記（gitignored）
```

</details>

<details>
<summary><strong>CLI 指令詳細說明</strong></summary>

各 CLI 目錄另有獨立 README，說明建置與設定細節。以下為指令一覽。

### timecard-cli — 舊版工時表（已退役）

> **已退役：** 此 CLI 對應的 skill 已自 plugin 移除、不再隨 release 發布，Claude Code 不會再觸發它。原始碼保留於 `timecard-cli/` 供參考與必要時手動建置，工時填報請改用 nouveau-timecard-cli。

| 指令         | 說明                   |
|--------------|------------------------|
| `projects`   | 列出可用專案           |
| `activities` | 列出活動項目           |
| `timesheet`  | 查看當週工時表         |
| `summary`    | 工時摘要               |
| `save`       | 儲存工時（draft only） |
| `version`    | 版本資訊               |

### nouveau-timecard-cli — 工時系統工時填報

針對改版後的智慧工時系統（Razor Pages），透過 HTTP + HTML 解析操作，沿用 `TIMECARD_*` 環境變數。僅支援草稿儲存，刻意不提供送出審核。

| 指令         | 說明                           |
|--------------|--------------------------------|
| `projects`   | 列出可填報的專案               |
| `activities` | 列出專案底下的活動             |
| `timesheet`  | 查看指定月份的填報狀況         |
| `save`       | 儲存工時（draft only，原子式） |
| `sync-leave` | 同步休假至「休假」活動（草稿） |
| `version`    | 版本資訊                       |

### wedaka-cli — 打卡出勤

REST API client，操作 WeDaka 出勤系統。

| 指令            | 說明             |
|-----------------|------------------|
| `clock-in`      | 上班打卡         |
| `clock-out`     | 下班打卡         |
| `timelog`       | 查看出勤記錄     |
| `check-workday` | 檢查是否為工作日 |
| `version`       | 版本資訊         |

### azuredevops-cli — Azure DevOps 操作

REST API client，操作 Azure DevOps Server（on-premises, IIS Basic Auth）。

| 指令              | 說明                |
|-------------------|---------------------|
| `projects`        | 列出所有專案        |
| `repos`           | 列出儲存庫          |
| `prs`             | 列出 Pull Requests  |
| `my-prs`          | 跨專案查詢我的 PR   |
| `pr`              | 檢視 PR 詳情        |
| `pr-create`       | 建立 PR             |
| `pr-update`       | 更新 PR             |
| `pr-approve`      | 核准 PR             |
| `pr-reject`       | 拒絕 PR             |
| `pr-comment`      | 新增留言 / 回覆留言 |
| `pr-comments`     | 列出留言            |
| `pr-attachment`   | 上傳圖片附件至 PR   |
| `pr-reviewers`    | 列出審閱者          |
| `pr-add-reviewer` | 新增審閱者          |
| `version`         | 版本資訊            |

### chpw-cli — 變更 on-prem AD 密碼

Go CLI，透過離線自助服務入口變更 on-prem AD 密碼，單一旗標驅動（非子指令）。密碼一律經 `--pass-stdin` 或互動隱藏提示輸入，session 檔只保存 cookie 與表單 token、不含密碼。`--method` 選 OTP 遞送方式（`APP` = i-daka/Email，預設；`SMS` = 手機簡訊）。chpw 入口路徑 Windows 與 Mac 皆可用；domain-joined 且連得到 DC 的情境改走 PowerShell（ADSI）本機變更（僅 Windows、免 OTP），由 `change-password` skill 自動選路。

| 指令 / 旗標                  | 說明                                                                          |
|------------------------------|-------------------------------------------------------------------------------|
| `chpw -i`                    | 互動一鍵（人用）：提示 目前密碼 → OTP → 新密碼（預設兩次確認，`--no-confirm` 跳過） |
| `chpw` + `--pass-stdin`      | 兩段式 step1（自動化）：驗證目前密碼、觸發 OTP、印出 `next` 指令               |
| `chpw --continue --otp <碼>` | 兩段式 step2：以 `--pass-stdin` 送出新密碼與 OTP（需在效期內完成）            |
| `version`                    | 版本資訊                                                                      |

### 共通設計

- 所有輸出皆為 JSON 格式（stdout 為成功結果，stderr 為錯誤訊息）
- 密碼透過 `--pass-stdin` pipe 輸入，不使用 command-line flag
- Exit codes：`0` 成功、`1` 一般錯誤、`2` 認證/設定、`3` 驗證、`4` 網路

</details>

<details>
<summary><strong>從原始碼建置</strong></summary>

所有 CLI 皆為 Go module（Go 1.25），使用 [cobra](https://github.com/spf13/cobra) 處理 CLI 命令與 flag 解析。

```bash
# 單一平台建置（開發用）
cd nouveau-timecard-cli && go build -o nouveau-timecard .
cd wedaka-cli && go build -o wedaka .
cd azuredevops-cli && go build -o azuredevops .
cd chpw-cli && go build -o chpw .
# 已退役的 timecard-cli 仍可手動建置：cd timecard-cli && go build -o timecard .

# 跨平台建置（產出至 dist/）
./scripts/build.sh v0.3.0
```

**Release 流程** — Push version tag 後 GitHub Actions 會自動 cross-compile 並建立 Release（6 個平台：linux/darwin/windows × amd64/arm64）：

```bash
git tag v0.3.0
git push origin v0.3.0
```

</details>

## 授權

[MIT License](LICENSE)
