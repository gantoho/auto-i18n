package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"auto_i18n/internal/extractor"
	"auto_i18n/internal/generator"
	"auto_i18n/internal/xlsx"
)

var (
	targetLangs string
	outputDir   string
)

var rootCmd = &cobra.Command{
	Use:   "auto-i18n",
	Short: "Auto i18n - 自动化国际化翻译工作流工具",
	Long: `Auto i18n 帮助开发者自动化国际化翻译工作流。

工作流程:
  1. extract  - 从源语言 JSON 提取可翻译文案，生成 xlsx 模板
  2. generate - 从翻译完成的 xlsx 回填生成各语言 JSON 文件`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var extractCmd = &cobra.Command{
	Use:   "extract <json_file>",
	Short: "从 JSON 文件提取可翻译文案，生成 xlsx 翻译模板",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonPath := args[0]

		if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", jsonPath)
		}

		ext := extractor.New(jsonPath)
		result, err := ext.Run()
		if err != nil {
			return fmt.Errorf("extract failed: %w", err)
		}

		if len(result.Entries) == 0 {
			return fmt.Errorf("no translatable content found in %s", jsonPath)
		}

		langs := []string{}
		if targetLangs != "" {
			for _, l := range strings.Split(targetLangs, ",") {
				l = strings.TrimSpace(l)
				if l != "" && l != result.SourceLang {
					langs = append(langs, l)
				}
			}
		}

		xlsxPath := strings.TrimSuffix(jsonPath, filepath.Ext(jsonPath)) + ".xlsx"

		values := make([]string, len(result.Entries))
		for i, e := range result.Entries {
			values[i] = e.Value
		}

		writer := xlsx.NewWriter(xlsxPath)
		if err := writer.Write(result.SourceLang, values, langs); err != nil {
			return fmt.Errorf("write xlsx failed: %w", err)
		}

		fmt.Printf("✓ Extracted %d entries from %s\n", len(result.Entries), filepath.Base(jsonPath))
		if result.SourceLang != "" {
			fmt.Printf("  Source language: %s\n", result.SourceLang)
		}
		fmt.Printf("  Output: %s\n", xlsxPath)

		return nil
	},
}

var generateCmd = &cobra.Command{
	Use:   "generate <xlsx_file>",
	Short: "从翻译完成的 xlsx 生成各语言 JSON 文件",
	Long: `根据翻译完成的 xlsx 文件，为每个目标语言生成对应的 JSON 文件。

程序会：
  1. 自动从同目录寻找原始 JSON 文件
  2. 读取 xlsx 中的翻译内容
  3. 为每个目标语言生成完整 JSON 文件`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		xlsxPath := args[0]

		if _, err := os.Stat(xlsxPath); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", xlsxPath)
		}

		reader := xlsx.NewReader(xlsxPath)
		data, err := reader.Read()
		if err != nil {
			return fmt.Errorf("read xlsx failed: %w", err)
		}

		if len(data.TargetLangs) == 0 {
			return fmt.Errorf("xlsx must have at least 2 language columns")
		}

		sourceLang := data.SourceLang
		jsonPath := deriveJSONPath(xlsxPath, sourceLang)

		if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
			return fmt.Errorf("cannot find original JSON file: %s (looked for %s)", jsonPath, jsonPath)
		}

		outDir := outputDir
		if outDir == "" {
			outDir = filepath.Dir(xlsxPath)
		}

		gen := generator.New(xlsxPath, jsonPath, outDir)
		if err := gen.Run(); err != nil {
			return fmt.Errorf("generate failed: %w", err)
		}

		fmt.Printf("✓ Generated %d languages from %s\n", len(data.TargetLangs), filepath.Base(xlsxPath))
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("auto-i18n v0.1.0")
	},
}

func deriveJSONPath(xlsxPath, sourceLang string) string {
	dir := filepath.Dir(xlsxPath)
	base := strings.TrimSuffix(filepath.Base(xlsxPath), filepath.Ext(xlsxPath))

	candidates := []string{
		filepath.Join(dir, base+"_"+sourceLang+".json"),
		filepath.Join(dir, base+".json"),
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	return candidates[0]
}

func init() {
	extractCmd.Flags().StringVarP(&targetLangs, "target-langs", "t", "",
		"目标语言列表，逗号分隔 (如 zh-CN,ja,ko)")

	generateCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "",
		"JSON 输出目录 (默认与 xlsx 同目录)")

	rootCmd.AddCommand(extractCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
