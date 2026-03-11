// Package cmd 提供命令行入口。
//
// 本文件实现版本打印命令。
package cmd

import (
	"fmt"

	"github.com/cy77cc/OpsPilot/version"
	"github.com/spf13/cobra"
)

// versionCMD 打印应用程序版本。
var (
	versionCMD = &cobra.Command{
		Use:   "version",
		Short: "print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.VERSION)
		},
	}
)

func init() {
	rootCMD.AddCommand(versionCMD)
}
