package main

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestCommitMetadataTextIsHorizontallyCenteredWithinEachColumn(t *testing.T) {
	a := uiFixture()
	tt := ui.NewTester(a.view, 1380, 900)
	frame := tt.Image()
	color := a.uiTheme.Text
	for _, label := range []string{a.dateText(a.snapshot.Commits[0]), "Ada", shortSHA(a.snapshot.Commits[0].SHA)} {
		rect, ok := tt.Find(label)
		if !ok {
			t.Fatalf("missing column text %s", label)
		}
		left, right := int(rect.X+rect.W), int(rect.X)-1
		for y := int(rect.Y); y < int(rect.Y+rect.H); y++ {
			for x := int(rect.X); x < int(rect.X+rect.W); x++ {
				p := frame.RGBAAt(x, y)
				if abs(int(p.R)-int(color.R)) < 20 && abs(int(p.G)-int(color.G)) < 20 && abs(int(p.B)-int(color.B)) < 20 {
					left = min(left, x)
					right = max(right, x)
				}
			}
		}
		if right < left {
			t.Fatalf("column text %s was not painted", label)
		}
		center := float64(left+right+1) / 2
		want := float64(rect.X + rect.W/2)
		if math.Abs(center-want) > 2 {
			t.Fatalf("%s is not centered: ink center=%.2f column center=%.2f", label, center, want)
		}
	}
	captureMigrationPreview(t, tt, "centered-metadata-columns")
}

func TestCommitMetadataHasBalancedVerticalSpace(t *testing.T) {
	a := uiFixture()
	a.snapshot.Commits[0].Author = "Yong Liu"
	a.snapshot.Commits[0].SHA = "b2d74e3d12345678"
	a.selected = 0
	a.listSelected = 0
	tt := ui.NewTester(a.view, 1380, 900)
	frame := tt.Image()
	selected := a.listTheme.Accent
	rowTop, rowBottom := 900, -1
	for y := 54; y < 180; y++ {
		pixel := frame.RGBAAt(980, y)
		if pixel.R == selected.R && pixel.G == selected.G && pixel.B == selected.B {
			rowTop = min(rowTop, y)
			rowBottom = max(rowBottom, y)
		}
	}
	if rowBottom-rowTop+1 < 28 {
		t.Fatalf("commit row did not gain breathing room: top=%d bottom=%d", rowTop, rowBottom)
	}
	for _, label := range []string{a.dateText(a.snapshot.Commits[0]), "Yong Liu", shortSHA(a.snapshot.Commits[0].SHA)} {
		rect, ok := tt.Find(label)
		if !ok {
			t.Fatal("missing metadata")
		}
		inkTop, inkBottom := 900, -1
		color := a.uiTheme.Text
		for y := int(rect.Y); y < int(rect.Y+rect.H); y++ {
			for x := int(rect.X); x < int(rect.X+rect.W); x++ {
				pixel := frame.RGBAAt(x, y)
				if abs(int(pixel.R)-int(color.R)) < 20 && abs(int(pixel.G)-int(color.G)) < 20 && abs(int(pixel.B)-int(color.B)) < 20 {
					inkTop = min(inkTop, y)
					inkBottom = max(inkBottom, y)
				}
			}
		}
		above, below := inkTop-rowTop, rowBottom-inkBottom
		t.Logf("%s: top gap=%d bottom gap=%d", label, above, below)
		if inkBottom < inkTop || above < 4 || below < 6 || math.Abs(float64(above-below)) > 4 {
			t.Fatalf("unbalanced metadata spacing for %s: top=%d bottom=%d", label, above, below)
		}
	}
	captureMigrationPreview(t, tt, "balanced-row-spacing")
}
