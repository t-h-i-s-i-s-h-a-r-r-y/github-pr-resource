package resource_test

import (
	"testing"

	"github.com/shurcooL/githubv4"
	"github.com/stretchr/testify/assert"
	resource "github.com/telia-oss/github-pr-resource"
	"github.com/telia-oss/github-pr-resource/fakes"
)

var (
	testPullRequests = []*CheckTestPR{
		createCheckTestPR(1, "master", true, false, 0, nil, false, githubv4.PullRequestStateOpen, [][]resource.ChangedFileObject{}),
		createCheckTestPR(2, "master", false, false, 0, nil, false, githubv4.PullRequestStateOpen, [][]resource.ChangedFileObject{
			{{Path: "README.md"}, {Path: "travis.yml"}},
		}),
		createCheckTestPR(3, "master", false, false, 0, nil, true, githubv4.PullRequestStateOpen, [][]resource.ChangedFileObject{
			{{Path: "terraform/modules/ecs/main.tf"}, {Path: "README.md"}},
		}),
		createCheckTestPR(4, "master", false, false, 0, nil, false, githubv4.PullRequestStateOpen, [][]resource.ChangedFileObject{
			{{Path: "terraform/modules/variables.tf"}, {Path: "travis.yml"}},
		}),
		createCheckTestPR(5, "master", false, true, 0, nil, false, githubv4.PullRequestStateOpen, [][]resource.ChangedFileObject{}),
		createCheckTestPR(6, "master", false, false, 0, nil, false, githubv4.PullRequestStateOpen, [][]resource.ChangedFileObject{}),
		createCheckTestPR(7, "develop", false, false, 0, []string{"enhancement"}, false, githubv4.PullRequestStateOpen, [][]resource.ChangedFileObject{}),
		createCheckTestPR(8, "master", false, false, 1, []string{"wontfix"}, false, githubv4.PullRequestStateOpen, [][]resource.ChangedFileObject{}),
		createCheckTestPR(9, "master", false, false, 0, nil, false, githubv4.PullRequestStateOpen, [][]resource.ChangedFileObject{}),
		createCheckTestPR(10, "master", false, false, 0, nil, false, githubv4.PullRequestStateClosed, [][]resource.ChangedFileObject{}),
		createCheckTestPR(11, "master", false, false, 0, nil, false, githubv4.PullRequestStateMerged, [][]resource.ChangedFileObject{}),
		createCheckTestPR(12, "master", false, false, 0, nil, false, githubv4.PullRequestStateOpen, [][]resource.ChangedFileObject{}),
	}
)

func TestCheck(t *testing.T) {
	tests := []struct {
		description  string
		source       resource.Source
		version      resource.Version
		pullRequests []*CheckTestPR
		expected     resource.CheckResponse
	}{
		{
			description: "check returns the latest version if there is no previous",
			source: resource.Source{
				Repository:  "itsdalmo/test-repository",
				AccessToken: "oauthtoken",
			},
			version:      resource.Version{},
			pullRequests: testPullRequests,
			expected: resource.CheckResponse{
				resource.NewVersion(testPullRequests[1].PR),
			},
		},

		{
			description: "check returns the previous version when its still latest",
			source: resource.Source{
				Repository:  "itsdalmo/test-repository",
				AccessToken: "oauthtoken",
			},
			version:      resource.NewVersion(testPullRequests[1].PR),
			pullRequests: testPullRequests,
			expected: resource.CheckResponse{
				resource.NewVersion(testPullRequests[1].PR),
			},
		},

		{
			description: "check returns all new versions since the last",
			source: resource.Source{
				Repository:  "itsdalmo/test-repository",
				AccessToken: "oauthtoken",
			},
			version:      resource.NewVersion(testPullRequests[3].PR),
			pullRequests: testPullRequests,
			expected: resource.CheckResponse{
				resource.NewVersion(testPullRequests[2].PR),
				resource.NewVersion(testPullRequests[1].PR),
			},
		},

		{
			description: "check will only return versions that match the specified paths",
			source: resource.Source{
				Repository:  "itsdalmo/test-repository",
				AccessToken: "oauthtoken",
				Paths:       []string{"terraform/*/*.tf", "terraform/*/*/*.tf"},
			},
			version:      resource.NewVersion(testPullRequests[3].PR),
			pullRequests: testPullRequests,
			expected: resource.CheckResponse{
				resource.NewVersion(testPullRequests[2].PR),
			},
		},

		{
			description: "check will skip versions which only match the ignore paths",
			source: resource.Source{
				Repository:  "itsdalmo/test-repository",
				AccessToken: "oauthtoken",
				IgnorePaths: []string{"*.md", "*.yml"},
			},
			version:      resource.NewVersion(testPullRequests[3].PR),
			pullRequests: testPullRequests,
			expected: resource.CheckResponse{
				resource.NewVersion(testPullRequests[2].PR),
			},
		},

		{
			description: "check correctly ignores [skip ci] when specified",
			source: resource.Source{
				Repository:    "itsdalmo/test-repository",
				AccessToken:   "oauthtoken",
				DisableCISkip: true,
			},
			version:      resource.NewVersion(testPullRequests[1].PR),
			pullRequests: testPullRequests,
			expected: resource.CheckResponse{
				resource.NewVersion(testPullRequests[0].PR),
			},
		},

		{
			description: "check correctly ignores drafts when drafts are ignored",
			source: resource.Source{
				Repository:   "itsdalmo/test-repository",
				AccessToken:  "oauthtoken",
				IgnoreDrafts: true,
			},
			version:      resource.NewVersion(testPullRequests[3].PR),
			pullRequests: testPullRequests,
			expected: resource.CheckResponse{
				resource.NewVersion(testPullRequests[1].PR),
			},
		},

		{
			description: "check does not ignore drafts when drafts are not ignored",
			source: resource.Source{
				Repository:   "itsdalmo/test-repository",
				AccessToken:  "oauthtoken",
				IgnoreDrafts: false,
			},
			version:      resource.NewVersion(testPullRequests[3].PR),
			pullRequests: testPullRequests,
			expected: resource.CheckResponse{
				resource.NewVersion(testPullRequests[2].PR),
				resource.NewVersion(testPullRequests[1].PR),
			},
		},

		{
			description: "check correctly ignores cross repo pull requests",
			source: resource.Source{
				Repository:   "itsdalmo/test-repository",
				AccessToken:  "oauthtoken",
				DisableForks: true,
			},
			version:      resource.NewVersion(testPullRequests[5].PR),
			pullRequests: testPullRequests,
			expected: resource.CheckResponse{
				resource.NewVersion(testPullRequests[3].PR),
				resource.NewVersion(testPullRequests[2].PR),
				resource.NewVersion(testPullRequests[1].PR),
			},
		},

		{
			description: "check supports specifying base branch",
			source: resource.Source{
				Repository:  "itsdalmo/test-repository",
				AccessToken: "oauthtoken",
				BaseBranch:  "develop",
			},
			version:      resource.Version{},
			pullRequests: testPullRequests,
			expected: resource.CheckResponse{
				resource.NewVersion(testPullRequests[6].PR),
			},
		},

		{
			description: "check correctly ignores PRs with no approved reviews when specified",
			source: resource.Source{
				Repository:              "itsdalmo/test-repository",
				AccessToken:             "oauthtoken",
				RequiredReviewApprovals: 1,
			},
			version:      resource.NewVersion(testPullRequests[8].PR),
			pullRequests: testPullRequests,
			expected: resource.CheckResponse{
				resource.NewVersion(testPullRequests[7].PR),
			},
		},

		{
			description: "check returns latest version from a PR with at least one of the desired labels on it",
			source: resource.Source{
				Repository:  "itsdalmo/test-repository",
				AccessToken: "oauthtoken",
				Labels:      []string{"enhancement"},
			},
			version:      resource.Version{},
			pullRequests: testPullRequests,
			expected: resource.CheckResponse{
				resource.NewVersion(testPullRequests[6].PR),
			},
		},

		{
			description: "check returns latest version from a PR with a single state filter",
			source: resource.Source{
				Repository:  "itsdalmo/test-repository",
				AccessToken: "oauthtoken",
				States:      []githubv4.PullRequestState{githubv4.PullRequestStateClosed},
			},
			version:      resource.Version{},
			pullRequests: testPullRequests,
			expected: resource.CheckResponse{
				resource.NewVersion(testPullRequests[9].PR),
			},
		},

		{
			description: "check filters out versions from a PR which do not match the state filter",
			source: resource.Source{
				Repository:  "itsdalmo/test-repository",
				AccessToken: "oauthtoken",
				States:      []githubv4.PullRequestState{githubv4.PullRequestStateOpen},
			},
			version:      resource.Version{},
			pullRequests: testPullRequests[9:11],
			expected:     resource.CheckResponse(nil),
		},

		{
			description: "check returns versions from a PR with multiple state filters",
			source: resource.Source{
				Repository:  "itsdalmo/test-repository",
				AccessToken: "oauthtoken",
				States:      []githubv4.PullRequestState{githubv4.PullRequestStateClosed, githubv4.PullRequestStateMerged},
			},
			version:      resource.NewVersion(testPullRequests[11].PR),
			pullRequests: testPullRequests,
			expected: resource.CheckResponse{
				resource.NewVersion(testPullRequests[9].PR),
				resource.NewVersion(testPullRequests[10].PR),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			github := new(fakes.FakeGithub)
			pullRequests := []*resource.PullRequest{}
			filterStates := []githubv4.PullRequestState{githubv4.PullRequestStateOpen}
			if len(tc.source.States) > 0 {
				filterStates = tc.source.States
			}
			var changedFilesCall int
			for i, pr := range tc.pullRequests {
				for j := range filterStates {
					if filterStates[j] == tc.pullRequests[i].PR.PullRequestObject.State {
						pullRequests = append(pullRequests, tc.pullRequests[i].PR)
						break
					}
				}

				for _, cfo := range pr.AdditionalFiles {
					hasNext := len(pr.AdditionalFiles) > (i + 1)
					github.GetChangedFilesReturnsOnCall(changedFilesCall, cfo, hasNext, "", nil)
					changedFilesCall += 1
				}
			}
			github.ListPullRequestsReturns(pullRequests, nil)

			input := resource.CheckRequest{Source: tc.source, Version: tc.version}
			output, err := resource.Check(input, github)

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expected, output)
			}
			assert.Equal(t, 1, github.ListPullRequestsCallCount())
		})
	}
}

func TestContainsSkipCI(t *testing.T) {
	tests := []struct {
		description string
		message     string
		want        bool
	}{
		{
			description: "does not just match any symbol in the regexp",
			message:     "(",
			want:        false,
		},
		{
			description: "does not match when it should not",
			message:     "test",
			want:        false,
		},
		{
			description: "matches [ci skip]",
			message:     "[ci skip]",
			want:        true,
		},
		{
			description: "matches [skip ci]",
			message:     "[skip ci]",
			want:        true,
		},
		{
			description: "matches trailing skip ci",
			message:     "trailing [skip ci]",
			want:        true,
		},
		{
			description: "matches leading skip ci",
			message:     "[skip ci] leading",
			want:        true,
		},
		{
			description: "is case insensitive",
			message:     "case[Skip CI]insensitive",
			want:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			got := resource.ContainsSkipCI(tc.message)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFilterPath(t *testing.T) {
	cases := []struct {
		description string
		pattern     string
		files       []resource.ChangedFileObject
		want        []resource.ChangedFileObject
	}{
		{
			description: "returns all matching files",
			pattern:     "*.txt",
			files: []resource.ChangedFileObject{
				{Path: "file1.txt"},
				{Path: "test/file2.txt"},
			},
			want: []resource.ChangedFileObject{
				{Path: "file1.txt"},
			},
		},
		{
			description: "works with wildcard",
			pattern:     "test/*",
			files: []resource.ChangedFileObject{
				{Path: "file1.txt"},
				{Path: "test/file2.txt"},
			},
			want: []resource.ChangedFileObject{
				{Path: "test/file2.txt"},
			},
		},
		{
			description: "excludes unmatched files",
			pattern:     "*/*.txt",
			files: []resource.ChangedFileObject{
				{Path: "test/file1.go"},
				{Path: "test/file2.txt"},
			},
			want: []resource.ChangedFileObject{
				{Path: "test/file2.txt"},
			},
		},
		{
			description: "handles prefix matches",
			pattern:     "foo/",
			files: []resource.ChangedFileObject{
				{Path: "foo/a"},
				{Path: "foo/a.txt"},
				{Path: "foo/a/b/c/d.txt"},
				{Path: "foo"},
				{Path: "bar"},
				{Path: "bar/a.txt"},
			},
			want: []resource.ChangedFileObject{
				{Path: "foo/a"},
				{Path: "foo/a.txt"},
				{Path: "foo/a/b/c/d.txt"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			got, err := resource.FilterPath(tc.files, tc.pattern)
			if assert.NoError(t, err) {
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

func TestFilterIgnorePath(t *testing.T) {
	cases := []struct {
		description string
		pattern     string
		files       []resource.ChangedFileObject
		want        []resource.ChangedFileObject
	}{
		{
			description: "excludes all matching files",
			pattern:     "*.txt",
			files: []resource.ChangedFileObject{
				{Path: "file1.txt"},
				{Path: "test/file2.txt"},
			},
			want: []resource.ChangedFileObject{
				{Path: "test/file2.txt"},
			},
		},
		{
			description: "works with wildcard",
			pattern:     "test/*",
			files: []resource.ChangedFileObject{
				{Path: "file1.txt"},
				{Path: "test/file2.txt"},
			},
			want: []resource.ChangedFileObject{
				{Path: "file1.txt"},
			},
		},
		{
			description: "includes unmatched files",
			pattern:     "*/*.txt",
			files: []resource.ChangedFileObject{
				{Path: "test/file1.go"},
				{Path: "test/file2.txt"},
			},
			want: []resource.ChangedFileObject{
				{Path: "test/file1.go"},
			},
		},
		{
			description: "handles prefix matches",
			pattern:     "foo/",
			files: []resource.ChangedFileObject{
				{Path: "foo/a"},
				{Path: "foo/a.txt"},
				{Path: "foo/a/b/c/d.txt"},
				{Path: "foo"},
				{Path: "bar"},
				{Path: "bar/a.txt"},
			},
			want: []resource.ChangedFileObject{
				{Path: "foo"},
				{Path: "bar"},
				{Path: "bar/a.txt"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			got, err := resource.FilterIgnorePath(tc.files, tc.pattern)
			if assert.NoError(t, err) {
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

func TestIsInsidePath(t *testing.T) {
	cases := []struct {
		description string
		parent      string

		expectChildren    []string
		expectNotChildren []string

		want bool
	}{
		{
			description: "basic test",
			parent:      "foo/bar",
			expectChildren: []string{
				"foo/bar",
				"foo/bar/baz",
			},
			expectNotChildren: []string{
				"foo/barbar",
				"foo/baz/bar",
			},
		},
		{
			description: "does not match parent directories against child files",
			parent:      "foo/",
			expectChildren: []string{
				"foo/bar",
			},
			expectNotChildren: []string{
				"foo",
			},
		},
		{
			description: "matches parents without trailing slash",
			parent:      "foo/bar",
			expectChildren: []string{
				"foo/bar",
				"foo/bar/baz",
			},
		},
		{
			description: "handles children that are shorter than the parent",
			parent:      "foo/bar/baz",
			expectNotChildren: []string{
				"foo",
				"foo/bar",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			for _, expectedChild := range tc.expectChildren {
				if !resource.IsInsidePath(tc.parent, expectedChild) {
					t.Errorf("Expected \"%s\" to be inside \"%s\"", expectedChild, tc.parent)
				}
			}

			for _, expectedNotChild := range tc.expectNotChildren {
				if resource.IsInsidePath(tc.parent, expectedNotChild) {
					t.Errorf("Expected \"%s\" to not be inside \"%s\"", expectedNotChild, tc.parent)
				}
			}
		})
	}
}

func TestHasWantedFiles(t *testing.T) {
	cases := []struct {
		description string

		paths       []string
		ignorePaths []string

		files [][]string

		expected bool
	}{
		{
			description: "true when paths in first page of files",
			paths:       []string{"README.md"},
			ignorePaths: []string{},
			files: [][]string{
				{"README.md"},
			},
			expected: true,
		},
		{
			description: "true when paths in second page of files",
			paths:       []string{"*.md"},
			ignorePaths: []string{},
			files: [][]string{
				{"travis.yml"},
				{"README.md"},
			},
			expected: true,
		},
		{
			description: "true when multiple paths but only one file matches",
			paths:       []string{"*.md"},
			ignorePaths: []string{},
			files: [][]string{
				{"travis.yml", "README.md"},
			},
			expected: true,
		},
		{
			description: "false when paths not in any page",
			paths:       []string{"*.md"},
			ignorePaths: []string{},
			files: [][]string{
				{"travis.yml"},
				{"travis.yml"},
			},
			expected: false,
		},
		{
			description: "true when paths on first page not in ignore",
			paths:       []string{},
			ignorePaths: []string{"*.yml"},
			files: [][]string{
				{"README.md"},
			},
			expected: true,
		},
		{
			description: "true when paths on second page not in ignore",
			paths:       []string{},
			ignorePaths: []string{"*.yml"},
			files: [][]string{
				{"travis.yml"},
				{"README.md"},
			},
			expected: true,
		},
		{
			description: "false when multiple ignore paths and both match",
			paths:       []string{},
			ignorePaths: []string{"*.md", "*.yml"},
			files: [][]string{
				{"travis.yml", "README.md"},
			},
			expected: false,
		},
		{
			description: "false when all pages in ignore",
			paths:       []string{},
			ignorePaths: []string{"*.yml"},
			files: [][]string{
				{"travis.yml"},
				{"travis2.yml"},
			},
			expected: false,
		},
		{
			description: "false when in both paths and ignore",
			paths:       []string{"*.yml"},
			ignorePaths: []string{"*.yml"},
			files: [][]string{
				{"travis.yml"},
			},
			expected: false,
		},
	}

	for _, tc := range cases {
		manager := new(fakes.FakeGithub)

		t.Run(tc.description, func(t *testing.T) {
			var initialFiles []resource.ChangedFileObject

			for i, files := range tc.files {
				if i == 0 {
					// The first page of files is included in the first function call
					for _, path := range files {
						initialFiles = append(initialFiles, resource.ChangedFileObject{Path: path})
					}
					continue
				}

				// subsequent pages are retrieved from GitHub
				var cfo []resource.ChangedFileObject
				for _, path := range files {
					cfo = append(cfo, resource.ChangedFileObject{Path: path})
				}
				hasNextPage := len(tc.files) > (i + 1)
				manager.GetChangedFilesReturnsOnCall(i-1, cfo, hasNextPage, "", nil)
			}

			initialHasNextPage := len(tc.files) > 1

			actual, err := resource.HasWantedFiles("foo", tc.paths, tc.ignorePaths, initialFiles, initialHasNextPage, "", manager)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

type CheckTestPR struct {
	PR              *resource.PullRequest
	AdditionalFiles [][]resource.ChangedFileObject
}

func createCheckTestPR(
	count int,
	baseName string,
	skipCI bool,
	isCrossRepo bool,
	approvedReviews int,
	labels []string,
	isDraft bool,
	state githubv4.PullRequestState,
	files [][]resource.ChangedFileObject,
) *CheckTestPR {
	var additionalFiles [][]resource.ChangedFileObject

	pr := createTestPR(count, baseName, skipCI, isCrossRepo, approvedReviews, labels, isDraft, state)

	// PRs retrieved by ListPullRequests - as is the case with check - include some changed files.
	// We want to write tests that include changed files beyond those on the first page, however.
	if len(files) > 0 {
		additionalFiles = files[1:]
		pr.Files = files[0]
		pr.FilesPageInfo.HasNextPage = len(additionalFiles) > 0
	}

	return &CheckTestPR{
		PR:              pr,
		AdditionalFiles: additionalFiles,
	}
}
