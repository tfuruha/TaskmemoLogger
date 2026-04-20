# TaskmemoLogger

デスクトップで素早くタスクの実績を記録し、後で生成AIに要約させるための  
Markdown ログを生成する「Fire & Forget 型」アプリケーションです。

## 特徴

- **爆速の UX**: `Ctrl+Enter` を押すだけで即座に保存してアプリ終了。次の作業にすぐ戻れます
- **チャット風 UI**: 本日分のタスクがチャットバブル形式でスクロール表示され、追記感覚で入力できます
- **タグ pill 入力 + オートコンプリート**: タグを入力中に過去のタグが候補として表示されます。キーボードのみで選択でき、選択済みタグは pill として表示されます
- **Markdown 出力**: 月ごとに `YYYY-MM_log.md` 形式で保存。ChatGPT / Claude などの AI にそのまま読み込ませて月次・半期レポートの要約が行えます
- **単一 EXE 配布**: Python ランタイム不要。`TaskmemoLogger.exe` 1ファイルで動作します

## 動作環境

| 項目 | 要件 |
|------|------|
| OS | Windows 10 (Build 19041以降) / Windows 11 |
| ランタイム | WebView2 (Windows 11 は標準搭載 / Win10 は自動インストール) |
| その他 | 追加インストール不要 |

## セットアップ

1. [GitHub Releases (最新版)](https://github.com/tfuruha/TaskmemoLogger/releases/latest) から `TaskmemoLogger.exe` をダウンロードします  
2. ダウンロードしたファイルを任意のフォルダに配置して実行します
3. 初回起動時、以下のフォルダが自動的に作成されます:
   - `%USERPROFILE%\Documents\TaskmemoLogger\` — タグ候補データ・ログ（同一フォルダ配下に統合）

## 使い方

```
起動
  ↓
タグ欄にフォーカス（自動）
  ↓
タグを入力（候補が出たら ↑↓ + Enter で選択 / そのまま Enter で新規追加）
  ↓
Tab キーでタスク内容欄へ移動
  ↓
タスク内容を入力
  ↓
Ctrl+Enter → 保存して即終了
```

## キーボードショートカット

| キー | 動作 |
|------|------|
| `Tab` | タグ入力欄 → タスク内容欄へ移動 |
| `↑` / `↓` | タグ候補リストを移動 |
| `Enter`（候補表示中） | タグを選択して pill 化 |
| `Enter`（候補なし） | 入力中のタグを新規追加して pill 化 |
| `Backspace`（タグ欄・空） | 末尾の pill を削除 |
| `Escape` | 候補リストを閉じる |
| `Ctrl+Enter` | タスクを保存してアプリ終了 |

## 出力ファイルの形式

`%USERPROFILE%\Documents\TaskmemoLogger\log\YYYY-MM_log.md` に以下の形式で追記されます:

```markdown
## 2026-04-18 15:30
- [会議] [プロジェクトA]
- 進捗報告MTG。次回アクションを確認。

## 2026-04-18 17:45
- [開発]
- PR レビュー対応。コメント3件を修正して再レビュー依頼。
```

## タグ候補ファイル

`%USERPROFILE%\Documents\TaskmemoLogger\tags.json` にタグが蓄積されます。  
テキストエディタで直接編集できます（不要なタグの削除など）。

```json
[
  "会議",
  "開発",
  "ドキュメント",
  "レビュー"
]
```

## 開発・ビルド

```
# 必要ツール
Go 1.23+
Node.js 18+
Wails v2  (go install github.com/wailsapp/wails/v2/cmd/wails@latest)

# 開発モード（ホットリロード）
wails dev

# リリースビルド
wails build
# → build\bin\TaskmemoLogger.exe が生成されます
```

## ドキュメント

- [仕様書 (doc/specification.md)](doc/specification.md)
- [設計書 (doc/design.md)](doc/design.md)
- [旧 Python 版仕様書 (doc_org/)](doc_org/)
