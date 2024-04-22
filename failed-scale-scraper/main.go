package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/go-github/v60/github"
	"github.com/samber/lo"
)

type Info struct {
	ID int64
	Timestamp time.Time
}

func main() {
	token := os.Getenv("GH_PAT")
	if token == "" {
		log.Fatalf("creating client, GH_PAT not set")
	}
	client := github.NewClient(nil).WithAuthToken(token)

	ctx := context.Background()

	runs, err := getLastFailedWorkflows(ctx, client, 2)
	if err != nil {
		log.Fatalf("fetching workflows, %s", err)
	}
	log.Printf("discoverd %d failed workflow runs", len(runs))
	err = os.Mkdir("logs", 0777)
	if err != nil && !os.IsExist(err){
		log.Fatalf("creating log dir, %s", err)
	}
	for _, run := range runs {
		err := downloadRunLogs(ctx, client, run.ID, fmt.Sprintf("logs/%s-%d.zip", run.Timestamp.Format("2006_01_02_15_04_05"), run.ID))
		if err != nil {
			log.Printf("failed to download logs for run %d, %s", run.ID, err)
			continue
		}
	}
}

func downloadRunLogs(ctx context.Context, client *github.Client, runID int64, path string) error {
	url, _, err := client.Actions.GetWorkflowRunLogs(ctx, "aws", "karpenter-provider-aws", runID, 20)
	if err != nil {
		return err
	}
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	resp, err := http.Get(url.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func getLastFailedWorkflows(ctx context.Context, client *github.Client, count int) ([]Info, error) {
	const WORKFLOW_NAME = "E2EScaleTrigger"
	const CONCLUSION_FAILURE = "failure"
	const CONCLUSION_SUCCESS = "success"
	ids := []Info{}

	page := 0
	for {
		runs, resp, err := client.Actions.ListRepositoryWorkflowRuns(ctx, "aws", "karpenter-provider-aws", &github.ListWorkflowRunsOptions{
			Event: "schedule",
			ListOptions: github.ListOptions{
				Page: page,
			},
		})
		if err != nil {
			return nil, err
		}
		page = resp.NextPage
		if lo.FromPtr(runs.TotalCount) == 0 || page == 0 {
			break
		}
		ids = append(ids, lo.FilterMap(runs.WorkflowRuns, func(run *github.WorkflowRun, _ int) (Info, bool) {
			if lo.FromPtr(run.Name) == WORKFLOW_NAME && lo.FromPtr(run.Conclusion) == CONCLUSION_SUCCESS {
				return Info{
					ID: lo.FromPtr(run.ID),
					Timestamp: lo.FromPtr(run.CreatedAt).Time,
				}, true
			}
			return Info{}, false
		})...)
		if len(ids) >= count {
			break
		}
	}
	if len(ids) > count {
		return ids[0:count], nil
	}
	return ids, nil
}
