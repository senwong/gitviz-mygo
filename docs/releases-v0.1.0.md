# Gitviz 0.1.0

The first downloadable release of Gitviz MyGo: a native Git graph viewer for macOS.

## Download

- `Gitviz-macOS-universal.zip`: universal macOS application containing Apple Silicon (arm64) and Intel (x86_64) binaries.
- `SHA256SUMS.txt`: SHA-256 checksum for the ZIP.
- Requires macOS 13 or later and an installed Git executable.

Unzip the archive and drag `gitviz.app` to Applications. This release is **ad-hoc signed and not notarized by Apple**. After the first launch attempt, macOS may require you to allow it through **System Settings → Privacy & Security → Open Anyway**. See the [installation guide](https://senwong.github.io/gitviz-mygo/install/).

## Included

- Colored commit graph with local/remote branch labels and tags.
- Inline commit details and changed-file navigation.
- Full-size code diffs, text search and multiline copy.
- Before/after previews for supported raster images.
- Repository switching and VS Code workspace support.
- Remote branch/tag freshness checks and optional automatic Fetch.
- Light/dark themes, keyboard shortcuts and last-repository restore.

Gitviz is a separate project inspired by the workflow of VS Code Git Graph; complete feature parity is not claimed. Automatic Fetch does not pull or merge into the current local branch.

## Validation and current limits

The app is built for both architectures. The Intel executable can be launch-tested on the build machine; the arm64 executable is cross-compiled and is not physically tested on Apple Silicon here. The local build, static checks and focused core race tests passed. The full race suite hit its default 10-minute total time limit during the initial repository import, so a complete full-suite pass is not claimed for this release.

The macOS application is the only distributed platform. GIF image previews show the first frame. Image previews have size and pixel-count limits, and unavailable Git LFS contents require fetching the actual files separately.

[Website](https://senwong.github.io/gitviz-mygo/) · [Source and issues](https://github.com/senwong/gitviz-mygo)
