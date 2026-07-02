package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type AppConfig struct {
	Rules           []string `json:"rules"`
	SummaryTemplate string   `json:"summary_template"`
	PriorityTags    []string `json:"priority_tags"`
}

// LoadConfig は設定ファイルを読み込みます。ファイルが存在しない場合はデフォルト設定を自動生成して保存します。
func LoadConfig(dir string) (*AppConfig, error) {
	configPath := filepath.Join(dir, "config.json")
	var config AppConfig

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// デフォルトの設定ルール
			config.Rules = []string{
				"このファイルは{year}年{month}月のタスクログです。",
				"時刻はタスク終了を示すこと。",
				"タスク開始は直前のタスクに紐づいている時刻（始業タグまたは前タスクの終了時刻）であること。",
				"昼休みは12:00から13:00。",
				"始業タグがない日は8:00始業とすること。",
				"工数は15分単位（0.25時間単位）に丸めて集計すること。",
				"「[休憩]」または「[私用]」タグが含まれるタスクは、工数集計の対象外とすること。",
			}
			config.SummaryTemplate = "以下のフォーマットで出力してください：\n### 1. タグ別工数集計\n- [タグ名]: XX時間 (全体に対する割合%)\n### 2. 主な業務成果の要約\n- [タグ名]: 業務の要約と成果"
			config.PriorityTags = []string{}

			newData, err := json.MarshalIndent(config, "", "  ")
			if err == nil {
				_ = os.WriteFile(configPath, newData, 0644)
			}
			return &config, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
