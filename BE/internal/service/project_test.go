package service

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
	"github.com/metaforismo/sprintorio/BE/internal/dto"
	"github.com/metaforismo/sprintorio/BE/internal/repository"
	"github.com/stretchr/testify/require"
	"testing"
)

type projectMetadataFake struct {
	repository.ProjectRepo
	project         *domain.Project
	err             error
	validReferences bool
	writes          int
}

func (f *projectMetadataFake) GetByID(context.Context, uuid.UUID) (*domain.Project, error) {
	return f.project, f.err
}
func (f *projectMetadataFake) Create(_ context.Context, p *domain.Project) error {
	f.writes++
	f.project = p
	return f.err
}
func (f *projectMetadataFake) Update(_ context.Context, p *domain.Project) error {
	f.writes++
	f.project = p
	return f.err
}
func (f *projectMetadataFake) ValidateReferences(context.Context, uuid.UUID, *uuid.UUID, *uuid.UUID) (bool, error) {
	return f.validReferences, f.err
}

func TestProjectMetadataDatesAndNullablePatch(t *testing.T) {
	ctx := context.Background()
	ws := uuid.New()
	team := uuid.New().String()
	lead := uuid.New().String()
	start := "2026-10-02"
	target := "2026-10-10"
	repo := &projectMetadataFake{validReferences: true}
	svc := NewProjectService(repo)
	p, err := svc.Create(ctx, ws, dto.CreateProjectRequest{Name: " Launch ", TeamID: &team, LeadID: &lead, StartDate: &start, TargetDate: &target})
	require.NoError(t, err)
	require.Equal(t, "Launch", p.Name)
	require.Equal(t, start, p.StartDate.Format("2006-01-02"))
	require.Equal(t, target, p.TargetDate.Format("2006-01-02"))
	var req dto.UpdateProjectRequest
	require.NoError(t, json.Unmarshal([]byte(`{"name":"Renamed"}`), &req))
	p, err = svc.Update(ctx, ws, p.ID, req)
	require.NoError(t, err)
	require.NotNil(t, p.TeamID)
	require.NotNil(t, p.StartDate)
	require.NoError(t, json.Unmarshal([]byte(`{"team_id":null,"lead_id":null,"start_date":null,"target_date":null}`), &req))
	p, err = svc.Update(ctx, ws, p.ID, req)
	require.NoError(t, err)
	require.Nil(t, p.TeamID)
	require.Nil(t, p.LeadID)
	require.Nil(t, p.StartDate)
	require.Nil(t, p.TargetDate)
}
func TestProjectMetadataRejectsInvalidChangesWithoutWriting(t *testing.T) {
	ws := uuid.New()
	for _, body := range []string{`{"name":" "}`, `{"name":"<b></b>"}`, `{"status":"bogus"}`, `{"team_id":"bad"}`, `{"lead_id":"00000000-0000-0000-0000-000000000000"}`, `{"start_date":"2026-02-30"}`, `{"start_date":"2026-1-2"}`, `{"start_date":"2026-10-03","target_date":"2026-10-02"}`} {
		t.Run(body, func(t *testing.T) {
			repo := &projectMetadataFake{project: &domain.Project{ID: uuid.New(), WorkspaceID: ws, Name: "Original"}, validReferences: true}
			var req dto.UpdateProjectRequest
			require.NoError(t, json.Unmarshal([]byte(body), &req))
			_, err := NewProjectService(repo).Update(context.Background(), ws, repo.project.ID, req)
			require.ErrorIs(t, err, ErrInvalidProject)
			require.Zero(t, repo.writes)
			require.Equal(t, "Original", repo.project.Name)
		})
	}
	repo := &projectMetadataFake{project: &domain.Project{ID: uuid.New(), WorkspaceID: ws}}
	id := uuid.New().String()
	_, err := NewProjectService(repo).Update(context.Background(), ws, repo.project.ID, dto.UpdateProjectRequest{TeamID: dto.OptionalString{Set: true, Value: &id}})
	require.ErrorIs(t, err, ErrInvalidProject)
	require.Zero(t, repo.writes)
	_, err = NewProjectService(repo).Update(context.Background(), uuid.New(), repo.project.ID, dto.UpdateProjectRequest{})
	require.ErrorIs(t, err, ErrProjectNotFound)
	repo.err = errors.New("database unavailable")
	_, err = NewProjectService(repo).Update(context.Background(), ws, repo.project.ID, dto.UpdateProjectRequest{})
	require.ErrorIs(t, err, repo.err)
}

func TestProjectMetadataPatchDoesNotRevalidateUnchangedDepartedLead(t *testing.T) {
	ws, id, formerLead := uuid.New(), uuid.New(), uuid.New()
	repo := &projectMetadataFake{project: &domain.Project{ID: id, WorkspaceID: ws, Name: "Original", LeadID: &formerLead}, validReferences: false}
	svc := NewProjectService(repo)
	var rename dto.UpdateProjectRequest
	require.NoError(t, json.Unmarshal([]byte(`{"name":"Renamed","status":"in_progress"}`), &rename))
	updated, err := svc.Update(context.Background(), ws, id, rename)
	require.NoError(t, err)
	require.Equal(t, "Renamed", updated.Name)
	require.Equal(t, domain.ProjectStatusInProgress, updated.Status)
	require.Equal(t, &formerLead, updated.LeadID)
	// Explicitly assigning the same departed lead is a new association request,
	// so it must still fail membership validation without another write.
	lead := formerLead.String()
	_, err = svc.Update(context.Background(), ws, id, dto.UpdateProjectRequest{LeadID: dto.OptionalString{Set: true, Value: &lead}})
	require.ErrorIs(t, err, ErrInvalidProject)
	require.Equal(t, 1, repo.writes)
	// Clearing that association repairs it without requiring former membership.
	updated, err = svc.Update(context.Background(), ws, id, dto.UpdateProjectRequest{LeadID: dto.OptionalString{Set: true}})
	require.NoError(t, err)
	require.Nil(t, updated.LeadID)
	require.Equal(t, 2, repo.writes)
	// Creation still validates every supplied reference.
	_, err = svc.Create(context.Background(), ws, dto.CreateProjectRequest{Name: "New", LeadID: &lead})
	require.ErrorIs(t, err, ErrInvalidProject)
	require.Equal(t, 2, repo.writes)
}
