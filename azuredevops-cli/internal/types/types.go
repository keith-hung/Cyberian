// Package types defines request/response structures for the Azure DevOps REST API.
package types

// --- Azure DevOps API response types ---

// ProjectListResponse is the API response for listing projects.
type ProjectListResponse struct {
	Count int          `json:"count"`
	Value []APIProject `json:"value"`
}

// APIProject represents an Azure DevOps project.
type APIProject struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	State       string `json:"state"`
	URL         string `json:"url"`
}

// RepoListResponse is the API response for listing repositories.
type RepoListResponse struct {
	Count int       `json:"count"`
	Value []APIRepo `json:"value"`
}

// APIRepo represents a Git repository.
type APIRepo struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	URL           string     `json:"url"`
	RemoteURL     string     `json:"remoteUrl"`
	DefaultBranch string     `json:"defaultBranch"`
	Project       APIProject `json:"project"`
	Size          int64      `json:"size"`
	IsDisabled    bool       `json:"isDisabled"`
}

// PRListResponse is the API response for listing pull requests.
type PRListResponse struct {
	Count int     `json:"count"`
	Value []APIPR `json:"value"`
}

// APIPR represents a pull request.
type APIPR struct {
	PullRequestID int           `json:"pullRequestId"`
	Title         string        `json:"title"`
	Description   string        `json:"description"`
	Status        string        `json:"status"`
	CreatedBy     APIIdentity   `json:"createdBy"`
	CreationDate  string        `json:"creationDate"`
	ClosedDate    string        `json:"closedDate,omitempty"`
	SourceRefName string        `json:"sourceRefName"`
	TargetRefName string        `json:"targetRefName"`
	MergeStatus   string        `json:"mergeStatus"`
	IsDraft       bool          `json:"isDraft"`
	Reviewers     []APIReviewer `json:"reviewers"`
	URL           string        `json:"url"`
	Repository    APIRepo       `json:"repository"`
}

// APIIdentity represents a user identity.
type APIIdentity struct {
	DisplayName string `json:"displayName"`
	ID          string `json:"id"`
	UniqueName  string `json:"uniqueName"`
}

// APIReviewer represents a PR reviewer.
type APIReviewer struct {
	DisplayName string `json:"displayName"`
	ID          string `json:"id"`
	UniqueName  string `json:"uniqueName"`
	Vote        int    `json:"vote"`
	IsRequired  bool   `json:"isRequired"`
}

// APIConnectionData is the response from _apis/connectionData.
type APIConnectionData struct {
	AuthenticatedUser APIIdentity `json:"authenticatedUser"`
}

// ThreadListResponse is the API response for listing comment threads.
type ThreadListResponse struct {
	Count int         `json:"count"`
	Value []APIThread `json:"value"`
}

// APIThread represents a PR comment thread.
type APIThread struct {
	ID       int          `json:"id"`
	Comments []APIComment `json:"comments"`
	Status   int          `json:"status"`
}

// APIComment represents a single comment in a thread.
type APIComment struct {
	ID              int         `json:"id"`
	Content         string      `json:"content"`
	CommentType     string      `json:"commentType"`
	ParentCommentID int         `json:"parentCommentId"`
	Author          APIIdentity `json:"author"`
	PublishedDate   string      `json:"publishedDate"`
	IsDeleted       bool        `json:"isDeleted"`
}

// ReviewerListResponse is the API response for listing reviewers.
type ReviewerListResponse struct {
	Count int           `json:"count"`
	Value []APIReviewer `json:"value"`
}

// APIIdentitySearchResponse is the response from identity search.
type APIIdentitySearchResponse struct {
	Count int           `json:"count"`
	Value []APIIdentity `json:"value"`
}

// APIErrorResponse is the Azure DevOps API error format.
type APIErrorResponse struct {
	Message  string `json:"message"`
	TypeName string `json:"typeName"`
	TypeKey  string `json:"typeKey"`
}

// --- API request body types ---

// PRCreateBody is the request body for creating a PR.
type PRCreateBody struct {
	SourceRefName string `json:"sourceRefName"`
	TargetRefName string `json:"targetRefName"`
	Title         string `json:"title"`
	Description   string `json:"description,omitempty"`
}

// PRUpdateBody is the request body for updating a PR.
type PRUpdateBody struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
}

// VoteBody is the request body for voting on a PR.
type VoteBody struct {
	Vote int `json:"vote"`
}

// ReviewerAddBody is the request body for adding a reviewer.
type ReviewerAddBody struct {
	ID         string `json:"id,omitempty"`
	UniqueName string `json:"uniqueName,omitempty"`
}

// ThreadBody is the request body for creating a comment thread.
type ThreadBody struct {
	Comments []ThreadComment `json:"comments"`
	Status   int             `json:"status"`
}

// ThreadComment is a comment within a thread body.
type ThreadComment struct {
	ParentCommentID int    `json:"parentCommentId"`
	Content         string `json:"content"`
	CommentType     string `json:"commentType"`
}

// --- CLI output types ---

// ErrorOutput is the standard error JSON written to stderr.
type ErrorOutput struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// ProjectsOutput is the output for the projects command.
type ProjectsOutput struct {
	Success  bool            `json:"success"`
	Projects []ProjectOutput `json:"projects"`
	Count    int             `json:"count"`
}

// ProjectOutput is a single project in the output.
type ProjectOutput struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	State       string `json:"state"`
}

// ReposOutput is the output for the repos command.
type ReposOutput struct {
	Success bool         `json:"success"`
	Project string       `json:"project"`
	Repos   []RepoOutput `json:"repos"`
	Count   int          `json:"count"`
}

// RepoOutput is a single repository in the output.
type RepoOutput struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	DefaultBranch string `json:"default_branch"`
	RemoteURL     string `json:"remote_url"`
	Size          int64  `json:"size"`
}

// PRsOutput is the output for the prs command.
type PRsOutput struct {
	Success      bool       `json:"success"`
	PullRequests []PROutput `json:"pull_requests"`
	Count        int        `json:"count"`
}

// PROutput is a single pull request in the output.
type PROutput struct {
	ID           int              `json:"id"`
	Title        string           `json:"title"`
	Description  string           `json:"description,omitempty"`
	Status       string           `json:"status"`
	CreatedBy    string           `json:"created_by"`
	CreationDate string           `json:"creation_date"`
	SourceBranch string           `json:"source_branch"`
	TargetBranch string           `json:"target_branch"`
	MergeStatus  string           `json:"merge_status"`
	IsDraft      bool             `json:"is_draft"`
	Reviewers    []ReviewerOutput `json:"reviewers,omitempty"`
}

// ReviewerOutput is a single reviewer in the output.
type ReviewerOutput struct {
	DisplayName string `json:"display_name"`
	ID          string `json:"id"`
	UniqueName  string `json:"unique_name"`
	Vote        int    `json:"vote"`
	VoteLabel   string `json:"vote_label"`
	IsRequired  bool   `json:"is_required"`
}

// PRDetailOutput is the output for the pr command.
type PRDetailOutput struct {
	Success     bool     `json:"success"`
	PullRequest PROutput `json:"pull_request"`
}

// PRCreateOutput is the output for the pr-create command.
type PRCreateOutput struct {
	Success       bool   `json:"success"`
	PullRequestID int    `json:"pull_request_id"`
	URL           string `json:"url"`
}

// PRUpdateOutput is the output for the pr-update command.
type PRUpdateOutput struct {
	Success       bool   `json:"success"`
	PullRequestID int    `json:"pull_request_id"`
	Message       string `json:"message"`
}

// PRVoteOutput is the output for pr-approve / pr-reject commands.
type PRVoteOutput struct {
	Success       bool   `json:"success"`
	PullRequestID int    `json:"pull_request_id"`
	Vote          string `json:"vote"`
	Message       string `json:"message"`
}

// PRCommentsOutput is the output for the pr-comments command.
type PRCommentsOutput struct {
	Success       bool           `json:"success"`
	PullRequestID int            `json:"pull_request_id"`
	Threads       []ThreadOutput `json:"threads"`
	Count         int            `json:"count"`
}

// ThreadOutput is a single comment thread in the output.
type ThreadOutput struct {
	ID       int             `json:"id"`
	Status   string          `json:"status"`
	Comments []CommentOutput `json:"comments"`
}

// CommentOutput is a single comment in the output.
type CommentOutput struct {
	ID            int    `json:"id"`
	Author        string `json:"author"`
	Content       string `json:"content"`
	PublishedDate string `json:"published_date"`
}

// PRCommentOutput is the output for the pr-comment command.
type PRCommentOutput struct {
	Success       bool   `json:"success"`
	PullRequestID int    `json:"pull_request_id"`
	ThreadID      int    `json:"thread_id,omitempty"`
	Message       string `json:"message"`
}

// ReviewersOutput is the output for the pr-reviewers command.
type ReviewersOutput struct {
	Success       bool             `json:"success"`
	PullRequestID int              `json:"pull_request_id"`
	Reviewers     []ReviewerOutput `json:"reviewers"`
	Count         int              `json:"count"`
}

// AddReviewerOutput is the output for the pr-add-reviewer command.
type AddReviewerOutput struct {
	Success       bool   `json:"success"`
	PullRequestID int    `json:"pull_request_id"`
	Reviewer      string `json:"reviewer"`
	Message       string `json:"message"`
}

// APIAttachment represents an uploaded PR attachment.
type APIAttachment struct {
	ID          int    `json:"id"`
	DisplayName string `json:"displayName"`
	URL         string `json:"url"`
}

// PRAttachmentOutput is the output for the pr-attachment command.
type PRAttachmentOutput struct {
	Success       bool   `json:"success"`
	PullRequestID int    `json:"pull_request_id"`
	Filename      string `json:"filename"`
	URL           string `json:"url"`
}

// VersionOutput is the output for the version command.
type VersionOutput struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
}

// MyPROutput is a single pull request in the my-prs output, with cross-project context.
type MyPROutput struct {
	ID           int              `json:"id"`
	Title        string           `json:"title"`
	Status       string           `json:"status"`
	CreatedBy    string           `json:"created_by"`
	CreationDate string           `json:"creation_date"`
	SourceBranch string           `json:"source_branch"`
	TargetBranch string           `json:"target_branch"`
	MergeStatus  string           `json:"merge_status"`
	IsDraft      bool             `json:"is_draft"`
	Project      string           `json:"project"`
	Repo         string           `json:"repo"`
	Roles        []string         `json:"roles"`
	Reviewers    []ReviewerOutput `json:"reviewers,omitempty"`
}

// MyPRsOutput is the output for the my-prs command.
type MyPRsOutput struct {
	Success      bool         `json:"success"`
	Role         string       `json:"role"`
	Status       string       `json:"status"`
	Project      string       `json:"project,omitempty"`
	PullRequests []MyPROutput `json:"pull_requests"`
	Count        int          `json:"count"`
}
