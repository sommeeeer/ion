package js

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	esbuild "github.com/evanw/esbuild/pkg/api"
)

type EvalOptions struct {
	Dir    string
	Code   string
	Env    []string
	Banner string
	Inject []string
	Define map[string]string
}

func Build(input EvalOptions) (esbuild.BuildResult, error) {
	outfile := filepath.Join(input.Dir, ".sst", "platform", "sst.config.mjs")
	slog.Info("esbuild building")
	result := esbuild.Build(esbuild.BuildOptions{
		Banner: map[string]string{
			"js": `
import { createRequire as topLevelCreateRequire } from 'module';
const require = topLevelCreateRequire(import.meta.url);
import { fileURLToPath as topLevelFileUrlToPath, URL as topLevelURL } from "url"
const __filename = topLevelFileUrlToPath(import.meta.url)
const __dirname = topLevelFileUrlToPath(new topLevelURL(".", import.meta.url))
` + input.Banner,
		},
		MainFields: []string{"module", "main"},
		Format:     esbuild.FormatESModule,
		Platform:   esbuild.PlatformNode,
		Sourcemap:  esbuild.SourceMapLinked,
		Stdin: &esbuild.StdinOptions{
			Contents:   input.Code,
			ResolveDir: input.Dir,
			Sourcefile: "eval.ts",
			Loader:     esbuild.LoaderTS,
		},
		NodePaths: []string{
			filepath.Join(input.Dir, ".sst", ".platform", "node_modules"),
		},
		External: []string{
			"@pulumi/*",
			"@pulumiverse/*",
			"@sst-provider/*",
			"@aws-sdk/*",
			"esbuild",
			"archiver",
			"glob",
		},
		Define:   input.Define,
		Inject:   input.Inject,
		Outfile:  outfile,
		Write:    true,
		Bundle:   true,
		Metafile: true,
	})
	if len(result.Errors) > 0 {
		for _, err := range result.Errors {
			slog.Error("esbuild error", "text", err.Text)
		}
		return result, fmt.Errorf("%s", FormatError(result.Errors))
	}
	slog.Info("esbuild built", "outfile", outfile)

	return result, nil
}

func FormatError(input []esbuild.Message) string {
	lines := []string{}
	for _, err := range input {
		if err.Location == nil {
			lines = append(lines, fmt.Sprintf("%v", err.Text))
			continue
		}
		lines = append(lines, fmt.Sprintf("%v:%v:%v: %v", err.Location.File, err.Location.Line, err.Location.Column, err.Text))
	}
	return strings.Join(lines, "\n")
}
