package main

import (
	"flag"
	"log"
	"os"

	"github.com/egoist/mygo"
)

func main() {
	screenshot := flag.String("screenshot", "", "Save a loaded native-window PNG, then exit")
	flag.Parse()
	prefs := readPreferences()
	roots := flag.Args()
	if len(roots) == 0 {
		roots = prefs.Roots
	}
	if len(roots) == 0 {
		if cwd, err := os.Getwd(); err == nil {
			home, _ := os.UserHomeDir()
			if cwd != home && cwd != "/" {
				roots = []string{cwd}
			}
		}
	}
	a := newApp(prefs)
	mygo.Theme.SetSource(a.nativeThemeSource())
	a.screenshot = *screenshot
	mygo.App.SetName("Gitviz")
	a.installWindowLifecycle()
	mygo.App.WhenReady(func() {
		a.createMainWindow()
		a.installMenu()
		if len(roots) > 0 {
			a.discoverRoots(roots)
		} else {
			a.scheduleCapture()
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
