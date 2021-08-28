package resource

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/shurcooL/githubv4"
)

// Check (business logic)
func Check(request CheckRequest, manager Github) (CheckResponse, error) {
	var response CheckResponse

	// Filter out pull request if it does not have a filtered state
	filterStates := []githubv4.PullRequestState{githubv4.PullRequestStateOpen}
	if len(request.Source.States) > 0 {
		filterStates = request.Source.States
	}

	pulls, err := manager.ListPullRequests(filterStates)
	if err != nil {
		return nil, fmt.Errorf("failed to get last commits: %s", err)
	}

	disableSkipCI := request.Source.DisableCISkip

Loop:
	for _, p := range pulls {
		// [ci skip]/[skip ci] in Pull request title
		if !disableSkipCI && ContainsSkipCI(p.Title) {
			continue
		}

		// [ci skip]/[skip ci] in Commit message
		if !disableSkipCI && ContainsSkipCI(p.Tip.Message) {
			continue
		}

		// Filter pull request if the BaseBranch does not match the one specified in source
		if request.Source.BaseBranch != "" && p.PullRequestObject.BaseRefName != request.Source.BaseBranch {
			continue
		}

		// Filter out commits that are too old.
		if !p.UpdatedDate().Time.After(request.Version.CommittedDate) {
			continue
		}

		// Filter out pull request if it does not contain at least one of the desired labels
		if len(request.Source.Labels) > 0 {
			labelFound := false

		LabelLoop:
			for _, wantedLabel := range request.Source.Labels {
				for _, targetLabel := range p.Labels {
					if targetLabel.Name == wantedLabel {
						labelFound = true
						break LabelLoop
					}
				}
			}

			if !labelFound {
				continue Loop
			}
		}

		// Filter out forks.
		if request.Source.DisableForks && p.IsCrossRepository {
			continue
		}

		// Filter out drafts.
		if request.Source.IgnoreDrafts && p.IsDraft {
			continue
		}

		// Filter pull request if it does not have the required number of approved review(s).
		if p.ApprovedReviewCount < request.Source.RequiredReviewApprovals {
			continue
		}

		// Filter pull request if paths or ignorePaths is specified and no wanted paths were found
		if len(request.Source.Paths) > 0 || len(request.Source.IgnorePaths) > 0 {
			found, err := HasWantedFiles(
				strconv.Itoa(p.Number),
				request.Source.Paths,
				request.Source.IgnorePaths,
				p.Files,
				p.FilesPageInfo.HasNextPage,
				string(p.FilesPageInfo.EndCursor),
				manager,
			)

			if err != nil {
				return nil, err
			}

			if !found {
				continue Loop
			}
		}

		response = append(response, NewVersion(p))
	}

	// Sort the commits by date
	sort.Sort(response)

	// If there are no new but an old version = return the old
	if len(response) == 0 && request.Version.PR != "" {
		response = append(response, request.Version)
	}
	// If there are new versions and no previous = return just the latest
	if len(response) != 0 && request.Version.PR == "" {
		response = CheckResponse{response[len(response)-1]}
	}
	return response, nil
}

func HasWantedFiles(prNumber string, paths []string, ignorePaths []string, files []ChangedFileObject, hasMoreFiles bool, nextFileCursor string, manager Github) (bool, error) {
	// construct a slice that contains 'wanted' files and use this to determine if we should continue
	// files are wanted either when they appear in the paths list or don't appear in the ignore paths list
	var wanted []ChangedFileObject
	var err error

	if len(paths) > 0 {
		for _, pattern := range paths {
			w, err := FilterPath(files, pattern)
			if err != nil {
				return false, fmt.Errorf("path match failed: %s", err)
			}
			wanted = append(wanted, w...)
		}
	} else {
		wanted = files
	}

	for _, pattern := range ignorePaths {
		wanted, err = FilterIgnorePath(wanted, pattern)
		if err != nil {
			return false, fmt.Errorf("ignore path match failed: %s", err)
		}
	}

	if len(wanted) > 0 {
		// wanted files were found
		return true, nil
	}

	if !hasMoreFiles {
		// no wanted files were found and there are no more files to examine
		return false, nil
	}

	// no wanted files were found, but there are more files to check
	// fetch them now and then check them in another iteration of this function
	files, hasMoreFiles, nextFileCursor, err = manager.GetChangedFiles(
		prNumber,
		100,
		nextFileCursor,
	)
	if err != nil {
		return false, fmt.Errorf("get more files failed: %s", err)
	}

	return HasWantedFiles(prNumber, paths, ignorePaths, files, hasMoreFiles, nextFileCursor, manager)
}

// ContainsSkipCI returns true if a string contains [ci skip] or [skip ci].
func ContainsSkipCI(s string) bool {
	re := regexp.MustCompile("(?i)\\[(ci skip|skip ci)\\]")
	return re.MatchString(s)
}

// FilterIgnorePath ...
func FilterIgnorePath(files []ChangedFileObject, pattern string) ([]ChangedFileObject, error) {
	var out []ChangedFileObject
	for _, cfo := range files {
		file := cfo.Path
		match, err := filepath.Match(pattern, file)
		if err != nil {
			return nil, err
		}
		if !match && !IsInsidePath(pattern, file) {
			out = append(out, cfo)
		}
	}
	return out, nil
}

// FilterPath ...
func FilterPath(files []ChangedFileObject, pattern string) ([]ChangedFileObject, error) {
	var out []ChangedFileObject
	for _, cfo := range files {
		file := cfo.Path
		match, err := filepath.Match(pattern, file)
		if err != nil {
			return nil, err
		}
		if match || IsInsidePath(pattern, file) {
			out = append(out, cfo)
		}
	}
	return out, nil
}

// IsInsidePath checks whether the child path is inside the parent path.
//
// /foo/bar is inside /foo, but /foobar is not inside /foo.
// /foo is inside /foo, but /foo is not inside /foo/
func IsInsidePath(parent, child string) bool {
	if parent == child {
		return true
	}

	// we add a trailing slash so that we only get prefix matches on a
	// directory separator
	parentWithTrailingSlash := parent
	if !strings.HasSuffix(parentWithTrailingSlash, string(filepath.Separator)) {
		parentWithTrailingSlash += string(filepath.Separator)
	}

	return strings.HasPrefix(child, parentWithTrailingSlash)
}

// CheckRequest ...
type CheckRequest struct {
	Source  Source  `json:"source"`
	Version Version `json:"version"`
}

// CheckResponse ...
type CheckResponse []Version

func (r CheckResponse) Len() int {
	return len(r)
}

func (r CheckResponse) Less(i, j int) bool {
	return r[j].CommittedDate.After(r[i].CommittedDate)
}

func (r CheckResponse) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}
