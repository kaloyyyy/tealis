package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func countLinesInFile(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	lines := 0
	buf := make([]byte, 1024)
	for {
		n, err := file.Read(buf)
		if err != nil {
			break
		}
		lines += strings.Count(string(buf[:n]), "\n")
	}
	return lines, nil
}

func countLinesInDir(dirPath string) (int, error) {
	totalLines := 0

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only count lines in .go files
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			lines, err := countLinesInFile(path)
			if err != nil {
				log.Printf("Error counting lines in file %s: %v", path, err)
				return nil
			}
			fmt.Printf("path: %s  lines: %d\n", path, lines)
			totalLines += lines
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return totalLines, nil
}

func main() {
	dirPath := "." // Start in the current directory, change if needed

	lines, err := countLinesInDir(dirPath)
	if err != nil {
		log.Fatalf("Error counting lines: %v", err)
	}

	fmt.Printf("Total lines of Go code: %d\n", lines)
}
