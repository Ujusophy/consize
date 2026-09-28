package repositorysecurity

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func LoadGitIndex(ctx context.Context, root string) ([]TrackedFile, string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, "", err
	}
	revisionBytes, err := gitOutput(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		return nil, "", err
	}
	revision := strings.TrimSpace(string(revisionBytes))
	if revision == "" {
		return nil, "", fmt.Errorf("git returned an empty revision")
	}

	index, err := gitOutput(ctx, root, "ls-files", "--stage", "-z")
	if err != nil {
		return nil, "", err
	}
	entries := bytes.Split(index, []byte{0})
	files := make([]TrackedFile, 0, len(entries))
	for _, entry := range entries {
		if len(entry) == 0 {
			continue
		}
		metadata, pathBytes, ok := bytes.Cut(entry, []byte{'\t'})
		if !ok {
			return nil, "", fmt.Errorf("invalid git index entry")
		}
		fields := strings.Fields(string(metadata))
		if len(fields) != 3 || fields[2] != "0" {
			return nil, "", fmt.Errorf("unsupported staged git entry for %q", string(pathBytes))
		}
		path := filepath.ToSlash(string(pathBytes))
		if !exactRelativePath(path) {
			return nil, "", fmt.Errorf("git returned unsafe tracked path %q", path)
		}
		sizeBytes, err := gitOutput(ctx, root, "cat-file", "-s", fields[1])
		if err != nil {
			return nil, "", fmt.Errorf("read size for %s: %w", path, err)
		}
		size, err := strconv.ParseInt(strings.TrimSpace(string(sizeBytes)), 10, 64)
		if err != nil || size < 0 {
			return nil, "", fmt.Errorf("invalid blob size for %s", path)
		}
		file := TrackedFile{Path: path, Mode: fields[0], GitHash: fields[1], Size: size}
		if size <= HardMaxBytes {
			content, err := gitOutput(ctx, root, "cat-file", "blob", fields[1])
			if err != nil {
				return nil, "", fmt.Errorf("read blob for %s: %w", path, err)
			}
			file.Content = content
			file.ContentSet = true
			file.SHA256 = digest(content)
		}
		files = append(files, file)
	}
	return files, revision, nil
}

func gitOutput(ctx context.Context, root string, arguments ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, "git", append([]string{"-C", root}, arguments...)...)
	output, err := command.Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return nil, fmt.Errorf("git %s failed: %s", arguments[0], strings.TrimSpace(string(exitError.Stderr)))
		}
		return nil, fmt.Errorf("git %s failed: %w", arguments[0], err)
	}
	return output, nil
}
