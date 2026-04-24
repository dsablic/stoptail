package es

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func (c *Client) FetchTasks(ctx context.Context) ([]TaskInfo, error) {
	res, err := c.es.Tasks.List(
		c.es.Tasks.List.WithContext(ctx),
		c.es.Tasks.List.WithDetailed(true),
		c.es.Tasks.List.WithActions("*reindex*", "*byquery*", "*forcemerge*", "*snapshot*"),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching tasks: %w", err)
	}
	defer res.Body.Close()

	body, err := readBody(res, "tasks")
	if err != nil {
		return nil, err
	}

	return parseTasksResponse(body)
}

func (c *Client) CancelTask(ctx context.Context, taskID string) error {
	res, err := c.es.Tasks.Cancel(
		c.es.Tasks.Cancel.WithContext(ctx),
		c.es.Tasks.Cancel.WithTaskID(taskID),
	)
	if err != nil {
		return fmt.Errorf("cancelling task: %w", err)
	}
	defer res.Body.Close()

	return checkError(res)
}

func parseTasksResponse(data []byte) ([]TaskInfo, error) {
	var response struct {
		Nodes map[string]struct {
			Name  string `json:"name"`
			Tasks map[string]struct {
				Node             string `json:"node"`
				ID               int64  `json:"id"`
				Action           string `json:"action"`
				Description      string `json:"description"`
				RunningTimeNanos int64  `json:"running_time_in_nanos"`
				Cancellable      bool   `json:"cancellable"`
				ParentTaskID     string `json:"parent_task_id"`
			} `json:"tasks"`
		} `json:"nodes"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parsing tasks response: %w", err)
	}

	actionPrefixes := []string{
		"indices:data/write/reindex",
		"indices:data/write/update/byquery",
		"indices:data/write/delete/byquery",
		"indices:admin/forcemerge",
		"cluster:admin/snapshot",
	}

	isTargetAction := func(action string) bool {
		for _, prefix := range actionPrefixes {
			if strings.HasPrefix(action, prefix) {
				return true
			}
		}
		return false
	}

	var tasks []TaskInfo
	for _, nodeData := range response.Nodes {
		for taskID, task := range nodeData.Tasks {
			if !isTargetAction(task.Action) || task.ParentTaskID != "" {
				continue
			}

			runningMs := task.RunningTimeNanos / 1_000_000
			tasks = append(tasks, TaskInfo{
				ID:            taskID,
				Action:        task.Action,
				Node:          nodeData.Name,
				Description:   task.Description,
				RunningTime:   formatDuration(runningMs),
				RunningTimeMs: runningMs,
				Cancellable:   task.Cancellable,
			})
		}
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].RunningTimeMs > tasks[j].RunningTimeMs
	})

	return tasks, nil
}

func (c *Client) FetchPendingTasks(ctx context.Context) ([]PendingTask, error) {
	res, err := c.es.Cluster.PendingTasks(
		c.es.Cluster.PendingTasks.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching pending tasks: %w", err)
	}
	defer res.Body.Close()

	body, err := readBody(res, "pending tasks")
	if err != nil {
		return nil, err
	}

	var response struct {
		Tasks []PendingTask `json:"tasks"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parsing pending tasks: %w", err)
	}

	return response.Tasks, nil
}
