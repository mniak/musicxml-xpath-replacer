package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/beevik/etree"
	"github.com/spf13/cobra"
	"go.uber.org/multierr"
)

type XMLReplacement struct {
	Path     string
	Pattern  *regexp.Regexp
	NewValue string
}

type MusicXMLProcessor struct {
	Replacements []XMLReplacement
}

var rootCmd = cobra.Command{
	Use:  "xr <directory>",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		directory := args[0]

		mxp := MusicXMLProcessor{
			Replacements: []XMLReplacement{
				{
					Path:     "credit/credit-words",
					Pattern:  regexp.MustCompile(`Arr\.:`),
					NewValue: "Arranjo:",
				},
				{
					Path:     "credit/credit-words",
					Pattern:  regexp.MustCompile(`!!(20\d+)!!! -`),
					NewValue: "$1.",
				},
			},
		}

		err := ForeachFile(directory, func(reader io.Reader, writer io.Writer) error {
			return mxp.Process(reader, writer)
		})
		if err != nil {
			log.Fatalln(err)
		}
	},
}

func main() {
	rootCmd.Execute()
}

func ForeachFile(directory string, fn func(reader io.Reader, writer io.Writer) error) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}
	err = os.Mkdir(filepath.Join(directory, "output"), 0o755)
	if os.IsExist(err) {
		// directory already exists, fine
		err = nil
	} else if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		inputPath := filepath.Join(directory, entry.Name())
		outputPath := filepath.Join(directory, "output", entry.Name())
		fmt.Printf("Processing %s... ", entry.Name())
		err2 := func() error {
			inputFile, err := os.Open(inputPath)
			if err != nil {
				return err
			}
			defer inputFile.Close()

			var outputBuffer bytes.Buffer
			if err = fn(inputFile, &outputBuffer); err != nil {
				return err
			}
			outputFile, err := os.Create(outputPath)
			if err != nil {
				return err
			}
			defer outputFile.Close()
			_, err = io.Copy(outputFile, &outputBuffer)
			return err
		}()
		if err2 == nil {
			fmt.Println("Done!")
		} else {
			fmt.Printf("Error: %s\n", err2.Error())
		}
		err = multierr.Append(err, err2)
		fmt.Println(entry.Name())
	}
	return err
}

func (m *MusicXMLProcessor) Process(reader io.Reader, writer io.Writer) error {
	doc := etree.NewDocument()
	if _, err := doc.ReadFrom(reader); err != nil {
		return err
	}

	root := doc.Root()

	for _, repl := range m.Replacements {
		for _, element := range root.FindElements(repl.Path) {

			newText := repl.Pattern.ReplaceAllString(element.Text(), repl.NewValue)
			element.SetText(newText)
		}
	}

	_, err := doc.WriteTo(writer)
	return err
}
