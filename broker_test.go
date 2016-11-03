package main

import (
	"context"

	"code.cloudfoundry.org/lager/lagertest"
	"github.com/pivotal-cf/brokerapi"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

type FakeUAAClient struct {
	mock.Mock
	userGUID string
}

func (c *FakeUAAClient) GetUser(userID string) (User, error) {
	c.Called(userID)
	return User{ID: c.userGUID}, nil
}

func (c *FakeUAAClient) CreateUser(user User) (User, error) {
	c.Called(user)
	return User{ID: c.userGUID}, nil
}

func (c *FakeUAAClient) DeleteUser(userID string) error {
	c.Called(userID)
	return nil
}

type FakeCFClient struct {
	mock.Mock
}

func (c *FakeCFClient) CreateUser(userID string) error {
	c.Called(userID)
	return nil
}

func (c *FakeCFClient) DeleteUser(userID string) error {
	c.Called(userID)
	return nil
}

func (c *FakeCFClient) AddUserToOrg(userID, orgID string) error {
	c.Called(userID, orgID)
	return nil
}

func (c *FakeCFClient) AddUserToSpace(userID, spaceID string) error {
	c.Called(userID, spaceID)
	return nil
}

type FakeCredentialSender struct {
	mock.Mock
	link string
}

func (s FakeCredentialSender) Send(message string) (string, error) {
	s.Called(message)
	return s.link, nil
}

var _ = Describe("broker", func() {
	var (
		uaaClient        FakeUAAClient
		cfClient         FakeCFClient
		credentialSender FakeCredentialSender
		broker           DeployerAccountBroker
	)

	BeforeEach(func() {
		uaaClient = FakeUAAClient{userGUID: "user-guid"}
		cfClient = FakeCFClient{}
		credentialSender = FakeCredentialSender{link: "https://fugacio.us/m/42"}
		broker = DeployerAccountBroker{
			uaaClient:        &uaaClient,
			cfClient:         &cfClient,
			credentialSender: &credentialSender,
			logger:           lagertest.NewTestLogger("broker-test"),
			generatePassword: func(int) string {
				return "password"
			},
			config: Config{
				EmailAddress:   "fake@fake.org",
				PasswordLength: 32,
			},
		}
	})

	Describe("provision", func() {
		It("returns a provision service spec", func() {
			credentialSender.On("Send", "instance-guid | password")
			uaaClient.On("CreateUser", User{
				UserName: "instance-guid",
				Password: "password",
				Emails: []Email{{
					Value:   "fake@fake.org",
					Primary: true,
				}},
			}).Return(User{ID: "user-guid"}, nil)
			cfClient.On("CreateUser", "user-guid").Return(User{ID: "user-guid"}, nil)
			cfClient.On("AddUserToOrg", "user-guid", "org-guid").Return(nil)
			cfClient.On("AddUserToSpace", "user-guid", "space-guid").Return(nil)

			spec, err := broker.Provision(
				context.Background(),
				"instance-guid",
				brokerapi.ProvisionDetails{
					OrganizationGUID: "org-guid",
					SpaceGUID:        "space-guid",
				},
				false,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(spec.IsAsync).To(Equal(false))
			Expect(spec.DashboardURL).To(Equal("https://fugacio.us/m/42"))

			credentialSender.AssertExpectations(GinkgoT())
			uaaClient.AssertExpectations(GinkgoT())
			cfClient.AssertExpectations(GinkgoT())
		})
	})

	Describe("deprovision", func() {
		It("returns a deprovision service spec", func() {
			uaaClient.On("GetUser", "instance-guid").Return(User{ID: "user-guid"}, nil)
			uaaClient.On("DeleteUser", "user-guid").Return(nil)
			cfClient.On("DeleteUser", "user-guid").Return(nil)

			spec, err := broker.Deprovision(
				context.Background(),
				"instance-guid",
				brokerapi.DeprovisionDetails{},
				false,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(spec.IsAsync).To(Equal(false))

			uaaClient.AssertExpectations(GinkgoT())
			cfClient.AssertExpectations(GinkgoT())
		})
	})
})
