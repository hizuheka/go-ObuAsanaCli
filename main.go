package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config は設定ファイルの構造体です
type Config struct {
	PersonalAccessToken string            `yaml:"personal_access_token"`
	WorkspaceID         string            `yaml:"workspace_id"`
	ProjectID           string            `yaml:"project_id"`
	Assignees           map[string]string `yaml:"assignees"` // 追加: 担当者のショートカットとGIDのマッピング
}

// TaskRequest はAsana APIへ送信するJSONリクエストの構造体です
type TaskRequest struct {
	Data TaskData `json:"data"`
}

// TaskData はリクエスト内の実際のデータ部分です
type TaskData struct {
	Name      string   `json:"name"`
	Workspace string   `json:"workspace"`
	Projects  []string `json:"projects"`
	Notes     string   `json:"notes,omitempty"`
	Assignee  string   `json:"assignee,omitempty"` // 追加: 担当者(GID)
	DueOn     string   `json:"due_on,omitempty"`
}

// TaskResponse はAsana APIからのレスポンスを受け取る構造体です
type TaskResponse struct {
	Data struct {
		GID          string `json:"gid"`
		PermalinkURL string `json:"permalink_url"`
	} `json:"data"`
}

func main() {
	// 1. 設定ファイルのパスを決定
	configDir, err := os.UserConfigDir() // Windowsでは %AppData% になります
	if err != nil {
		fmt.Printf("エラー: ユーザー設定ディレクトリの取得に失敗しました: %v\n", err)
		os.Exit(1)
	}
	appDir := filepath.Join(configDir, "asana-cli")
	configPath := filepath.Join(appDir, "config.yaml")

	// 2. 設定ファイルの読み込みと存在チェック
	cfg, err := loadConfig(configPath, appDir)
	if err != nil {
		fmt.Printf("エラー: 設定ファイルの処理中に問題が発生しました: %v\n", err)
		os.Exit(1)
	}

	// 設定値の簡易チェック（PATなどが空でないか）
	if cfg.PersonalAccessToken == "" || cfg.WorkspaceID == "" || cfg.ProjectID == "" {
		fmt.Printf("エラー: 設定ファイル(%s)の必須項目が未入力です。\n", configPath)
		os.Exit(1)
	}

	// Scannerの準備 (標準入力から受け取る)
	scanner := bufio.NewScanner(os.Stdin)

	// 3. タスク名の入力 (必須)
	name := promptInput(scanner, "タスク名を入力してください: ", true)

	// 4. タスクの説明の入力 (任意)
	notes := promptInput(scanner, "タスクの説明を入力してください (省略可): ", false)

	// 5. 担当者の入力 (任意・ショートカット変換あり)
	var assigneeGID string
	for {
		assigneeInput := promptInput(scanner, "担当者を入力してください (me / 設定名 / 省略可): ", false)
		if assigneeInput == "" {
			break // 省略時
		}

		// 設定ファイルのマップからGIDを検索
		if gid, ok := cfg.Assignees[assigneeInput]; ok {
			assigneeGID = gid
			break
		} else {
			fmt.Printf("エラー: '%s' は設定ファイルの assignees に登録されていません。\n", assigneeInput)
			// 再入力させる
		}
	}

	// 6. 期日の入力 (任意・today変換・形式チェックあり)
	var dueOn string
	for {
		inputDue := promptInput(scanner, "期日を入力してください (today / YYYY-MM-DD / 省略可): ", false)

		if inputDue == "" {
			break // 省略時
		}

		// 'today' ショートカットの処理
		if strings.ToLower(inputDue) == "today" {
			dueOn = time.Now().Format("2006-01-02")
			fmt.Printf("  -> 期日を %s に設定しました\n", dueOn)
			break
		}

		// 日付形式(YYYY-MM-DD)のバリデーション
		_, err := time.Parse("2006-01-02", inputDue)
		if err != nil {
			fmt.Println("エラー: 日付の形式が正しくありません。today または YYYY-MM-DD の形式で入力してください。")
			continue
		}
		dueOn = inputDue
		break
	}

	// 7. Asana APIへのリクエスト実行
	fmt.Println("\nタスクを登録しています...")
	taskData := TaskData{
		Name:      name,
		Workspace: cfg.WorkspaceID,
		Projects:  []string{cfg.ProjectID},
	}
	if notes != "" {
		taskData.Notes = notes
	}
	if assigneeGID != "" {
		taskData.Assignee = assigneeGID
	}
	if dueOn != "" {
		taskData.DueOn = dueOn
	}

	reqBody := TaskRequest{Data: taskData}
	createTask(cfg.PersonalAccessToken, reqBody)
}

// loadConfig は設定ファイルを読み込みます。無ければ作成フローに移行します。
func loadConfig(configPath, appDir string) (*Config, error) {
	// ファイルが存在するか確認
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("設定ファイルが見つかりません: %s\n", configPath)
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("新しい設定ファイルの雛形を作成しますか？ (Y/n): ")
		scanner.Scan()
		ans := strings.TrimSpace(strings.ToLower(scanner.Text()))

		if ans == "y" || ans == "yes" || ans == "" { // デフォルトYESとする
			if err := createConfigTemplate(configPath, appDir); err != nil {
				return nil, fmt.Errorf("雛形の作成に失敗しました: %w", err)
			}
			fmt.Printf("\n設定ファイルの雛形を作成しました。\n")
			fmt.Printf("お手数ですが、エディタで %s を開き、PATやID、担当者のGIDを入力して再度実行してください。\n", configPath)
			os.Exit(0) // 雛形作成後は終了する
		} else {
			fmt.Println("設定ファイルが存在しないため、ツールを終了します。")
			os.Exit(1)
		}
	}

	// ファイルを読み込む
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("ファイルの読み込みに失敗: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("YAMLのパースに失敗: %w", err)
	}

	// Assigneesマップがnilの場合に初期化しておく（設定ファイルに記載がない場合への備え）
	if cfg.Assignees == nil {
		cfg.Assignees = make(map[string]string)
	}

	return &cfg, nil
}

// createConfigTemplate は設定ファイルの雛形を生成します
func createConfigTemplate(configPath, appDir string) error {
	// ディレクトリが存在しない場合は作成
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return err
	}

	// 担当者の雛形を含めた構造体
	template := Config{
		PersonalAccessToken: "",
		WorkspaceID:         "",
		ProjectID:           "",
		Assignees: map[string]string{
			"me":   "YOUR_GID_HERE",
			"john": "JOHNS_GID_HERE",
		},
	}

	data, err := yaml.Marshal(&template)
	if err != nil {
		return err
	}

	// コメント付きで書き込むために調整
	content := []byte(
		"# Asana API Configuration\n" +
			"# Personal Access Token: https://app.asana.com/0/developer-console から取得\n" +
			"# Assignees: ショートカット名とAsanaのユーザーGIDを紐付けます\n" +
			string(data),
	)

	return os.WriteFile(configPath, content, 0644)
}

// promptInput はCUIプロンプトを表示し、ユーザーの入力を受け取ります
func promptInput(scanner *bufio.Scanner, message string, required bool) string {
	for {
		fmt.Print(message)
		scanner.Scan()
		input := strings.TrimSpace(scanner.Text())

		if required && input == "" {
			fmt.Println("エラー: この項目は入力必須です。")
			continue
		}
		return input
	}
}

// createTask はAsana APIにPOSTリクエストを送信してタスクを作成します
func createTask(pat string, taskReq TaskRequest) {
	apiURL := "https://app.asana.com/api/1.0/tasks"

	// 構造体をJSONにエンコード
	jsonData, err := json.Marshal(taskReq)
	if err != nil {
		fmt.Printf("エラー: リクエストデータのJSONエンコードに失敗しました: %v\n", err)
		os.Exit(1)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("エラー: リクエストの作成に失敗しました: %v\n", err)
		os.Exit(1)
	}

	// ヘッダーの設定
	req.Header.Set("Authorization", "Bearer "+pat)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// リクエストの送信
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("エラー: Asana APIへの接続に失敗しました: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// レスポンスの読み込み
	body, _ := io.ReadAll(resp.Body)

	// ステータスコードのチェック
	if resp.StatusCode != http.StatusCreated {
		fmt.Printf("エラー: タスクの作成に失敗しました。\nステータスコード: %d\nレスポンス: %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	// 成功時のレスポンス解析
	var taskResp TaskResponse
	if err := json.Unmarshal(body, &taskResp); err != nil {
		// API側で作成自体は成功しているが解析に失敗した場合
		fmt.Printf("タスクは作成されましたが、レスポンスの解析に失敗しました: %v\n", err)
		return
	}

	fmt.Println("\n✅ タスクの作成に成功しました！")
	fmt.Printf("タスクURL: %s\n", taskResp.Data.PermalinkURL)
}
