package main

type RepoConfig struct {
	Branches          bool              `json:"branches"`
	Remotes           bool              `json:"remotes"`
	Tags              bool              `json:"tags"`
	FirstParent       bool              `json:"first_parent"`
	ShowStashes       bool              `json:"show_stashes"`
	ShowUncommitted   bool              `json:"show_uncommitted"`
	IncludeUntracked  bool              `json:"include_untracked"`
	CombineRefs       bool              `json:"combine_refs"`
	DimNonHeadCommits bool              `json:"dim_non_head_commits"`
	ShowCommitTooltip bool              `json:"show_commit_tooltip"`
	Emoji             bool              `json:"emoji"`
	Markdown          bool              `json:"markdown"`
	DateCommit        bool              `json:"date_commit"`
	ColumnsDate       bool              `json:"columns_date"`
	ColumnsAuthor     bool              `json:"columns_author"`
	ColumnsCommit     bool              `json:"columns_commit"`
	FileTree          bool              `json:"file_tree"`
	CompactFolders    bool              `json:"compact_folders"`
	DateShort         bool              `json:"date_short"`
	RelativeDates     bool              `json:"relative_dates"`
	FullRefs          bool              `json:"full_refs"`
	ScrollToHead      bool              `json:"scroll_to_head"`
	DateWidth         float64           `json:"date_width"`
	AuthorWidth       float64           `json:"author_width"`
	CommitWidth       float64           `json:"commit_width"`
	ColorPreset       int               `json:"color_preset"`
	UseMailmap        bool              `json:"use_mailmap"`
	IncludeReflogs    bool              `json:"include_reflogs"`
	RemoteHeads       bool              `json:"remote_heads"`
	OnlyTags          bool              `json:"only_tags"`
	FetchPrune        bool              `json:"fetch_prune"`
	FetchPruneTags    bool              `json:"fetch_prune_tags"`
	ShowDetail        bool              `json:"show_detail"`
	RepoOrder         string            `json:"repo_order"`
	BranchGlobs       []string          `json:"branch_globs"`
	LaneColors        []string          `json:"lane_colors"`
	HiddenActions     []string          `json:"hidden_actions"`
	EmojiMappings     []string          `json:"emoji_mappings"`
	GraphStyle        string            `json:"graph_style"`
	PrProvider        string            `json:"pr_provider"`
	IssueProvider     string            `json:"issue_provider"`
	Extra             map[string]string `json:"extra,omitempty"`
}

func defaultConfig() RepoConfig {
	return RepoConfig{
		Branches:          true,
		Remotes:           true,
		Tags:              true,
		FirstParent:       false,
		ShowStashes:       true,
		ShowUncommitted:   true,
		IncludeUntracked:  true,
		CombineRefs:       true,
		DimNonHeadCommits: false,
		ShowCommitTooltip: false,
		Emoji:             true,
		Markdown:          true,
		DateCommit:        false,
		ColumnsDate:       true,
		ColumnsAuthor:     true,
		ColumnsCommit:     true,
		FileTree:          false,
		CompactFolders:    true,
		DateShort:         false,
		RelativeDates:     false,
		FullRefs:          false,
		ScrollToHead:      false,
		DateWidth:         150,
		AuthorWidth:       130,
		CommitWidth:       80,
		ColorPreset:       0,
		UseMailmap:        false,
		IncludeReflogs:    false,
		RemoteHeads:       false,
		OnlyTags:          false,
		FetchPrune:        false,
		FetchPruneTags:    false,
		ShowDetail:        true,
		RepoOrder:         "name",
		GraphStyle:        "rounded",
	}
}
