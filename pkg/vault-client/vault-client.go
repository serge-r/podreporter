package vault_client

import (
	"errors"
	"fmt"
	"github.com/hashicorp/vault/api"
	"net/http"
	"net/url"
	"time"
)

func VaultAuth(vaultAddr url.URL, vaultTimeout time.Duration, vaultSecretID string, vaultRoleID string) (*api.Client, error) {

	httpClient := &http.Client{Timeout: vaultTimeout}

	client, err := api.NewClient(&api.Config{Address: vaultAddr.String(), HttpClient: httpClient})
	if err != nil {
		return nil,err
	}
	response,err := client.Logical().Write("auth/approle/login", map[string]interface{}{
		"role_id":   vaultRoleID,
		"secret_id": vaultSecretID,
	})
	if err != nil {
		return nil,err
	}
	client.SetToken(response.Auth.ClientToken)

	return client, nil
}

func VaultReturnSecret(client *api.Client, secretPath string, secretKey string) ([]byte, error) {
	secret,err := client.Logical().Read(secretPath)
	if err != nil {
		return nil, err
	}
	if secret != nil {
		data, ok := secret.Data["data"].(map[string]interface{})[secretKey]
		if ok {
			return []byte(fmt.Sprintf("%s", data)), nil
		}
		return nil, errors.New("have a problem with return secret - check secret key")
	}
	return nil, errors.New("cannot find secret data check secret path")
}