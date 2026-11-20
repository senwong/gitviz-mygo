package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
	_ "golang.org/x/image/tiff"
)

const maxImageDiffBytes = 32 << 20

type imageDiffSide struct {
	Bitmap               *ui.Bitmap
	Width, Height, Bytes int
	Present              bool
	Error                string
}
type imageFileDiff struct{ Before, After imageDiffSide }

func isPreviewImage(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".tif", ".tiff":
		return true
	}
	return false
}

// Read the blob at the selected revision, never today's working copy of a commit.
func imageBlob(ctx context.Context, repo, revision, path string) ([]byte, error) {
	if !filepath.IsLocal(path) {
		return nil, fmt.Errorf("invalid repository-relative image path")
	}
	if revision == "" {
		return nil, nil
	}
	if revision == "worktree" {
		full := filepath.Join(repo, path)
		info, err := os.Lstat(full)
		if os.IsNotExist(err) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("not a regular image file")
		}
		// Resolve parent directories too, so repository symlinks cannot escape the repository.
		resolved, err := filepath.EvalSymlinks(full)
		if err != nil {
			return nil, err
		}
		root, err := filepath.EvalSymlinks(repo)
		if err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(root, resolved)
		if err != nil || !filepath.IsLocal(rel) {
			return nil, fmt.Errorf("image path is outside the repository")
		}
		if info.Size() > maxImageDiffBytes {
			return nil, fmt.Errorf("image exceeds the 32 MB preview limit")
		}
		f, err := os.Open(resolved)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return io.ReadAll(io.LimitReader(f, maxImageDiffBytes+1))
	}
	listing, err := gitCommand(ctx, repo, "--literal-pathspecs", "ls-tree", "-z", revision, "--", path)
	if err != nil {
		return nil, err
	}
	if listing == "" {
		return nil, nil
	}
	fields := strings.Fields(strings.SplitN(listing, "\t", 2)[0])
	if len(fields) != 3 || fields[1] != "blob" || (fields[0] != "100644" && fields[0] != "100755") {
		return nil, fmt.Errorf("not a regular image blob")
	}
	size, err := gitCommand(ctx, repo, "cat-file", "-s", fields[2])
	if err != nil {
		return nil, err
	}
	var n int64
	if _, err = fmt.Sscan(size, &n); err != nil {
		return nil, err
	}
	if n > maxImageDiffBytes {
		return nil, fmt.Errorf("image exceeds the 32 MB preview limit")
	}
	data, err := gitCommand(ctx, repo, "cat-file", "blob", fields[2])
	return []byte(data), err
}

func decodeImageSide(data []byte, err error) imageDiffSide {
	s := imageDiffSide{Present: data != nil || err != nil, Bytes: len(data)}
	if err != nil {
		s.Error = err.Error()
		return s
	}
	if data == nil {
		return s
	}
	if len(data) > maxImageDiffBytes {
		s.Error = "Image exceeds the 32 MB preview limit"
		return s
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		s.Error = "This image cannot be previewed (unsupported, damaged, or a Git LFS pointer)."
		return s
	}
	s.Width, s.Height = cfg.Width, cfg.Height
	if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 40_000_000 {
		s.Error = "Image exceeds the 40 megapixel preview limit"
		return s
	}
	s.Bitmap, err = ui.DecodeBitmap(data)
	if err != nil {
		s.Error = "Unable to decode this image: " + err.Error()
	} else {
		s.Width, s.Height = s.Bitmap.Size()
	}
	return s
}

func loadImageDiff(ctx context.Context, repo, revision, compare string, file FileChange) *imageFileDiff {
	if !isPreviewImage(file.Path) && !isPreviewImage(file.OldPath) {
		return nil
	}
	before, after := compare, revision
	if revision == "worktree" {
		after = "worktree"
		if before == "" {
			if _, err := gitCommand(ctx, repo, "rev-parse", "--verify", "HEAD"); err == nil {
				before = "HEAD"
			}
		}
		if file.Untracked {
			before = ""
		}
	} else if before == "" {
		parents, err := gitCommand(ctx, repo, "rev-list", "--parents", "-n", "1", revision)
		if err != nil {
			return &imageFileDiff{Before: decodeImageSide(nil, err), After: decodeImageSide(nil, err)}
		}
		if ids := strings.Fields(parents); len(ids) > 1 {
			before = ids[1]
		}
	}
	oldPath := file.Path
	if file.OldPath != "" {
		oldPath = file.OldPath
	}
	b, err := imageBlob(ctx, repo, before, oldPath)
	result := &imageFileDiff{Before: decodeImageSide(b, err)}
	b, err = imageBlob(ctx, repo, after, file.Path)
	result.After = decodeImageSide(b, err)
	return result
}

func (a *App) imageDiffView(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().MinHeight(0).AlignItems(ui.Stretch).Gap(1).Background(t.Border).Label("Image comparison").Children(func() {
		sides := []struct {
			title string
			side  imageDiffSide
		}{{"Before", a.imageDiff.Before}, {"After", a.imageDiff.After}}
		for _, item := range sides {
			ui.Column(c).FillHeight().Grow(1).Basis(0).MinWidth(0).MinHeight(0).Background(t.Background).Children(func() {
				ui.Row(c).FillWidth().Height(32).Shrink(0).PaddingX(12).Gap(8).AlignItems(ui.Center).Background(t.Surface).Children(func() {
					ui.Text(c, item.title).FontSize(12).Bold()
					if item.side.Bitmap != nil {
						ui.Textf(c, "%d × %d · %.1f KB", item.side.Width, item.side.Height, float64(item.side.Bytes)/1024).FontSize(11).TextColor(t.TextMuted)
					}
				})
				ui.Box(c).Fill().MinHeight(0).Padding(16).Children(func() {
					switch {
					case item.side.Bitmap != nil:
						ui.Image(c, item.side.Bitmap).Fill().Fit(ui.Contain).Label(item.title + " image")
					case item.side.Error != "":
						ui.Column(c).Fill().Center().Padding(12).Children(func() { ui.Text(c, item.side.Error).FontSize(12).TextColor(t.TextMuted).Selectable() })
					default:
						message := "File did not exist"
						if item.title == "After" {
							message = "File deleted"
						}
						ui.Column(c).Fill().Center().Children(func() { ui.Text(c, message).FontSize(12).TextColor(t.TextMuted) })
					}
				})
			})
		}
	})
}
