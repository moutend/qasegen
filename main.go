package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

type Bulk struct {
	TestCases []*TestCase `json:"cases"`
}

type TestCase struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"-"`
	SuiteId     int64  `json:"suite_id" yaml:"suite_id"`
}

func main() {
	if err := run(); err != nil {
		log.New(os.Stderr, "error: ", 0).Fatal(err)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return nil
	}

	paths, err := getPaths(os.Args[1])

	if err != nil {
		return err
	}

	var bulk Bulk

	for _, path := range paths {
		testCase, err := generateTestCase(path)

		if err != nil {
			return err
		}

		bulk.TestCases = append(bulk.TestCases, testCase)
	}

	outputFile, err := os.Create("bulk.json")

	if err != nil {
		return err
	}

	defer outputFile.Close()

	if err := json.NewEncoder(outputFile).Encode(bulk); err != nil {
		return err
	}

	return nil
}

func getPaths(rootPath string) (paths []string, err error) {
	err = filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, e error) error {
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		paths = append(paths, path)

		return nil
	})

	if err != nil {
		return nil, err
	}

	slices.Sort(paths)

	return paths, nil
}

func generateTestCase(path string) (*TestCase, error) {
	file, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	lines := []string{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(lines) < 4 {
		return nil, fmt.Errorf("file might be broken: %s", path)
	}

	endOfFrontMatter := 0

	for i, line := range lines[1:] {
		if line == "---" {
			endOfFrontMatter = i + 1

			break
		}
	}
	if endOfFrontMatter == 0 {
		return nil, fmt.Errorf("failed to find end of front matter")
	}

	var testCase TestCase

	frontMatter := bytes.NewBufferString(strings.Join(lines[1:endOfFrontMatter], "\n"))

	if err := yaml.NewDecoder(frontMatter).Decode(&testCase); err != nil {
		return nil, err
	}

	testCase.Description = strings.Join(lines[endOfFrontMatter+1:], "\n")

	return &testCase, nil
}
