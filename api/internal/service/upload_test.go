package service_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/evadeplayer/api/internal/model"
	"github.com/evadeplayer/api/internal/service"
)

type uploadRecord struct {
	path        string
	contentType string
}

type fakeStorage struct {
	uploadErr error
	uploads   []uploadRecord
	deletes   []string
}

func (f *fakeStorage) Upload(_ context.Context, path string, _ io.Reader, contentType string) error {
	f.uploads = append(f.uploads, uploadRecord{path: path, contentType: contentType})
	return f.uploadErr
}
func (f *fakeStorage) Delete(_ context.Context, path string) error {
	f.deletes = append(f.deletes, path)
	return nil
}

type fakeProducer struct {
	task *model.TranscodeTask
}

func (f *fakeProducer) Enqueue(_ context.Context, task *model.TranscodeTask) error {
	f.task = task
	return nil
}

func newUploadSvc(videos *fakeVideoStore, series *fakeSeriesStore) *service.UploadService {
	return service.NewUploadService(videos, series, &fakeStorage{}, &fakeProducer{})
}

func uploadInput(overrides ...func(*service.UploadInput)) *service.UploadInput {
	in := &service.UploadInput{
		UserID:  "user-1",
		Title:   "My Video",
		FileExt: ".mp4",
		Size:    1024,
		Reader:  strings.NewReader("fake video data"),
	}
	for _, fn := range overrides {
		fn(in)
	}
	return in
}

// --- Upload without series ---

func TestUploadService_Upload_NoSeries(t *testing.T) {
	svc := newUploadSvc(&fakeVideoStore{}, newFakeSeriesStore())

	v, err := svc.Upload(context.Background(), uploadInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ID == "" {
		t.Error("video ID must be set")
	}
	if v.SeriesID != nil {
		t.Error("series_id must be nil when not provided")
	}
	if v.SeasonNumber != nil || v.EpisodeNumber != nil {
		t.Error("season/episode numbers must be nil when not provided")
	}
}

func TestUploadService_Upload_WithPreviewOverride(t *testing.T) {
	videos := &fakeVideoStore{}
	storage := &fakeStorage{}
	producer := &fakeProducer{}
	svc := service.NewUploadService(videos, newFakeSeriesStore(), storage, producer)

	v, err := svc.Upload(context.Background(), uploadInput(func(in *service.UploadInput) {
		in.PreviewReader = strings.NewReader("jpeg")
		in.PreviewContentType = "image/jpeg"
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(storage.uploads) != 2 {
		t.Fatalf("expected original and preview uploads, got %#v", storage.uploads)
	}
	wantPreviewPath := "thumbnails/" + v.ID + "/preview.jpg"
	if storage.uploads[1].path != wantPreviewPath {
		t.Errorf("preview upload path = %q, want %q", storage.uploads[1].path, wantPreviewPath)
	}
	if storage.uploads[1].contentType != "image/jpeg" {
		t.Errorf("preview content type = %q", storage.uploads[1].contentType)
	}
	if producer.task == nil || !producer.task.PreviewOverride {
		t.Fatalf("expected transcode task with PreviewOverride=true, got %#v", producer.task)
	}
}

// --- Upload with valid series ---

func TestUploadService_Upload_WithValidSeries(t *testing.T) {
	seriesStore := newFakeSeriesStore()
	ctx := context.Background()
	s, _ := newSeriesSvc(seriesStore).CreateSeries(ctx, &service.CreateSeriesInput{
		UserID: "user-1",
		Title:  "My Show",
	})

	sn, ep := 2, 3
	svc := newUploadSvc(&fakeVideoStore{}, seriesStore)
	v, err := svc.Upload(ctx, uploadInput(func(in *service.UploadInput) {
		in.SeriesID = &s.ID
		in.SeasonNumber = &sn
		in.EpisodeNumber = &ep
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.SeriesID == nil || *v.SeriesID != s.ID {
		t.Errorf("series_id: got %v, want %q", v.SeriesID, s.ID)
	}
	if v.SeasonNumber == nil || *v.SeasonNumber != 2 {
		t.Errorf("season_number: got %v, want 2", v.SeasonNumber)
	}
	if v.EpisodeNumber == nil || *v.EpisodeNumber != 3 {
		t.Errorf("episode_number: got %v, want 3", v.EpisodeNumber)
	}
}

// --- Upload with non-existent series ---

func TestUploadService_Upload_SeriesNotFound(t *testing.T) {
	svc := newUploadSvc(&fakeVideoStore{}, newFakeSeriesStore())
	nonExistent := "00000000-0000-0000-0000-000000000000"

	_, err := svc.Upload(context.Background(), uploadInput(func(in *service.UploadInput) {
		in.SeriesID = &nonExistent
	}))
	if !errors.Is(err, service.ErrSeriesNotFound) {
		t.Errorf("expected ErrSeriesNotFound, got %v", err)
	}
}

// --- Upload with series belonging to another user ---

func TestUploadService_Upload_SeriesWrongUser(t *testing.T) {
	seriesStore := newFakeSeriesStore()
	ctx := context.Background()
	s, _ := newSeriesSvc(seriesStore).CreateSeries(ctx, &service.CreateSeriesInput{
		UserID: "owner-user",
		Title:  "Owner's Show",
	})

	svc := newUploadSvc(&fakeVideoStore{}, seriesStore)
	_, err := svc.Upload(ctx, uploadInput(func(in *service.UploadInput) {
		in.UserID = "other-user"
		in.SeriesID = &s.ID
	}))
	if !errors.Is(err, service.ErrSeriesNotFound) {
		t.Errorf("expected ErrSeriesNotFound for wrong user, got %v", err)
	}
}

// --- Version uploads ---

func TestUploadService_Upload_WithVersion(t *testing.T) {
	videos := &fakeVideoStore{}
	ctx := context.Background()

	// create a primary video in the store
	primaryID := "primary-123"
	videos.video = &model.Video{ID: primaryID, UserID: "user-1", Status: model.StatusReady}

	label := "RU dub"
	svc := newUploadSvc(videos, newFakeSeriesStore())
	v, err := svc.Upload(ctx, uploadInput(func(in *service.UploadInput) {
		in.VersionOf = &primaryID
		in.VersionLabel = &label
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.VersionOf == nil || *v.VersionOf != primaryID {
		t.Errorf("version_of: got %v, want %q", v.VersionOf, primaryID)
	}
	if v.VersionLabel == nil || *v.VersionLabel != "RU dub" {
		t.Errorf("version_label: got %v, want %q", v.VersionLabel, "RU dub")
	}
	// series fields must be cleared
	if v.SeriesID != nil || v.SeasonNumber != nil || v.EpisodeNumber != nil {
		t.Error("series fields must be nil when version_of is set")
	}
}

func TestUploadService_Upload_VersionParentNotFound(t *testing.T) {
	svc := newUploadSvc(&fakeVideoStore{}, newFakeSeriesStore())
	nonExistent := "no-such-video"

	_, err := svc.Upload(context.Background(), uploadInput(func(in *service.UploadInput) {
		in.VersionOf = &nonExistent
	}))
	if !errors.Is(err, service.ErrVideoNotFound) {
		t.Errorf("expected ErrVideoNotFound, got %v", err)
	}
}

func TestUploadService_Upload_VersionChainingRejected(t *testing.T) {
	videos := &fakeVideoStore{}
	parentID := "parent-id"
	alreadyVersionOf := "grandparent-id"
	// parent is itself a version (version_of is set)
	videos.video = &model.Video{
		ID:        parentID,
		UserID:    "user-1",
		VersionOf: &alreadyVersionOf,
	}

	svc := newUploadSvc(videos, newFakeSeriesStore())
	_, err := svc.Upload(context.Background(), uploadInput(func(in *service.UploadInput) {
		in.VersionOf = &parentID
	}))
	if !errors.Is(err, service.ErrVersionChaining) {
		t.Errorf("expected ErrVersionChaining, got %v", err)
	}
}

// --- Storage failure ---

func TestUploadService_Upload_StorageError(t *testing.T) {
	videos := &fakeVideoStore{}
	svc := service.NewUploadService(videos, newFakeSeriesStore(), &fakeStorage{uploadErr: errors.New("disk full")}, &fakeProducer{})

	_, err := svc.Upload(context.Background(), uploadInput())
	if err == nil {
		t.Error("expected error on storage failure")
	}
}
