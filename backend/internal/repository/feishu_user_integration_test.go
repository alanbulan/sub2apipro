//go:build integration

package repository

import (
	"fmt"

	"github.com/Wei-Shaw/sub2api/ent/authidentity"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (s *UserRepoSuite) TestFeishuPlaceholdersDoNotBecomeEmailAuthenticationIdentities() {
	for i := 0; i < 3; i++ {
		user := s.mustCreateUser(&service.User{
			Email:        fmt.Sprintf("employee%d%s", i, service.FeishuConnectSyntheticEmailDomain),
			SignupSource: "feishu",
		})
		stored, err := s.repo.GetByID(s.ctx, user.ID)
		s.Require().NoError(err)
		s.Require().Equal("feishu", stored.SignupSource)
		s.Require().Equal(user.Email, stored.Email)
	}
	count, err := s.client.AuthIdentity.Query().Where(authidentity.ProviderTypeEQ("email")).Count(s.ctx)
	s.Require().NoError(err)
	s.Require().Zero(count, "Feishu identifiers must not become email authentication identities")
	// Canonical email registration still creates its own authentication identity.
	publicUser := s.mustCreateUser(&service.User{Email: "public@example.com"})
	identity, err := s.client.AuthIdentity.Query().Where(
		authidentity.ProviderTypeEQ("email"),
		authidentity.ProviderSubjectEQ(publicUser.Email),
	).Only(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(publicUser.ID, identity.UserID)
}
