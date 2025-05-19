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

var RestoreCmd = &cobra.Command{
	Use:     "restore ",
	Short:   "Restore data ",
	Example: utils.RestoreExample,
	Run: func(cmd *cobra.Command, args []string) {
		err := pkg.Restore(cmd)
		if err != nil {
			slog.Error("Restore error", "error", err)
			os.Exit(1)
		}
	},
}

func init() {
	// Backup
	RestoreCmd.PersistentFlags().StringP("path", "p", "", "S3 Storage path`")
	RestoreCmd.PersistentFlags().StringP("dest", "d", "", "Destination path`")
	RestoreCmd.PersistentFlags().StringP("file", "f", "", "File to restore`")
	RestoreCmd.PersistentFlags().BoolP("decompress", "", false, "Enable decompression, only for compressed file, when using --file flag")
	RestoreCmd.PersistentFlags().BoolP("ignore-errors", "", false, "Ignore errors when restoring files")

}
