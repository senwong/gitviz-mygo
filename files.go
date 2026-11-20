package main

import (
	"sort"
	"strings"
)

type FileRow struct {
	Path, Name  string
	Depth, File int
	Dir         bool
}
type fileNode struct {
	dirs  map[string]*fileNode
	files []int
}

func buildFileRows(files []FileChange, tree, compact bool, collapsed map[string]bool) []FileRow {
	if !tree {
		rows := make([]FileRow, 0, len(files))
		for i, f := range files {
			rows = append(rows, FileRow{Path: f.Path, Name: f.Path, File: i})
		}
		return rows
	}
	root := &fileNode{dirs: map[string]*fileNode{}}
	for i, f := range files {
		parts := strings.Split(f.Path, "/")
		node := root
		for _, part := range parts[:len(parts)-1] {
			if node.dirs[part] == nil {
				node.dirs[part] = &fileNode{dirs: map[string]*fileNode{}}
			}
			node = node.dirs[part]
		}
		node.files = append(node.files, i)
	}
	var rows []FileRow
	var walk func(*fileNode, string, int)
	walk = func(node *fileNode, prefix string, depth int) {
		names := make([]string, 0, len(node.dirs))
		for name := range node.dirs {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			child := node.dirs[name]
			path := prefix + name
			label := name
			for compact && len(child.dirs) == 1 && len(child.files) == 0 {
				for next, n := range child.dirs {
					label += "/" + next
					path += "/" + next
					child = n
					break
				}
			}
			rows = append(rows, FileRow{Path: path, Name: label, Depth: depth, File: -1, Dir: true})
			if !collapsed[path] {
				walk(child, path+"/", depth+1)
			}
		}
		sort.Slice(node.files, func(i, j int) bool { return files[node.files[i]].Path < files[node.files[j]].Path })
		for _, i := range node.files {
			p := files[i].Path
			name := p
			if index := strings.LastIndex(p, "/"); index >= 0 {
				name = p[index+1:]
			}
			rows = append(rows, FileRow{Path: p, Name: name, Depth: depth, File: i})
		}
	}
	walk(root, "", 0)
	return rows
}
func (a *App) rebuildFileRows() {
	a.fileRows = buildFileRows(a.files, a.config.FileTree, a.config.CompactFolders, a.collapsed)
	a.fileRowSelected = -1
	for i, r := range a.fileRows {
		if !r.Dir && r.File == a.selectedFile {
			a.fileRowSelected = i
			break
		}
	}
}
