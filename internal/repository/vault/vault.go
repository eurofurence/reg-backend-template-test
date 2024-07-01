package vault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	auconfigapi "github.com/StephanHCB/go-autumn-config-api"
	auconfigenv "github.com/StephanHCB/go-autumn-config-env"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	aurestclientprometheus "github.com/StephanHCB/go-autumn-restclient-prometheus"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	auresthttpclient "github.com/StephanHCB/go-autumn-restclient/implementation/httpclient"
	aurestlogging "github.com/StephanHCB/go-autumn-restclient/implementation/requestlogging"
	"github.com/go-http-utils/headers"
	"net/http"
	"os"
	"strings"
	"time"
)

type Vault interface {
	Setup(ctx context.Context) error
	Authenticate(ctx context.Context) error
	ObtainSecrets(ctx context.Context) error
}

func New() Vault {
	return &Impl{}
}

const (
	VaultEnabled                 = "VAULT_ENABLED"
	VaultServer                  = "VAULT_SERVER"
	VaultAuthToken               = "VAULT_AUTH_TOKEN"
	VaultAuthKubernetesRole      = "VAULT_AUTH_KUBERNETES_ROLE"
	VaultAuthKubernetesTokenPath = "VAULT_AUTH_KUBERNETES_TOKEN_PATH"
	VaultAuthKubernetesBackend   = "VAULT_AUTH_KUBERNETES_BACKEND"
	VaultSecretsConfig           = "VAULT_SECRETS_CONFIG"
)

func ConfigItems() []auconfigapi.ConfigItem {
	return []auconfigapi.ConfigItem{
		{
			Key:         VaultEnabled,
			Default:     "0",
			Description: "enable Vault/OpenBAO integration. Set to 1 to enable. Defaults to 0.",
			Validate:    auconfigenv.ObtainUintRangeValidator(0, 1),
		}, {
			Key:         VaultServer,
			EnvName:     VaultServer,
			Default:     "",
			Description: "fqdn of the vault server - do not add any other part of the URL",
			Validate:    auconfigenv.ObtainPatternValidator("^(|[a-z0-9.-]+)$"),
		}, {
			Key:         VaultAuthToken,
			Default:     "",
			Description: "authentication token used to fetch secrets. Use this to directly provide a token, for example during local development.",
			Validate:    auconfigapi.ConfigNeedsNoValidation,
		}, {
			Key:         VaultAuthKubernetesRole,
			Default:     "",
			Description: "role binding to use for vault kubernetes authentication.",
			Validate:    auconfigapi.ConfigNeedsNoValidation,
		}, {
			Key:         VaultAuthKubernetesTokenPath,
			Default:     "/var/run/secrets/kubernetes.io/serviceaccount/token",
			Description: "file path to the service-account token",
			Validate:    auconfigapi.ConfigNeedsNoValidation,
		}, {
			Key:         VaultAuthKubernetesBackend,
			Default:     "",
			Description: "authentication path for the kubernetes cluster",
			Validate:    auconfigapi.ConfigNeedsNoValidation,
		}, {
			Key:         VaultSecretsConfig,
			Default:     "{}",
			Description: "configuration consisting of vault paths and keys to fetch from the corresponding path. values will be written to the global configuration object.",
			Validate: func(key string) error {
				value := auconfigenv.Get(key)
				_, err := parseSecretsConfig(value)
				return err
			},
		},
	}
}

type vaultSecretsDef map[string][]vaultSecretDef

type vaultSecretDef struct {
	VaultKey  string  `json:"vaultKey"`
	ConfigKey *string `json:"configKey,omitempty"`
}

func parseSecretsConfig(jsonString string) (vaultSecretsDef, error) {
	secretsConfig := vaultSecretsDef{}
	if err := json.Unmarshal([]byte(jsonString), &secretsConfig); err != nil {
		return nil, err
	}
	return secretsConfig, nil
}

type Impl struct {
	VaultEnabled                 bool
	VaultProtocol                string
	VaultServer                  string
	VaultAuthToken               string
	VaultAuthKubernetesRole      string
	VaultAuthKubernetesTokenPath string
	VaultAuthKubernetesBackend   string
	VaultSecretsConfig           vaultSecretsDef

	VaultClient aurestclientapi.Client
}

func (v *Impl) Setup(ctx context.Context) error {
	aulogging.Logger.Ctx(ctx).Info().Print("setting up vault")

	if v.VaultEnabled || auconfigenv.Get(VaultEnabled) == "1" {
		v.VaultEnabled = true

		v.VaultProtocol = "https"

		if v.VaultServer == "" && auconfigenv.Get(VaultServer) != "" {
			v.VaultServer = auconfigenv.Get(VaultServer)
		}

		if v.VaultSecretsConfig == nil {
			secretsConfig, err := parseSecretsConfig(auconfigenv.Get(VaultSecretsConfig))
			if err != nil {
				return err
			}
			v.VaultSecretsConfig = secretsConfig
		}

		if v.VaultAuthToken != "" || auconfigenv.Get(VaultAuthToken) != "" {
			// user has directly specified a token, completely skip everything related to kubernetes auth
			if v.VaultAuthToken == "" {
				v.VaultAuthToken = auconfigenv.Get(VaultAuthToken)
			}
		} else {
			// assume kubernetes auth
			if v.VaultAuthKubernetesRole == "" {
				v.VaultAuthKubernetesRole = auconfigenv.Get(VaultAuthKubernetesRole)
			}
			if v.VaultAuthKubernetesTokenPath == "" {
				v.VaultAuthKubernetesTokenPath = auconfigenv.Get(VaultAuthKubernetesTokenPath)
			}
			if v.VaultAuthKubernetesBackend == "" {
				v.VaultAuthKubernetesBackend = auconfigenv.Get(VaultAuthKubernetesBackend)
			}
		}
	}

	client, err := auresthttpclient.New(15*time.Second, nil, v.vaultRequestHeaderManipulator())
	if err != nil {
		return err
	}
	aurestclientprometheus.InstrumentHttpClient(client)

	logWrapper := aurestlogging.New(client)

	v.VaultClient = logWrapper
	return nil
}

func (v *Impl) vaultRequestHeaderManipulator() func(ctx context.Context, r *http.Request) {
	return func(ctx context.Context, r *http.Request) {
		r.Header.Set(headers.Accept, aurestclientapi.ContentTypeApplicationJson)
		if v.VaultAuthToken != "" {
			r.Header.Set("X-Vault-Token", v.VaultAuthToken)
		}
	}
}

type K8sAuthRequest struct {
	Jwt  string `json:"jwt"`
	Role string `json:"role"`
}

type K8sAuthResponse struct {
	Auth   *K8sAuth `json:"auth"`
	Errors []string `json:"errors"`
}

type K8sAuth struct {
	ClientToken string `json:"client_token"`
}

func (v *Impl) Authenticate(ctx context.Context) error {
	if v.VaultAuthToken != "" {
		aulogging.Logger.Ctx(ctx).Info().Print("using passed in vault token, skipping authentication with vault")
	} else if v.VaultEnabled {
		aulogging.Logger.Ctx(ctx).Info().Print("authenticating with vault")

		remoteUrl := fmt.Sprintf("%s://%s/v1/auth/%s/login", v.VaultProtocol, v.VaultServer, v.VaultAuthKubernetesBackend)

		k8sToken, err := os.ReadFile(v.VaultAuthKubernetesTokenPath)
		if err != nil {
			return fmt.Errorf("unable to read vault token file from path %s: %s", v.VaultAuthKubernetesTokenPath, err.Error())
		}

		requestDto := &K8sAuthRequest{
			Jwt:  string(k8sToken),
			Role: v.VaultAuthKubernetesRole,
		}

		responseDto := &K8sAuthResponse{}
		response := &aurestclientapi.ParsedResponse{
			Body: responseDto,
		}

		err = v.VaultClient.Perform(ctx, http.MethodPost, remoteUrl, requestDto, response)
		if err != nil {
			return err
		}

		if response.Status != http.StatusOK {
			return errors.New("did not receive http 200 from vault")
		}

		if len(responseDto.Errors) > 0 {
			aulogging.Logger.Ctx(ctx).Warn().WithErr(err).Printf("failed to authenticate with vault: %v", responseDto.Errors)
			return errors.New("got an errors array from vault")
		}

		if responseDto.Auth == nil || responseDto.Auth.ClientToken == "" {
			return errors.New("response from vault did not include a client_token")
		}

		v.VaultAuthToken = responseDto.Auth.ClientToken
	} else {
		aulogging.Logger.Ctx(ctx).Info().Print("vault disabled - skipping")
	}

	return nil
}

type SecretsResponse struct {
	Data   *SecretsResponseData `json:"data"`
	Errors []string             `json:"errors"`
}

type SecretsResponseData struct {
	Data map[string]string `json:"data"`
}

func (v *Impl) ObtainSecrets(ctx context.Context) error {
	if v.VaultEnabled {
		for path, secretsConfig := range v.VaultSecretsConfig {
			secrets, err := v.lowlevelObtainSecrets(ctx, path)
			if err != nil {
				return err
			}
			for _, secretConfig := range secretsConfig {
				vaultKey := secretConfig.VaultKey
				if secret, ok := secrets[vaultKey]; ok {
					configKey := vaultKey
					if secretConfig.ConfigKey != nil && *secretConfig.ConfigKey != "" {
						configKey = *secretConfig.ConfigKey
					}
					if keys := strings.Split(configKey, "."); len(keys) > 1 {
						secretsMap, err := appendSecretToMap(auconfigenv.Get(keys[0]), keys[1], secret)
						if err != nil {
							return fmt.Errorf("nested secret key %s from vault path %s is not valid", configKey, path)
						}
						auconfigenv.Set(keys[0], secretsMap)
					} else {
						auconfigenv.Set(configKey, secret)
					}
				} else {
					return fmt.Errorf("key %s does not exist at vault path %s", vaultKey, path)
				}
			}
		}
	}

	return nil
}

func (v *Impl) lowlevelObtainSecrets(ctx context.Context, fullSecretsPath string) (map[string]string, error) {
	emptyMap := make(map[string]string)

	aulogging.Logger.Ctx(ctx).Info().Printf("querying vault for secrets, secret path %s", fullSecretsPath)

	remoteUrl := fmt.Sprintf("%s://%s/v1/system_kv/data/v1/%s", v.VaultProtocol, v.VaultServer, fullSecretsPath)

	responseDto := &SecretsResponse{}
	response := &aurestclientapi.ParsedResponse{
		Body: responseDto,
	}

	err := v.VaultClient.Perform(ctx, http.MethodGet, remoteUrl, nil, response)
	if err != nil {
		return emptyMap, err
	}

	if response.Status != http.StatusOK {
		return emptyMap, errors.New("did not receive http 200 from vault")
	}

	if len(responseDto.Errors) > 0 {
		aulogging.Logger.Ctx(ctx).Warn().WithErr(err).Printf("failed to obtain secrets from vault: %v", responseDto.Errors)
		return emptyMap, errors.New("got an errors array from vault")
	}

	if responseDto.Data == nil {
		return emptyMap, errors.New("got no top level data structure from vault")
	}
	if responseDto.Data.Data == nil {
		return emptyMap, errors.New("got no second level data structure from vault")
	}

	return responseDto.Data.Data, nil
}

func appendSecretToMap(secretMapJson string, secretKey string, secretValue string) (string, error) {
	secretMap := make(map[string]string)
	if secretMapJson != "" {
		if err := json.Unmarshal([]byte(secretMapJson), &secretMap); err != nil {
			return "{}", err
		}
	}
	secretMap[secretKey] = secretValue
	result, err := json.Marshal(secretMap)
	return string(result), err
}
