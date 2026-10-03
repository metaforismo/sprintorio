package agentclient

import (
	"reflect"
	"strings"

	"github.com/metaforismo/sprintorio/BE/internal/domain"
	"github.com/metaforismo/sprintorio/BE/internal/dto"
)

// Response field names come from known public DTOs. Dynamic objects use a separate
// credential-filtered projection; arbitrary API fields are not exposed by default.
func init() {
	values := []any{dto.IssueResponse{}, dto.ProjectResponse{}, dto.TeamResponse{}, dto.TeamStatusResponse{}, dto.CycleResponse{}, dto.LabelResponse{}, dto.IssueTemplateResponse{}, dto.CommentResponse{}, dto.IssueRelationResponse{}, dto.IssueHistoryResponse{}, dto.WorkspaceResponse{}, dto.WorkspaceMemberResponse{}, dto.ViewResponse{}, dto.WebhookResponse{}, dto.AISettingsResponse{}, dto.FavoriteResponse{}, dto.GitHubStatusResponse{}, dto.GitHubRepoResponse{}, dto.GitHubAutoTransitionResponse{}, dto.GitHubIssueActivityResponse{}, dto.AnalyticsOverview{}, dto.AnalyticsIssueDistribution{}, dto.AnalyticsInsightsResponse{}, dto.AnalyticsBurnupResponse{}, dto.AgentRunTraceResponse{}, dto.DevMachineOperationResponse{}, dto.TerminalSessionResponse{}, dto.AgentProviderResponse{}, domain.DevMachine{}}
	seen := map[reflect.Type]bool{}
	var collect func(reflect.Type)
	collect = func(t reflect.Type) {
		for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct || seen[t] {
			return
		}
		seen[t] = true
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			name := strings.Split(f.Tag.Get("json"), ",")[0]
			if name == "" || name == "-" || secretField(name) {
				continue
			}
			safeFields[name] = true
			collect(f.Type)
		}
	}
	for _, v := range values {
		collect(reflect.TypeOf(v))
	}
	for _, field := range strings.Fields("operation retry_after_seconds next_event_id next_log_id has_more_events has_more_logs run logs events status_info role current_user_role count open_issues completed_issues total_projects total_members started_issues overdue_issues unassigned_issues completion_rate by_status by_priority") {
		safeFields[field] = true
		summaryFields[field] = true
	}
}
func secretField(name string) bool {
	s := strings.ToLower(name)
	return strings.Contains(s, "token") || strings.Contains(s, "secret") || strings.Contains(s, "password") || strings.Contains(s, "encrypted") || s == "api_key" || s == "config" || s == "env_vars" || s == "launch_url" || s == "web_socket_url"
}
