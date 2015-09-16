package vault

import "github.com/hashicorp/vault/api"

var DefaultConfig *api.Config

func Client(token string) (*api.Client, error) {
	client, err := api.NewClient(DefaultConfig)
	if err != nil {
		return nil, err
	}
	client.SetToken(token)
	return client, nil
}
