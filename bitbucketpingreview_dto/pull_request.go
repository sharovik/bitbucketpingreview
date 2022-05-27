package bitbucketpingreview_dto

//PullRequest the pull-request item
type PullRequest struct {
	ID             int64
	CanBeMerged    bool
	RepositorySlug string
	BranchName     string
	Workspace      string
	Title          string
	Description    string
}
