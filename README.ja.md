# hayate（疾風）

![hayate（疾風）](./hayate.jpg)

[English](README.md)

Windows 11 と WSL2 のプロジェクトをまとめて検索する Go 製 CLI です。選択したディレクトリへ移動し、設定したアクションを実行できます。

## インストール

リリース公開後は、[GitHub Releases](https://github.com/withnic/hayate/releases) から Windows・Linux 用のビルド済みバイナリを入手できます。

Go 1.23 以上で、任意のディレクトリから次を実行します。

```sh
go install github.com/withnic/hayate/cmd/hayate@latest
```

すでにクローン済みのソースからインストールする場合は、そのリポジトリのルートで `go install ./cmd/hayate` を実行します。

配置先を PATH に追加します。配置先は `GOBIN` を設定していればそのディレクトリ、未設定なら `go env GOPATH` 配下の `bin` です。

- Windows の PowerShell から使う場合は、Windows 版だけで Windows と WSL2 の両方を検索できます。WSL 側に hayate をインストールする必要はありません。
- WSL 内の Bash や Zsh から hayate を直接実行する場合は、WSL 側にもインストールし、WSL の PATH に追加します。

Windows の対応対象は Windows 11 です。Windows 10 での動作は未確認です。

## 設定

`hayate init-config` で `~/.hayate.jsonc` を作成し、`roots` のパスを実際の環境に合わせます。

```jsonc
{
  "roots": [
    {"env": "windows", "path": "C:\\dev", "mode": "children"},
    {"env": "wsl", "distro": "Ubuntu", "path": "/home/alice/projects", "mode": "children"},
  ],
  "actions": {
    "c": {"description": "VS Code を開く", "windows": "code .", "wsl": "code ."},
  },
  // 必要ならコメントを外すと、j だけで VS Code も開きます。
  // "default_action": "c",
}
```

### WSL の検索先を設定する

1. WSL のターミナルを開き、検索したいプロジェクトが並んでいる親フォルダーに移動します。
2. そのフォルダーで `pwd` を実行し、表示された絶対パスを `roots` の WSL 項目の `path` にそのまま書きます。たとえば `/home/alice/projects` と表示されたら、次のように設定します。

   ```jsonc
   {"env": "wsl", "distro": "Ubuntu", "path": "/home/alice/projects", "mode": "children"}
   ```

3. `distro` には実際のディストリビューション名を指定します。PowerShell の `wsl -l -q` で確認できます。設定を保存したら、PowerShell で `hayate.exe list --refresh` を実行して候補を確認できます。

## `j` を登録して実行する

### PowerShell

次の2行を PowerShell で実行すると、今のセッションに `j` が登録され、検索画面が開きます。

```powershell
hayate.exe init powershell | Out-String | Invoke-Expression
j
```

新しい PowerShell でも `j` を使うには、1行目を PowerShell のプロファイル（`$PROFILE`）に追加します。

### WSL の Bash または Zsh

WSL 内に hayate をインストールした場合は、Bash で次の2行を実行します。Zsh の場合は1行目の `bash` を `zsh` に置き換えます。

```bash
eval "$(hayate init bash)"
j
```

新しいシェルでも `j` を使うには、1行目を Bash なら `~/.bashrc`、Zsh なら `~/.zshrc` に追加します。

## 使い方

- `j` で検索を開き、Enter で移動します。Esc または Ctrl+C でキャンセルできます。
- `j -c` で移動後に VS Code を開きます。上の例のように `actions` へ追加すると、新しいフラグが使えます。
- 設定のトップレベルに `"default_action": "c"` を追加すると、`j` だけで VS Code も開きます。アクションを明示した場合はそちらが優先されます。既定のアクションを省略した場合、`j` だけならディレクトリ移動のみです。
- たとえば `withnic/knit` を `/home/alice/projects/knit` に置いている場合、`j --query knit` で `knit` を入力済みの状態で検索を開けます。
- `hayate list --json` で候補を確認できます。`--refresh` を付けると再走査します。
- ラッパー名を変更するには `hayate init` に `--cmd <name>` を指定します。

## 開発時の確認

GNU Make を使える環境では、`make fmt` で Go ソースを整形し、`make check` で整形チェック、`go vet`、レース検出テストを実行します。レース検出テストには CGO と C コンパイラが必要です。

Make を使わない場合は、プロジェクトのルートで `gofmt -s -l cmd internal` を実行します。ファイル名が表示されたら、`gofmt -s -w cmd internal` で整形します。静的解析は `go vet ./...`、通常のテストは `go test ./...` で実行できます。

## ライセンス

[MIT License](LICENSE)
