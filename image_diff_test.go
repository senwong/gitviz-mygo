package main

import (
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func writeTestImage(t *testing.T, repo, path string, w, h int) {
	t.Helper()
	f, err := os.Create(filepath.Join(repo, path))
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{uint8(x * 255 / w), uint8(y * 255 / h), 120, 255})
		}
	}
	if strings.EqualFold(filepath.Ext(path), ".jpg") {
		err = jpeg.Encode(f, img, nil)
	} else {
		err = png.Encode(f, img)
	}
	closeErr := f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}

func TestImageDiffUsesHistoricalBlobsIncludingRootRenameAndDeletion(t *testing.T) {
	repo := testRepo(t)
	path := "hero [1].png"
	writeTestImage(t, repo, path, 120, 80)
	testGit(t, repo, "add", "--", path)
	testGit(t, repo, "commit", "-m", "add image")
	added := testGit(t, repo, "rev-parse", "HEAD")
	added = strings.TrimSpace(added)
	writeTestImage(t, repo, path, 160, 90)
	testGit(t, repo, "add", "--", path)
	testGit(t, repo, "commit", "-m", "modify image")
	modified := strings.TrimSpace(testGit(t, repo, "rev-parse", "HEAD"))
	// Today's working image must never replace either historical side.
	writeTestImage(t, repo, path, 12, 8)
	root := loadImageDiff(context.Background(), repo, added, "", FileChange{Path: path, Status: "A"})
	if root.Before.Present || root.After.Bitmap == nil || root.After.Width != 120 {
		t.Fatalf("root image: %+v", root)
	}
	changed := loadImageDiff(context.Background(), repo, modified, "", FileChange{Path: path, Status: "M"})
	if changed.Before.Bitmap == nil || changed.After.Bitmap == nil || changed.Before.Width != 120 || changed.After.Width != 160 {
		t.Fatalf("historical comparison: %+v", changed)
	}
	comparison := loadImageDiff(context.Background(), repo, modified, added, FileChange{Path: path, Status: "M"})
	if comparison.Before.Width != 120 || comparison.After.Width != 160 {
		t.Fatal("explicit comparison uses wrong revisions")
	}
	testGit(t, repo, "restore", "--", path)
	testGit(t, repo, "mv", "--", path, "renamed.png")
	testGit(t, repo, "commit", "-m", "rename image")
	rename := strings.TrimSpace(testGit(t, repo, "rev-parse", "HEAD"))
	renamed := loadImageDiff(context.Background(), repo, rename, "", FileChange{Path: "renamed.png", OldPath: path, Status: "R"})
	if renamed.Before.Width != 160 || renamed.After.Width != 160 {
		t.Fatalf("rename failed: %+v", renamed)
	}
	testGit(t, repo, "rm", "renamed.png")
	testGit(t, repo, "commit", "-m", "delete image")
	deleted := loadImageDiff(context.Background(), repo, "HEAD", "", FileChange{Path: "renamed.png", Status: "D"})
	if deleted.Before.Bitmap == nil || deleted.After.Present {
		t.Fatalf("deletion failed: %+v", deleted)
	}
	for _, tc := range []struct {
		name    string
		preview *imageFileDiff
	}{{"added", root}, {"modified", changed}, {"deleted", deleted}} {
		a := diffFindFixture()
		a.files = []FileChange{{Path: path, Status: "M"}}
		a.selectedFile = 0
		a.rebuildFileRows()
		a.imageDiff = tc.preview
		tt := ui.NewTester(a.view, 1000, 620)
		if !tt.HasText("Before") || !tt.HasText("After") {
			t.Fatal("missing before/after headings")
		}
		beforeRect, _ := tt.Find("Before")
		afterRect, _ := tt.Find("After")
		if beforeRect.Y != afterRect.Y {
			t.Fatalf("image headers do not align: %+v %+v", beforeRect, afterRect)
		}
		captureMigrationPreview(t, tt, "image-diff-"+tc.name)
		if err := tt.Click("Show text diff"); err != nil {
			t.Fatal(err)
		}
		if !a.imageDiffText || !tt.HasText("Show image diff") {
			t.Fatal("raw diff toggle failed")
		}
		if err := tt.Click("Show image diff"); err != nil {
			t.Fatal(err)
		}
		tt.Key(ui.Cmd, ui.KeyF)
		if !a.imageDiffText || !a.showSearch || !a.searchDiff {
			t.Fatal("image text search did not reveal raw diff")
		}
		if err := tt.Click("Show image diff"); err != nil {
			t.Fatal(err)
		}
		if a.showSearch {
			t.Fatal("image preview retained a text search with invisible results")
		}
	}
}

func TestImageDiffWorkingTreeUntrackedAndInvalidImages(t *testing.T) {
	repo := testRepo(t)
	writeTestImage(t, repo, "hero.png", 16, 12)
	testGit(t, repo, "add", "hero.png")
	testGit(t, repo, "commit", "-m", "image")
	writeTestImage(t, repo, "hero.png", 32, 24)
	preview := loadImageDiff(context.Background(), repo, "worktree", "", FileChange{Path: "hero.png", Status: "M"})
	if preview.Before.Width != 16 || preview.After.Width != 32 {
		t.Fatalf("worktree sides: %+v", preview)
	}
	writeTestImage(t, repo, "photo.jpg", 48, 36)
	testGit(t, repo, "add", "photo.jpg")
	testGit(t, repo, "commit", "-m", "add JPEG")
	jpegPreview := loadImageDiff(context.Background(), repo, "HEAD", "", FileChange{Path: "photo.jpg", Status: "A"})
	if jpegPreview.Before.Present || jpegPreview.After.Bitmap == nil || jpegPreview.After.Width != 48 {
		t.Fatalf("JPEG preview failed: %+v", jpegPreview)
	}
	writeTestImage(t, repo, "new.png", 24, 18)
	preview = loadImageDiff(context.Background(), repo, "worktree", "", FileChange{Path: "new.png", Untracked: true})
	if preview.Before.Present || preview.After.Width != 24 {
		t.Fatalf("untracked sides: %+v", preview)
	}
	if err := os.WriteFile(filepath.Join(repo, "bad.png"), []byte("not an image"), 0600); err != nil {
		t.Fatal(err)
	}
	preview = loadImageDiff(context.Background(), repo, "worktree", "", FileChange{Path: "bad.png", Untracked: true})
	if preview.After.Error == "" || preview.After.Bitmap != nil {
		t.Fatal("invalid image did not show fallback")
	}
	if loadImageDiff(context.Background(), repo, "HEAD", "", FileChange{Path: "code.go"}) != nil {
		t.Fatal("text file was treated as an image")
	}
	h := newActionHarness(t, repo)
	h.a.revision = "worktree"
	h.a.files = []FileChange{{Path: "hero.png", Status: "M"}, {Path: "file.txt", Untracked: true}}
	if err := os.WriteFile(filepath.Join(repo, "file.txt"), []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	h.a.selectFile(0)
	h.drain()
	if h.a.imageDiff == nil || h.a.imageDiff.After.Width != 32 {
		t.Fatal("selectFile did not load preview")
	}
	h.a.selectFile(1)
	h.drain()
	if h.a.imageDiff != nil {
		t.Fatal("image leaked into text diff")
	}
}
