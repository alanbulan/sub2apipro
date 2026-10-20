package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/feishu"
)

const feishuDirectoryJobKey = "directory:active"
const feishuDirectoryTimeout = 15 * time.Minute

var ErrFeishuSetupStateNotFound = errors.New("feishu state not found")

type FeishuDirectoryJob struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Total     int    `json:"total"`
	Processed int    `json:"processed"`
	Created   int    `json:"created"`
	Updated   int    `json:"updated"`
	Skipped   int    `json:"skipped"`
	Failed    int    `json:"failed"`
	ErrorCode string `json:"error_code,omitempty"`
	StartedAt int64  `json:"started_at"`
	Deadline  int64  `json:"deadline"`
}

func (j *FeishuDirectoryJob) Running() bool {
	return j.Status == "preparing" || j.Status == "reading" || j.Status == "syncing"
}

type FeishuDirectoryService struct {
	auth  *AuthService
	setup *FeishuSetupService
}

func NewFeishuDirectoryService(auth *AuthService, setup *FeishuSetupService) *FeishuDirectoryService {
	return &FeishuDirectoryService{auth: auth, setup: setup}
}

func (s *FeishuDirectoryService) Current(ctx context.Context) (*FeishuDirectoryJob, error) {
	if s == nil || s.setup == nil || s.setup.store == nil {
		return nil, ErrServiceUnavailable
	}
	data, err := s.setup.store.Get(ctx, feishuDirectoryJobKey)
	if errors.Is(err, ErrFeishuSetupStateNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, ErrServiceUnavailable
	}
	if len(data) == 0 {
		return nil, nil
	}
	var job FeishuDirectoryJob
	if json.Unmarshal(data, &job) != nil || job.ID == "" {
		return nil, ErrServiceUnavailable
	}
	if job.Running() && job.Deadline <= time.Now().Unix() {
		job.Status, job.ErrorCode = "failed", "expired"
	}
	return &job, nil
}

func (s *FeishuDirectoryService) Start(ctx context.Context) (*FeishuDirectoryJob, error) {
	if s == nil || s.auth == nil || s.setup == nil || s.setup.store == nil {
		return nil, ErrServiceUnavailable
	}
	release, err := s.setup.Lock(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if release != nil {
			release()
		}
	}()
	if _, err := s.setup.settings.GetFeishuOAuthConfig(ctx); err != nil {
		return nil, err
	}
	current, err := s.Current(ctx)
	if err != nil {
		return nil, err
	}
	if current != nil && current.Running() {
		return current, nil
	}
	id, err := randomHexString(24)
	if err != nil {
		return nil, ErrServiceUnavailable
	}
	job := &FeishuDirectoryJob{ID: id, Status: "preparing", StartedAt: time.Now().Unix(), Deadline: time.Now().Add(feishuDirectoryTimeout).Unix()}
	if err := s.save(ctx, job); err != nil {
		return nil, err
	}
	// A copy keeps the HTTP response independent of progress writes. All admin
	// sessions observe the same persisted task; disconnecting does not stop it.
	work := *job
	release()
	release = nil
	go s.run(&work)
	return job, nil
}

func (s *FeishuDirectoryService) save(ctx context.Context, job *FeishuDirectoryJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return s.setup.store.Set(ctx, feishuDirectoryJobKey, data, time.Hour)
}

func (s *FeishuDirectoryService) persist(ctx context.Context, job *FeishuDirectoryJob) error {
	store, ok := s.setup.store.(interface {
		SaveDirectoryProgress(context.Context, string, []byte, time.Duration) (bool, error)
	})
	if !ok {
		return ErrServiceUnavailable
	}
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	updated, err := store.SaveDirectoryProgress(ctx, job.ID, data, time.Hour)
	if err != nil {
		return err
	}
	if !updated {
		return infraerrors.Conflict("FEISHU_SYNC_REPLACED", "The directory task was replaced")
	}
	return nil
}

func (s *FeishuDirectoryService) run(job *FeishuDirectoryJob) {
	ctx, cancel := context.WithTimeout(context.Background(), feishuDirectoryTimeout)
	defer cancel()
	defer func() {
		if recover() != nil {
			job.Status, job.ErrorCode = "failed", "service_error"
		}
		persistCtx, persistCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer persistCancel()
		_ = s.persist(persistCtx, job)
	}()
	if err := s.execute(ctx, job); err != nil {
		job.Status, job.ErrorCode = "failed", "service_error"
		var upstream *feishu.Error
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			job.ErrorCode = "expired"
		case infraerrors.Reason(err) == "FEISHU_CONFIG_CHANGED":
			job.ErrorCode = "config_changed"
		case errors.As(err, &upstream):
			job.ErrorCode = "upstream_error"
			if upstream.Code == "210040" || upstream.Code == "99991672" || upstream.Code == "99991671" || upstream.Code == "40004" || upstream.Code == "41050" || upstream.Code == "missing_member_fields" {
				job.Status, job.ErrorCode = "approval_required", "permission_required"
			}
		}
	}
}

func (s *FeishuDirectoryService) execute(ctx context.Context, job *FeishuDirectoryJob) error {
	// Upgrading permissions uses the same lock as QR setup and enable/disable.
	cfg, token, err := s.prepare(ctx)
	if err != nil {
		return err
	}
	job.Status = "reading"
	if err := s.persist(ctx, job); err != nil {
		return err
	}
	members, err := s.setup.api.DirectoryMembers(ctx, token)
	if err != nil {
		return err
	}
	job.Total, job.Status = len(members), "syncing"
	if err := s.persist(ctx, job); err != nil {
		return err
	}
	for _, member := range members {
		if err := ctx.Err(); err != nil {
			return err
		}
		current, err := s.setup.settings.GetFeishuOAuthConfig(ctx)
		if err != nil || current.Fingerprint() != cfg.Fingerprint() {
			return infraerrors.Conflict("FEISHU_CONFIG_CHANGED", "Feishu configuration changed during synchronization")
		}
		job.Processed++
		if !member.Active() {
			job.Skipped++
		} else {
			info := &feishu.UserInfo{OpenID: member.OpenID, TenantKey: cfg.TenantKey, Name: member.Name, AvatarURL: member.Avatar.URL, Email: member.Email, EnterpriseEmail: member.EnterpriseEmail}
			_, created, err := s.auth.ensureFeishuEnterpriseUser(ctx, cfg, info, true)
			switch {
			case errors.Is(err, ErrUserNotActive):
				job.Skipped++
			case err != nil:
				job.Failed++
			case created:
				job.Created++
			default:
				job.Updated++
			}
		}
		if err := s.persist(ctx, job); err != nil {
			return err
		}
	}
	job.Status = "succeeded"
	if job.Failed > 0 {
		job.Status = "partial"
	}
	return nil
}

func (s *FeishuDirectoryService) prepare(ctx context.Context) (*FeishuConnectSettings, string, error) {
	release, err := s.setup.Lock(ctx)
	if err != nil {
		return nil, "", err
	}
	defer release()
	cfg, err := s.setup.settings.GetFeishuOAuthConfig(ctx)
	if err != nil {
		return nil, "", err
	}
	token, err := s.setup.api.TenantToken(ctx, cfg.AppID, cfg.AppSecret)
	if err != nil {
		return nil, "", err
	}
	if err := s.setup.api.ConfigureDirectoryRead(ctx, token, cfg.AppID); err != nil {
		return nil, "", err
	}
	if err := s.setup.api.Publish(ctx, token, cfg.AppID); err != nil {
		return nil, "", err
	}
	return cfg, strings.TrimSpace(token), nil
}
