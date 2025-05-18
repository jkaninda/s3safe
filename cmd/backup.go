/*
MIT License

Copyright (c) 2025 Jonas Kaninda

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package cmd

import (
	"github.com/jkaninda/s3safe/pkg"
	"github.com/jkaninda/s3safe/utils"
	"github.com/spf13/cobra"
	"log/slog"
	"os"
)

var BackupCmd = &cobra.Command{
	Use:     "backup ",
	Short:   "Backup data ",
	Example: utils.BackupExample,
	Run: func(cmd *cobra.Command, args []string) {
		err := pkg.Backup(cmd)
		if err != nil {
			slog.Error("Backup error", "error", err)
			os.Exit(1)
		}
	},
}

func init() {
	// Backup
	BackupCmd.PersistentFlags().BoolP("compress", "c", false, "Enable backup compression")
	BackupCmd.PersistentFlags().BoolP("timestamp", "t", false, "Enable timestamp in backup file name, only for compression")
	BackupCmd.PersistentFlags().StringP("path", "p", "", "Storage path`")
	BackupCmd.PersistentFlags().StringP("dest", "d", "", "S3 destination path`")
	BackupCmd.PersistentFlags().StringP("file", "f", "", "Backup a single file`")
}
