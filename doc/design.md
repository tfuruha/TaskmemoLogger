# 設計書 (Architecture & Design)

## 1. アーキテクチャ概観

本アプリは Wails v2 フレームワークを採用し、**Go バックエンド** と  
**Vanilla HTML/CSS/JS フロントエンド** を明確に分離する構成をとる。

```
┌─────────────────────────────────────────────────────┐
│  フロントエンド (WebView2 内)                         │
│  index.html / style.css / main.js                    │
│  ↕ JS バインディング（自動生成）                       │
├─────────────────────────────────────────────────────┤
│  Go バックエンド                                      │
│  app.go (バインディング層 + ライフサイクルフック)      │
│    ├── logger.go        (ファイル I/O 層)             │
│    └── tags_manager.go  (タグ永続化層)                │
│  focus_windows.go       (Win32 フォーカス制御)        │
│  single_instance.go     (Windows Mutex ロック)        │
│  main.go                (エントリポイント)            │
└─────────────────────────────────────────────────────┘
         ↓ 書き込み
%USERPROFILE%\Documents\TaskmemoLogger\log\YYYY-MM_log.md
%USERPROFILE%\Documents\TaskmemoLogger\tags.json
```

## 2. ファイル別・クラス別構成

### `main.go` — エントリポイント

- シングルインスタンスガードを実行してから `wails.Run()` を呼び出す
- ウィンドウ設定（サイズ・背景色・タイトル）を定義する

### `single_instance.go` — シングルインスタンス制御

- Windows 名前付き Mutex `Local\TaskmemoLogger_SingleInstance` を用いて二重起動を防止する
- `acquireSingleInstanceLock()` は取得成功時にハンドルを、二重起動検知時に `handle=0, err=nil` を返す
- 呼び出し元（`main.go`）は `handle == 0` の場合に `os.Exit(0)` で即終了する

### `app.go` — Go/JS バインディング層

フロントエンドの JS から呼び出せる Public メソッドを定義する。

| メソッド | 引数 | 戻り値 | 説明 |
|---------|------|--------|------|
| `SaveLog` | `tags []string, text string` | `error` | Markdown 追記 + 新規タグ保存 |
| `GetTodayLogs` | なし | `[]LogEntry, error` | 今日分ログの読み取り |
| `GetTagSuggestions` | `prefix string` | `[]string` | 前方一致タグ候補 |
| `AddTag` | `tag string` | `error` | タグの単独追加 |

`startup()` で `os.UserHomeDir()` からパスを確定し、`TaskLogger` と `TagsManager` を初期化する。

### `logger.go` — ファイル I/O 層

```go
type LogEntry struct {
    Timestamp string   `json:"timestamp"` // "YYYY-MM-DD HH:MM"
    Tags      []string `json:"tags"`
    Text      string   `json:"text"`
}

type TaskLogger struct { logDir string }

func (l *TaskLogger) Append(entry LogEntry) error   // O_APPEND でファイルに追記
func (l *TaskLogger) ReadToday() ([]LogEntry, error) // 今月ファイルから当日分を解析
```

- `Append()`: `os.O_APPEND | os.O_CREATE | os.O_WRONLY` で開いて追記する
- `ReadToday()`: 正規表現で `## YYYY-MM-DD HH:MM` ヘッダーを検出、当日分のみを返す
- Markdown フォーマットは Python 版と完全互換（既存ログを壊さない）

### `tags_manager.go` — タグ永続化層

```go
type TagsManager struct { filePath string }

func (t *TagsManager) Load() ([]string, error)
func (t *TagsManager) Add(tag string) error         // 重複排除（大小文字非区別）+ 原子書き込み
func (t *TagsManager) GetSuggestions(prefix string) []string // 前方一致検索
```

#### 原子書き込み（Rename 方式）

```
tags.json.tmp へ書き込み → os.Rename(tmp, tags.json)
```

`truncate 後・書き込み前`にプロセスが終了しても既存データが消えない。  
Python 版で潜在していたタグ全消失リスクを根本排除している。

### `frontend/index.html` — UI 構造

```
<main class="chat-area">   ← 今日のログをチャットバブルで表示（スクロール・読み取り専用）
<footer class="input-panel">
  <div class="tag-pills-area">  ← タグ pill + テキスト入力
    <ul class="suggestion-list"> ← サジェストドロップダウン（position: absolute）
  <textarea class="task-textarea">  ← タスク内容
  <button class="submit-btn">保存して閉じる</button>
```

### `frontend/src/style.css` — スタイル設計

CSS カスタムプロパティ（変数）でデザイントークンを一元管理する。

```css
:root {
  --bg-primary:   #1a1b1e;  /* メイン背景 */
  --bg-secondary: #25262b;  /* 入力パネル背景 */
  --accent:       #5c7cfa;  /* フォーカスリング・ボタン */
  --tag-pill-bg:  #3b4254;  /* タグ pill 背景 */
}
```

主なアニメーション:
- チャットバブル出現: `bubble-in` (translateY + opacity, 0.18s)
- タグ pill 出現: `pill-in` (scale + opacity, 0.12s)
- サジェストリスト出現: `dropdown-in` (translateY + opacity, 0.10s)

### `frontend/src/main.js` — フロントエンド制御

```javascript
import { SaveLog, GetTodayLogs, GetTagSuggestions } from '../wailsjs/go/main/App.js';
import { Quit, EventsOn } from '../wailsjs/runtime/runtime.js';
```

Wails が `wails build` 時に自動生成するバインディングをインポートする。

主な処理フロー:

```
DOMContentLoaded
  → GetTodayLogs()  → チャットバブルを追加
  → EventsOn('app:ready-focus', ...) → リスナー登録

  ※ Go 側 OnDomReady が完了すると 'app:ready-focus' イベントが発火
  → focusTagInput()  → tagInput.blur() + requestAnimationFrame(focus + select)

window 'focus' イベント（Alt+Tab 復帰時など）
  → focusTagInput()

tagInput keyup
  → GetTagSuggestions(prefix) → <ul> を更新

submitBtn click / Ctrl+Enter
  → SaveLog(tags, text)  // Go 側で Append + Add を順次実行
  → Quit()               // アプリ終了
```

## 3. データフロー

```
[ユーザー入力] ─Ctrl+Enter→ [SaveLog (JS)]
                                   ↓ await
                             [App.SaveLog (Go)]
                                   ├─→ TaskLogger.Append()  → YYYY-MM_log.md (追記)
                                   └─→ TagsManager.Add()    → tags.json (原子書き込み)
                                   ↓ 完了後
                             [Quit()]  → プロセス終了
```

## 4. ファイル・ディレクトリ構成

```
TaskmemoLogger/
├── main.go                  エントリポイント。Wails 設定・シングルインスタンスガード
├── app.go                   Go/JS バインディング層（Public メソッド + ライフサイクルフック）
├── focus_windows.go         Win32 API による WebView2 フォーカス強制制御
├── logger.go                Markdown ファイル I/O
├── tags_manager.go          tags.json 永続化・前方一致検索
├── single_instance.go       Windows 名前付き Mutex によるシングルインスタンス制御
├── logger_tags_test.go      Go ユニットテスト（8ケース）
├── wails.json               Wails プロジェクト設定
├── go.mod / go.sum          Go モジュール
├── frontend/
│   ├── index.html           チャット UI の構造
│   ├── src/
│   │   ├── style.css        ダークテーマ CSS（カスタムプロパティ）
│   │   └── main.js          フロントエンド制御ロジック
│   ├── dist/                Vite ビルド成果物（バイナリに埋め込み）
│   └── wailsjs/             Wails 自動生成バインディング（編集不要）
├── build/
│   └── bin/
│       └── TaskmemoLogger.exe  リリースバイナリ
├── doc/
│   ├── specification.md     仕様書（本ドキュメントと対）
│   └── design.md            本ドキュメント
└── doc_org/                 Python 版の旧仕様書群（参照用）
```

## 5. 設計上の決定事項

### なぜ SQLite を使わないか
出力結果を「プレーンなテキスト」としてそのまま AI プロンプトに貼り付けることが目的のため、  
変換ロジック不要で人間も AI も読めるMarkdown ファイルへの直接保存を採用している。

### なぜログと `tags.json` を `%USERPROFILE%\Documents\TaskmemoLogger\` 以下に統合するか

- `%APPDATA%` は隠しフォルダで、ユーザーが手動編集しにくい
- `%USERPROFILE%\Documents\TaskmemoLogger\` に統一することで、エクスプローラーから直接開いてログやタグを確認・編集できる
- `Documents` のルートに汎用名フォルダ（`log` など）を作るのは他アプリとの競合リスクがあるため、アプリ専用フォルダ内にサブフォルダとして配置する
- ショートカット/タスクバーなどどこから起動しても同一パスを参照できる

フォルダ構成:

```
%USERPROFILE%\Documents\TaskmemoLogger\
  ├── tags.json       ← タグ永続化
  └── log\
        └── YYYY-MM_log.md  ← タスクログ（月別）
```

### なぜ tags.json を `os.Rename()` で原子的に書くか
Python 版では「ファイルを truncate してから書き込む」パターンのため、  
`truncate 後・書き込み完了前`のプロセス強制終了でタグが全消失するリスクがあった。  
`tmp ファイル → os.Rename()` は NTFS 上でアトミックな操作となり、このリスクを根本排除する。

### なぜ `SaveLog` と `AddTag` を同一スレッドで順次実行するか
Wails バインディングは各呼び出しを Promise として完了を待つため、  
`SaveLog` 内で `Append` → `Add` の順に同期実行し、  
JS 側からは `await SaveLog()` → `Quit()` の順に呼び出すだけでデータ整合性が保たれる。

### なぜシングルインスタンスロックが必要か
タグ追加時に `Load → 追記判定 → Save(Rename)` という非アトミックなシーケンスがあるため、  
2プロセスが同時に実行すると一方の `Save` が他方の変更を上書きしてタグが消える恐れがある。  
Windows 名前付き Mutex でプロセスを1つに制限することでリスクをゼロにしている。

### サジェストドロップダウンに `position: absolute` を採用した理由
Python 版では `tk.Listbox + place（絶対座標）` で同様の機能を実装していたが、  
Z-index の競合やウィンドウ非アクティブ時の描画残りが問題になっていた。  
Web 標準の `position: absolute` + CSS ドロップダウンではそれらの問題は発生しない。

### 起動時キャレット非表示（ゴーストフォーカス）問題の解決策

Wails の WebView2 では、`StartHidden: true` + `WindowShow()` でウィンドウを表示しても、
WebView2 の内部 HWND（ウィンドウハンドル）に Win32 レベルのキーボードフォーカスが渡らない。
この状態では DOM 側の `element.focus()` を呼んでも「DOMフォーカスはあるがOSのキーボード入力が届かない」
ゴーストフォーカスになり、キャレットが表示されずIME入力がスクリーン左上に流れるという症状が発生する。

**解決策 (`focus_windows.go`):**

`OnDomReady` ライフサイクルフック内で以下の Win32 API シーケンスを実行する。

```
1. WindowShow(ctx)               ← Wails 経由で top-level HWND を表示
2. FindWindowW("TaskmemoLogger") ← HWND を取得
3. SetForegroundWindow(hwnd)     ← 最前面へ
4. AttachThreadInput(fgThread, myThread, true)
                                 ← 異スレッドからの SetFocus 制限を解除
5. EnumChildWindows + GetClassNameW("Chrome_WidgetWin_1")
                                 ← WebView2 の Chromium ホスト HWND を特定
   （最大 500ms リトライ）
6. SetFocus(webview2Hwnd)        ← Win32 レベルでキーボードフォーカスを直接付与
7. EventsEmit("app:ready-focus") ← フロントエンドに element.focus() を指示
```

JS 側は `EventsOn('app:ready-focus')` を受けて `requestAnimationFrame` 内で
`tagInput.focus()` + `tagInput.select()` を呼ぶことでキャレットを確実に描画する。

`GetCurrentThreadId` は `kernel32.dll` に属するため、`user32.dll` ではなく
`kernel32.dll` から呼び出す必要がある点に注意する。

## 6. テスト設計

### Go ユニットテスト（`logger_tags_test.go`）

| テストケース | 検証内容 |
|------------|---------|
| `TestTaskLogger_AppendAndReadToday` | Append → ReadToday のラウンドトリップ |
| `TestTaskLogger_MultilineText` | マルチラインテキストのファイル書き込み |
| `TestTaskLogger_EmptyLogFile` | ログファイルなし時の空配列返却 |
| `TestTagsManager_AddAndSuggest` | 追加後の前方一致候補取得 |
| `TestTagsManager_DuplicatePrevention` | 完全一致重複の排除 |
| `TestTagsManager_CaseInsensitiveDuplicate` | 大文字小文字を無視した重複排除 |
| `TestTagsManager_AtomicWrite` | 再読み込み後のデータ保持（原子書き込み確認） |
| `TestTagsManager_EmptyTag` | 空文字タグが保存されないこと |

```bash
# 実行コマンド
go test ./... -v
```
