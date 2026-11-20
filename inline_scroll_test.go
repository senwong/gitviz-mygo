package main

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestClickingCommitKeepsItsPositionWhenInlineDetailsFit(t *testing.T) {
	a := uiFixture()
	a.prefs.InlineDetails = true
	a.config.ShowDetail = false
	a.detailVisible = false
	a.post = func(func()) {}
	defer func() {
		if a.cancelDetail != nil {
			a.cancelDetail()
		}
	}()
	base := a.snapshot.Commits[0]
	a.snapshot.Commits = nil
	for i := 0; i < 100; i++ {
		c := base
		c.SHA = fmt.Sprintf("commit-%03d", i)
		c.Subject = fmt.Sprintf("History commit %03d", i)
		c.Refs = nil
		a.snapshot.Commits = append(a.snapshot.Commits, c)
	}
	a.rebuildVisible()
	tt := ui.NewTester(a.view, 1380, 900)
	tt.Scroll(500, 400, 0, 480)
	tt.Frame()
	label := "History commit 026"
	before, ok := tt.Find(label)
	if !ok || before.Y < 100 || before.Y > 400 {
		t.Fatalf("test did not start in a scrolled history: %+v", before)
	}
	captureMigrationPreview(t, tt, "before-click")
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	after, ok := tt.Find(label)
	if !ok || after.Y != before.Y {
		t.Fatalf("opening details jumped the clicked row: before=%+v after=%+v", before, after)
	}
	if _, ok := tt.Find("Inline commit details"); !ok {
		t.Fatal("details did not open")
	}
	captureMigrationPreview(t, tt, "after-click")
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	closed, _ := tt.Find(label)
	if a.detailVisible || closed.Y != before.Y {
		t.Fatalf("closing details moved the row: before=%+v closed=%+v", before, closed)
	}

	nearBottom := ""
	for i := 0; i < 100; i++ {
		candidate := fmt.Sprintf("History commit %03d", i)
		if rect, ok := tt.Find(candidate); ok && rect.Y >= 700 && rect.Y+rect.H <= 850 {
			nearBottom = candidate
		}
	}
	if nearBottom == "" {
		t.Fatal("missing visible commit near viewport bottom")
	}
	bottomBefore, ok := tt.Find(nearBottom)
	if !ok || bottomBefore.Y < 700 {
		t.Fatalf("expected a commit near viewport bottom: %+v", bottomBefore)
	}
	if err := tt.Click(nearBottom); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	bottomAfter, _ := tt.Find(nearBottom)
	if bottomAfter.Y != bottomBefore.Y {
		t.Fatalf("opening near viewport bottom scrolled the clicked row: before=%+v after=%+v", bottomBefore, bottomAfter)
	}
	captureMigrationPreview(t, tt, "near-bottom-open")
	if err := tt.Click(nearBottom); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	bottomClosed, _ := tt.Find(nearBottom)
	if a.detailVisible || bottomClosed.Y != bottomBefore.Y {
		t.Fatalf("closing near viewport bottom scrolled: before=%+v after=%+v", bottomBefore, bottomClosed)
	}
}
