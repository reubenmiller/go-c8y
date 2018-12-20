package c8y

// MicroserviceService api
type MicroserviceService service

// SetServiceUsers sets the service users which can then be used later for following requests
// The service users are retrieved by using the bootstrap credentials stored in environment variables
func (s *MicroserviceService) SetServiceUsers() error {
	c := s.client
	serviceUsers, err := c.Microservice.GetServiceUsers()

	if err != nil {
		return err
	}

	c.clientMu.Lock()
	c.ServiceUsers = serviceUsers.Users
	c.clientMu.Unlock()
	return nil
}

// GetServiceUsers returns a list of the subscriped tenant where the application is running
// along with the service user subscriptions for each tenant
func (s *MicroserviceService) GetServiceUsers() (*ApplicationSubscriptions, error) {
	ctx := s.client.Context.BootstrapUserFromEnvironment()
	resp, _, err := s.client.Application.GetCurrentApplicationSubscriptions(ctx)
	return resp, err
}
