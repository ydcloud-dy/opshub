package asset

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeHostRepo struct {
	byID           map[uint]*Host
	byAgentID      map[string]*Host
	byInstallToken map[string]*Host
	updates        int
}

func (r *fakeHostRepo) Create(ctx context.Context, host *Host) error {
	return nil
}

func (r *fakeHostRepo) CreateOrUpdate(ctx context.Context, host *Host) error {
	return nil
}

func (r *fakeHostRepo) Update(ctx context.Context, host *Host) error {
	r.updates++
	if r.byID == nil {
		r.byID = make(map[uint]*Host)
	}
	r.byID[host.ID] = host
	if host.AgentID != "" {
		if r.byAgentID == nil {
			r.byAgentID = make(map[string]*Host)
		}
		r.byAgentID[host.AgentID] = host
	}
	if host.AgentInstallTokenHash != "" {
		if r.byInstallToken == nil {
			r.byInstallToken = make(map[string]*Host)
		}
		r.byInstallToken[host.AgentInstallTokenHash] = host
	}
	return nil
}

func (r *fakeHostRepo) Delete(ctx context.Context, id uint) error {
	return nil
}

func (r *fakeHostRepo) GetByID(ctx context.Context, id uint) (*Host, error) {
	host, ok := r.byID[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return host, nil
}

func (r *fakeHostRepo) List(ctx context.Context, page, pageSize int, keyword string, groupIDs []uint, accessibleHostIDs []uint, status *int, collectMode, agentStatus string) ([]*Host, int64, error) {
	return nil, 0, nil
}

func (r *fakeHostRepo) GetByGroupID(ctx context.Context, groupID uint) ([]*Host, error) {
	return nil, nil
}

func (r *fakeHostRepo) GetByIP(ctx context.Context, ip string) (*Host, error) {
	return nil, errors.New("not found")
}

func (r *fakeHostRepo) GetByCloudInstanceID(ctx context.Context, instanceID string) (*Host, error) {
	return nil, errors.New("not found")
}

func (r *fakeHostRepo) GetByAgentID(ctx context.Context, agentID string) (*Host, error) {
	host, ok := r.byAgentID[agentID]
	if !ok {
		return nil, errors.New("not found")
	}
	return host, nil
}

func (r *fakeHostRepo) GetByAgentInstallTokenHash(ctx context.Context, tokenHash string) (*Host, error) {
	host, ok := r.byInstallToken[tokenHash]
	if !ok {
		return nil, errors.New("not found")
	}
	return host, nil
}

func (r *fakeHostRepo) CountByCredentialID(ctx context.Context, credentialID uint) (int64, error) {
	return 0, nil
}

func TestCloudHostInfoIsSSHOnlyAndHidesStaleAgentState(t *testing.T) {
	now := time.Now()
	host := &Host{
		Type:                       "cloud",
		CloudProvider:              "aliyun",
		CloudInstanceID:            "i-test",
		AgentID:                    "agt_stale",
		AgentVersion:               "0.1.0",
		AgentStatus:                "online",
		AgentLastSeen:              &now,
		AgentLastCollectAt:         &now,
		AgentInstallTokenHash:      "stale",
		AgentInstallTokenExpiresAt: &now,
	}

	vo := (&HostUseCase{}).toInfoVO(host)

	if vo.CollectMode != "ssh" || vo.CollectModeText != "SSH采集" {
		t.Fatalf("expected cloud host collect mode ssh, got %s/%s", vo.CollectMode, vo.CollectModeText)
	}
	if vo.AgentSupported {
		t.Fatal("expected cloud host to be unsupported by agent")
	}
	if vo.AgentDisabledReason != cloudHostAgentDisabledReason {
		t.Fatalf("unexpected disabled reason: %s", vo.AgentDisabledReason)
	}
	if vo.AgentStatusText != "仅SSH采集" {
		t.Fatalf("expected ssh-only status text, got %s", vo.AgentStatusText)
	}
	if vo.AgentID != "" || vo.AgentVersion != "" || vo.AgentStatus != "" || vo.AgentLastSeen != "" || vo.AgentLastCollectAt != "" {
		t.Fatalf("expected stale agent fields hidden, got id=%q version=%q status=%q lastSeen=%q lastCollect=%q",
			vo.AgentID, vo.AgentVersion, vo.AgentStatus, vo.AgentLastSeen, vo.AgentLastCollectAt)
	}
}

func TestCreateAgentInstallCommandRejectsCloudHostAndClearsAgentState(t *testing.T) {
	now := time.Now()
	host := &Host{
		Type:                       "cloud",
		CloudProvider:              "tencent",
		AgentID:                    "agt_stale",
		AgentVersion:               "0.1.0",
		AgentStatus:                "online",
		AgentLastSeen:              &now,
		AgentLastCollectAt:         &now,
		AgentTokenHash:             "token",
		AgentInstallTokenHash:      "install-token",
		AgentInstallTokenExpiresAt: &now,
	}
	host.ID = 7
	repo := &fakeHostRepo{byID: map[uint]*Host{host.ID: host}}
	uc := NewHostUseCase(repo, nil, nil, nil)

	_, err := uc.CreateAgentInstallCommand(context.Background(), host.ID, "https://opshub.example.com", 30)

	if err == nil || !strings.Contains(err.Error(), "仅支持SSH采集") {
		t.Fatalf("expected ssh-only error, got %v", err)
	}
	if repo.updates != 1 {
		t.Fatalf("expected stale agent state update, got %d updates", repo.updates)
	}
	assertAgentStateCleared(t, host)
}

func TestRegisterAgentRejectsCloudHostWithStaleInstallToken(t *testing.T) {
	now := time.Now()
	token := "enroll-token"
	host := &Host{
		Type:                       "cloud",
		CloudAccountID:             100,
		AgentInstallTokenHash:      hashSecret(token),
		AgentInstallTokenExpiresAt: &now,
		AgentStatus:                "pending",
	}
	host.ID = 8
	repo := &fakeHostRepo{
		byID:           map[uint]*Host{host.ID: host},
		byInstallToken: map[string]*Host{hashSecret(token): host},
	}
	uc := NewHostUseCase(repo, nil, nil, nil)

	_, err := uc.RegisterAgent(context.Background(), &AgentRegisterRequest{EnrollmentToken: token})

	if err == nil || !strings.Contains(err.Error(), "仅支持SSH采集") {
		t.Fatalf("expected ssh-only error, got %v", err)
	}
	if repo.updates != 1 {
		t.Fatalf("expected stale install token cleanup, got %d updates", repo.updates)
	}
	assertAgentStateCleared(t, host)
}

func TestAuthenticateAgentRejectsCloudHostAndClearsStaleBinding(t *testing.T) {
	agentToken := "agent-token"
	host := &Host{
		Type:           "cloud",
		AgentID:        "agt_stale",
		AgentTokenHash: hashSecret(agentToken),
		AgentStatus:    "online",
	}
	host.ID = 9
	repo := &fakeHostRepo{
		byID:      map[uint]*Host{host.ID: host},
		byAgentID: map[string]*Host{host.AgentID: host},
	}
	uc := NewHostUseCase(repo, nil, nil, nil)

	_, err := uc.authenticateAgent(context.Background(), host.AgentID, agentToken)

	if err == nil || !strings.Contains(err.Error(), "仅支持SSH采集") {
		t.Fatalf("expected ssh-only error, got %v", err)
	}
	if repo.updates != 1 {
		t.Fatalf("expected stale agent binding cleanup, got %d updates", repo.updates)
	}
	assertAgentStateCleared(t, host)
}

func assertAgentStateCleared(t *testing.T, host *Host) {
	t.Helper()
	if host.AgentID != "" ||
		host.AgentVersion != "" ||
		host.AgentStatus != "" ||
		host.AgentLastSeen != nil ||
		host.AgentLastCollectAt != nil ||
		host.AgentTokenHash != "" ||
		host.AgentInstallTokenHash != "" ||
		host.AgentInstallTokenExpiresAt != nil {
		t.Fatalf("expected agent state cleared, got %+v", host)
	}
}
