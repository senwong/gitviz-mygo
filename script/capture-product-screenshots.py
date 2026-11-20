#!/usr/bin/env python3
"""Render the real app using a disposable, synthetic Git repository."""
from pathlib import Path
import datetime, os, shutil, subprocess, tempfile

root=Path(__file__).resolve().parent.parent
output=root/'docs/assets'
output.mkdir(parents=True,exist_ok=True)
with tempfile.TemporaryDirectory(prefix='gitviz-product-') as scratch:
    scratch=Path(scratch)
    source=scratch/'source'
    source.mkdir()
    for path in root.glob('*.go'):
        if not path.name.endswith('_test.go') and path.name!='main.go':
            shutil.copy2(path,source/path.name)
    for name in ['go.mod','go.sum']:
        shutil.copy2(root/name,source/name)
    shutil.copy2(root/'script/capture-product-screenshots.go.txt',source/'main.go')
    repo=scratch/'gitviz-demo'
    repo.mkdir()
    env={**os.environ,'GIT_CONFIG_GLOBAL':os.devnull,'GIT_AUTHOR_NAME':'Gitviz Demo','GIT_AUTHOR_EMAIL':'demo@example.com','GIT_COMMITTER_NAME':'Gitviz Demo','GIT_COMMITTER_EMAIL':'demo@example.com','GIT_TERMINAL_PROMPT':'0'}
    step=0
    def git(*args):
        return subprocess.check_output(['git','-C',str(repo),*args],env=env,text=True).strip()
    def tick():
        global step
        step+=1
        date=(datetime.datetime(2026,11,1,9,0,tzinfo=datetime.timezone.utc)+datetime.timedelta(minutes=step*25)).isoformat()
        env.update(GIT_AUTHOR_DATE=date,GIT_COMMITTER_DATE=date)
    def commit(message,files):
        for name,content in files.items():
            path=repo/name
            path.parent.mkdir(parents=True,exist_ok=True)
            path.write_text(content)
        git('add','--all');tick();git('-c','commit.gpgsign=false','commit','-qm',message)
    def merge(branch,message):
        tick();git('-c','commit.gpgsign=false','merge','--no-ff',branch,'-m',message)
    git('init','-q','-b','main')
    commit('Start a native Git graph viewer',{'README.md':'# Gitviz Demo\n\nA synthetic repository for product screenshots.\n','src/app.go':'package main\n\nfunc main() {\n    openGraph()\n}\n'})
    commit('Lay out the commit graph',{'src/graph.go':'package main\n\nconst rowHeight = 28\n'})
    git('switch','-qc','feature/search')
    commit('Find commits by message, author and SHA',{'src/search.go':'package main\n\nimport "strings"\n\nfunc matches(message, query string) bool {\n    return strings.Contains(message, query)\n}\n'})
    commit('Highlight matching text in references',{'src/highlight.go':'package main\n\nfunc highlight(text, query string) []Span {\n    return findSpans(text, query)\n}\n'})
    git('switch','-q','main')
    commit('Restore the last opened repository',{'src/workspace.go':'package main\n\nfunc restoreWorkspace() {\n    openRepository(preferences.LastRepository)\n}\n'})
    commit('Show the current branch in the window title',{'src/title.go':'package main\n\nfunc title(repo, branch string) string {\n    return repo + " · " + branch + " — Gitviz"\n}\n'})
    merge('feature/search','Merge search and reference highlights')
    git('tag','v0.1.0-preview')
    git('switch','-qc','feature/image-diff')
    commit('Load image versions from Git blobs',{'src/image_diff.go':'package main\n\nfunc imageVersions(commit, path string) (Image, Image) {\n    return readImage(commit + "^", path), readImage(commit, path)\n}\n'})
    commit('Compare images side by side',{'src/image_view.go':'package main\n\nfunc showImages(before, after Image) {\n    imagePanel("Before", before)\n    imagePanel("After", after)\n}\n'})
    git('switch','-q','main')
    commit('Add branch actions to the context menu',{'src/menu.go':'package main\n\nvar branchActions = []string{"Checkout", "Merge", "Rebase"}\n'})
    commit('Keep commit details inside the graph',{'src/details.go':'package main\n\nfunc showDetails(commit Commit) {\n    inlineDetails(commit)\n}\n'})
    merge('feature/image-diff','Merge image comparison')
    git('tag','image-preview')
    git('switch','-qc','feature/quick-open')
    commit('Filter repositories by name',{'src/quick_open.go':'package main\n\nfunc repositoryMatches(name, query string) bool {\n    return matches(name, query)\n}\n'})
    commit('Anchor the repository picker to the toolbar',{'src/picker.go':'package main\n\nfunc showRepositories() {\n    anchoredPicker(repositoryButton)\n}\n'})
    commit('Show keyboard shortcuts in the toolbar',{'src/shortcuts.go':'package main\n\nvar shortcuts = map[string]string{"Find": "⌘F", "Repositories": "⌘P"}\n'})
    git('switch','-q','main')
    commit('Check remote branches and tags in the background',{'src/remote.go':'package main\n\nfunc checkRemote(repo string) Status {\n    return compareRefs(localRefs(repo), remoteRefs(repo))\n}\n'})
    commit('Keep the graph steady when searching',{'src/filter.go':'package main\n\nfunc searchGraph(query string) {\n    highlightMatches(query)\n}\n'})
    commit('Copy selected text across diff lines',{'src/selection.go':'package main\n\nfunc diffSelection() {\n    selectableDiffContent()\n}\n'})
    commit('Return to the graph with Command W',{'src/navigation.go':'package main\n\nfunc closeDiff() {\n    showGraph()\n}\n'})
    commit('Make the current branch easier to spot',{'src/references.go':'package main\n\nfunc referenceLabel(ref Reference) {\n    drawReference(ref, ref.Current)\n}\n'})
    commit('Preserve spaces and newlines when copying',{'src/copy.go':'package main\n\nfunc copyLines(lines []string) {\n    clipboard.Write(strings.Join(lines, "\\n"))\n}\n'})
    commit('Refine the native application icon',{'assets/icon.txt':'A continuous mainline with parallel branch lanes.\n'})
    commit('Improve contrast across light and dark themes',{'src/theme.go':'package main\n\nconst minimumContrast = 4.5\n'})
    commit('Keep search highlights consistent across branches',{
        'src/search.go':'package main\n\nimport "strings"\n\n// Search is case-insensitive for commit messages and references.\nfunc matches(message, query string) bool {\n    query = strings.TrimSpace(query)\n    if query == "" {\n        return true\n    }\n    return strings.Contains(strings.ToLower(message), strings.ToLower(query))\n}\n\nfunc findCommit(commit Commit, query string) bool {\n    if matches(commit.Subject, query) {\n        return true\n    }\n    for _, ref := range commit.Refs {\n        if matches(ref.Name, query) {\n            return true\n        }\n    }\n    return false\n}\n',
        'src/highlight.go':'package main\n\nfunc highlight(text, query string) []Span {\n    if query == "" {\n        return []Span{{Text: text}}\n    }\n    return findSpans(text, query)\n}\n'
    })
    git('remote','add','origin','https://github.com/senwong/gitviz-mygo.git')
    git('update-ref','refs/remotes/origin/main',git('rev-parse','main'))
    git('config','branch.main.remote','origin')
    git('config','branch.main.merge','refs/heads/main')
    subprocess.run(['go','run','.',str(repo),str(output)],cwd=source,env=env,check=True)
print(output)
