// Copyright 2017 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github

import (
	"context"
	"fmt"
	"time"

	"github.com/drone/go-scm/scm"
)

type gitService struct {
	client *wrapper
}

func (s *gitService) CreateBranch(ctx context.Context, repo string, params *scm.ReferenceInput) (*scm.Response, error) {
	path := fmt.Sprintf("repos/%s/git/refs", repo)
	in := &createBranch{
		Ref: scm.ExpandRef(params.Name, "refs/heads"),
		Sha: params.Sha,
	}
	return s.client.do(ctx, "POST", path, in, nil)
}

func (s *gitService) FindBranch(ctx context.Context, repo, name string) (*scm.Reference, *scm.Response, error) {
	path := fmt.Sprintf("repos/%s/branches/%s", repo, name)
	out := new(branch)
	res, err := s.client.do(ctx, "GET", path, nil, out)
	return convertBranch(out), res, err
}

func (s *gitService) CreateCommit(ctx context.Context, repo string, opts *scm.CommitInput) (*scm.Commit, *scm.Response, error) {
	tree, res, err := s.createTree(ctx, repo, opts)
	if err != nil {
		return nil, res, err
	}

	path := fmt.Sprintf("repos/%s/git/commits", repo)
	in := new(commitInput)
	in.Message = opts.Message
	in.Tree = tree.Sha
	in.Parents = []string{opts.Base}
	out := new(commit)
	res, err = s.client.do(ctx, "POST", path, in, out)
	if err != nil {
		return nil, res, err
	}

	return convertCommit(out), res, err
}

func (s *gitService) createTree(ctx context.Context, repo string, opts *scm.CommitInput) (*gitTree, *scm.Response, error) {
	path := fmt.Sprintf("repos/%s/git/trees", repo)
	owner, repo := scm.Split(repo)
	in := new(gitTree)
	in.BaseTree = opts.Base
	in.Owner = owner
	in.Repo = repo
	for _, b := range opts.Blobs {
		in.Tree = append(in.Tree, blob{
			Path:    b.Path,
			Mode:    b.Mode,
			Type:    "commit",
			Content: b.Content,
		})
	}
	out := new(gitTree)
	res, err := s.client.do(ctx, "POST", path, in, out)
	return out, res, err
}

func (s *gitService) FindCommit(ctx context.Context, repo, ref string) (*scm.Commit, *scm.Response, error) {
	path := fmt.Sprintf("repos/%s/commits/%s", repo, ref)
	out := new(commit)
	res, err := s.client.do(ctx, "GET", path, nil, out)
	return convertCommit(out), res, err
}

func (s *gitService) FindTag(ctx context.Context, repo, name string) (*scm.Reference, *scm.Response, error) {
	path := fmt.Sprintf("repos/%s/git/ref/tags/%s", repo, name)
	out := new(ref)
	res, err := s.client.do(ctx, "GET", path, nil, out)
	return convertRef(out), res, err
}

func (s *gitService) ListBranches(ctx context.Context, repo string, opts scm.ListOptions) ([]*scm.Reference, *scm.Response, error) {
	path := fmt.Sprintf("repos/%s/branches?%s", repo, encodeListOptions(opts))
	out := []*branch{}
	res, err := s.client.do(ctx, "GET", path, nil, &out)
	return convertBranchList(out), res, err
}

func (s *gitService) ListCommits(ctx context.Context, repo string, opts scm.CommitListOptions) ([]*scm.Commit, *scm.Response, error) {
	path := fmt.Sprintf("repos/%s/commits?%s", repo, encodeCommitListOptions(opts))
	out := []*commit{}
	res, err := s.client.do(ctx, "GET", path, nil, &out)
	return convertCommitList(out), res, err
}

func (s *gitService) ListTags(ctx context.Context, repo string, opts scm.ListOptions) ([]*scm.Reference, *scm.Response, error) {
	path := fmt.Sprintf("repos/%s/tags?%s", repo, encodeListOptions(opts))
	out := []*branch{}
	res, err := s.client.do(ctx, "GET", path, nil, &out)
	return convertTagList(out), res, err
}

func (s *gitService) ListChanges(ctx context.Context, repo, ref string, _ scm.ListOptions) ([]*scm.Change, *scm.Response, error) {
	path := fmt.Sprintf("repos/%s/commits/%s", repo, ref)
	out := new(commit)
	res, err := s.client.do(ctx, "GET", path, nil, &out)
	return convertChangeList(out.Files), res, err
}

func (s *gitService) CompareChanges(ctx context.Context, repo, source, target string, _ scm.ListOptions) ([]*scm.Change, *scm.Response, error) {
	path := fmt.Sprintf("repos/%s/compare/%s...%s", repo, source, target)
	out := new(compare)
	res, err := s.client.do(ctx, "GET", path, nil, &out)
	return convertChangeList(out.Files), res, err
}

type createBranch struct {
	Ref string `json:"ref"`
	Sha string `json:"sha"`
}

type branch struct {
	Name      string `json:"name"`
	Commit    commit `json:"commit"`
	Protected bool   `json:"protected"`
}

type commit struct {
	Sha    string `json:"sha"`
	URL    string `json:"html_url"`
	Commit struct {
		Author struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"author"`
		Committer struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"committer"`
		Message string `json:"message"`
	} `json:"commit"`
	Author struct {
		AvatarURL string `json:"avatar_url"`
		Login     string `json:"login"`
	} `json:"author"`
	Committer struct {
		AvatarURL string `json:"avatar_url"`
		Login     string `json:"login"`
	} `json:"committer"`
	Files []*file `json:"files"`
}

type commitInput struct {
	Message string   `json:"message"`
	Tree    string   `json:"tree"`
	Parents []string `json:"parents"`
}

type gitTree struct {
	Sha      string `json:"sha"`
	URL      string `json:"url"`
	Owner    string `json:"owner"`
	Repo     string `json:"repo"`
	Tree     []blob `json:"tree"`
	BaseTree string `json:"base_tree"`
}

type blob struct {
	Path    string `json:"path,omitempty"`
	Mode    string `json:"mode,omitempty"`
	Type    string `json:"type,omitempty"`
	Sha     string `json:"sha,omitempty"`
	Content string `json:"content,omitempty"`
}

type ref struct {
	Ref    string `json:"ref"`
	Object struct {
		Type string `json:"type"`
		Sha  string `json:"sha"`
	} `json:"object"`
}

type compare struct {
	Files []*file `json:"files"`
}

func convertCommitList(from []*commit) []*scm.Commit {
	to := []*scm.Commit{}
	for _, v := range from {
		to = append(to, convertCommit(v))
	}
	return to
}

func convertCommit(from *commit) *scm.Commit {
	return &scm.Commit{
		Message: from.Commit.Message,
		Sha:     from.Sha,
		Link:    from.URL,
		Author: scm.Signature{
			Name:   from.Commit.Author.Name,
			Email:  from.Commit.Author.Email,
			Date:   from.Commit.Author.Date,
			Login:  from.Author.Login,
			Avatar: from.Author.AvatarURL,
		},
		Committer: scm.Signature{
			Name:   from.Commit.Committer.Name,
			Email:  from.Commit.Committer.Email,
			Date:   from.Commit.Committer.Date,
			Login:  from.Committer.Login,
			Avatar: from.Committer.AvatarURL,
		},
	}
}

func convertBranchList(from []*branch) []*scm.Reference {
	to := []*scm.Reference{}
	for _, v := range from {
		to = append(to, convertBranch(v))
	}
	return to
}

func convertBranch(from *branch) *scm.Reference {
	return &scm.Reference{
		Name: scm.TrimRef(from.Name),
		Path: scm.ExpandRef(from.Name, "refs/heads/"),
		Sha:  from.Commit.Sha,
	}
}

func convertRef(from *ref) *scm.Reference {
	return &scm.Reference{
		Name: scm.TrimRef(from.Ref),
		Path: from.Ref,
		Sha:  from.Object.Sha,
	}
}

func convertTagList(from []*branch) []*scm.Reference {
	to := []*scm.Reference{}
	for _, v := range from {
		to = append(to, convertTag(v))
	}
	return to
}

func convertTag(from *branch) *scm.Reference {
	return &scm.Reference{
		Name: scm.TrimRef(from.Name),
		Path: scm.ExpandRef(from.Name, "refs/tags/"),
		Sha:  from.Commit.Sha,
	}
}
