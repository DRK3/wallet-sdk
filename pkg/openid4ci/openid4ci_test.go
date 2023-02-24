/*
Copyright Avast Software. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package openid4ci_test

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/hyperledger/aries-framework-go/pkg/doc/did"
	"github.com/hyperledger/aries-framework-go/pkg/doc/jose"
	arieskms "github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/wallet-sdk/internal/testutil"
	"github.com/trustbloc/wallet-sdk/pkg/api"
	"github.com/trustbloc/wallet-sdk/pkg/localkms"
	"github.com/trustbloc/wallet-sdk/pkg/openid4ci"
)

const (
	sampleTokenResponse = `{"access_token":"eyJhbGciOiJSUzI1NiIsInR5cCI6Ikp..sHQ",` +
		`"token_type":"bearer","expires_in":86400,"c_nonce":"tZignsnFbp","c_nonce_expires_in":86400}`
	mockDID   = "did:test:foo"
	mockKeyID = "did:example:12345#testId"
)

var (
	//go:embed testdata/sample_credential_offer.json
	sampleCredentialOffer []byte

	//go:embed testdata/sample_credential_response.json
	sampleCredentialResponse []byte
)

type mockIssuerServerHandler struct {
	t                                                       *testing.T
	credentialOffer                                         *openid4ci.CredentialOffer
	credentialOfferEndpointShouldFail                       bool
	credentialOfferEndpointShouldGiveUnmarshallableResponse bool
	openIDConfig                                            *openid4ci.OpenIDConfig
	issuerMetadata                                          string
	tokenRequestShouldFail                                  bool
	tokenRequestShouldGiveUnmarshallableResponse            bool
	credentialRequestShouldFail                             bool
	credentialRequestShouldGiveUnmarshallableResponse       bool
	credentialResponse                                      []byte
}

//nolint:gocyclo // test file
func (m *mockIssuerServerHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var err error

	switch request.URL.Path {
	case "/credential-offer":
		switch {
		case m.credentialOfferEndpointShouldFail:
			writer.WriteHeader(http.StatusInternalServerError)
			_, err = writer.Write([]byte("test failure"))
		case m.credentialOfferEndpointShouldGiveUnmarshallableResponse:
			_, err = writer.Write([]byte("invalid"))
		default:
			var credentialOfferBytes []byte

			credentialOfferBytes, err = json.Marshal(m.credentialOffer)
			require.NoError(m.t, err)

			_, err = writer.Write(credentialOfferBytes)
		}
	case "/.well-known/openid-configuration":
		var openIDConfigBytes []byte

		openIDConfigBytes, err = json.Marshal(m.openIDConfig)
		if err != nil {
			break
		}

		_, err = writer.Write(openIDConfigBytes)
	case "/.well-known/openid-credential-issuer":
		_, err = writer.Write([]byte(m.issuerMetadata))
	case "/oidc/token":
		switch {
		case m.tokenRequestShouldFail:
			writer.WriteHeader(http.StatusInternalServerError)
			_, err = writer.Write([]byte("test failure"))
		case m.tokenRequestShouldGiveUnmarshallableResponse:
			_, err = writer.Write([]byte("invalid"))
		default:
			_, err = writer.Write([]byte(sampleTokenResponse))
		}
	case "/credential":
		switch {
		case m.credentialRequestShouldFail:
			writer.WriteHeader(http.StatusInternalServerError)
			_, err = writer.Write([]byte("test failure"))
		case m.credentialRequestShouldGiveUnmarshallableResponse:
			_, err = writer.Write([]byte("invalid"))
		default:
			_, err = writer.Write(m.credentialResponse)
		}
	}

	require.NoError(m.t, err)
}

func TestNewInteraction(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		t.Run("Credential format is jwt_vc_json", func(t *testing.T) {
			newInteraction(t, createCredentialOfferIssuanceURI(t, "example.com"))
		})
		t.Run("Credential format is jwt_vc_json-ld", func(t *testing.T) {
			credentialOffer := createSampleCredentialOffer(t)

			credentialOffer.Credentials[0].Format = "jwt_vc_json-ld"

			credentialOfferBytes, err := json.Marshal(credentialOffer)
			require.NoError(t, err)

			credentialOfferEscaped := url.QueryEscape(string(credentialOfferBytes))

			credentialOfferIssuanceURI := "openid-vc://?credential_offer=" + credentialOfferEscaped

			newInteraction(t, credentialOfferIssuanceURI)
		})
	})
	t.Run("Fail to parse URI", func(t *testing.T) {
		config := getTestClientConfig(t)

		interaction, err := openid4ci.NewInteraction("%", config)
		testutil.RequireErrorContains(t, err, `parse "%": invalid URL escape "%"`)
		require.Nil(t, interaction)
	})
	t.Run("Missing client config", func(t *testing.T) {
		interaction, err := openid4ci.NewInteraction("", nil)
		testutil.RequireErrorContains(t, err, "no client config provided")
		require.Nil(t, interaction)
	})
	t.Run("Missing DID resolver", func(t *testing.T) {
		testConfig := getTestClientConfig(t)

		testConfig.DIDResolver = nil

		interaction, err := openid4ci.NewInteraction("", testConfig)
		testutil.RequireErrorContains(t, err, "no DID resolver provided")
		require.Nil(t, interaction)
	})
	t.Run("Fail to get credential offer", func(t *testing.T) {
		t.Run("Credential offer query parameter missing", func(t *testing.T) {
			interaction, err := openid4ci.NewInteraction("", getTestClientConfig(t))
			require.EqualError(t, err, "INVALID_ISSUANCE_URI(OCI0-0004):credential offer query "+
				"parameter missing from initiate issuance URI")
			require.Nil(t, interaction)
		})
		t.Run("Bad server URL", func(t *testing.T) {
			escapedCredentialOfferURI := url.QueryEscape("BadURL")

			credentialOfferIssuanceURI := "openid-credential-offer://?credential_offer_uri=" + escapedCredentialOfferURI

			interaction, err := openid4ci.NewInteraction(credentialOfferIssuanceURI, getTestClientConfig(t))
			require.EqualError(t, err, "INVALID_CREDENTIAL_OFFER(OCI0-0005):failed to get credential "+
				"offer from the endpoint specified in the credential_offer_uri URL query parameter: "+
				`Get "BadURL": unsupported protocol scheme ""`)
			require.Nil(t, interaction)
		})
		t.Run("Server error", func(t *testing.T) {
			issuerServerHandler := &mockIssuerServerHandler{
				t:                                 t,
				credentialOfferEndpointShouldFail: true,
			}

			server := httptest.NewServer(issuerServerHandler)
			defer server.Close()

			escapedCredentialOfferURI := url.QueryEscape(server.URL + "/credential-offer")

			credentialOfferIssuanceURI := "openid-credential-offer://?credential_offer_uri=" + escapedCredentialOfferURI

			interaction, err := openid4ci.NewInteraction(credentialOfferIssuanceURI, getTestClientConfig(t))
			require.EqualError(t, err, "INVALID_CREDENTIAL_OFFER(OCI0-0005):received status code "+
				"[500] with body [test failure] from the endpoint specified in the credential_offer_uri "+
				"URL query parameter")
			require.Nil(t, interaction)
		})
		t.Run("Fail to unmarshal credential offer", func(t *testing.T) {
			//nolint:gosec // false positive
			credentialOfferIssuanceURI := "openid-credential-offer://?credential_offer="

			interaction, err := openid4ci.NewInteraction(credentialOfferIssuanceURI, getTestClientConfig(t))
			require.EqualError(t, err, "INVALID_CREDENTIAL_OFFER(OCI0-0005):failed to unmarshal "+
				"credential offer JSON into a credential offer object: unexpected end of JSON input")
			require.Nil(t, interaction)
		})
	})
	t.Run("Missing pre-authorized grant type", func(t *testing.T) {
		credentialOffer := openid4ci.CredentialOffer{}

		credentialOfferBytes, err := json.Marshal(credentialOffer)
		require.NoError(t, err)

		credentialOfferEscaped := url.QueryEscape(string(credentialOfferBytes))

		credentialOfferIssuanceURI := "openid-vc://?credential_offer=" + credentialOfferEscaped

		interaction, err := openid4ci.NewInteraction(credentialOfferIssuanceURI, getTestClientConfig(t))
		require.EqualError(t, err, "PRE_AUTHORIZED_GRANT_TYPE_REQUIRED(OCI0-0003):pre-authorized grant "+
			"type is required in the credential offer (support for other grant types not implemented)")
		require.Nil(t, interaction)
	})
	t.Run("Unsupported credential type", func(t *testing.T) {
		credentialOffer := createSampleCredentialOffer(t)

		credentialOffer.Credentials[0].Format = "UnsupportedType"

		credentialOfferBytes, err := json.Marshal(credentialOffer)
		require.NoError(t, err)

		credentialOfferEscaped := url.QueryEscape(string(credentialOfferBytes))

		credentialOfferIssuanceURI := "openid-vc://?credential_offer=" + credentialOfferEscaped

		interaction, err := openid4ci.NewInteraction(credentialOfferIssuanceURI, getTestClientConfig(t))
		require.EqualError(t, err, "UNSUPPORTED_CREDENTIAL_TYPE_IN_OFFER(OCI0-0006):unsupported "+
			"credential type (UnsupportedType) in credential offer at index 0 of credentials object "+
			"(must be jwt_vc_json or jwt_vc_json-ld)")
		require.Nil(t, interaction)
	})
}

func TestInteraction_Authorize(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		interaction := newInteraction(t, createCredentialOfferIssuanceURI(t, "example.com"))

		result, err := interaction.Authorize()
		require.NoError(t, err)
		require.NotNil(t, result)
	})
	t.Run("Interaction not instantiated", func(t *testing.T) {
		interaction := openid4ci.Interaction{}

		result, err := interaction.Authorize()
		require.EqualError(t, err, "interaction not instantiated")
		require.Nil(t, result)
	})
}

type mockResolver struct {
	keyWriter api.KeyWriter
}

func (m *mockResolver) Resolve(string) (*did.DocResolution, error) {
	didDoc, err := makeMockDoc(m.keyWriter)
	if err != nil {
		return nil, err
	}

	return &did.DocResolution{DIDDocument: didDoc}, err
}

func TestInteraction_RequestCredential(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		t.Run("Using credential_offer", func(t *testing.T) {
			issuerServerHandler := &mockIssuerServerHandler{
				t:                  t,
				credentialResponse: sampleCredentialResponse,
			}

			server := httptest.NewServer(issuerServerHandler)
			defer server.Close()

			issuerServerHandler.openIDConfig = &openid4ci.OpenIDConfig{
				TokenEndpoint: fmt.Sprintf("%s/oidc/token", server.URL),
			}

			issuerServerHandler.issuerMetadata = fmt.Sprintf(`{"credential_endpoint":"%s/credential"}`,
				server.URL)

			interaction := newInteraction(t, createCredentialOfferIssuanceURI(t, server.URL))

			credentialRequest := &openid4ci.CredentialRequestOpts{UserPIN: "1234"}

			credentials, err := interaction.RequestCredential(credentialRequest, &jwtSignerMock{
				keyID: mockKeyID,
			})
			require.NoError(t, err)
			require.Len(t, credentials, 1)
			require.NotEmpty(t, credentials[0])
		})
		t.Run("Using credential_offer_uri", func(t *testing.T) {
			issuerServerHandler := &mockIssuerServerHandler{
				t:                  t,
				credentialResponse: sampleCredentialResponse,
			}

			server := httptest.NewServer(issuerServerHandler)
			defer server.Close()

			issuerServerHandler.openIDConfig = &openid4ci.OpenIDConfig{
				TokenEndpoint: fmt.Sprintf("%s/oidc/token", server.URL),
			}

			issuerServerHandler.credentialOffer = createCredentialOffer(t, server.URL)

			issuerServerHandler.issuerMetadata = fmt.Sprintf(`{"credential_endpoint":"%s/credential"}`,
				server.URL)

			escapedCredentialOfferURI := url.QueryEscape(server.URL + "/credential-offer")

			credentialOfferIssuanceURI := "openid-credential-offer://?credential_offer_uri=" + escapedCredentialOfferURI

			interaction := newInteraction(t, credentialOfferIssuanceURI)

			credentialRequest := &openid4ci.CredentialRequestOpts{UserPIN: "1234"}

			credentials, err := interaction.RequestCredential(credentialRequest, &jwtSignerMock{
				keyID: mockKeyID,
			})
			require.NoError(t, err)
			require.Len(t, credentials, 1)
			require.NotEmpty(t, credentials[0])
		})
	})
	t.Run("Invalid user PIN", func(t *testing.T) {
		config := getTestClientConfig(t)

		interaction, err := openid4ci.NewInteraction(createCredentialOfferIssuanceURI(t, "example.com"), config)
		require.NoError(t, err)

		credentialRequest := &openid4ci.CredentialRequestOpts{}

		credentialResponses, err := interaction.RequestCredential(credentialRequest, &jwtSignerMock{
			keyID: mockKeyID,
		})
		testutil.RequireErrorContains(t, err, "invalid user PIN")
		require.Nil(t, credentialResponses)
	})
	t.Run("Fail to fetch issuer's OpenID configuration", func(t *testing.T) {
		requestURI := createCredentialOfferIssuanceURI(t, "BadURL")

		interaction := newInteraction(t, requestURI)

		credentialRequest := &openid4ci.CredentialRequestOpts{UserPIN: "1234"}

		credentialResponses, err := interaction.RequestCredential(credentialRequest, &jwtSignerMock{
			keyID: mockKeyID,
		})
		require.Contains(t, err.Error(), "ISSUER_OPENID_FETCH_FAILED(OCI1-0008):failed to fetch issuer's "+
			`OpenID configuration: Get "BadURL/.well-known/openid-configuration": unsupported protocol scheme ""`)
		require.Nil(t, credentialResponses)
	})
	t.Run("Fail to reach issuer token endpoint", func(t *testing.T) {
		issuerServerHandler := &mockIssuerServerHandler{
			t:            t,
			openIDConfig: &openid4ci.OpenIDConfig{TokenEndpoint: "http://BadURL"},
		}
		server := httptest.NewServer(issuerServerHandler)

		requestURI := createCredentialOfferIssuanceURI(t, server.URL)

		interaction := newInteraction(t, requestURI)

		credentialRequest := &openid4ci.CredentialRequestOpts{UserPIN: "1234"}

		credentialResponses, err := interaction.RequestCredential(credentialRequest, &jwtSignerMock{
			keyID: mockKeyID,
		})
		require.Contains(t, err.Error(), `failed to get token response: Post `+
			`"http://BadURL": dial tcp: lookup BadURL:`)
		require.Nil(t, credentialResponses)
	})
	t.Run("Fail to get token response: server failure", func(t *testing.T) {
		issuerServerHandler := &mockIssuerServerHandler{t: t, tokenRequestShouldFail: true}
		server := httptest.NewServer(issuerServerHandler)
		defer server.Close()

		issuerServerHandler.openIDConfig = &openid4ci.OpenIDConfig{
			TokenEndpoint: fmt.Sprintf("%s/oidc/token", server.URL),
		}

		requestURI := createCredentialOfferIssuanceURI(t, server.URL)

		interaction := newInteraction(t, requestURI)

		credentialRequest := &openid4ci.CredentialRequestOpts{UserPIN: "1234"}

		credentialResponses, err := interaction.RequestCredential(credentialRequest, &jwtSignerMock{
			keyID: mockKeyID,
		})
		testutil.RequireErrorContains(t, err, "failed to get token response: received status code [500] with body "+
			"[test failure] from issuer's token endpoint")
		require.Nil(t, credentialResponses)
	})
	t.Run("Fail to unmarshal response from issuer token endpoint", func(t *testing.T) {
		issuerServerHandler := &mockIssuerServerHandler{t: t, tokenRequestShouldGiveUnmarshallableResponse: true}
		server := httptest.NewServer(issuerServerHandler)
		defer server.Close()

		issuerServerHandler.openIDConfig = &openid4ci.OpenIDConfig{
			TokenEndpoint: fmt.Sprintf("%s/oidc/token", server.URL),
		}

		requestURI := createCredentialOfferIssuanceURI(t, server.URL)

		interaction := newInteraction(t, requestURI)

		credentialRequest := &openid4ci.CredentialRequestOpts{UserPIN: "1234"}

		credentialResponses, err := interaction.RequestCredential(credentialRequest, &jwtSignerMock{
			keyID: mockKeyID,
		})
		testutil.RequireErrorContains(t, err, "failed to get token response: failed to unmarshal response from the "+
			"issuer's token endpoint: invalid character 'i' looking for beginning of value")
		require.Nil(t, credentialResponses)
	})
	t.Run("Fail to get credential response: server failure", func(t *testing.T) {
		issuerServerHandler := &mockIssuerServerHandler{t: t, credentialRequestShouldFail: true}
		server := httptest.NewServer(issuerServerHandler)
		defer server.Close()

		issuerServerHandler.openIDConfig = &openid4ci.OpenIDConfig{
			TokenEndpoint: fmt.Sprintf("%s/oidc/token", server.URL),
		}

		issuerServerHandler.issuerMetadata = fmt.Sprintf(`{"credential_endpoint":"%s/credential"}`, server.URL)

		requestURI := createCredentialOfferIssuanceURI(t, server.URL)

		interaction := newInteraction(t, requestURI)

		credentialRequest := &openid4ci.CredentialRequestOpts{UserPIN: "1234"}

		credentialResponses, err := interaction.RequestCredential(credentialRequest, &jwtSignerMock{
			keyID: mockKeyID,
		})
		testutil.RequireErrorContains(t, err, "failed to get credential response: received status code [500] "+
			"with body [test failure] from issuer's credential endpoint")
		require.Nil(t, credentialResponses)
	})
	t.Run("Fail to get credential response: signature error", func(t *testing.T) {
		issuerServerHandler := &mockIssuerServerHandler{t: t, credentialRequestShouldFail: true}
		server := httptest.NewServer(issuerServerHandler)
		defer server.Close()

		issuerServerHandler.openIDConfig = &openid4ci.OpenIDConfig{
			TokenEndpoint: fmt.Sprintf("%s/oidc/token", server.URL),
		}

		issuerServerHandler.issuerMetadata = fmt.Sprintf(`{"credential_endpoint":"%s/credential"}`, server.URL)

		requestURI := createCredentialOfferIssuanceURI(t, server.URL)

		interaction := newInteraction(t, requestURI)

		credentialRequest := &openid4ci.CredentialRequestOpts{UserPIN: "1234"}

		credentialResponses, err := interaction.RequestCredential(credentialRequest, &jwtSignerMock{
			Err: errors.New("signature error"),
		})
		testutil.RequireErrorContains(t, err, "JWT_SIGNING_FAILED")
		require.Nil(t, credentialResponses)
	})
	t.Run("Fail to reach issuer's credential endpoint", func(t *testing.T) {
		issuerServerHandler := &mockIssuerServerHandler{t: t}
		server := httptest.NewServer(issuerServerHandler)
		defer server.Close()

		issuerServerHandler.openIDConfig = &openid4ci.OpenIDConfig{
			TokenEndpoint: fmt.Sprintf("%s/oidc/token", server.URL),
		}

		issuerServerHandler.issuerMetadata = `{"credential_endpoint":"http://BadURL"}`

		requestURI := createCredentialOfferIssuanceURI(t, server.URL)

		interaction := newInteraction(t, requestURI)

		credentialRequest := &openid4ci.CredentialRequestOpts{UserPIN: "1234"}

		credentialResponses, err := interaction.RequestCredential(credentialRequest, &jwtSignerMock{
			keyID: mockKeyID,
		})
		require.Contains(t, err.Error(), `failed to get credential response: `+
			`Post "http://BadURL": dial tcp: lookup BadURL:`)
		require.Nil(t, credentialResponses)
	})
	t.Run("Fail to get credential response: kid not containing did part", func(t *testing.T) {
		issuerServerHandler := &mockIssuerServerHandler{t: t}
		server := httptest.NewServer(issuerServerHandler)
		defer server.Close()

		issuerServerHandler.openIDConfig = &openid4ci.OpenIDConfig{
			TokenEndpoint: fmt.Sprintf("%s/oidc/token", server.URL),
		}

		requestURI := createCredentialOfferIssuanceURI(t, server.URL)

		interaction := newInteraction(t, requestURI)

		credentialRequest := &openid4ci.CredentialRequestOpts{UserPIN: "1234"}

		credentialResponses, err := interaction.RequestCredential(credentialRequest, &jwtSignerMock{
			keyID: "did:example:12345",
		})
		testutil.RequireErrorContains(t, err, "KEY_ID_NOT_CONTAIN_DID_PART")
		require.Nil(t, credentialResponses)
	})
	t.Run("Fail to unmarshal response from issuer credential endpoint", func(t *testing.T) {
		issuerServerHandler := &mockIssuerServerHandler{t: t, credentialRequestShouldGiveUnmarshallableResponse: true}
		server := httptest.NewServer(issuerServerHandler)
		defer server.Close()

		issuerServerHandler.openIDConfig = &openid4ci.OpenIDConfig{
			TokenEndpoint: fmt.Sprintf("%s/oidc/token", server.URL),
		}

		issuerServerHandler.issuerMetadata = fmt.Sprintf(`{"credential_endpoint":"%s/credential"}`, server.URL)

		requestURI := createCredentialOfferIssuanceURI(t, server.URL)

		interaction := newInteraction(t, requestURI)

		credentialRequest := &openid4ci.CredentialRequestOpts{UserPIN: "1234"}

		credentialResponses, err := interaction.RequestCredential(credentialRequest, &jwtSignerMock{
			keyID: mockKeyID,
		})
		testutil.RequireErrorContains(t, err, "failed to get credential response: failed to unmarshal response "+
			"from the issuer's credential endpoint: invalid character 'i' looking for beginning of value")
		require.Nil(t, credentialResponses)
	})
	t.Run("Fail to parse VC", func(t *testing.T) {
		var credentialResponse openid4ci.CredentialResponse

		credentialResponse.Credential = ""

		credentialResponseBytes, err := json.Marshal(credentialResponse)
		require.NoError(t, err)

		issuerServerHandler := &mockIssuerServerHandler{t: t, credentialResponse: credentialResponseBytes}
		server := httptest.NewServer(issuerServerHandler)
		defer server.Close()

		issuerServerHandler.openIDConfig = &openid4ci.OpenIDConfig{
			TokenEndpoint: fmt.Sprintf("%s/oidc/token", server.URL),
		}

		issuerServerHandler.issuerMetadata = fmt.Sprintf(`{"credential_endpoint":"%s/credential"}`, server.URL)

		requestURI := createCredentialOfferIssuanceURI(t, server.URL)

		interaction := newInteraction(t, requestURI)

		credentialRequest := &openid4ci.CredentialRequestOpts{UserPIN: "1234"}

		vcs, err := interaction.RequestCredential(credentialRequest, &jwtSignerMock{
			keyID: mockKeyID,
		})
		require.EqualError(t, err, "failed to parse credential from credential response at index 0: "+
			"unmarshal new credential: unexpected end of JSON input")
		require.Nil(t, vcs)
	})
	t.Run("Fail VC proof check - public key not found for issuer DID", func(t *testing.T) {
		issuerServerHandler := &mockIssuerServerHandler{t: t, credentialResponse: sampleCredentialResponse}
		server := httptest.NewServer(issuerServerHandler)
		defer server.Close()

		issuerServerHandler.openIDConfig = &openid4ci.OpenIDConfig{
			TokenEndpoint: fmt.Sprintf("%s/oidc/token", server.URL),
		}

		issuerServerHandler.issuerMetadata = fmt.Sprintf(`{"credential_endpoint":"%s/credential"}`, server.URL)

		requestURI := createCredentialOfferIssuanceURI(t, server.URL)

		localKMS, err := localkms.NewLocalKMS(localkms.Config{Storage: localkms.NewMemKMSStore()})
		require.NoError(t, err)

		didResolver := &mockResolver{keyWriter: localKMS}

		config := &openid4ci.ClientConfig{
			ClientID:    "ClientID",
			DIDResolver: didResolver,
		}

		interaction, err := openid4ci.NewInteraction(requestURI, config)
		require.NoError(t, err)
		require.NotNil(t, interaction)

		credentialRequest := &openid4ci.CredentialRequestOpts{UserPIN: "1234"}

		credentials, err := interaction.RequestCredential(credentialRequest, &jwtSignerMock{
			keyID: mockKeyID,
		})
		require.EqualError(t, err, "failed to parse credential from credential response at index 0: "+
			"decode new JWT credential: JWS decoding: unmarshal VC JWT claims: parse JWT: "+
			"parse JWT from compact JWS: public key with KID d3cfd36b-4f75-4041-b416-f0a7a3c6b9f6 is not "+
			"found for DID did:orb:uAAA:EiDpzs0hy0q0If4ZfJA1kxBQd9ed6FoBFhhqDWSiBeKaIg")
		require.Nil(t, credentials)
	})
}

func TestInteraction_IssuerURI(t *testing.T) {
	testIssuerURI := "https://example.com"
	requestURI := createCredentialOfferIssuanceURI(t, testIssuerURI)

	interaction := newInteraction(t, requestURI)

	issuerURI := interaction.IssuerURI()

	require.Equal(t, testIssuerURI, issuerURI)
}

func newInteraction(t *testing.T, requestURI string) *openid4ci.Interaction {
	t.Helper()

	config := getTestClientConfig(t)

	interaction, err := openid4ci.NewInteraction(requestURI, config)
	require.NoError(t, err)
	require.NotNil(t, interaction)

	return interaction
}

func getTestClientConfig(t *testing.T) *openid4ci.ClientConfig {
	t.Helper()

	localKMS, err := localkms.NewLocalKMS(localkms.Config{Storage: localkms.NewMemKMSStore()})
	require.NoError(t, err)

	didResolver := &mockResolver{keyWriter: localKMS}

	return &openid4ci.ClientConfig{
		ClientID:             "ClientID",
		DIDResolver:          didResolver,
		DisableVCProofChecks: true,
	}
}

// makeMockDoc creates a key in the given KMS and returns a mock DID Doc with a verification method.
func makeMockDoc(keyWriter api.KeyWriter) (*did.Doc, error) {
	_, pkJWK, err := keyWriter.Create(arieskms.ED25519Type)
	if err != nil {
		return nil, err
	}

	pkb, err := pkJWK.PublicKeyBytes()
	if err != nil {
		return nil, err
	}

	vm := &did.VerificationMethod{
		ID:         "#key-1",
		Controller: mockDID,
		Type:       "Ed25519VerificationKey2018",
		Value:      pkb,
	}

	newDoc := &did.Doc{
		ID: mockDID,
		AssertionMethod: []did.Verification{
			{
				VerificationMethod: *vm,
				Relationship:       did.AssertionMethod,
			},
		},
		VerificationMethod: []did.VerificationMethod{
			*vm,
		},
	}

	return newDoc, nil
}

type jwtSignerMock struct {
	keyID string
	Err   error
}

func (s *jwtSignerMock) GetKeyID() string {
	return s.keyID
}

func (s *jwtSignerMock) Sign(data []byte) ([]byte, error) {
	return []byte("test signature"), s.Err
}

func (s *jwtSignerMock) Headers() jose.Headers {
	return jose.Headers{
		jose.HeaderKeyID:     "KeyID",
		jose.HeaderAlgorithm: "ES384",
	}
}

func createCredentialOfferIssuanceURI(t *testing.T, issuerURL string) string {
	t.Helper()

	credentialOffer := createCredentialOffer(t, issuerURL)

	credentialOfferBytes, err := json.Marshal(credentialOffer)
	require.NoError(t, err)

	credentialOfferEscaped := url.QueryEscape(string(credentialOfferBytes))

	return "openid-vc://?credential_offer=" + credentialOfferEscaped
}

func createCredentialOffer(t *testing.T, issuerURL string) *openid4ci.CredentialOffer {
	t.Helper()

	credentialOffer := createSampleCredentialOffer(t)

	credentialOffer.CredentialIssuer = issuerURL

	return credentialOffer
}

func createSampleCredentialOffer(t *testing.T) *openid4ci.CredentialOffer {
	t.Helper()

	var credentialOffer openid4ci.CredentialOffer

	err := json.Unmarshal(sampleCredentialOffer, &credentialOffer)
	require.NoError(t, err)

	return &credentialOffer
}
