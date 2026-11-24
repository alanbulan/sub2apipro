//go:build integration

package repository

import (
	"fmt"

	"github.com/Wei-Shaw/sub2api/ent/authidentity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/authguard"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (s *UserRepoSuite) TestFeishuEmployeesShareOfficeIPWithoutEmailIdentities() {
	publicCtx := authguard.WithRegistrationIP(s.ctx, "192.0.2.40")
	enterpriseCtx := authguard.WithoutRegistrationIP(publicCtx)
	for i := 0; i < 3; i++ {
		user := registrationIPTestUser(fmt.Sprintf("employee%d%s", i, service.FeishuConnectSyntheticEmailDomain))
		user.SignupSource = "feishu"
		s.Require().NoError(s.repo.Create(enterpriseCtx, user))
	}
	count, err := s.client.AuthIdentity.Query().Where(authidentity.ProviderTypeEQ("email")).Count(s.ctx)
	s.Require().NoError(err)
	s.Require().Zero(count, "Feishu identifiers must not become email authentication identities")
	// The enterprise exemption must not weaken public signup from the same IP.
	s.Require().NoError(s.repo.Create(publicCtx, registrationIPTestUser("public1@example.com")))
	s.Require().NoError(s.repo.Create(publicCtx, registrationIPTestUser("public2@example.com")))
	s.Require().ErrorIs(s.repo.Create(publicCtx, registrationIPTestUser("public3@example.com")), service.ErrRegistrationIPLimit)
}
