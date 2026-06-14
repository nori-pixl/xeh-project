# xeh — XML設計図で動くポリグロット実行エンジン

`xeh` は、1つの XML ファイル(`app.xeh`)と1つの設定ファイル(`set.json`)を中心に、
複数言語のコード・HTTPサーバー・共有メモリを同時に定義・実行できる「実行環境(ランタイム)」です。

Goで書かれた `main.go` 以下のプログラムが、この2つのファイルを読み込み、
書かれている内容に従って実際の処理(外部言語の実行、HTTPサーバー起動、変数操作など)を行います。

---

## 1. 全体イメージ

```
set.json (設計図)  ──┐
                      ├─→  xehエンジン (Go製) ──→ 実際の処理を実行
app.xeh (アプリ本体) ─┘
```

- **`app.xeh`**: 「何をするか」を書くファイル。共有変数の初期値、HTTPサーバーの定義、
  実行したい各言語のコードなどをXMLタグで記述する。
- **`set.json`**: 「どのタグをどう処理するか」を定義する設計図。
  - タグ名 → 処理担当(ハンドラー)のマッピング
  - 外部言語(Python, Goなど)の実行コマンド設定
  - 出力先ファイルパスなどのエンジン設定
- **xehエンジン本体(Go)**: `set.json` を見て `app.xeh` の各タグをディスパッチ(振り分け)し、
  対応する処理を実行する。

---

## 2. set.json — 設計図の構造

```json
{
  "meta": {
    "name": "xeh-core-engine",
    "version": "1.0.9",
    "license": "MIT",
    "charset": "UTF-8"
  },
  "handlers": {
    "xeh-logic": "core.logic",
    "ai-logic": "core.engine"
  },
  "runtimes": {
    "py": {
      "command": "python",
      "args": ["-u", "{src}"]
    }
  },
  "engines": {
    "ai-logic": {
      "type": "py",
      "src": "plugins/ai_core.py"
    }
  },
  "plugins": {
    "ai-mod": {
      "src": "plugins/ai_plugin.xeh"
    }
  }
}
```

### 各セクションの役割

| セクション | 役割 |
|---|---|
| `meta` | アプリ名・バージョン・ライセンスなどの基本情報。`--version` `--license` で表示される |
| `handlers` | **設計図の本体**。`app.xeh` 内のタグ名 → エンジン内部のハンドラーID のマッピング |
| `runtimes` | 外部言語の起動コマンド定義(`{src}` はファイルパスに置換される) |
| `engines` | タグごとに「どの言語(`runtimes`のキー)で」「どのファイルに」コードを書き出すかの設定 |
| `plugins` | `<subframework import="...">` で読み込める外部 `.xeh` ファイルの定義 |

### `handlers` の値(ハンドラーID)一覧

| ハンドラーID | 処理内容 |
|---|---|
| `core.logic` | `<xeh-logic>` 内のネイティブ命令(`<print>` など)を実行 |
| `core.engine` | `engines` 設定に従ってタグの内容をファイル出力し、外部プロセスとして実行 |

`handlers` に定義がなくても、同名タグが `engines` に存在する場合は自動的に `core.engine` として処理される(フォールバック)。

---

## 3. app.xeh — アプリ本体の構造

```xml
<xeh>
  <root memory_name="default_space">
    [
      {"name": "username", "type": "string", "value": "taro"},
      {"name": "count", "type": "int", "value": 42}
    ]
  </root>

  <server id="main" port="8080">
    <!DOCTYPE html>
    <html><body><h1>xeh server</h1></body></html>
  </server>

  <subframework roteapp="main">
    <xeh-logic>
      <print>Hello, ${username}! count = ${count}</print>
    </xeh-logic>

    <ai-logic>
import sys
print("Hello from python engine")
    </ai-logic>
  </subframework>
</xeh>
```

### 主要タグ

| タグ | 役割 |
|---|---|
| `<xeh>` | ルート要素。アプリ全体を囲む |
| `<root memory_name="...">` | 共有メモリ空間の初期値をJSON配列で定義。`memory_name` は空間の識別名 |
| `<server id="..." port="...">` | HTTPサーバーを起動。中身のHTMLがそのまま `/` で返される。`/submit` でフォーム送信を受け取れる |
| `<subframework>` | 実行したい処理のまとまり。`import` 属性で他の `.xeh` をプラグインとして読み込める |
| `<subframework>` 内の各タグ | `set.json` の `handlers` で指定されたハンドラーに振り分けられる |

### 共有メモリと変数展開

`<root>` で定義した変数は、全ての処理から共有される「共有メモリ空間」に格納される。

- `<xeh-logic>` 内の `<print>` では `${変数名}` で展開できる
  - 例: `${username}` → `taro`
- 外部言語(Python等)には、起動時に標準入力(stdin)経由でJSONとして渡される

```json
{
  "memory_name": "default_space",
  "data": {
    "username": "taro",
    "count": 42
  }
}
```

---

## 4. 実行の流れ(全体シーケンス)

1. `set.json` を読み込む(壊れていたら起動失敗)
2. `--version` / `--license` フラグがあれば表示して終了
3. `app.xeh` を読み込む
4. `<root>` のJSONをパースし、共有メモリ空間を初期化
5. Ctrl+C 等のシグナルハンドラを準備(子プロセスを全停止できるように)
6. `<server>` タグがあれば、それぞれHTTPサーバーを非同期起動
7. `<subframework>` を順に処理
   - `import` 属性があれば、対応する `.xeh` プラグインを読み込み、変数をマージ
   - 子タグそれぞれについて:
     1. `set.json` の `handlers` でタグ名に対応するハンドラーIDを調べる
     2. 該当ハンドラーが内部レジストリにあれば実行
     3. なければ `engines` にタグ定義があるか確認し、あれば `core.engine` として実行
8. 外部言語プロセス(`core.engine`で起動したもの)の終了を待つ
9. HTTPサーバーが1つ以上あれば、プロセスを維持し続ける

---

## 5. Go側のコード構成(エンジン本体)

```
xeh/
├── main.go                       … エントリーポイント(設定読込→ディスパッチのみ)
├── set.json                       … 設計図
├── app.xeh                         … アプリ本体
└── internal/
    ├── core/
    │   ├── types.go               … XehApp / SetConfig などの型定義
    │   ├── state.go               … SharedStore(共有メモリ)・ProcessList(子プロセス管理)
    │   ├── registry.go            … ハンドラー登録レジストリ・Context型
    │   └── loader.go              … set.json / .xeh ファイルの読み込み
    └── handlers/
        ├── xeh.go                 … <root> 共有メモリの初期化・マージ
        ├── subframework.go        … <subframework> 走査・set.jsonに基づくディスパッチ
        ├── logic.go                … <xeh-logic> 担当 (core.logic)
        ├── engine.go               … 外部言語エンジン担当 (core.engine)
        └── server.go               … <server> HTTPサーバー担当
```

### なぜこの構成にしたか

以前は `main.go` に全部のロジックが詰まっていたが、以下の理由で分割した。

- **set.jsonが「設計図」として機能する**:タグ名とハンドラーIDの対応表を見れば、
  「このタグは誰が処理するか」が一目でわかる。
- **新しいタグの追加が、既存コードに触れずにできる**:
  1. `internal/handlers/` に新しいファイルを作る(例: `database.go`)
  2. `init()` で `core.Register("core.database", RunDatabase)` のように登録
  3. `set.json` の `handlers` に `"db": "core.database"` を追記
  4. `app.xeh` に `<db>...</db>` を書く

  → `main.go` や `subframework.go` は一切変更不要。

### コンポーネント間の関係

```
main.go
  ├─ core.LoadSetConfig("set.json")   … 設計図を読み込む
  ├─ core.MustLoadXehFile("app.xeh")  … アプリ本体を読み込む
  ├─ core.NewSharedStore()            … 共有メモリを生成
  ├─ handlers.LoadRootVariables(...)  … <root> を共有メモリに展開
  ├─ handlers.StartHTTPServer(...)    … <server> を起動
  └─ handlers.DispatchSubFrameworks() … <subframework> をディスパッチ
        └─ core.Get(handlerID)        … set.jsonのhandlersからハンドラーを取得
              ├─ handlers.RunLogic    (core.logic)
              └─ handlers.RunEngine   (core.engine)
```

---

## 6. 現状でできること / 制約

### できること

- XMLファイル1つで「変数定義 + Webサーバー + 複数言語の実行」を表現できる
- 各言語(Python, Go, Rust, Java, PHPなど)を `set.json` の `runtimes` に追加すれば、
  対応する `<タグ>` をそのまま書くだけで実行できる
- 共有メモリ経由で、全エンジン・全プロセスに同じ変数値を渡せる
- `<subframework import="...">` で他の `.xeh` をプラグインとして合成できる
- 新しいタグ処理を追加しても、既存ファイルに手を入れる必要がない(レジストリ方式)

### 現状の制約・注意点

- 外部言語の実行は `sh -c` / `cmd /C` 経由のため、**`app.xeh` の内容をそのまま実行する**設計になっている。
  信頼できない入力を `app.xeh` に渡す用途では、コマンドインジェクションのリスクがある。
- `core.engine` は実行結果を共有メモリに**書き戻す仕組みがない**(現状は一方向: xeh → 外部プロセス)。
  双方向にしたい場合は、stdoutやファイル経由での結果取り込み機構が必要。
- HTTPサーバーは起動するが、Ctrl+C時に `srv.Shutdown()` などで明示的にクローズしていない
  (プロセス自体をkillして終了する想定)。
- `rs`(Rust)や `cpp` の `runtimes` 設定はコンパイル+実行を `&&` で連結しているが、
  `sh -c` 経由なのでシェルが対応している必要がある(Windowsの `cmd /C` では `&&` の挙動が異なる場合あり)。

---

## 7. 今後拡張しやすいポイント(アイデア)

- `core.condition`: `<if>` `<switch>` のような分岐タグ
- `core.database`: SQLite/Postgresなどへの読み書きタグ
- `core.http_request`: 外部APIを呼び出すタグ
- 外部エンジンからの結果を共有メモリに書き戻す `core.engine` の拡張(stdout JSONをパースしてマージ)
- HTTPサーバーのグレースフルシャットダウン対応
